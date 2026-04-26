// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package variable centralizes resolution of templated variables used in
// tool argument mapping, observe webhook payloads, and agent prompt building.
//
// Two consumer shapes share one data model and namespace registry:
//   - Apply(mapping, src, ctx)  : flat lookup, tool/observe callers
//   - Expand(src, ctx)          : nested map, agent prompt builder
package variable

import "time"

// AssistantInfo is the subset of assistant fields exposed to templates.
type AssistantInfo struct {
	ID          uint64
	VersionID   uint64
	Name        string
	Language    string
	Description string
}

// ConversationInfo is the subset of conversation fields exposed to templates.
type ConversationInfo struct {
	ID          uint64
	Identifier  string
	Source      string
	Direction   string
	CreatedDate time.Time
}

// HistoryEntry is a simplified message for templated payloads.
type HistoryEntry struct {
	Role    string
	Content string
}

// Source is the data provider behind every variable lookup. Adapters wrap a
// live Communication or a frozen ConversationSnapshot.
type Source interface {
	Assistant() *AssistantInfo
	Conversation() *ConversationInfo
	Histories() []HistoryEntry
	Arguments() map[string]any
	Metadata() map[string]any
	Options() map[string]any
	Mode() string
	SessionSource() string
	Now() time.Time
}
