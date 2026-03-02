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

// NewGoogleTraceExporter creates an OTLPExporter pre-configured for
// Google Cloud Trace via its OTLP HTTP ingestion endpoint.
//
// Required options:
//
//	endpoint — Cloud Trace OTLP endpoint
//	           e.g. "cloudtrace.googleapis.com" (TLS) or a local collector
//
// Optional options:
//
//	access_token  — Bearer token for authentication; stored as "Authorization"
//	                header. Mutually exclusive with api_key.
//	api_key       — API key stored as "x-goog-api-key" header.
//	insecure      — "true" to skip TLS (only for local collectors)
//	headers       — Comma-separated additional "Key=Value" headers
//
// For production use, prefer Application Default Credentials via a local
// OpenTelemetry Collector configured with the Google Cloud exporter plugin.
func NewGoogleTraceExporter(ctx context.Context, opts utils.Option) (*OTLPExporter, error) {
	endpoint, _ := opts["endpoint"].(string)
	if endpoint == "" {
		return nil, fmt.Errorf("observe/google_trace: missing required option 'endpoint'")
	}

	var headers []string

	if token, ok := opts["access_token"].(string); ok && token != "" {
		headers = append(headers, "Authorization=Bearer "+token)
	}
	if apiKey, ok := opts["api_key"].(string); ok && apiKey != "" {
		headers = append(headers, "x-goog-api-key="+apiKey)
	}
	if extra, ok := opts["headers"].(string); ok && extra != "" {
		headers = append(headers, strings.Split(extra, ",")...)
	}

	insecureStr, _ := opts["insecure"].(string)
	insecure := insecureStr == "true" || insecureStr == "1"

	cfg := OTLPConfig{
		Endpoint: endpoint,
		Protocol: "http/protobuf",
		Headers:  headers,
		Insecure: insecure,
	}
	return NewOTLPExporter(ctx, cfg)
}
