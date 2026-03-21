// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package telemetry

import (
	"time"

	"github.com/rapidaai/protos"
)

// SessionMeta holds stable per-session identifiers passed to every exporter.
type SessionMeta struct {
	AssistantID             uint64
	AssistantConversationID uint64
	ProjectID               uint64
	OrganizationID          uint64
}

// EventRecord represents a named event fired during a voice session.
// MessageID identifies the interaction turn (context ID) in which the event occurred.
type EventRecord struct {
	MessageID string // turn/interaction context ID
	Name      string
	Data      map[string]string
	Time      time.Time
}

// MetricRecord is a sealed interface for typed metrics.
// Implementations: ConversationMetricRecord, MessageMetricRecord.
type MetricRecord interface{ isMetricRecord() }

// ConversationMetricRecord carries metrics scoped to an entire conversation.
type ConversationMetricRecord struct {
	ConversationID string
	Metrics        []*protos.Metric
	Time           time.Time
}

func (ConversationMetricRecord) isMetricRecord() {}

// MessageMetricRecord carries metrics scoped to a single message/turn.
type MessageMetricRecord struct {
	MessageID      string // turn context ID
	ConversationID string // for correlation
	Metrics        []*protos.Metric
	Time           time.Time
}

func (MessageMetricRecord) isMetricRecord() {}
