package internal_sip_telephony

import (
	"context"
	"testing"
	"time"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSIPStreamer(t *testing.T) *Streamer {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	_, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(logger, &callcontext.CallContext{}, nil),
		cancelParent:          cancel,
	}
}

func TestSend_EndConversation_PushesToolResult(t *testing.T) {
	s := newTestSIPStreamer(t)

	err := s.Send(&protos.ConversationToolCall{
		Id:     "ctx-1",
		ToolId: "tool-1",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	})
	require.NoError(t, err)

	select {
	case msg := <-s.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "ctx-1", result.GetId())
		assert.Equal(t, "tool-1", result.GetToolId())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	// Context should remain open; disconnect is owned by handleToolResult in adapter layer.
	select {
	case <-s.Context().Done():
		t.Fatal("streamer context should remain open; teardown is owned by Talk loop")
	default:
	}
}

func TestSend_ConversationDisconnection_RequeuesToTalkLoop(t *testing.T) {
	s := newTestSIPStreamer(t)

	err := s.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_IDLE_TIMEOUT,
	})
	require.NoError(t, err)

	select {
	case msg := <-s.CriticalCh:
		disc, ok := msg.(*protos.ConversationDisconnection)
		require.True(t, ok, "expected ConversationDisconnection, got %T", msg)
		assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_IDLE_TIMEOUT, disc.GetType())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for requeued ConversationDisconnection")
	}
}

func TestSend_ConversationDisconnection_PreservesExplicitReason(t *testing.T) {
	s := newTestSIPStreamer(t)

	err := s.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_MAX_DURATION,
	})
	require.NoError(t, err)

	select {
	case msg := <-s.CriticalCh:
		disc, ok := msg.(*protos.ConversationDisconnection)
		require.True(t, ok, "expected ConversationDisconnection, got %T", msg)
		assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_MAX_DURATION, disc.GetType())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for requeued ConversationDisconnection")
	}
}

func TestSend_TransferConversation_UsesTransferToKey(t *testing.T) {
	s := newTestSIPStreamer(t)

	var gotTargets []string
	var gotMessage string
	var gotPostTransferAction string
	s.SetOnTransferInitiated(func(targets []string, message string, postTransferAction string) {
		gotTargets = append([]string(nil), targets...)
		gotMessage = message
		gotPostTransferAction = postTransferAction
	})

	err := s.Send(&protos.ConversationToolCall{
		Id:     "ctx-transfer",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args: map[string]string{
			"transfer_to":          "+15550001111" + commons.SEPARATOR + "sip:agent@example.com",
			"message":              "Please hold while I transfer your call.",
			"post_transfer_action": "end_call",
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"+15550001111", "sip:agent@example.com"}, gotTargets)
	assert.Equal(t, "Please hold while I transfer your call.", gotMessage)
	assert.Equal(t, "end_call", gotPostTransferAction)
}

func TestSend_TransferConversation_MissingTransferTarget(t *testing.T) {
	s := newTestSIPStreamer(t)

	err := s.Send(&protos.ConversationToolCall{
		Id:     "ctx-transfer-missing",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": ""},
	})
	require.NoError(t, err)

	select {
	case msg := <-s.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "ctx-transfer-missing", result.GetId())
		assert.Equal(t, "tool-transfer", result.GetToolId())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "missing transfer target")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}
}
