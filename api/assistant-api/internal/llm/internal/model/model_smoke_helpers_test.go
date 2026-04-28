package internal_model

import (
	"context"
	"io"
	"sync"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_agent_tool "github.com/rapidaai/api/assistant-api/internal/tool"
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
	sendErr   error
	closeSent bool
}

func (m *testStream) Send(req *protos.ChatRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls = append(m.sendCalls, req)
	return m.sendErr
}
func (m *testStream) Recv() (*protos.ChatResponse, error) { return nil, io.EOF }
func (m *testStream) CloseSend() error {
	m.mu.Lock()
	m.closeSent = true
	m.mu.Unlock()
	return nil
}
func (m *testStream) Header() (metadata.MD, error) { return nil, nil }
func (m *testStream) Trailer() metadata.MD         { return nil }
func (m *testStream) Context() context.Context     { return context.Background() }
func (m *testStream) SendMsg(any) error            { return nil }
func (m *testStream) RecvMsg(any) error            { return nil }

type testToolExecutor struct {
	internal_agent_tool.ToolExecutor
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

func findPackets[T any](pkts []internal_type.Packet) []T {
	out := make([]T, 0)
	for _, p := range pkts {
		if v, ok := p.(T); ok {
			out = append(out, v)
		}
	}
	return out
}
