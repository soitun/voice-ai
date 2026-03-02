// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observe_exporters

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/observe"
	"github.com/rapidaai/pkg/commons"
)

// LoggingExporter logs events and metrics at INFO level.
// It implements both observe.EventExporter and observe.MetricExporter.
type LoggingExporter struct {
	logger commons.Logger
}

func NewLoggingExporter(logger commons.Logger) *LoggingExporter {
	return &LoggingExporter{logger: logger}
}

func (e *LoggingExporter) ExportEvent(_ context.Context, meta observe.SessionMeta, rec observe.EventRecord) error {
	e.logger.Infof("[observe/event] assistant=%d conversation=%d message=%s name=%s data=%v",
		meta.AssistantID, meta.AssistantConversationID, rec.MessageID, rec.Name, rec.Data)
	return nil
}

func (e *LoggingExporter) ExportMetric(_ context.Context, meta observe.SessionMeta, rec observe.MetricRecord) error {
	switch m := rec.(type) {
	case observe.ConversationMetricRecord:
		e.logger.Infof("[observe/metric/conversation] assistant=%d conversation=%s metrics=%v",
			meta.AssistantID, m.ConversationID, m.Metrics)
	case observe.MessageMetricRecord:
		e.logger.Infof("[observe/metric/message] assistant=%d message=%s conversation=%s metrics=%v",
			meta.AssistantID, m.MessageID, m.ConversationID, m.Metrics)
	}
	return nil
}

func (e *LoggingExporter) Shutdown(_ context.Context) error { return nil }
