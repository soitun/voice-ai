// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observe

import (
	"time"

	"github.com/rapidaai/api/assistant-api/internal/variable"
)

// SnapshotSource adapts a ConversationSnapshot to variable.Source so the
// shared resolver can drive observe webhook payloads.
type SnapshotSource struct {
	snap *ConversationSnapshot
	now  func() time.Time
}

// NewSnapshotSource wraps a snapshot. now is optional; nil defaults to time.Now.
func NewSnapshotSource(snap *ConversationSnapshot) *SnapshotSource {
	return &SnapshotSource{snap: snap, now: time.Now}
}

// WithClock overrides the clock for tests.
func (s *SnapshotSource) WithClock(now func() time.Time) *SnapshotSource {
	if now != nil {
		s.now = now
	}
	return s
}

func (s *SnapshotSource) Assistant() *variable.AssistantInfo {
	a := s.snap.Assistant
	if a == nil {
		return nil
	}
	return &variable.AssistantInfo{
		ID:          a.Id,
		VersionID:   a.AssistantProviderId,
		Name:        a.Name,
		Language:    a.Language,
		Description: a.Description,
	}
}

func (s *SnapshotSource) Conversation() *variable.ConversationInfo {
	c := s.snap.Conversation
	if c == nil {
		return nil
	}
	return &variable.ConversationInfo{
		ID: c.ID,
	}
}

func (s *SnapshotSource) Histories() []variable.HistoryEntry {
	out := make([]variable.HistoryEntry, 0, len(s.snap.Histories))
	for _, m := range s.snap.Histories {
		out = append(out, variable.HistoryEntry{Role: m.Role, Content: m.Content})
	}
	return out
}

func (s *SnapshotSource) Arguments() map[string]any { return s.snap.Arguments }
func (s *SnapshotSource) Metadata() map[string]any  { return s.snap.Metadata }
func (s *SnapshotSource) Options() map[string]any   { return s.snap.Options }
func (s *SnapshotSource) Mode() string              { return "" }
func (s *SnapshotSource) SessionSource() string     { return "" }
func (s *SnapshotSource) Now() time.Time            { return s.now() }
