// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package providers

import (
	"context"

	rapida_config "github.com/rapidaai/config"
)

// AzureMonitorConfig configures Azure Monitor OTLP HTTP export.
type AzureMonitorConfig = rapida_config.TelemetryAzureMonitorConfig

// AzureMonitorConfigFromOptions parses Azure Monitor options.
func AzureMonitorConfigFromOptions(opts map[string]interface{}) (AzureMonitorConfig, error) {
	return rapida_config.AzureMonitorTelemetryConfigFromOptions(opts)
}

// NewAzureMonitorExporter creates an OTLP exporter pre-configured for Azure Monitor.
func NewAzureMonitorExporter(ctx context.Context, cfg AzureMonitorConfig) (*OTLPExporter, error) {
	headers := append([]string{}, cfg.Headers...)
	if cfg.APIKey != "" {
		headers = append([]string{"api-key=" + cfg.APIKey}, headers...)
	}
	return NewOTLPExporter(ctx, OTLPConfig{
		Endpoint: cfg.Endpoint,
		Protocol: "http/protobuf",
		Headers:  headers,
		Insecure: cfg.Insecure,
	})
}
