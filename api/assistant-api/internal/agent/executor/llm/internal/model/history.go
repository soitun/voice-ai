// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_model

import (
	"strings"
	"sync"

	"github.com/rapidaai/protos"
)

type toolBlockState int

const (
	toolBlockNone       toolBlockState = iota
	toolBlockOpen
	toolBlockSuperseded
	toolBlockResolved
)

type ConversationHistory struct {
	mu       sync.RWMutex
	messages []*protos.Message

	toolState     toolBlockState
	toolContextID string
	toolAssistant *protos.Message
	toolExpected  map[string]struct{}
	toolResults   map[string]*protos.ToolMessage_Tool
}

func NewConversationHistory() *ConversationHistory {
	return &ConversationHistory{
		messages: make([]*protos.Message, 0, 64),
	}
}

func (h *ConversationHistory) Snapshot() []*protos.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*protos.Message, len(h.messages))
	copy(out, h.messages)
	return out
}

func (h *ConversationHistory) AppendUser(text string) {
	h.mu.Lock()
	h.messages = append(h.messages, &protos.Message{
		Role:    "user",
		Message: &protos.Message_User{User: &protos.UserMessage{Content: text}},
	})
	h.mu.Unlock()
}

// AppendAssistant commits a plain assistant message directly. If tool_calls
// are present, the message is held in a pending block until all results
// arrive via AcceptToolResult + FlushToolBlock.
// Returns the contextID of any open block that was superseded (empty if none).
func (h *ConversationHistory) AppendAssistant(contextID string, msg *protos.Message) (supersededCtx string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	toolCalls := msg.GetAssistant().GetToolCalls()
	if len(toolCalls) == 0 {
		h.messages = append(h.messages, msg)
		return ""
	}

	if h.toolState == toolBlockOpen {
		supersededCtx = h.toolContextID
	}

	expected := make(map[string]struct{}, len(toolCalls))
	for _, tc := range toolCalls {
		if id := strings.TrimSpace(tc.GetId()); id != "" {
			expected[id] = struct{}{}
		}
	}
	if len(expected) == 0 {
		h.messages = append(h.messages, msg)
		return supersededCtx
	}

	h.toolState = toolBlockOpen
	h.toolContextID = contextID
	h.toolAssistant = msg
	h.toolExpected = expected
	h.toolResults = make(map[string]*protos.ToolMessage_Tool, len(expected))
	return supersededCtx
}

func (h *ConversationHistory) AppendInjected(text string) {
	h.mu.Lock()
	h.messages = append(h.messages, &protos.Message{
		Role:    "assistant",
		Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{text}}},
	})
	h.mu.Unlock()
}

// AcceptToolResult stores a tool result into the pending block.
// Returns (accepted, resolved). Rejects if no open block, wrong context,
// unknown tool ID, or block already superseded/resolved.
func (h *ConversationHistory) AcceptToolResult(contextID, toolID, name, content string) (accepted bool, resolved bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.toolState != toolBlockOpen {
		return false, false
	}
	if h.toolContextID != contextID {
		return false, false
	}
	if _, ok := h.toolExpected[toolID]; !ok {
		return false, false
	}
	h.toolResults[toolID] = &protos.ToolMessage_Tool{Name: name, Id: toolID, Content: content}
	if len(h.toolResults) == len(h.toolExpected) {
		h.toolState = toolBlockResolved
		return true, true
	}
	return true, false
}

// FlushToolBlock commits a resolved block to history or discards a
// superseded one. Returns (contextID, followUp). Only valid after
// AcceptToolResult returns resolved=true or after SupersedePending.
func (h *ConversationHistory) FlushToolBlock() (contextID string, followUp bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.toolState != toolBlockResolved && h.toolState != toolBlockSuperseded {
		return "", false
	}
	ctx := h.toolContextID

	if h.toolState == toolBlockSuperseded {
		h.clearToolBlock()
		return ctx, false
	}

	tools := make([]*protos.ToolMessage_Tool, 0, len(h.toolExpected))
	for _, tc := range h.toolAssistant.GetAssistant().GetToolCalls() {
		if t, ok := h.toolResults[tc.GetId()]; ok {
			tools = append(tools, t)
		}
	}
	h.messages = append(h.messages,
		h.toolAssistant,
		&protos.Message{
			Role:    "tool",
			Message: &protos.Message_Tool{Tool: &protos.ToolMessage{Tools: tools}},
		},
	)
	h.clearToolBlock()
	return ctx, true
}

func (h *ConversationHistory) SupersedePending() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.toolState != toolBlockOpen {
		return ""
	}
	h.toolState = toolBlockSuperseded
	return h.toolContextID
}

func (h *ConversationHistory) PendingContextID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.toolState == toolBlockNone {
		return ""
	}
	return h.toolContextID
}

func (h *ConversationHistory) Reset() {
	h.mu.Lock()
	h.messages = h.messages[:0]
	h.clearToolBlock()
	h.mu.Unlock()
}

func (h *ConversationHistory) clearToolBlock() {
	h.toolState = toolBlockNone
	h.toolContextID = ""
	h.toolAssistant = nil
	h.toolExpected = nil
	h.toolResults = nil
}
