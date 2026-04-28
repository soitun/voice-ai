package internal_model

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// Smoke tests for lifecycle/state behavior: interruption, context checks, metrics, listen-loop errors, and close semantics.

func TestModel_Interruption_SupersedesPending(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"}
	e.history.AppendAssistant("ctx-1", testToolAssistantMessage("t1"))

	err := e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-1"})
	require.NoError(t, err)
	require.Equal(t, "ctx-1", e.currentContextID())
	require.Empty(t, comm.pkts)

	ctx, followUp := e.history.FlushToolBlock()
	require.Equal(t, "ctx-1", ctx)
	require.False(t, followUp)
}

func TestModel_ValidateHistorySequence(t *testing.T) {
	e, _, _, _ := newModelTestEnv(t)

	valid := []*protos.Message{
		testToolAssistantMessage("t1"),
		{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{Tools: []*protos.ToolMessage_Tool{{Id: "t1"}}}}},
	}
	require.NoError(t, e.validateHistorySequence(valid))

	missing := []*protos.Message{testToolAssistantMessage("t1")}
	require.ErrorContains(t, e.validateHistorySequence(missing), "not followed")

	orphan := []*protos.Message{{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{Tools: []*protos.ToolMessage_Tool{{Id: "t1"}}}}}}
	require.ErrorContains(t, e.validateHistorySequence(orphan), "orphan")
}

func TestModel_CurrentContextAndStaleCheck(t *testing.T) {
	e, _, _, _ := newModelTestEnv(t)
	require.True(t, e.isStaleResponse("ctx-1"))
	require.Equal(t, "", e.currentContextID())

	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}
	require.False(t, e.isStaleResponse("ctx-1"))
	require.True(t, e.isStaleResponse("ctx-2"))
	require.Equal(t, "ctx-1", e.currentContextID())
}

func TestModel_BuildCompletionMetrics_AddsLatencyMs(t *testing.T) {
	e, _, _, _ := newModelTestEnv(t)
	out := e.buildCompletionMetrics([]*protos.Metric{{Name: "time_to_first_token", Value: "1000000"}, {Name: "token_count", Value: "9"}})
	require.Len(t, out, 3)
	require.Equal(t, "agent_time_to_first_token", out[0].GetName())
	require.Equal(t, "llm_latency_ms", out[1].GetName())
	require.Equal(t, "1", out[1].GetValue())
	require.Equal(t, "agent_token_count", out[2].GetName())
}

func TestModel_Listen_RecvError_EmitsSystemPanic(t *testing.T) {
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	errStream := &listenErrorStream{recvErr: errors.New("stream broke")}
	e := &modelAssistantExecutor{
		logger:        logger,
		history:       NewConversationHistory(),
		stream:        errStream,
		currentPacket: &internal_type.UserInputPacket{ContextID: "ctx-1"},
	}
	comm := &testComm{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.listen(ctx, comm)

	errPkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", errPkt.ContextID)
	require.Equal(t, internal_type.LLMSystemPanic, errPkt.Type)
}

func TestModel_Close_ResetsAndClosesStream(t *testing.T) {
	e, _, stream, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-close"}
	e.history.AppendUser("u")

	require.NoError(t, e.Close(context.Background()))
	require.Nil(t, e.currentPacket)
	require.Nil(t, e.stream)
	require.Len(t, e.history.Snapshot(), 0)
	require.True(t, stream.closeSent)
}

func TestModel_Close_ThenLatePackets_NoCrash(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-close2"}

	require.NoError(t, e.Close(context.Background()))

	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "ctx-close2",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"late"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "token_count", Value: "1"}},
	})
	require.NoError(t, e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-close2", ToolID: "t1", Name: "fn", Result: map[string]string{"ok": "1"},
	}))

	require.Empty(t, findPackets[internal_type.LLMResponseDonePacket](comm.pkts))
}

type listenErrorStream struct {
	recvErr error
}

func (m *listenErrorStream) Send(*protos.ChatRequest) error { return nil }
func (m *listenErrorStream) Recv() (*protos.ChatResponse, error) {
	if m.recvErr != nil {
		err := m.recvErr
		m.recvErr = nil
		return nil, err
	}
	time.Sleep(5 * time.Millisecond)
	return nil, io.EOF
}
func (m *listenErrorStream) CloseSend() error             { return nil }
func (m *listenErrorStream) Header() (metadata.MD, error) { return nil, nil }
func (m *listenErrorStream) Trailer() metadata.MD         { return nil }
func (m *listenErrorStream) Context() context.Context     { return context.Background() }
func (m *listenErrorStream) SendMsg(any) error            { return nil }
func (m *listenErrorStream) RecvMsg(any) error            { return nil }
