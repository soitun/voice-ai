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

// GoogleTraceConfig configures Google Cloud Trace export via OTLP HTTP.
type GoogleTraceConfig = rapida_config.TelemetryGoogleTraceConfig

// GoogleTraceConfigFromOptions parses Google Trace options.
func GoogleTraceConfigFromOptions(opts map[string]interface{}) (GoogleTraceConfig, error) {
	return rapida_config.GoogleTraceTelemetryConfigFromOptions(opts)
}

// NewGoogleTraceExporter creates an OTLP exporter pre-configured for Google Trace.
func NewGoogleTraceExporter(ctx context.Context, cfg GoogleTraceConfig) (*OTLPExporter, error) {
	headers := append([]string{}, cfg.Headers...)
	if cfg.AccessToken != "" {
		headers = append([]string{"Authorization=Bearer " + cfg.AccessToken}, headers...)
	}
	if cfg.APIKey != "" {
		headers = append([]string{"x-goog-api-key=" + cfg.APIKey}, headers...)
	}
	return NewOTLPExporter(ctx, OTLPConfig{
		Endpoint: cfg.Endpoint,
		Protocol: "http/protobuf",
		Headers:  headers,
		Insecure: cfg.Insecure,
	})
}
