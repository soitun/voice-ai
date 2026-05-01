// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"fmt"
	"sort"
	"strings"

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

	type candidate struct {
		record     Record
		deployment uint64
		assistant  uint64
		sipStatus  string
	}

	grouped := make(map[string][]candidate, len(deployments))
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

		rec := Record{
			DID:          did,
			AssistantID:  dep.AssistantId,
			DeploymentID: dep.Id,
			CredentialID: credentialID,
			Status:       sipStatus,
		}
		key := normalizeDIDForCollision(did)
		grouped[key] = append(grouped[key], candidate{
			record:     rec,
			deployment: dep.Id,
			assistant:  dep.AssistantId,
			sipStatus:  sipStatus,
		})
	}

	selected := make([]Record, 0, len(grouped))
	for didKey, list := range grouped {
		if len(list) == 1 {
			selected = append(selected, list[0].record)
			continue
		}

		// Deterministic winner:
		// 1) keep already active deployment to avoid flapping
		// 2) otherwise latest deployment id
		// 3) finally highest assistant id as stable tie-break
		sort.Slice(list, func(i, j int) bool {
			iActive := list[i].sipStatus == StatusActive
			jActive := list[j].sipStatus == StatusActive
			if iActive != jActive {
				return iActive
			}
			if list[i].deployment != list[j].deployment {
				return list[i].deployment > list[j].deployment
			}
			return list[i].assistant > list[j].assistant
		})

		winner := list[0]
		selected = append(selected, winner.record)

		m.logger.Warnw("Duplicate SIP DID detected; keeping one deployment and dropping others",
			"did", didKey,
			"winner_assistant_id", winner.assistant,
			"winner_deployment_id", winner.deployment,
			"dropped_count", len(list)-1)

		for _, loser := range list[1:] {
			reason := fmt.Sprintf(
				"Duplicate DID %s. Inbound registration skipped: kept assistant=%d deployment=%d",
				didKey, winner.assistant, winner.deployment,
			)
			m.markStatus(ctx, loser.deployment, StatusConfigError, reason)
			m.upsertOption(ctx, loser.deployment, OptKeySIPRetry, "0")
		}
	}

	return selected, nil
}

func normalizeDIDForCollision(did string) string {
	v := strings.TrimSpace(did)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "+") {
		return v
	}
	// Keep short internal extensions unchanged; normalize phone-like values.
	if len(v) > 5 {
		return "+" + v
	}
	return v
}
