// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_telemetry_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/utils"
)

// AssistantTelemetryProvider links an assistant to an external observability
// backend. Connection details are stored as key-value option pairs in the
// child AssistantTelemetryProviderOption table (no vault dependency).
//
// ProviderType values:
//
//	"otlp_http"      — OTLP HTTP/protobuf (generic)
//	"otlp_grpc"      — OTLP gRPC (generic)
//	"xray"           — AWS X-Ray via ADOT Collector
//	"google_trace"   — Google Cloud Trace via OTLP
//	"azure_monitor"  — Azure Monitor via OTLP
//	"datadog"        — Datadog APM via OTLP agent
//	"opensearch"     — Rapida OpenSearch events/metrics index
//	"logging"        — Structured log output (debug / audit)
type AssistantTelemetryProvider struct {
	gorm_model.Audited
	gorm_model.Organizational
	AssistantId  uint64 `json:"assistantId"  gorm:"type:bigint;not null"`
	ProviderType string `json:"providerType" gorm:"type:varchar(50);not null"`
	Enabled      bool   `json:"enabled"      gorm:"default:true"`

	Options []*AssistantTelemetryProviderOption `json:"options" gorm:"foreignKey:AssistantTelemetryProviderId"`
}

func (AssistantTelemetryProvider) TableName() string {
	return "assistant_telemetry_providers"
}

// GetOptions returns all provider options as a flat key-value map.
func (p *AssistantTelemetryProvider) GetOptions() utils.Option {
	opts := make(utils.Option, len(p.Options))
	for _, o := range p.Options {
		opts[o.Key] = o.Value
	}
	return opts
}

// AssistantTelemetryProviderOption holds a single configuration key-value pair
// for a telemetry provider (e.g. endpoint, headers, insecure flag).
type AssistantTelemetryProviderOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantTelemetryProviderId uint64 `json:"assistantTelemetryProviderId" gorm:"type:bigint;not null"`
}

func (AssistantTelemetryProviderOption) TableName() string {
	return "assistant_telemetry_provider_options"
}
