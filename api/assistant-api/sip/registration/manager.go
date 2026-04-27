// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/redis/go-redis/v9"
)

// Manager is the SIP registration orchestrator. It runs a periodic reconcile
// loop that drives a typed Pipeline chain:
//
//	ClaimOwnershipPipeline -> RegisterPipeline -> MarkActivePipeline
//
// Distribution across instances is achieved via Redis SETNX on a per-DID key
// whose value is the server's externalIP@hostname identity. Each instance
// only owns the DIDs it successfully claims; peers skip those records.
// Ownership self-heals via TTL.
type Manager struct {
	logger     commons.Logger
	postgres   connectors.PostgresConnector
	redis      *redis.Client
	vault      web_client.VaultClient
	regClient  *sip_infra.RegistrationClient
	externalIP string
	opDefaults func(*sip_infra.Config)
}

// NewManager wires the dependencies and resolves a stable instance identity
// (externalIP@hostname) for the Redis ownership keys. Bare externalIP is not
// enough — two replicas behind a shared LB or with a "0.0.0.0" bind-address
// fallback can collapse to the same value and mistakenly treat each other's
// DIDs as self-owned. Combining with hostname always distinguishes pods.
func NewManager(cfg Config) *Manager {
	hostname, _ := os.Hostname()
	ip := strings.TrimSpace(cfg.ExternalIP)
	if ip == "" && hostname == "" {
		panic("sip_registration: ExternalIP and Hostname both empty — cannot derive instance identity")
	}
	identity := ip + "@" + hostname

	m := &Manager{
		logger:     cfg.Logger,
		postgres:   cfg.Postgres,
		redis:      cfg.Redis.GetConnection(),
		vault:      cfg.Vault,
		regClient:  cfg.RegistrationClient,
		externalIP: identity,
		opDefaults: cfg.ApplyOpDefaults,
	}
	cfg.Logger.Infow("SIP registration manager initialized",
		"instance_id", identity,
		"external_ip", cfg.ExternalIP,
		"poll_interval", PollInterval,
		"ownership_ttl", OwnershipTTL,
		"max_concurrent", MaxConcurrent)
	return m
}

// Start blocks running the periodic reconcile loop until ctx is cancelled.
func (m *Manager) Start(ctx context.Context) {
	m.logger.Infow("SIP registration watcher started", "interval", PollInterval)
	t := time.NewTicker(PollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			m.logger.Infow("SIP registration watcher stopped")
			return
		case <-t.C:
			m.Reconcile(ctx)
		}
	}
}

// Reconcile runs one full pipeline tick: load records, drive each through the
// typed stage chain in bounded parallel, and unregister any locally-active
// DIDs that no longer appear in the desired set.
func (m *Manager) Reconcile(ctx context.Context) {
	tickStart := time.Now()

	records, err := m.loadRecords(ctx)
	if err != nil {
		m.logger.Warnw("Failed to load registration records", "error", err)
		return
	}

	desired := make(map[string]bool, len(records))
	var wg sync.WaitGroup
	sem := make(chan struct{}, MaxConcurrent)

	for i := range records {
		rec := &records[i]
		desired[rec.DID] = true

		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			m.Run(ctx, rec)
		}()
	}
	wg.Wait()

	// Unregister anything we currently hold that the DB no longer wants.
	unregistered := 0
	for _, did := range m.regClient.GetRegisteredDIDs() {
		if desired[did] {
			continue
		}
		m.logger.Infow("Unregistering removed DID", "did", did)
		if err := m.regClient.Unregister(ctx, did); err != nil {
			m.logger.Warnw("Failed to unregister", "did", did, "error", err)
			continue
		}
		m.releaseOwner(ctx, did)
		unregistered++
	}

	counts := map[string]int{}
	for _, r := range records {
		counts[r.Outcome]++
	}
	m.logger.Infow("Registration reconcile complete",
		"loaded", len(records),
		"registered", counts[OutcomeRegistered],
		"already_active", counts[OutcomeAlreadyActive],
		"peer_owned", counts[OutcomePeerOwned],
		"rejected", counts[OutcomeRejected],
		"auth_failed", counts[OutcomeAuthFailed],
		"config_error", counts[OutcomeConfigError],
		"transient", counts[OutcomeTransient],
		"claim_error", counts[OutcomeClaimError],
		"unregistered", unregistered,
		"active_local", m.regClient.ActiveCount(),
		"owner", m.externalIP,
		"duration_ms", time.Since(tickStart).Milliseconds())
}

// ReleaseAll drops every Redis ownership key this instance currently holds so
// peers can claim those DIDs immediately on their next reconcile tick instead
// of waiting OwnershipTTL. Intended for graceful shutdown — call BEFORE
// RegistrationClient.UnregisterAll, since that drains the active-DID set.
func (m *Manager) ReleaseAll(ctx context.Context) {
	dids := m.regClient.GetRegisteredDIDs()
	for _, did := range dids {
		m.releaseOwner(ctx, did)
	}
	m.logger.Infow("SIP registration ownership released",
		"count", len(dids), "owner", m.externalIP)
}

// Run drives the typed Pipeline chain for one Record, starting at
// ClaimOwnershipPipeline. dispatch returns the next Pipeline or nil to stop.
func (m *Manager) Run(ctx context.Context, rec *Record) {
	var next Pipeline = ClaimOwnershipPipeline{Record: rec}
	for next != nil {
		next = m.dispatch(ctx, next)
	}
}

// dispatch routes a typed Pipeline to the matching handler. Mirrors the
// switch-on-type pattern of sip/pipeline/dispatcher.go.
func (m *Manager) dispatch(ctx context.Context, p Pipeline) Pipeline {
	switch v := p.(type) {
	case ClaimOwnershipPipeline:
		return m.handleClaimOwnership(ctx, v)
	case RegisterPipeline:
		return m.handleRegister(ctx, v)
	case MarkActivePipeline:
		return m.handleMarkActive(ctx, v)
	default:
		m.logger.Warnw("dispatch: unknown pipeline type",
			"did", p.DID(), "type", fmt.Sprintf("%T", p))
		return nil
	}
}
