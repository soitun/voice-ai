// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"errors"
	"fmt"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/types"
)

// stageRegister implements the "Register" pipeline step. Skips if the DID is
// already registered by this instance (renewal loop is healthy). On terminal
// failure modes (rejected, auth, config) the stage writes the matching status
// itself and returns the error to halt the chain so stageMarkActive does not
// overwrite it.
func (m *Manager) stageRegister(ctx context.Context, rec *Record) error {
	if m.regClient.IsRegistered(rec.DID) {
		rec.Outcome = OutcomeAlreadyActive
		m.logger.Debugw("SIP DID already registered — renewal loop active",
			"did", rec.DID, "assistant_id", rec.AssistantID)
		return nil
	}

	db := m.postgres.DB(ctx)
	var assistant internal_assistant_entity.Assistant
	if err := db.Where("id = ?", rec.AssistantID).First(&assistant).Error; err != nil {
		rec.Outcome = OutcomeConfigError
		m.logger.Warnw("Failed to load assistant for registration",
			"assistant_id", rec.AssistantID, "did", rec.DID, "error", err)
		m.markStatus(ctx, rec.DeploymentID, StatusConfigError, "assistant not found")
		return fmt.Errorf("load assistant %d: %w", rec.AssistantID, err)
	}

	auth := &types.ProjectScope{
		ProjectId:      &assistant.ProjectId,
		OrganizationId: &assistant.OrganizationId,
	}

	vaultCred, err := m.vault.GetCredential(ctx, auth, rec.CredentialID)
	if err != nil {
		rec.Outcome = OutcomeConfigError
		m.logger.Warnw("Failed to fetch vault credential for registration",
			"assistant_id", rec.AssistantID, "did", rec.DID,
			"credential_id", rec.CredentialID, "error", err)
		m.markStatus(ctx, rec.DeploymentID, StatusConfigError, "vault credential not found")
		return fmt.Errorf("vault credential %d: %w", rec.CredentialID, err)
	}

	sipConfig, err := sip_infra.ParseConfigFromVault(vaultCred)
	if err != nil {
		rec.Outcome = OutcomeConfigError
		m.logger.Warnw("Failed to parse SIP config for registration",
			"assistant_id", rec.AssistantID, "did", rec.DID, "error", err)
		m.markStatus(ctx, rec.DeploymentID, StatusConfigError, "invalid SIP config: "+err.Error())
		return err
	}
	if m.opDefaults != nil {
		m.opDefaults(sipConfig)
	}

	regErr := m.regClient.Register(ctx, &sip_infra.Registration{
		DID:         rec.DID,
		Config:      sipConfig,
		AssistantID: rec.AssistantID,
	})
	if regErr == nil {
		rec.Outcome = OutcomeRegistered
		m.logger.Infow("SIP DID registered",
			"did", rec.DID,
			"assistant_id", rec.AssistantID,
			"server", sipConfig.Server,
			"owner", m.externalIP)
		return nil
	}

	switch {
	case errors.Is(regErr, sip_infra.ErrPermanentFailure):
		rec.Outcome = OutcomeRejected
		m.logger.Errorw("SIP registration permanently rejected — will not retry",
			"did", rec.DID, "assistant_id", rec.AssistantID, "error", regErr)
		m.markStatus(ctx, rec.DeploymentID, StatusRejected, regErr.Error())
	case errors.Is(regErr, sip_infra.ErrAuthFailed):
		rec.Outcome = OutcomeAuthFailed
		m.logger.Errorw("SIP registration auth failed — marking deployment as failed",
			"did", rec.DID, "assistant_id", rec.AssistantID, "error", regErr)
		m.markStatus(ctx, rec.DeploymentID, StatusFailed, regErr.Error())
	default:
		rec.Outcome = OutcomeTransient
		m.handleTransient(ctx, rec, regErr)
	}
	return regErr
}
