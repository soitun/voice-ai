// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"errors"
	"time"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
)

const (
	PollInterval   = 5 * time.Minute
	OwnershipTTL   = 10 * time.Minute
	OwnerKeyPrefix = "sip:registration:owner:"

	MaxConcurrent       = 10
	MaxTransientRetries = 10

	StatusActive       = "active"
	StatusFailed       = "failed"
	StatusRejected     = "rejected"
	StatusConfigError  = "config_error"
	StatusUnreachable  = "unreachable"
	StatusDisabled     = "disabled"
	OptKeyPhone        = "phone"
	OptKeyCredentialID = "rapida.credential_id"
	OptKeySIPStatus    = "rapida.sip_status"
	OptKeySIPError     = "rapida.sip_error"
	OptKeySIPRetry     = "rapida.sip_retry_count"
	OptKeySIPInbound   = "rapida.sip_inbound"
)

// Record is a single DID-registration work item flowing through the pipeline.
// Outcome is written by stages (claimed/peer/registered/...) so Reconcile can
// emit a single structured tick-summary log instead of N per-record lines.
type Record struct {
	DID          string
	AssistantID  uint64
	DeploymentID uint64
	CredentialID uint64
	Status       string
	Outcome      string
}

// Outcome values written by stages.
const (
	OutcomePeerOwned     = "peer_owned"
	OutcomeAlreadyActive = "already_active"
	OutcomeRegistered    = "registered"
	OutcomeRejected      = "rejected"
	OutcomeAuthFailed    = "auth_failed"
	OutcomeConfigError   = "config_error"
	OutcomeTransient     = "transient"
	OutcomeClaimError    = "claim_error"
)

// Stage is a single step of the per-record pipeline. Returning an error stops
// the chain — each stage is responsible for writing any status it cares about.
type Stage func(ctx context.Context, rec *Record) error

// Config wires the manager's external dependencies. ApplyOpDefaults overlays
// the operational SIP defaults (port, transport, RTP range) onto the per-DID
// vault config and is supplied by the SIP engine.
type Config struct {
	Logger             commons.Logger
	Postgres           connectors.PostgresConnector
	Redis              connectors.RedisConnector
	Vault              web_client.VaultClient
	RegistrationClient *sip_infra.RegistrationClient
	ExternalIP         string
	ApplyOpDefaults    func(*sip_infra.Config)
}

// errPeerOwned signals a record is owned by another instance; pipeline stops silently.
var errPeerOwned = errors.New("did owned by peer instance")
