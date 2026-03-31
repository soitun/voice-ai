// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package telemetry

import (
	"context"
	"sync"

	"github.com/rapidaai/pkg/commons"
)

// EventCollector fans out EventRecord to all registered exporters.
type EventCollector interface {
	Collect(ctx context.Context, rec EventRecord)
	Shutdown(ctx context.Context)
}

// MetricCollector fans out MetricRecord to all registered exporters.
type MetricCollector interface {
	Collect(ctx context.Context, rec MetricRecord)
	Shutdown(ctx context.Context)
}

// =============================================================================
// Fan-out EventCollector
// =============================================================================

type fanoutEventCollector struct {
	logger    commons.Logger
	meta      SessionMeta
	exporters []Exporter
	wg        sync.WaitGroup
}

// NewEventCollector returns a fan-out EventCollector. When no exporters are
// provided, a no-op implementation is returned to avoid any allocations.
func NewEventCollector(logger commons.Logger, meta SessionMeta, exporters ...Exporter) EventCollector {
	if len(exporters) == 0 {
		return noopEventCollector{}
	}
	return &fanoutEventCollector{logger: logger, meta: meta, exporters: exporters}
}

func (c *fanoutEventCollector) Collect(ctx context.Context, rec EventRecord) {
	for _, exp := range c.exporters {
		exp := exp
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			if err := exp.ExportEvent(ctx, c.meta, rec); err != nil {
				c.logger.Errorf("telemetry: event export error: %v", err)
			}
		}()
	}
}

// Shutdown waits for all in-flight export goroutines to finish, then shuts down
// each exporter in order.
func (c *fanoutEventCollector) Shutdown(ctx context.Context) {
	c.wg.Wait()
	for _, exp := range c.exporters {
		if err := exp.Shutdown(ctx); err != nil {
			c.logger.Errorf("telemetry: event exporter shutdown error: %v", err)
		}
	}
}

// =============================================================================
// Fan-out MetricCollector
// =============================================================================

type fanoutMetricCollector struct {
	logger    commons.Logger
	meta      SessionMeta
	exporters []Exporter
	wg        sync.WaitGroup
}

// NewMetricCollector returns a fan-out MetricCollector. When no exporters are
// provided, a no-op implementation is returned to avoid any allocations.
func NewMetricCollector(logger commons.Logger, meta SessionMeta, exporters ...Exporter) MetricCollector {
	if len(exporters) == 0 {
		return noopMetricCollector{}
	}
	return &fanoutMetricCollector{logger: logger, meta: meta, exporters: exporters}
}

func (c *fanoutMetricCollector) Collect(ctx context.Context, rec MetricRecord) {
	for _, exp := range c.exporters {
		exp := exp
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			if err := exp.ExportMetric(ctx, c.meta, rec); err != nil {
				c.logger.Errorf("telemetry: metric export error: %v", err)
			}
		}()
	}
}

// Shutdown waits for all in-flight export goroutines to finish, then shuts down
// each exporter in order.
func (c *fanoutMetricCollector) Shutdown(ctx context.Context) {
	c.wg.Wait()
	for _, exp := range c.exporters {
		if err := exp.Shutdown(ctx); err != nil {
			c.logger.Errorf("telemetry: metric exporter shutdown error: %v", err)
		}
	}
}

// =============================================================================
// Noop implementations
// =============================================================================

type noopEventCollector struct{}

func (noopEventCollector) Collect(_ context.Context, _ EventRecord) {}
func (noopEventCollector) Shutdown(_ context.Context)               {}

type noopMetricCollector struct{}

func (noopMetricCollector) Collect(_ context.Context, _ MetricRecord) {}
func (noopMetricCollector) Shutdown(_ context.Context)                {}
