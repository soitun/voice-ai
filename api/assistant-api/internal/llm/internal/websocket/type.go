// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_websocket

import "encoding/json"

// =============================================================================
// Message Types
// =============================================================================
//
// Communication Flow:
//   Initialize     → Connection established
//   Configuration  → Session ready
//   UserMessage    → Server processes (user can send multiple)
//   Stream         → Chunk response (one at a time)
//   Complete       → Final response with metrics
//   ToolCall       → Server requests action (disconnect, etc)
//   Interruption   → User interrupted response
//   Close          → End session
//
// =============================================================================

type MessageType string

const (
	// Client → Server
	TypeConfiguration MessageType = "configuration"
	TypeUserMessage   MessageType = "user_message"

	// Server → Client (sequential - one response at a time)
	TypeStream       MessageType = "stream"       // Streaming chunk
	TypeComplete     MessageType = "complete"     // Response complete with metrics
	TypeToolCall     MessageType = "tool_call"    // Server requests action
	TypeInterruption MessageType = "interruption" // User interrupted
	TypeError        MessageType = "error"
	TypeClose        MessageType = "close"

	// Bidirectional
	TypePing MessageType = "ping"
	TypePong MessageType = "pong"
)

// =============================================================================
// Message Envelope
// =============================================================================

type Request struct {
	Type      MessageType `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      any         `json:"data,omitempty"`
}

type Response struct {
	Type    MessageType     `json:"type"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *ErrorData      `json:"error,omitempty"`
}

// =============================================================================
// Client → Server
// =============================================================================

type ConfigurationData struct {
	AssistantID    uint64         `json:"assistant_id"`
	ConversationID uint64         `json:"conversation_id"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type UserMessageData struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// =============================================================================
// Server → Client
// =============================================================================

// StreamData - streaming text chunk
type StreamData struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Index   int    `json:"index"`
}

// CompleteData - response complete with full content and metrics
type CompleteData struct {
	ID      string        `json:"id"`
	Content string        `json:"content"`
	Metrics []*MetricData `json:"metrics,omitempty"`
}

// ToolCallData - server requests an action
type ToolCallData struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"` // "disconnect", "transfer", etc
	Params map[string]any `json:"params,omitempty"`
}

// InterruptionData - user interrupted the response
type InterruptionData struct {
	ID     string `json:"id"`
	Source string `json:"source"` // "word" or "vad"
}

// CloseData - session end
type CloseData struct {
	Reason string `json:"reason"`
	Code   int    `json:"code"`
}

// ErrorData - error info
type ErrorData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MetricData - performance metric
type MetricData struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}
