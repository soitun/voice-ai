// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observe

import "github.com/rapidaai/pkg/telemetry"

type EventExporter = telemetry.Exporter
type MetricExporter = telemetry.Exporter

// ExporterType enumerates supported telemetry exporter backends.
type ExporterType = telemetry.ExporterType

const (
	OTLP_HTTP     ExporterType = telemetry.OTLP_HTTP
	OTLP_GRPC     ExporterType = telemetry.OTLP_GRPC
	XRAY          ExporterType = telemetry.XRAY
	GOOGLE_TRACE  ExporterType = telemetry.GOOGLE_TRACE
	AZURE_MONITOR ExporterType = telemetry.AZURE_MONITOR
	DATADOG       ExporterType = telemetry.DATADOG
	OPENSEARCH    ExporterType = telemetry.OPENSEARCH
	LOGGING       ExporterType = telemetry.LOGGING
)
