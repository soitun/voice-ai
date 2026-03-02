// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observe

import "context"

// EventExporter writes event records to an external observability backend.
type EventExporter interface {
	ExportEvent(ctx context.Context, meta SessionMeta, rec EventRecord) error
	Shutdown(ctx context.Context) error
}

// MetricExporter writes metric records to an external observability backend.
// Implementations receive a MetricRecord and should type-switch to handle
// ConversationMetricRecord and MessageMetricRecord appropriately.
type MetricExporter interface {
	ExportMetric(ctx context.Context, meta SessionMeta, rec MetricRecord) error
	Shutdown(ctx context.Context) error
}
