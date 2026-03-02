// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observe_exporters

import (
	"context"
	"fmt"

	"github.com/rapidaai/pkg/utils"
)

// NewXRayExporter creates an OTLPExporter pre-configured for AWS X-Ray via the
// AWS Distro for OpenTelemetry (ADOT) Collector.
//
// Required options:
//
//	endpoint — ADOT Collector OTLP endpoint, e.g. "localhost:4318"
//
// Optional options:
//
//	protocol — "http/protobuf" (default) or "grpc"
//	insecure — "true" to skip TLS (typical for a local ADOT sidecar)
//	region   — AWS region tag added as a span attribute (informational only)
//
// The ADOT Collector handles the X-Ray segment conversion and signing before
// forwarding to the AWS X-Ray API — no SigV4 signing is needed here.
func NewXRayExporter(ctx context.Context, opts utils.Option) (*OTLPExporter, error) {
	endpoint, _ := opts["endpoint"].(string)
	if endpoint == "" {
		return nil, fmt.Errorf("observe/xray: missing required option 'endpoint'")
	}

	protocol, _ := opts["protocol"].(string)
	if protocol == "" {
		protocol = "http/protobuf"
	}

	insecureStr, _ := opts["insecure"].(string)
	insecure := insecureStr == "true" || insecureStr == "1"

	cfg := OTLPConfig{
		Endpoint: endpoint,
		Protocol: protocol,
		Insecure: insecure,
	}
	return NewOTLPExporter(ctx, cfg)
}
