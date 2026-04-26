package internal_model

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	internal_agent_executor "github.com/rapidaai/api/assistant-api/internal/agent/executor"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

type testStream struct {
	mu        sync.Mutex
	sendCalls []*protos.ChatRequest
}

func (m *testStream) Send(req *protos.ChatRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls = append(m.sendCalls, req)
	return nil
}
func (m *testStream) Recv() (*protos.ChatResponse, error) { return nil, io.EOF }
func (m *testStream) CloseSend() error                    { return nil }
func (m *testStream) Header() (metadata.MD, error)        { return nil, nil }
func (m *testStream) Trailer() metadata.MD                { return nil }
func (m *testStream) Context() context.Context            { return context.Background() }
func (m *testStream) SendMsg(any) error                   { return nil }
func (m *testStream) RecvMsg(any) error                   { return nil }

type testToolExecutor struct {
	internal_agent_executor.ToolExecutor
	calls []struct {
		contextID string
		tools     []*protos.ToolCall
	}
}

func (m *testToolExecutor) Initialize(context.Context, internal_type.Communication) error {
	return nil
}
func (m *testToolExecutor) GetFunctionDefinitions() []*protos.FunctionDefinition { return nil }
func (m *testToolExecutor) ExecuteAll(_ context.Context, contextID string, tools []*protos.ToolCall, _ internal_type.Communication) {
	m.calls = append(m.calls, struct {
		contextID string
		tools     []*protos.ToolCall
	}{contextID: contextID, tools: tools})
}
func (m *testToolExecutor) Close(context.Context) error { return nil }

type testComm struct {
	internal_type.Communication
	assistant    *internal_assistant_entity.Assistant
	conversation *internal_conversation_entity.AssistantConversation
	pkts         []internal_type.Packet
}

func (m *testComm) OnPacket(_ context.Context, pkts ...internal_type.Packet) error {
	m.pkts = append(m.pkts, pkts...)
	return nil
}
func (m *testComm) Assistant() *internal_assistant_entity.Assistant { return m.assistant }
func (m *testComm) Conversation() *internal_conversation_entity.AssistantConversation {
	return m.conversation
}
func (m *testComm) GetArgs() map[string]interface{}     { return map[string]interface{}{} }
func (m *testComm) GetMetadata() map[string]interface{} { return map[string]interface{}{} }
func (m *testComm) GetHistories() []internal_type.MessagePacket {
	return []internal_type.MessagePacket{}
}
func (m *testComm) GetMode() type_enums.MessageMode { return type_enums.TextMode }
func (m *testComm) GetSource() utils.RapidaSource   { return utils.WebPlugin }
func (m *testComm) GetOptions() utils.Option        { return nil }

func newModelTestEnv(t *testing.T) (*modelAssistantExecutor, *testComm, *testStream, *testToolExecutor) {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	stream := &testStream{}
	toolExec := &testToolExecutor{}
	e := &modelAssistantExecutor{
		logger:             logger,
		inputBuilder:       integration_client_builders.NewChatInputBuilder(logger),
		toolExecutor:       toolExec,
		history:            NewConversationHistory(),
		stream:             stream,
		providerCredential: &protos.VaultCredential{Id: 9, Value: &structpb.Struct{}},
	}

	comm := &testComm{
		assistant: &internal_assistant_entity.Assistant{
			Name: "assistant",
			AssistantProviderModel: &internal_assistant_entity.AssistantProviderModel{
				ModelProviderName:     "openai",
				Template:              gorm_types.PromptMap{},
				AssistantModelOptions: []*internal_assistant_entity.AssistantProviderModelOption{},
			},
		},
		conversation: &internal_conversation_entity.AssistantConversation{},
	}
	return e, comm, stream, toolExec
}

func testToolAssistantMessage(ids ...string) *protos.Message {
	calls := make([]*protos.ToolCall, 0, len(ids))
	for _, id := range ids {
		calls = append(calls, &protos.ToolCall{Id: id, Type: "function"})
	}
	return &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{ToolCalls: calls, Contents: []string{"calling tool"}}}}
}

func findPacket[T any](pkts []internal_type.Packet) (T, bool) {
	var zero T
	for _, p := range pkts {
		if v, ok := p.(T); ok {
			return v, true
		}
	}
	return zero, false
}

func TestModel_ExecuteUserTurn_SendsChatAndAppendsUser(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)
	msgs := stream.sendCalls[0].GetConversations()
	require.NotEmpty(t, msgs)
	require.Equal(t, "user", msgs[len(msgs)-1].GetRole())
	require.Equal(t, "hello", msgs[len(msgs)-1].GetUser().GetContent())
	require.Len(t, e.history.Snapshot(), 1)

	evt, ok := findPacket[internal_type.ConversationEventPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "llm", evt.Name)
	require.Equal(t, "executing", evt.Data["type"])
}

func TestModel_ExecuteUserTurn_BlocksInvalidHistory(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.messages = append(e.history.messages, &protos.Message{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{}}})

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 0)

	pkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", pkt.ContextID)
	require.ErrorContains(t, pkt.Error, "history integrity")
}

