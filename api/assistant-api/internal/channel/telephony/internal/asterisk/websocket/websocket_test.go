// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk_websocket

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStreamer creates a minimal asteriskWebsocketStreamer for unit testing.
// It has no real WebSocket connection and no AudioProcessor, so transport-level
// side effects (sendCommand, audio processing) are safely no-ops.
func newTestStreamer(t *testing.T) *asteriskWebsocketStreamer {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	return &asteriskWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.BaseTelephonyStreamer{
			BaseStreamer: channel_base.NewBaseStreamer(logger),
		},
		// connection is nil — sendCommand returns nil, Cancel skips close
		// audioProcessor is nil — stopAudioProcessing is a no-op (audioCancel is nil)
	}
}

func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	aws := newTestStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-123",
		ToolId: "tool-456",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := aws.Send(toolCall)
	require.NoError(t, err)

	// The ToolCallResult should be routed to CriticalCh by Input().
	select {
	case msg := <-aws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-123", result.GetId())
		assert.Equal(t, "tool-456", result.GetToolId())
		assert.Equal(t, "end_conversation", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, result.GetAction())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}
}

func TestSend_EndConversation_DoesNotCancelStreamer(t *testing.T) {
	aws := newTestStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-789",
		ToolId: "tool-001",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := aws.Send(toolCall)
	require.NoError(t, err)

	// Drain the tool call result.
	select {
	case <-aws.CriticalCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	// Context should remain open; disconnect is owned by handleToolResult in adapter layer.
	select {
	case <-aws.Context().Done():
		t.Fatal("streamer context should remain open after end-conversation")
	default:
	}
}

func TestSend_TransferConversation_MissingTarget(t *testing.T) {
	aws := newTestStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer-1",
		ToolId: "tool-transfer",
		Name:   "transfer_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": ""}, // empty target
	}

	err := aws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-aws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer-1", result.GetId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "missing target or channel name")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}
}

func TestSend_TextAssistantMessage_NoError(t *testing.T) {
	aws := newTestStreamer(t)

	// A text assistant message is a no-op in the switch (only Audio is handled).
	// Verify it returns nil without panicking.
	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Text{Text: "hello"},
	}

	err := aws.Send(msg)
	assert.NoError(t, err)
}

func TestSend_UnhandledType_NoError(t *testing.T) {
	aws := newTestStreamer(t)

	// An unrecognised message type (e.g. ConversationEvent) falls through the
	// switch and should return nil without error.
	msg := &protos.ConversationEvent{Name: "test"}

	err := aws.Send(msg)
	assert.NoError(t, err)
}

func TestDisconnectTypeFromReadError(t *testing.T) {
	assert.Equal(t,
		protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED,
		disconnectTypeFromReadError(nil),
	)

	assert.Equal(t,
		protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
		disconnectTypeFromReadError(io.EOF),
	)

	assert.Equal(t,
		protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
		disconnectTypeFromReadError(&websocket.CloseError{Code: websocket.CloseNormalClosure}),
	)

	assert.Equal(t,
		protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED,
		disconnectTypeFromReadError(errors.New("read failed")),
	)
}
