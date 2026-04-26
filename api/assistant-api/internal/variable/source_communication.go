// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import (
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// CommunicationSource adapts a live internal_type.Communication to the
// Source interface used by the variable resolver.
type CommunicationSource struct {
	c   internal_type.Communication
	now func() time.Time
}

// NewCommunicationSource wraps a Communication. now is optional; when nil
// it defaults to time.Now (the production path).
func NewCommunicationSource(c internal_type.Communication) *CommunicationSource {
	return &CommunicationSource{c: c, now: time.Now}
}

// WithClock overrides the clock for tests.
func (s *CommunicationSource) WithClock(now func() time.Time) *CommunicationSource {
	if now != nil {
		s.now = now
	}
	return s
}

func (s *CommunicationSource) Assistant() *AssistantInfo {
	a := s.c.Assistant()
	if a == nil {
		return nil
	}
	return &AssistantInfo{
		ID:          a.Id,
		VersionID:   a.AssistantProviderId,
		Name:        a.Name,
		Language:    a.Language,
		Description: a.Description,
	}
}

func (s *CommunicationSource) Conversation() *ConversationInfo {
	c := s.c.Conversation()
	if c == nil {
		return nil
	}
	return &ConversationInfo{
		ID:          c.Id,
		Identifier:  c.Identifier,
		Source:      string(c.Source),
		Direction:   c.Direction.String(),
		CreatedDate: time.Time(c.CreatedDate),
	}
}

func (s *CommunicationSource) Histories() []HistoryEntry {
	msgs := s.c.GetHistories()
	out := make([]HistoryEntry, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, HistoryEntry{Role: m.Role(), Content: m.Content()})
	}
	return out
}

func (s *CommunicationSource) Arguments() map[string]any {
	return s.c.GetArgs()
}

func (s *CommunicationSource) Metadata() map[string]any {
	return s.c.GetMetadata()
}

func (s *CommunicationSource) Options() map[string]any {
	return s.c.GetOptions()
}

func (s *CommunicationSource) Mode() string {
	return s.c.GetMode().String()
}

func (s *CommunicationSource) SessionSource() string {
	return string(s.c.GetSource())
}

func (s *CommunicationSource) Now() time.Time {
	return s.now()
}