func TestModel_InjectMessage_AppendsToHistory(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	err := e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{ContextID: "ctx-1", Text: "hello from inject"})
	require.NoError(t, err)
	require.Empty(t, comm.pkts)

	snap := e.history.Snapshot()
	require.Len(t, snap, 1)
	require.Equal(t, "assistant", snap[0].GetRole())
	require.Equal(t, "hello from inject", snap[0].GetAssistant().GetContents()[0])
}

func TestModel_ToolResultIgnored_EmitsEvent(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{ContextID: "ctx-1", ToolID: "t1", Name: "weather", Result: map[string]string{"ok": "1"}})
	require.NoError(t, err)

	evt, ok := findPacket[internal_type.ConversationEventPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "tool", evt.Name)
	require.Equal(t, "tool_result_ignored", evt.Data["type"])
	require.Equal(t, "no_pending_block", evt.Data["reason"])
}

func TestModel_ToolResultResolved_TriggersFollowUp(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.AppendAssistant("ctx-tool", testToolAssistantMessage("t1"))

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{ContextID: "ctx-tool", ToolID: "t1", Name: "weather", Result: map[string]string{"ok": "1"}})
	require.NoError(t, err)
	require.Len(t, stream.sendCalls, 1)
	require.Equal(t, "ctx-tool", stream.sendCalls[0].GetRequestId())

	snap := e.history.Snapshot()
	require.Len(t, snap, 2)
	require.Equal(t, "assistant", snap[0].GetRole())
	require.Equal(t, "tool", snap[1].GetRole())
}

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

func TestModel_ResponsePipeline_DropsStaleResponse(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	require.Nil(t, e.currentPacket)

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.ChatResponse{
		RequestId: "ctx-1",
		Success:   true,
		Data:      &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"ignored"}}}},
	}})

	require.Empty(t, comm.pkts)
	require.Empty(t, stream.sendCalls)
}

func TestModel_ResponsePipeline_Error_EmitsLLMErrorAndEvent(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.ChatResponse{
		RequestId: "ctx-1",
		Success:   false,
		Error:     &protos.Error{ErrorMessage: "provider down"},
	}})

	errPkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", errPkt.ContextID)
	require.EqualError(t, errPkt.Error, "provider down")
	evt, ok := findPacket[internal_type.ConversationEventPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "error", evt.Data["type"])
}

func TestModel_ResponsePipeline_Chunk_EmitsDeltaEvenWhenEmpty(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.ChatResponse{
		RequestId: "ctx-1",
		Success:   true,
		Data:      &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{""}}}},
	}})

	delta, ok := findPacket[internal_type.LLMResponseDeltaPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", delta.ContextID)
	require.Equal(t, "", delta.Text)
}

func TestModel_ResponsePipeline_DoneWithToolCalls_ExecutesToolsAndOpensBlock(t *testing.T) {
	e, comm, _, toolExec := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.ChatResponse{
		RequestId:    "ctx-1",
		Success:      true,
		FinishReason: "tool_calls",
		Metrics:      []*protos.Metric{{Name: "token_count", Value: "3"}},
		Data:         testToolAssistantMessage("t1", "t2"),
	}})

	require.Len(t, toolExec.calls, 1)
	require.Equal(t, "ctx-1", toolExec.calls[0].contextID)
	require.Len(t, toolExec.calls[0].tools, 2)
	require.Equal(t, "ctx-1", e.history.PendingContextID())

	done, ok := findPacket[internal_type.LLMResponseDonePacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", done.ContextID)
}

func TestModel_ResponsePipeline_DoneWithoutToolCalls_AppendsAssistant(t *testing.T) {
	e, comm, _, toolExec := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-1"}

	e.Run(context.Background(), comm, ResponsePipeline{Response: &protos.ChatResponse{
		RequestId:    "ctx-1",
		Success:      true,
		FinishReason: "stop",
		Metrics:      []*protos.Metric{{Name: "token_count", Value: "3"}},
		Data:         &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"final"}}}},
	}})

	require.Empty(t, toolExec.calls)
	require.Empty(t, e.history.PendingContextID())
	snap := e.history.Snapshot()
	require.Len(t, snap, 1)
	require.Equal(t, "final", snap[0].GetAssistant().GetContents()[0])
}

func TestModel_ToolFollowUp_ValidationFailureBlocksSend(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	e.history.messages = append(e.history.messages,
		&protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}}}}},
	)

	e.Run(context.Background(), comm, ToolFollowUpPipeline{ContextID: "ctx-1"})
	require.Empty(t, stream.sendCalls)
}

func TestModel_ExecuteUserTurn_SupersedesOpenToolBlock(t *testing.T) {
	e, comm, _, _ := newModelTestEnv(t)
	e.history.AppendAssistant("ctx-old", testToolAssistantMessage("t1"))

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-new", Text: "new user msg"})
	require.NoError(t, err)

	evt, ok := findPacket[internal_type.ConversationEventPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "tool", evt.Name)
	require.Equal(t, "tool_block_superseded", evt.Data["type"])
	require.Equal(t, "user_interrupted", evt.Data["reason"])
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
