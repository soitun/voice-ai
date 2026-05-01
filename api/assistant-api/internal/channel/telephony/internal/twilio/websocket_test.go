// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_twilio_telephony

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWSPair creates a connected WebSocket client/server pair for unit tests.
// The server side is discarded; only the client *websocket.Conn is returned.
// The caller should call cleanup() when done.
func testWSPair(t *testing.T) (clientConn *websocket.Conn, cleanup func()) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Drain messages so write side does not block.
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	}))

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	return conn, func() {
		conn.Close()
		server.Close()
	}
}

// newTestTwilioStreamer creates a twilioWebsocketStreamer with a real WebSocket
// connection (required because Send returns early if connection is nil) but
// without starting the background reader goroutine.
//
// ChannelUUID is intentionally left empty so the Twilio API call block inside
// END_CONVERSATION is skipped, isolating the ToolCallResult + Cancel logic.
func newTestTwilioStreamer(t *testing.T) (*twilioWebsocketStreamer, func()) {
	t.Helper()
	logger, _ := commons.NewApplicationLogger()
	cc := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		ChannelUUID:    "", // empty so the Twilio API block is skipped
	}

	conn, cleanup := testWSPair(t)

	tws := &twilioWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(
			logger, cc, nil,
			internal_telephony_base.WithSourceAudioConfig(internal_audio.NewMulaw8khzMonoAudioConfig()),
		),
		streamID:   "test-stream",
		connection: conn,
	}
	// Note: we do NOT start runWebSocketReader — tests exercise Send only.
	return tws, cleanup
}

func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	tws, cleanup := newTestTwilioStreamer(t)
	defer cleanup()

	toolCall := &protos.ConversationToolCall{
		Id:     "tool-call-id-123",
		ToolId: "tool-id-456",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := tws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-tws.CriticalCh:
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
}

func TestSend_EndConversation_DoesNotCancelStreamer(t *testing.T) {
	tws, cleanup := newTestTwilioStreamer(t)
	defer cleanup()

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-1",
		ToolId: "t-1",
		Name:   "hangup",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	_ = tws.Send(toolCall)

	// Drain the tool call result.
	select {
	case <-tws.CriticalCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	// Context should remain open; disconnect is owned by handleToolResult.
	select {
	case <-tws.Ctx.Done():
		t.Fatal("streamer context should remain open")
	default:
	}
	assert.False(t, tws.closed.Load(), "streamer should remain open")
}

func TestSend_TransferConversation_MissingTarget(t *testing.T) {
	tws, cleanup := newTestTwilioStreamer(t)
	defer cleanup()

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer-missing",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": ""},
	}

	err := tws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-tws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer-missing", result.GetId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "missing target or call ID")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}
}

func TestSend_TransferConversation_NoCallUUID(t *testing.T) {
	tws, cleanup := newTestTwilioStreamer(t)
	defer cleanup()

	// ChannelUUID is already empty from newTestTwilioStreamer
	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer-no-uuid",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": "+15551234567"},
	}

	err := tws.Send(toolCall)
	require.NoError(t, err)

	select {
	case msg := <-tws.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "Expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer-no-uuid", result.GetId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "missing target or call ID")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}
}
