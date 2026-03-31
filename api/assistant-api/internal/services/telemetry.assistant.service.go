// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_services

import (
	"context"

	internal_telemetry_entity "github.com/rapidaai/api/assistant-api/internal/entity/telemetry"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type AssistantTelemetryProviderService interface {
	Get(
		ctx context.Context,
		auth types.SimplePrinciple,
		telemetryProviderId uint64,
		assistantId uint64,
	) (*internal_telemetry_entity.AssistantTelemetryProvider, error)

	GetAll(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*protos.Criteria,
		paginate *protos.Paginate,
	) (int64, []*internal_telemetry_entity.AssistantTelemetryProvider, error)

	Create(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		providerType string,
		enabled bool,
		options []*protos.Metadata,
	) (*internal_telemetry_entity.AssistantTelemetryProvider, error)

	Update(
		ctx context.Context,
		auth types.SimplePrinciple,
		telemetryProviderId uint64,
		assistantId uint64,
		providerType string,
		enabled bool,
		options []*protos.Metadata,
	) (*internal_telemetry_entity.AssistantTelemetryProvider, error)

	Delete(
		ctx context.Context,
		auth types.SimplePrinciple,
		telemetryProviderId uint64,
		assistantId uint64,
	) (*internal_telemetry_entity.AssistantTelemetryProvider, error)
}
