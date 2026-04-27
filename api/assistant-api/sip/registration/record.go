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
// has SIP inbound enabled and has not entered a terminal status. Records
// whose DID collides with another assistant's are dropped with a WARN — the
// schema does not enforce phone-value uniqueness, so the misconfiguration
// fails loudly (DID unreachable) rather than routing non-deterministically.
func (m *Manager) loadRecords(ctx context.Context) ([]Record, error) {
	var deployments []internal_assistant_entity.AssistantPhoneDeployment
	if err := m.postgres.DB(ctx).
		Preload("TelephonyOption").
		Where("telephony_provider = ? AND status = ?", "sip", type_enums.RECORD_ACTIVE).
		Find(&deployments).Error; err != nil {
		return nil, err
	}

	byDID := make([]Record, 0, len(deployments))
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
		case StatusDisabled, StatusRejected, StatusConfigError:
			continue
		}

		sipInbound, _ := opts.GetString(OptKeySIPInbound)
		if sipInbound != "true" {
			continue
		}

		byDID = append(byDID, Record{
			DID:          did,
			AssistantID:  dep.AssistantId,
			DeploymentID: dep.Id,
			CredentialID: credentialID,
			Status:       sipStatus,
		})
	}

	return byDID, nil
}
