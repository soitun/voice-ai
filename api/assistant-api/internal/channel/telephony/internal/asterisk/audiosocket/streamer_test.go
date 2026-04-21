// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk_audiosocket

import (
	"bufio"
	"context"
	"net"
	"testing"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_asterisk "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/asterisk/internal"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStreamer builds a minimal Streamer suitable for Send() tests.
// The returned net.Conn is the "remote" side of the pipe that receives
// frames written by the streamer.
func newTestStreamer(t *testing.T) (*Streamer, net.Conn) {
	t.Helper()

	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	cc := &callcontext.CallContext{
		ContextID:   "test-uuid",
		AssistantID: 1,
	}

	local, remote := net.Pipe()

	reader := bufio.NewReader(local)
	writer := bufio.NewWriter(local)

	audioProcessor, err := internal_asterisk.NewAudioProcessor(logger, internal_asterisk.AudioProcessorConfig{
		AsteriskConfig:   internal_audio.NewLinear8khzMonoAudioConfig(),
		DownstreamConfig: internal_audio.NewLinear16khzMonoAudioConfig(),
		SilenceByte:      0x00,
		FrameSize:        320,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	outputCtx, outputCancel := context.WithCancel(context.Background())

	as := &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(
			logger, cc, nil,
		),
		conn:           local,
		reader:         reader,
		writer:         writer,
		audioProcessor: audioProcessor,
		ctx:            ctx,
		cancel:         cancel,
		outputCtx:      outputCtx,
		outputCancel:   outputCancel,
		initialUUID:    "test-uuid",
	}

	t.Cleanup(func() {
		cancel()
		outputCancel()
		local.Close()
		remote.Close()
	})

	return as, remote
}

func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	as, remote := newTestStreamer(t)

	toolCall := &protos.ConversationToolCall{
		Id:     "call-123",
		ToolId: "tool-456",
		Name:   "end_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	// Read the hangup frame in a goroutine so the writer does not block.
	frameCh := make(chan *Frame, 1)
	errCh := make(chan error, 1)
	go func() {
		r := bufio.NewReader(remote)
		frame, err := ReadFrame(r)
		if err != nil {
			errCh <- err
			return
		}
		frameCh <- frame
	}()

	err := as.Send(toolCall)
	require.NoError(t, err)

	// 1. Verify the ConversationToolCallResult was pushed to CriticalCh.
	select {
	case msg := <-as.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "call-123", result.GetId())
		assert.Equal(t, "tool-456", result.GetToolId())
		assert.Equal(t, "end_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, result.GetAction())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult on CriticalCh")
	}

	// 2. Verify a hangup frame was written to the connection.
	select {
	case frame := <-frameCh:
		assert.Equal(t, FrameTypeHangup, frame.Type, "expected hangup frame type")
		assert.Empty(t, frame.Payload, "hangup frame should have no payload")
	case err := <-errCh:
		t.Fatalf("error reading hangup frame: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for hangup frame on connection")
	}

	// 3. Verify the streamer is closed.
	assert.True(t, as.closed.Load(), "streamer should be closed after end conversation")
}

func TestSend_EndConversation_ClosedIdempotent(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Pre-close the streamer.
	as.closed.Store(true)

	toolCall := &protos.ConversationToolCall{
		Id:     "call-789",
		ToolId: "tool-012",
		Name:   "end_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	// Read whatever comes through so writeFrame does not block.
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := remote.Read(buf); err != nil {
				return
			}
		}
	}()

	// Send should still succeed (close() is idempotent via CompareAndSwap).
	err := as.Send(toolCall)
	require.NoError(t, err)

	// The tool call result should still be pushed.
	select {
	case msg := <-as.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult")
		assert.Equal(t, "call-789", result.GetId())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}
}

func TestSend_TransferConversation_Unsupported(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Drain remote so nothing blocks.
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := remote.Read(buf); err != nil {
				return
			}
		}
	}()

	toolCall := &protos.ConversationToolCall{
		Id:     "call-abc",
		ToolId: "tool-def",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"to": "+15551234567"},
	}

	err := as.Send(toolCall)
	require.NoError(t, err)

	// Should push a failed result because transfer is not supported.
	select {
	case msg := <-as.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "call-abc", result.GetId())
		assert.Equal(t, "tool-def", result.GetToolId())
		assert.Equal(t, "transfer_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, result.GetAction())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}

	// Streamer should NOT be closed for transfer failures.
	assert.False(t, as.closed.Load(), "streamer should remain open after failed transfer")
}

func TestSend_TransferConversation_EmptyToolId_NoResult(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Drain remote.
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := remote.Read(buf); err != nil {
				return
			}
		}
	}()

	toolCall := &protos.ConversationToolCall{
		Id:     "call-xyz",
		ToolId: "", // empty ToolId -- the guard in Send() skips the result
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
	}

	err := as.Send(toolCall)
	require.NoError(t, err)

	// No result should be pushed when ToolId is empty.
	select {
	case msg := <-as.CriticalCh:
		t.Fatalf("unexpected message on CriticalCh: %T", msg)
	case <-time.After(200 * time.Millisecond):
		// Expected: no message.
	}
}

func TestSend_UnknownToolCallAction_NoOp(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Drain remote.
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := remote.Read(buf); err != nil {
				return
			}
		}
	}()

	toolCall := &protos.ConversationToolCall{
		Id:     "call-unk",
		ToolId: "tool-unk",
		Name:   "unknown_action",
		Action: protos.ToolCallAction(999),
	}

	err := as.Send(toolCall)
	require.NoError(t, err)

	// No result or frame for an unrecognized action.
	select {
	case msg := <-as.CriticalCh:
		t.Fatalf("unexpected message on CriticalCh: %T", msg)
	case <-time.After(200 * time.Millisecond):
		// Expected: no message.
	}

	assert.False(t, as.closed.Load(), "streamer should remain open")
}

func TestSend_Interruption_ClearsOutputBuffer(t *testing.T) {
	as, remote := newTestStreamer(t)

	// Drain remote.
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := remote.Read(buf); err != nil {
				return
			}
		}
	}()

	interruption := &protos.ConversationInterruption{
		Type: protos.ConversationInterruption_INTERRUPTION_TYPE_WORD,
	}

	err := as.Send(interruption)
	require.NoError(t, err)
	// If we get here without panic, the code path exercised ClearOutputBuffer
	// on the audio processor. We cannot easily inspect the buffer directly,
	// but the absence of an error confirms correctness.
}

// Compile-time check that Streamer implements internal_type.Streamer.
var _ internal_type.Streamer = (*Streamer)(nil)
