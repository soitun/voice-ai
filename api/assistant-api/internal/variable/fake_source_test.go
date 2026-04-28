// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import "time"

// FakeSource is a deterministic Source for unit tests.
type FakeSource struct {
	AssistantInfo    *AssistantInfo
	ConversationInfo *ConversationInfo
	HistoryEntries   []HistoryEntry
	Args             map[string]any
	Meta             map[string]any
	Opts             map[string]any
	ModeValue        string
	SourceValue      string
	NowValue         time.Time
}

func (f *FakeSource) Assistant() *AssistantInfo       { return f.AssistantInfo }
func (f *FakeSource) Conversation() *ConversationInfo { return f.ConversationInfo }
func (f *FakeSource) Histories() []HistoryEntry       { return f.HistoryEntries }
func (f *FakeSource) Arguments() map[string]any       { return f.Args }
func (f *FakeSource) Metadata() map[string]any        { return f.Meta }
func (f *FakeSource) Options() map[string]any         { return f.Opts }
func (f *FakeSource) Mode() string                    { return f.ModeValue }
func (f *FakeSource) SessionSource() string           { return f.SourceValue }
func (f *FakeSource) Now() time.Time                  { return f.NowValue }

// fixedTime returns a stable UTC instant used across the test suite.
func fixedTime() time.Time {
	return time.Date(2026, 4, 26, 12, 30, 45, 0, time.UTC)
}

// newFixtureSource builds a Source preloaded with representative data so
// individual tests stay short.
func newFixtureSource() *FakeSource {
	return &FakeSource{
		AssistantInfo: &AssistantInfo{
			ID:          42,
			VersionID:   7,
			Name:        "Sage",
			Language:    "english",
			Description: "test assistant",
		},
		ConversationInfo: &ConversationInfo{
			ID:          100,
			Identifier:  "conv-abc",
			Source:      "phone",
			Direction:   "inbound",
			CreatedDate: fixedTime().Add(-2 * time.Minute),
		},
		HistoryEntries: []HistoryEntry{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
		Args: map[string]any{"foo": "bar", "count": 3},
		Meta: map[string]any{
			"client.direction":          "outbound",
			"client.telephony_provider": "sip",
			"client.phone":              "6001",
			"analysis.summary":          "ok",
			"loose":                     "value",
		},
		Opts:        map[string]any{"max_tokens": "1024"},
		ModeValue:   "audio",
		SourceValue: "phone",
		NowValue:    fixedTime(),
	}
}
