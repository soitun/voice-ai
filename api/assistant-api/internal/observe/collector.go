// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observe

import (
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/telemetry"
)

type EventCollector = telemetry.EventCollector
type MetricCollector = telemetry.MetricCollector

// NewEventCollector returns a fan-out EventCollector. When no exporters are
// provided, a no-op implementation is returned to avoid any allocations.
func NewEventCollector(logger commons.Logger, meta SessionMeta, exporters ...EventExporter) EventCollector {
	return telemetry.NewEventCollector(logger, meta, exporters...)
}

// NewMetricCollector returns a fan-out MetricCollector. When no exporters are
// provided, a no-op implementation is returned to avoid any allocations.
func NewMetricCollector(logger commons.Logger, meta SessionMeta, exporters ...MetricExporter) MetricCollector {
	return telemetry.NewMetricCollector(logger, meta, exporters...)
}
