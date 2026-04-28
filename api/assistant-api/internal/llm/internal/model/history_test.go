package internal_model

import (
	"testing"

	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

func toolAssistantMessage(ids ...string) *protos.Message {
	toolCalls := make([]*protos.ToolCall, 0, len(ids))
	for _, id := range ids {
		toolCalls = append(toolCalls, &protos.ToolCall{Id: id, Type: "function"})
	}
	return &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			Contents:  []string{"using tools"},
			ToolCalls: toolCalls,
		}},
	}
}

func TestConversationHistory_AppendUserAndInjected(t *testing.T) {
	h := NewConversationHistory()
	h.AppendUser("hello")
	h.AppendInjected("hi")

	snap := h.Snapshot()
	require.Len(t, snap, 2)
	require.Equal(t, "user", snap[0].GetRole())
	require.Equal(t, "hello", snap[0].GetUser().GetContent())
	require.Equal(t, "assistant", snap[1].GetRole())
	require.Equal(t, "hi", snap[1].GetAssistant().GetContents()[0])
}

func TestConversationHistory_AppendAssistantWithoutToolCalls(t *testing.T) {
	h := NewConversationHistory()
	msg := &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"plain"}}}}

	superseded := h.AppendAssistant("ctx-1", msg)
	require.Empty(t, superseded)
	require.Empty(t, h.PendingContextID())

	snap := h.Snapshot()
	require.Len(t, snap, 1)
	require.Equal(t, "plain", snap[0].GetAssistant().GetContents()[0])
}

func TestConversationHistory_AppendAssistantToolCallsWithoutIDs_AppendsDirectly(t *testing.T) {
	h := NewConversationHistory()
	msg := &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{ToolCalls: []*protos.ToolCall{{Type: "function"}}}}}

	superseded := h.AppendAssistant("ctx-1", msg)
	require.Empty(t, superseded)
	require.Empty(t, h.PendingContextID())
	require.Len(t, h.Snapshot(), 1)
}

func TestConversationHistory_SupersedeOnNewToolBlock(t *testing.T) {
	h := NewConversationHistory()
	require.Empty(t, h.AppendAssistant("ctx-1", toolAssistantMessage("t1")))
	superseded := h.AppendAssistant("ctx-2", toolAssistantMessage("t2"))

	require.Equal(t, "ctx-1", superseded)
	require.Equal(t, "ctx-2", h.PendingContextID())
}

func TestConversationHistory_AcceptToolResultRejectsWhenNoPendingBlock(t *testing.T) {
	h := NewConversationHistory()
	accepted, resolved := h.AcceptToolResult("ctx-1", "t1", "weather", `{"ok":true}`)
	require.False(t, accepted)
	require.False(t, resolved)
}

func TestConversationHistory_AcceptToolResultRejectsMismatch(t *testing.T) {
	h := NewConversationHistory()
	h.AppendAssistant("ctx-1", toolAssistantMessage("t1"))

	accepted, resolved := h.AcceptToolResult("ctx-2", "t1", "weather", `{"ok":true}`)
	require.False(t, accepted)
	require.False(t, resolved)

	accepted, resolved = h.AcceptToolResult("ctx-1", "other", "weather", `{"ok":true}`)
	require.False(t, accepted)
	require.False(t, resolved)
}

func TestConversationHistory_ResolveAndFlushToolBlock(t *testing.T) {
	h := NewConversationHistory()
	h.AppendAssistant("ctx-1", toolAssistantMessage("t1", "t2"))

	accepted, resolved := h.AcceptToolResult("ctx-1", "t2", "tool-2", `{"n":2}`)
	require.True(t, accepted)
	require.False(t, resolved)

	accepted, resolved = h.AcceptToolResult("ctx-1", "t1", "tool-1", `{"n":1}`)
	require.True(t, accepted)
	require.True(t, resolved)

	ctx, followUp := h.FlushToolBlock()
	require.Equal(t, "ctx-1", ctx)
	require.True(t, followUp)
	require.Empty(t, h.PendingContextID())

	snap := h.Snapshot()
	require.Len(t, snap, 2)
	require.NotNil(t, snap[0].GetAssistant())
	require.NotNil(t, snap[1].GetTool())
	require.Len(t, snap[1].GetTool().GetTools(), 2)
	require.Equal(t, "t1", snap[1].GetTool().GetTools()[0].GetId())
	require.Equal(t, "t2", snap[1].GetTool().GetTools()[1].GetId())
}

func TestConversationHistory_SupersedeThenFlushDiscards(t *testing.T) {
	h := NewConversationHistory()
	h.AppendAssistant("ctx-1", toolAssistantMessage("t1"))

	superseded := h.SupersedePending()
	require.Equal(t, "ctx-1", superseded)

	ctx, followUp := h.FlushToolBlock()
	require.Equal(t, "ctx-1", ctx)
	require.False(t, followUp)
	require.Len(t, h.Snapshot(), 0)
	require.Empty(t, h.PendingContextID())
}

func TestConversationHistory_ResetClearsEverything(t *testing.T) {
	h := NewConversationHistory()
	h.AppendUser("u")
	h.AppendAssistant("ctx-1", toolAssistantMessage("t1"))

	h.Reset()

	require.Len(t, h.Snapshot(), 0)
	require.Empty(t, h.PendingContextID())
	accepted, resolved := h.AcceptToolResult("ctx-1", "t1", "tool", `{"ok":true}`)
	require.False(t, accepted)
	require.False(t, resolved)
}
