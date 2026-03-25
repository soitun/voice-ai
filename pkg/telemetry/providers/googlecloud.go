// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package providers

import (
	"context"
)

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
