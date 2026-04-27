// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"errors"
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
// loop that builds a per-DID pipeline of stages:
//
//	GetRecordToRegister -> ClaimOwnership -> Register -> UpdateStatus
//
// Distribution across instances is achieved via Redis SETNX on a per-DID key
// whose value is the server's externalIP. Each instance only owns the DIDs it
// successfully claims; peers skip those records. Ownership self-heals via TTL.
type Manager struct {
	logger     commons.Logger
	postgres   connectors.PostgresConnector
	redis      *redis.Client
	vault      web_client.VaultClient
	regClient  *sip_infra.RegistrationClient
	externalIP string
	opDefaults func(*sip_infra.Config)

	stages []Stage
}

// NewManager wires the dependencies and assembles the default stage pipeline.
func NewManager(cfg Config) *Manager {
	m := &Manager{
		logger:     cfg.Logger,
		postgres:   cfg.Postgres,
		redis:      cfg.Redis.GetConnection(),
		vault:      cfg.Vault,
		regClient:  cfg.RegistrationClient,
		externalIP: resolveInstanceID(cfg.ExternalIP),
		opDefaults: cfg.ApplyOpDefaults,
	}
	m.stages = []Stage{
		m.stageClaimOwnership,
		m.stageRegister,
		m.stageMarkActive,
	}
	cfg.Logger.Infow("SIP registration manager initialized",
		"instance_id", m.externalIP,
		"external_ip", cfg.ExternalIP,
		"poll_interval", PollInterval,
		"ownership_ttl", OwnershipTTL,
		"max_concurrent", MaxConcurrent)
	return m
}

// resolveInstanceID composes a stable per-pod identity for the Redis ownership
// keys. The bare externalIP is not enough — two replicas behind a shared LB,
// or with the bind-address fallback (e.g. "0.0.0.0"), can collapse to the same
// value and mistakenly treat each other's DIDs as self-owned. Combining with
// the hostname always distinguishes pods even when the IPs collide.
func resolveInstanceID(externalIP string) string {
	hostname, _ := os.Hostname()
	ip := strings.TrimSpace(externalIP)
	if ip == "" && hostname == "" {
		panic("sip_registration: ExternalIP and Hostname both empty — cannot derive instance identity")
	}
	return ip + "@" + hostname
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

// Reconcile runs one full pipeline tick: load records, run the per-record
// stage chain in bounded parallel, and unregister any locally-active DIDs
// that no longer appear in the desired set.
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
			m.runPipeline(ctx, rec)
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

	tally := tally(records)
	m.logger.Infow("Registration reconcile complete",
		"loaded", len(records),
		"registered", tally[OutcomeRegistered],
		"already_active", tally[OutcomeAlreadyActive],
		"peer_owned", tally[OutcomePeerOwned],
		"rejected", tally[OutcomeRejected],
		"auth_failed", tally[OutcomeAuthFailed],
		"config_error", tally[OutcomeConfigError],
		"transient", tally[OutcomeTransient],
		"claim_error", tally[OutcomeClaimError],
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

// runPipeline executes the per-record stage chain. Stages stop early on error
// (or on the silent peer-owned skip).
func (m *Manager) runPipeline(ctx context.Context, rec *Record) {
	for _, stage := range m.stages {
		if err := stage(ctx, rec); err != nil {
			if !errors.Is(err, errPeerOwned) {
				m.logger.Debugw("Pipeline stopped",
					"did", rec.DID, "outcome", rec.Outcome, "error", err)
			}
			return
		}
	}
}

// tally counts outcomes across a finished tick.
func tally(records []Record) map[string]int {
	out := map[string]int{}
	for _, r := range records {
		out[r.Outcome]++
	}
	return out
}
