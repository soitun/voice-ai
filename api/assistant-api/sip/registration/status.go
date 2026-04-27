// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"strconv"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"gorm.io/gorm/clause"
)

// stageMarkActive implements the "Update status" pipeline step on the success
// path: clears any prior error and resets the transient retry counter.
// Skips DB writes if the DID was already active locally (renewal loop carries
// the binding) — avoids one upsert tuple per tick per DID at scale.
func (m *Manager) stageMarkActive(ctx context.Context, rec *Record) error {
	if rec.Outcome == OutcomeAlreadyActive {
		return nil
	}
	m.upsertOption(ctx, rec.DeploymentID, OptKeySIPStatus, StatusActive)
	m.upsertOption(ctx, rec.DeploymentID, OptKeySIPError, "")
	m.upsertOption(ctx, rec.DeploymentID, OptKeySIPRetry, "0")
	return nil
}

// markStatus writes a (sip_status, sip_error) pair. Used by stages that
// detect a terminal/failure condition before stageMarkActive runs.
func (m *Manager) markStatus(ctx context.Context, deploymentID uint64, status, errMsg string) {
	m.upsertOption(ctx, deploymentID, OptKeySIPStatus, status)
	m.upsertOption(ctx, deploymentID, OptKeySIPError, errMsg)
}

// upsertOption mirrors the upsert pattern used by CreatePhoneDeployment so
// existing rows are updated in place rather than duplicated.
func (m *Manager) upsertOption(ctx context.Context, deploymentID uint64, key, value string) {
	db := m.postgres.DB(ctx)
	opt := &internal_assistant_entity.AssistantDeploymentTelephonyOption{
		AssistantDeploymentTelephonyId: deploymentID,
		Metadata: gorm_models.Metadata{
			Key:   key,
			Value: value,
		},
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "assistant_deployment_telephony_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_date"}),
	}).Create(opt).Error; err != nil {
		m.logger.Warnw("Failed to upsert deployment option",
			"deployment_id", deploymentID, "key", key, "error", err)
	}
}

// handleTransient bumps the retry counter for transport / 5xx style errors.
// After MaxTransientRetries the deployment is marked unreachable so subsequent
// reconciles short-circuit it via the terminal-status filter in loadRecords.
func (m *Manager) handleTransient(ctx context.Context, rec *Record, err error) {
	db := m.postgres.DB(ctx)
	var opt internal_assistant_entity.AssistantDeploymentTelephonyOption
	retry := 0
	if dbErr := db.Where("assistant_deployment_telephony_id = ? AND key = ?",
		rec.DeploymentID, OptKeySIPRetry).First(&opt).Error; dbErr == nil {
		retry, _ = strconv.Atoi(opt.Value)
	}
	retry++

	if retry >= MaxTransientRetries {
		m.logger.Errorw("SIP registration unreachable after max retries — will not retry",
			"did", rec.DID, "assistant_id", rec.AssistantID, "retries", retry, "error", err)
		m.markStatus(ctx, rec.DeploymentID, StatusUnreachable, err.Error())
		m.upsertOption(ctx, rec.DeploymentID, OptKeySIPRetry, strconv.Itoa(retry))
		return
	}

	m.logger.Warnw("SIP registration failed (will retry)",
		"did", rec.DID, "assistant_id", rec.AssistantID, "retry", retry, "error", err)
	m.upsertOption(ctx, rec.DeploymentID, OptKeySIPRetry, strconv.Itoa(retry))
}
