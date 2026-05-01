// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_exotel_telephony

import (
	"testing"
	"time"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestExotelStreamer creates an exotelWebsocketStreamer without starting
// the background WebSocket reader goroutine. The connection is nil so Cancel()
// is a no-op on the transport side.
func newTestExotelStreamer(t *testing.T) *exotelWebsocketStreamer {
	t.Helper()
	logger, _ := commons.NewApplicationLogger()
	cc := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		ChannelUUID:    "test-channel-uuid",
	}
	return &exotelWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(
			logger, cc, nil,
			internal_telephony_base.WithSourceAudioConfig(exotelLinear8kConfig),
		),
		streamID:   "test-stream",
		connection: nil, // nil so Cancel() skips conn.Close()
	}
}

func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tool-call-id-123",
		ToolId: "tool-id-456",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := exotel.Send(toolCall)
	require.NoError(t, err)

	// The ToolCallResult should have been pushed to CriticalCh (since Input
	// routes ConversationToolCallResult to CriticalCh).
	select {
	case msg := <-exotel.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tool-call-id-123", result.GetId())
		assert.Equal(t, "tool-id-456", result.GetToolId())
		assert.Equal(t, "end_conversation", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, result.GetAction())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(time.Second):
		t.Fatal("Expected ConversationToolCallResult in CriticalCh but timed out")
	}

	// Context should remain open; disconnect is owned by handleToolResult in adapter layer.
	select {
	case <-exotel.Ctx.Done():
		t.Fatal("streamer context should remain open")
	default:
	}
}

func TestSend_EndConversation_DoesNotCancelStreamerImmediately(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-1",
		ToolId: "t-1",
		Name:   "hangup",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	_ = exotel.Send(toolCall)

	assert.False(t, exotel.closed.Load(), "streamer should remain open")
}

func TestSend_TransferConversation_PushesFailedResult(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer",
		ToolId: "t-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
	}

	err := exotel.Send(toolCall)
	require.NoError(t, err)

	// Transfer not supported for Exotel — should push a failed result.
	select {
	case msg := <-exotel.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer", result.GetId())
		assert.Equal(t, "t-transfer", result.GetToolId())
		assert.Equal(t, "transfer_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, result.GetAction())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported")
	case <-time.After(time.Second):
		t.Fatal("Expected ConversationToolCallResult in CriticalCh but timed out")
	}

	// Streamer should NOT be cancelled for transfer failure.
	select {
	case <-exotel.Ctx.Done():
		t.Fatal("Streamer context should NOT be cancelled on transfer failure")
	default:
		// expected - context is still alive
	}
}

func TestSend_TransferConversation_NoToolId_StillPushesFailedResult(t *testing.T) {
	exotel := newTestExotelStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-no-tool",
		ToolId: "", // empty ToolId
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
	}

	err := exotel.Send(toolCall)
	require.NoError(t, err)

	// Transfer failure should still emit a failed result even when ToolId is empty.
	select {
	case msg := <-exotel.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-no-tool", result.GetId())
		assert.Equal(t, "", result.GetToolId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported")
	case <-time.After(time.Second):
		t.Fatal("Expected ConversationToolCallResult in CriticalCh but timed out")
	}
}
