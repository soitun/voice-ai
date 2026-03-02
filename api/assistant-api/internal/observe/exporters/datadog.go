// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observe_exporters

import (
	"context"
	"fmt"
	"strings"

	"github.com/rapidaai/pkg/utils"
)

// NewDatadogExporter creates an OTLPExporter pre-configured for Datadog APM.
//
// Datadog accepts OTLP traces on the Datadog Agent (recommended) or directly
// via the Datadog intake API. The agent approach is preferred because it
// handles metadata enrichment, sampling, and environment tagging.
//
// Required options (one of):
//
//	endpoint    — OTLP endpoint of the Datadog Agent, e.g. "localhost:4317" (gRPC)
//	              or "localhost:4318" (HTTP). For the Datadog intake use
//	              "https://trace.agent.datadoghq.com" (or EU: "datadoghq.eu").
//
// Optional options:
//
//	api_key     — Datadog API key stored as "DD-API-KEY" header.
//	              Required when sending directly to the Datadog intake;
//	              not needed when routing through the local agent.
//	protocol    — "grpc" or "http/protobuf" (default: "grpc" for agent,
//	              "http/protobuf" for intake)
//	insecure    — "true" to skip TLS (use with local agent on loopback)
//	headers     — Comma-separated additional "Key=Value" headers
func NewDatadogExporter(ctx context.Context, opts utils.Option) (*OTLPExporter, error) {
	endpoint, _ := opts["endpoint"].(string)
	if endpoint == "" {
		return nil, fmt.Errorf("observe/datadog: missing required option 'endpoint'")
	}

	protocol, _ := opts["protocol"].(string)
	if protocol == "" {
		// Default to gRPC when talking to a local Datadog Agent; HTTP otherwise.
		if strings.HasPrefix(endpoint, "http") {
			protocol = "http/protobuf"
		} else {
			protocol = "grpc"
		}
	}

	var headers []string
	if apiKey, ok := opts["api_key"].(string); ok && apiKey != "" {
		headers = append(headers, "DD-API-KEY="+apiKey)
	}
	if extra, ok := opts["headers"].(string); ok && extra != "" {
		headers = append(headers, strings.Split(extra, ",")...)
	}

	insecureStr, _ := opts["insecure"].(string)
	insecure := insecureStr == "true" || insecureStr == "1"

	cfg := OTLPConfig{
		Endpoint: endpoint,
		Protocol: protocol,
		Headers:  headers,
		Insecure: insecure,
	}
	return NewOTLPExporter(ctx, cfg)
}
