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

// XRayConfig configures AWS X-Ray export via ADOT/OTLP.
type XRayConfig = rapida_config.TelemetryXRayConfig

// XRayConfigFromOptions parses X-Ray options.
func XRayConfigFromOptions(opts map[string]interface{}) (XRayConfig, error) {
	return rapida_config.XRayTelemetryConfigFromOptions(opts)
}

// NewXRayExporter creates an OTLP exporter pre-configured for AWS X-Ray.
func NewXRayExporter(ctx context.Context, cfg XRayConfig) (*OTLPExporter, error) {
	return NewOTLPExporter(ctx, OTLPConfig{
		Endpoint: cfg.Endpoint,
		Protocol: cfg.Protocol,
		Insecure: cfg.Insecure,
	})
}
