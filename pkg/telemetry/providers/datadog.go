// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package providers

import (
	"context"
)

// NewDatadogExporter creates an OTLP exporter pre-configured for Datadog APM.
func NewDatadogExporter(ctx context.Context, cfg DatadogConfig) (*OTLPExporter, error) {
	return NewOTLPExporter(ctx, OTLPConfig{
		Endpoint: cfg.Endpoint,
		Protocol: cfg.Protocol,
		Headers:  cfg.Headers,
		Insecure: cfg.Insecure,
	})
}
