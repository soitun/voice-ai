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

// NewAzureMonitorExporter creates an OTLPExporter pre-configured for
// Azure Monitor (Application Insights) via its OTLP ingestion endpoint.
//
// Required options:
//
//	endpoint — Azure Monitor OTLP endpoint
//	           e.g. "<region>.in.applicationinsights.azure.com"
//	           or the full URL path for a private-link workspace
//
// Optional options:
//
//	api_key  — Instrumentation key / connection string API key,
//	           stored as the "api-key" header required by Azure Monitor.
//	headers  — Comma-separated additional "Key=Value" headers
//	insecure — "true" to skip TLS (only for local collectors / testing)
//
// Recommended: route through an OpenTelemetry Collector with the
// azuremonitor exporter configured for full telemetry enrichment.
func NewAzureMonitorExporter(ctx context.Context, opts utils.Option) (*OTLPExporter, error) {
	endpoint, _ := opts["endpoint"].(string)
	if endpoint == "" {
		return nil, fmt.Errorf("observe/azure_monitor: missing required option 'endpoint'")
	}

	var headers []string

	if apiKey, ok := opts["api_key"].(string); ok && apiKey != "" {
		headers = append(headers, "api-key="+apiKey)
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
