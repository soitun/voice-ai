// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import (
	"fmt"
	"time"
)

// ConversationNamespace exposes the active conversation: id, identifier,
// source, direction, created_date, duration, messages.
type ConversationNamespace struct{}

func (n *ConversationNamespace) Get(suffix string, src Source, _ ResolveContext) (any, bool) {
	v, ok := n.fields(src.Conversation(), src.Histories())[suffix]
	return v, ok
}

func (n *ConversationNamespace) Enumerate(src Source, _ ResolveContext) map[string]any {
	return n.fields(src.Conversation(), src.Histories())
}

func (n *ConversationNamespace) fields(c *ConversationInfo, hist []HistoryEntry) map[string]any {
	out := map[string]any{
		"messages": n.simplifyHistory(hist),
	}
	if c == nil {
		return out
	}
	out["id"] = fmt.Sprintf("%d", c.ID)
	out["identifier"] = c.Identifier
	out["source"] = c.Source
	out["direction"] = c.Direction
	if !c.CreatedDate.IsZero() {
		out["created_date"] = c.CreatedDate.UTC().Format(time.RFC3339)
		out["duration"] = time.Since(c.CreatedDate).Truncate(time.Second).String()
	}
	return out
}

func (n *ConversationNamespace) simplifyHistory(msgs []HistoryEntry) []map[string]string {
	out := make([]map[string]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, map[string]string{"role": m.Role, "message": m.Content})
	}
	return out
}
