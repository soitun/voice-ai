// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

// loadRecords implements the "GetRecordToRegister" pipeline entry point.
// Returns the desired-state Records — only the latest active phone deployment
// per assistant (older versions are archived by CreatePhoneDeployment) that
// has SIP inbound enabled and has not entered a terminal status.
func (m *Manager) loadRecords(ctx context.Context) ([]Record, error) {
	var deployments []internal_assistant_entity.AssistantPhoneDeployment
	if err := m.postgres.DB(ctx).
		Preload("TelephonyOption").
		Where("telephony_provider = ? AND status = ?", "sip", type_enums.RECORD_ACTIVE).
		Find(&deployments).Error; err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(deployments))
	for _, dep := range deployments {
		opts := dep.GetOptions()

		did, _ := opts.GetString(OptKeyPhone)
		if did == "" {
			continue
		}

		credentialID, err := opts.GetUint64(OptKeyCredentialID)
		if err != nil {
			continue
		}

		sipStatus, _ := opts.GetString(OptKeySIPStatus)
		switch sipStatus {
		case StatusFailed, StatusDisabled, StatusRejected, StatusConfigError, StatusUnreachable:
			continue
		}

		sipInbound, _ := opts.GetString(OptKeySIPInbound)
		if sipInbound != "true" {
			continue
		}

		records = append(records, Record{
			DID:          did,
			AssistantID:  dep.AssistantId,
			DeploymentID: dep.Id,
			CredentialID: credentialID,
			Status:       sipStatus,
		})
	}
	return dedupeByDID(m, records), nil
}

// dedupeByDID drops every record whose DID is claimed by more than one
// assistant. Same-assistant duplicates can't happen (CreatePhoneDeployment
// archives prior versions), but the schema does not enforce uniqueness of the
// `phone` option value across assistants. If two assistants both claim the
// same DID, registering either one would route inbound calls non-deterministically.
// We skip the entire collision group so the misconfiguration fails loudly
// (the DID is unreachable until resolved) instead of silently picking a winner.
func dedupeByDID(m *Manager, records []Record) []Record {
	byDID := make(map[string][]int, len(records))
	for i := range records {
		byDID[records[i].DID] = append(byDID[records[i].DID], i)
	}
	out := make([]Record, 0, len(records))
	for _, idx := range byDID {
		if len(idx) > 1 {
			ids := make([]uint64, 0, len(idx))
			for _, i := range idx {
				ids = append(ids, records[i].AssistantID)
			}
			m.logger.Warnw("Duplicate DID across assistants — skipping all (resolve config)",
				"did", records[idx[0]].DID,
				"assistant_ids", ids,
				"count", len(idx))
			continue
		}
		out = append(out, records[idx[0]])
	}
	return out
}
