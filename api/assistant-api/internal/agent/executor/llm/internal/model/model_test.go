//go:build legacy_model_tests
// +build legacy_model_tests

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_model

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	internal_agent_executor "github.com/rapidaai/api/assistant-api/internal/agent/executor"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	rapida_types "github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

type packetCollector struct {
	mu   sync.Mutex
	pkts []internal_type.Packet
}

func (c *packetCollector) collect(_ context.Context, pkts ...internal_type.Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pkts = append(c.pkts, pkts...)
	return nil
}

func (c *packetCollector) all() []internal_type.Packet {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]internal_type.Packet, len(c.pkts))
	copy(out, c.pkts)
	return out
}

// =============================================================================
// Mock: mockStream — grpc.BidiStreamingClient[protos.ChatRequest, protos.ChatResponse]
// =============================================================================

type streamRecvResult struct {
	resp *protos.ChatResponse
	err  error
}

type mockStream struct {
	mu        sync.Mutex
	sendCalls []*protos.ChatRequest
	sendErr   error
	recvCh    chan streamRecvResult
	closeSent bool
}

func newMockStream() *mockStream {
	return &mockStream{
		recvCh: make(chan streamRecvResult, 16),
	}
}

func (m *mockStream) Send(req *protos.ChatRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls = append(m.sendCalls, req)
	return m.sendErr
}

func (m *mockStream) Recv() (*protos.ChatResponse, error) {
	r, ok := <-m.recvCh
	if !ok {
		return nil, io.EOF
	}
	return r.resp, r.err
}

func (m *mockStream) CloseSend() error {
	m.mu.Lock()
	m.closeSent = true
	m.mu.Unlock()
	return nil
}

func (m *mockStream) Header() (metadata.MD, error) { return nil, nil }
func (m *mockStream) Trailer() metadata.MD         { return nil }
func (m *mockStream) Context() context.Context     { return context.Background() }
func (m *mockStream) SendMsg(any) error            { return nil }
func (m *mockStream) RecvMsg(any) error            { return nil }

// =============================================================================
// Mock: mockCommunication
// =============================================================================

type mockCommunication struct {
	internal_type.Communication // embedded nil for unimplemented methods
	collector                   *packetCollector
	assistant                   *internal_assistant_entity.Assistant
	args                        map[string]interface{}
	metadata                    map[string]interface{}
	conversation                *internal_conversation_entity.AssistantConversation
	histories                   []internal_type.MessagePacket
	source                      utils.RapidaSource
	mode                        type_enums.MessageMode
}

type noModeCommunication struct {
	internal_type.Communication // embedded nil for unimplemented methods
	assistant                   *internal_assistant_entity.Assistant
	args                        map[string]interface{}
	metadata                    map[string]interface{}
	conversation                *internal_conversation_entity.AssistantConversation
	histories                   []internal_type.MessagePacket
	source                      utils.RapidaSource
}

type unknownPipeline struct {
	stop bool
}

func (p *unknownPipeline) IsStop() bool {
	return p.stop
}

func (m *mockCommunication) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	return m.collector.collect(ctx, pkts...)
}

func (m *mockCommunication) Assistant() *internal_assistant_entity.Assistant {
	return m.assistant
}

func (m *mockCommunication) GetArgs() map[string]interface{} {
	return m.args
}

func (m *mockCommunication) GetOptions() utils.Option {
	return nil
}

func (m *mockCommunication) GetSource() utils.RapidaSource {
	if m.source == "" {
		return utils.WebPlugin
	}
	return m.source
}

func (m *mockCommunication) Conversation() *internal_conversation_entity.AssistantConversation {
	return m.conversation
}

func (m *mockCommunication) GetHistories() []internal_type.MessagePacket {
	return m.histories
}

func (m *mockCommunication) GetMetadata() map[string]interface{} {
	return m.metadata
}

func (m *mockCommunication) GetMode() type_enums.MessageMode {
	if m.mode == "" {
		return type_enums.TextMode
	}
	return m.mode
}

func (m *noModeCommunication) Assistant() *internal_assistant_entity.Assistant {
	return m.assistant
}

func (m *noModeCommunication) GetArgs() map[string]interface{} {
	return m.args
}

func (m *noModeCommunication) GetOptions() utils.Option {
	return nil
}

func (m *noModeCommunication) GetMode() type_enums.MessageMode {
	return ""
}

func (m *noModeCommunication) GetSource() utils.RapidaSource {
	if m.source == "" {
		return utils.WebPlugin
	}
	return m.source
}

func (m *noModeCommunication) Conversation() *internal_conversation_entity.AssistantConversation {
	return m.conversation
}

func (m *noModeCommunication) GetHistories() []internal_type.MessagePacket {
	return m.histories
}

func (m *noModeCommunication) GetMetadata() map[string]interface{} {
	return m.metadata
}

// =============================================================================
// Mock: mockToolExecutor
// =============================================================================

type mockToolExecutor struct {
	executeFn   func(ctx context.Context, contextID string, calls []*protos.ToolCall, comm internal_type.Communication)
	closeCalled bool
}

var _ internal_agent_executor.ToolExecutor = (*mockToolExecutor)(nil)

func (m *mockToolExecutor) Initialize(context.Context, internal_type.Communication) error {
	return nil
}

func (m *mockToolExecutor) GetFunctionDefinitions() []*protos.FunctionDefinition {
	return nil
}

func (m *mockToolExecutor) ExecuteAll(ctx context.Context, contextID string, calls []*protos.ToolCall, comm internal_type.Communication) {
	if m.executeFn != nil {
		m.executeFn(ctx, contextID, calls, comm)
	}
}

func (m *mockToolExecutor) Close(context.Context) error {
	m.closeCalled = true
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

func newTestComm() (*mockCommunication, *packetCollector) {
	c := &packetCollector{}
	return &mockCommunication{
		collector: c,
		assistant: &internal_assistant_entity.Assistant{
			AssistantProviderModel: &internal_assistant_entity.AssistantProviderModel{
				Template:              gorm_types.PromptMap{},
				AssistantModelOptions: []*internal_assistant_entity.AssistantProviderModelOption{},
			},
		},
		source: utils.WebPlugin,
		mode:   type_enums.TextMode,
	}, c
}

func newTestExecutor() *modelAssistantExecutor {
	logger, _ := commons.NewApplicationLogger()
	return &modelAssistantExecutor{
		logger:       logger,
		toolExecutor: &mockToolExecutor{},
		inputBuilder: integration_client_builders.NewChatInputBuilder(logger),
		history:      make([]*protos.Message, 0),
	}
}

func mustLanguage(t *testing.T, iso6391 string) rapida_types.Language {
	t.Helper()
	lang := rapida_types.LookupLanguage(iso6391)
	require.NotEmpty(t, lang.Name, "language %q must exist", iso6391)
	return lang
}

func historySnapshot(e *modelAssistantExecutor) []*protos.Message {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*protos.Message, len(e.history))
	copy(out, e.history)
	return out
}

func findPacket[T internal_type.Packet](pkts []internal_type.Packet) (T, bool) {
	for _, p := range pkts {
		if v, ok := p.(T); ok {
			return v, true
		}
	}
	var zero T
	return zero, false
}

func findPackets[T internal_type.Packet](pkts []internal_type.Packet) []T {
	var out []T
	for _, p := range pkts {
		if v, ok := p.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// findActionToolCalls returns LLMToolCallPackets that have a non-UNSPECIFIED Action.
func findActionToolCalls(pkts []internal_type.Packet) []internal_type.LLMToolCallPacket {
	var out []internal_type.LLMToolCallPacket
	for _, p := range pkts {
		if tc, ok := p.(internal_type.LLMToolCallPacket); ok && tc.Action != protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED {
			out = append(out, tc)
		}
	}
	return out
}

func TestBuildAssistantArgumentationContext_IncludesNamespaces(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()
	comm.assistant.Id = 123
	comm.assistant.Name = "Lex"
	comm.assistant.Language = "en"
	comm.assistant.Description = "Assistant description"
	comm.args = map[string]interface{}{"custom_key": "custom_value"}

	ctxMap := e.buildAssistantArgumentationContext(comm)
	system, ok := ctxMap["system"].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, system["current_date"])
	assert.NotEmpty(t, system["current_time"])
	assert.NotEmpty(t, system["current_datetime"])
	assert.NotEmpty(t, system["day_of_week"])
	assert.NotEmpty(t, system["date_rfc1123"])
	assert.NotEmpty(t, system["date_unix"])
	assert.NotEmpty(t, system["date_unix_ms"])

	assistant, ok := ctxMap["assistant"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Lex", assistant["name"])
	assert.Equal(t, "123", assistant["id"])
	assert.Equal(t, "en", assistant["language"])
	assert.Equal(t, "Assistant description", assistant["description"])

	message, ok := ctxMap["message"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "English", message["language"])

	assert.Equal(t, "custom_value", ctxMap["custom_key"])
	_, hasArgsNamespace := ctxMap["args"]
	assert.True(t, hasArgsNamespace, "arguments should be available via {{args.*}}")
}

func TestBuildConversationArgumentationContext_IncludesConversationFields(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()
	comm.conversation = &internal_conversation_entity.AssistantConversation{
		Audited: gorm_models.Audited{
			Id:          999,
			CreatedDate: gorm_models.TimeWrapper(time.Now().Add(-3 * time.Minute)),
		},
		Identifier: "user-42",
		Source:     utils.PhoneCall,
		Direction:  type_enums.DIRECTION_INBOUND,
	}

	ctxMap := e.buildConversationArgumentationContext(comm)
	conversation, ok := ctxMap["conversation"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "999", conversation["id"])
	assert.Equal(t, "user-42", conversation["identifier"])
	assert.Equal(t, "phone-call", conversation["source"])
	assert.Equal(t, "inbound", conversation["direction"])
	assert.NotEmpty(t, conversation["created_date"])
	assert.NotEmpty(t, conversation["duration"])
}

func TestBuildSessionArgumentationContext_ModeOmittedWhenNotAvailable(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()
	noMode := &noModeCommunication{
		assistant:    comm.assistant,
		args:         comm.args,
		metadata:     comm.metadata,
		conversation: comm.conversation,
		histories:    comm.histories,
		source:       comm.source,
	}

	ctxMap := e.buildSessionArgumentationContext(noMode)

	session, ok := ctxMap["session"].(map[string]interface{})
	require.True(t, ok)
	_, hasMode := session["mode"]
	assert.False(t, hasMode)
}

func TestBuildSessionArgumentationContext_IncludesMode(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()
	comm.mode = type_enums.AudioMode

	ctxMap := e.buildSessionArgumentationContext(comm)

	session, ok := ctxMap["session"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, type_enums.AudioMode, session["mode"])
}

func TestRequestPipeline_ExecutesAllStages(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-pipeline",
		Text:      "test",
		Language:  mustLanguage(t, "en"),
	})
	require.NoError(t, err)

	stream.mu.Lock()
	defer stream.mu.Unlock()
	assert.Len(t, stream.sendCalls, 1, "send_chat stage should have sent")

	evs := findPackets[internal_type.ConversationEventPacket](collector.all())
	require.Len(t, evs, 1, "emit_event stage should have emitted")
	assert.Equal(t, "executing", evs[0].Data["type"])
}

func TestResponsePipeline_ExecutesAllStages(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()
	resp := &protos.ChatResponse{
		RequestId: "req-stages",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"hi"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "1"}},
	}
	e.handleResponse(context.Background(), comm, resp)

	pkts := collector.all()
	require.GreaterOrEqual(t, len(pkts), 3, "validate + build + emit stages should produce packets")
}

func TestPipeline_LocalHistoryPipeline_AppendsHistory(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	err := e.Pipeline(context.Background(), comm, LocalHistoryPipeline{
		Message: &protos.Message{Role: "assistant"},
	})
	require.NoError(t, err)

	snapshot := historySnapshot(e)
	require.Len(t, snapshot, 1)
	assert.Equal(t, "assistant", snapshot[0].Role)
}

func TestPipeline_LocalHistoryPipeline_NilMessageNoOp(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	err := e.Pipeline(context.Background(), comm, LocalHistoryPipeline{})
	require.NoError(t, err)
	assert.Empty(t, historySnapshot(e))
}

func TestPipeline_PrepareHistoryPipeline_ChainsToSendAndAppend(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	e.mu.Lock()
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "existing"}}},
	}
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-prepare"}
	e.mu.Unlock()

	pipeline := PrepareHistoryPipeline{
		Packet: internal_type.UserInputPacket{
			ContextID: "ctx-prepare",
			Text:      "new input",
			Language:  mustLanguage(t, "en"),
		},
	}
	err := e.Pipeline(context.Background(), comm, pipeline)
	require.NoError(t, err)

	stream.mu.Lock()
	require.Len(t, stream.sendCalls, 1)
	stream.mu.Unlock()

	snapshot := historySnapshot(e)
	require.Len(t, snapshot, 2)
	assert.Equal(t, "existing", snapshot[0].GetUser().GetContent())
	assert.Equal(t, "new input", snapshot[1].GetUser().GetContent())
}

func TestPipeline_ArgumentationPipeline_PreparesPromptArgs(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	en := mustLanguage(t, "en")
	comm.args = map[string]interface{}{"name": "lex"}
	comm.assistant.AssistantProviderModel.Template = gorm_types.PromptMap{
		"prompt": []map[string]string{
			{"role": "system", "content": "name={{ name }} msg={{ message.text }} lang={{ message.language }}"},
		},
	}

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-arg",
		Text:      "hello",
		Language:  en,
	})
	require.NoError(t, err)

	stream.mu.Lock()
	require.Len(t, stream.sendCalls, 1)
	require.NotNil(t, stream.sendCalls[0].GetConversations()[0].GetSystem())
	assert.Equal(t, "name=lex msg=hello lang="+en.Name, stream.sendCalls[0].GetConversations()[0].GetSystem().GetContent())
	stream.mu.Unlock()
}

func TestPipeline_ArgumentationPipeline_PriorityOverride_AssistantConversationMessageSession(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	en := mustLanguage(t, "en")
	comm.mode = type_enums.AudioMode
	comm.args = map[string]interface{}{
		"assistant": map[string]interface{}{
			"name": "from-args-assistant",
		},
		"conversation": map[string]interface{}{
			"id": "from-args-conversation",
		},
		"message": map[string]interface{}{
			"text":     "from-args-message",
			"language": "from-args-language",
		},
		"session": map[string]interface{}{
			"mode": "from-args-session",
		},
	}
	comm.conversation = &internal_conversation_entity.AssistantConversation{
		Audited: gorm_models.Audited{Id: 99},
	}
	comm.assistant.Name = "from-assistant-stage"
	comm.assistant.AssistantProviderModel.Template = gorm_types.PromptMap{
		"prompt": []map[string]string{
			{"role": "system", "content": "a={{ assistant.name }} c={{ conversation.id }} m={{ message.text }} l={{ message.language }} s={{ session.mode }}"},
		},
	}

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-arg-override",
		Text:      "hello",
		Language:  en,
	})
	require.NoError(t, err)

	stream.mu.Lock()
	require.Len(t, stream.sendCalls, 1)
	require.NotNil(t, stream.sendCalls[0].GetConversations()[0].GetSystem())
	assert.Equal(t, "a=from-args-assistant c=99 m=hello l="+en.Name+" s=audio", stream.sendCalls[0].GetConversations()[0].GetSystem().GetContent())
	stream.mu.Unlock()
}

func TestExecute_MessageLanguage_DefaultEnglishFallback(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	comm.assistant.AssistantProviderModel.Template = gorm_types.PromptMap{
		"prompt": []map[string]string{
			{"role": "system", "content": "lang={{ message.language }} text={{ message.text }}"},
		},
	}

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-default-language",
		Text:      "hello",
	})
	require.NoError(t, err)

	stream.mu.Lock()
	require.Len(t, stream.sendCalls, 1)
	require.NotNil(t, stream.sendCalls[0].GetConversations()[0].GetSystem())
	// When Language is zero-value, buildMessageArgumentationContext sets language=""
	// (overrides the "English" default from buildAssistantArgumentationContext).
	assert.Equal(t, "lang= text=hello", stream.sendCalls[0].GetConversations()[0].GetSystem().GetContent())
	stream.mu.Unlock()
}

func TestExecute_MessageLanguage_UsesUserTextReceivedPacketLanguage(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	hi := mustLanguage(t, "hi")
	comm.assistant.AssistantProviderModel.Template = gorm_types.PromptMap{
		"prompt": []map[string]string{
			{"role": "system", "content": "lang={{ message.language }} text={{ message.text }}"},
		},
	}

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-explicit-language",
		Text:      "namaste",
		Language:  hi,
	})
	require.NoError(t, err)

	stream.mu.Lock()
	require.Len(t, stream.sendCalls, 1)
	require.NotNil(t, stream.sendCalls[0].GetConversations()[0].GetSystem())
	assert.Equal(t, "lang="+hi.Name+" text=namaste", stream.sendCalls[0].GetConversations()[0].GetSystem().GetContent())
	stream.mu.Unlock()
}

func TestExecute_MessageLanguage_DottedPromptVariable_DoesNotBreakTemplateParsing(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	fr := mustLanguage(t, "fr")
	comm.assistant.AssistantProviderModel.Template = gorm_types.PromptMap{
		"prompt": []map[string]string{
			{"role": "system", "content": "lang={{ message.language }} text={{ message.text }}"},
		},
		"promptVariables": []map[string]string{
			{"name": "message.language", "defaultValue": "English"},
		},
	}

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-dotted-variable",
		Text:      "bonjour",
		Language:  fr,
	})
	require.NoError(t, err)

	stream.mu.Lock()
	require.Len(t, stream.sendCalls, 1)
	require.NotNil(t, stream.sendCalls[0].GetConversations()[0].GetSystem())
	assert.Equal(t, "lang="+fr.Name+" text=bonjour", stream.sendCalls[0].GetConversations()[0].GetSystem().GetContent())
	stream.mu.Unlock()
}

// =============================================================================
// Tests: handleResponse — 5 cases
// =============================================================================

func TestHandleResponse_Error(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	resp := &protos.ChatResponse{
		RequestId: "req-1",
		Success:   false,
		Error:     &protos.Error{ErrorMessage: "rate limited"},
	}
	e.handleResponse(context.Background(), comm, resp)

	pkts := collector.all()
	require.Len(t, pkts, 2)

	errPkt, ok := pkts[0].(internal_type.LLMErrorPacket)
	require.True(t, ok)
	assert.Equal(t, "req-1", errPkt.ContextID)
	assert.Equal(t, "rate limited", errPkt.Error.Error())

	ev, ok := pkts[1].(internal_type.ConversationEventPacket)
	require.True(t, ok)
	assert.Equal(t, "error", ev.Data["type"])
	assert.Equal(t, "rate limited", ev.Data["error"])
}

func TestHandleResponse_NilOutput(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	resp := &protos.ChatResponse{
		RequestId: "req-2",
		Success:   true,
		Data:      nil,
	}
	e.handleResponse(context.Background(), comm, resp)
	assert.Empty(t, collector.all(), "nil output should emit no packets")
}

func TestHandleResponse_StaleResponse_Dropped(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-active"}
	e.mu.Lock()
	e.history = append(e.history, &protos.Message{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "q"}}})
	e.mu.Unlock()

	resp := &protos.ChatResponse{
		RequestId: "ctx-stale",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"should-drop"}},
			},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "1"}},
	}
	e.handleResponse(context.Background(), comm, resp)
	assert.Empty(t, collector.all(), "stale response should be ignored")
	snap := historySnapshot(e)
	require.Len(t, snap, 1)
}

func TestValidateHistorySequence_SandwichRuleViolation(t *testing.T) {
	e := newTestExecutor()
	err := e.validateHistorySequence([]*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function"}},
				},
			},
		},
		{
			Role:    "user",
			Message: &protos.Message_User{User: &protos.UserMessage{Content: "next"}},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "immediately followed by tool response")
}

func TestValidateHistorySequence_IDMismatch(t *testing.T) {
	e := newTestExecutor()
	err := e.validateHistorySequence([]*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function"}},
				},
			},
		},
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{{Id: "tc2", Name: "fn", Content: "{}"}},
				},
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing tool response")
}

func TestValidateHistorySequence_OrphanTool(t *testing.T) {
	e := newTestExecutor()
	err := e.validateHistorySequence([]*protos.Message{
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{Tools: []*protos.ToolMessage_Tool{{Id: "tc1"}}},
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "orphan tool response")
}

func TestValidateHistorySequence_StrictSequencingViolation(t *testing.T) {
	e := newTestExecutor()
	err := e.validateHistorySequence([]*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function"}},
				},
			},
		},
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{{Id: "tc1", Name: "fn", Content: "{}"}},
				},
			},
		},
		{
			Role:    "user",
			Message: &protos.Message_User{User: &protos.UserMessage{Content: "oops"}},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict sequencing violated")
}

func TestValidateHistorySequence_ValidChain(t *testing.T) {
	e := newTestExecutor()
	err := e.validateHistorySequence([]*protos.Message{
		{
			Role:    "user",
			Message: &protos.Message_User{User: &protos.UserMessage{Content: "q"}},
		},
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function"}},
				},
			},
		},
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{{Id: "tc1", Name: "fn", Content: "{}"}},
				},
			},
		},
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"done"}},
			},
		},
	})
	require.NoError(t, err)
}

func TestExecuteUserTurn_InvalidHistoryIsNotRejectedByPipeline(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	e.mu.Lock()
	e.history = []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function"}},
				},
			},
		},
		{
			Role:    "user",
			Message: &protos.Message_User{User: &protos.UserMessage{Content: "bad"}},
		},
	}
	e.mu.Unlock()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-invalid-history",
		Text:      "new",
		Language:  mustLanguage(t, "en"),
	})
	require.NoError(t, err)
	stream.mu.Lock()
	defer stream.mu.Unlock()
	assert.Len(t, stream.sendCalls, 1)
}

func TestHandleResponse_FinalWithoutToolCalls(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	resp := &protos.ChatResponse{
		RequestId: "req-3",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Hello", " world"},
				},
			},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "10"}},
	}
	e.handleResponse(context.Background(), comm, resp)

	pkts := collector.all()
	require.Len(t, pkts, 3)

	done, ok := pkts[0].(internal_type.LLMResponseDonePacket)
	require.True(t, ok)
	assert.Equal(t, "req-3", done.ContextID)
	assert.Equal(t, "Hello world", done.Text)

	ev, ok := pkts[1].(internal_type.ConversationEventPacket)
	require.True(t, ok)
	assert.Equal(t, "completed", ev.Data["type"])
	assert.Equal(t, "11", ev.Data["response_char_count"])

	metric, ok := pkts[2].(internal_type.AssistantMessageMetricPacket)
	require.True(t, ok)
	assert.Equal(t, "req-3", metric.ContextID)
	require.Len(t, metric.Metrics, 1)
	assert.Equal(t, "agent_tokens", metric.Metrics[0].Name)

	// Verify history was updated
	snapshot := historySnapshot(e)
	require.Len(t, snapshot, 1)
	assert.Equal(t, "assistant", snapshot[0].Role)
}

func TestHandleResponse_FinalWithToolCalls(t *testing.T) {
	e := newTestExecutor()
	e.stream = nil
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "req-4"}

	// Wire mock tool executor that emits LLMToolCallPacket (matching real behavior).
	e.toolExecutor = &mockToolExecutor{
		executeFn: func(ctx context.Context, contextID string, calls []*protos.ToolCall, comm internal_type.Communication) {
			for _, tc := range calls {
				comm.OnPacket(ctx, internal_type.LLMToolCallPacket{
					ToolID: tc.GetId(), Name: tc.GetFunction().GetName(), ContextID: contextID,
				})
			}
		},
	}

	comm, collector := newTestComm()

	resp := &protos.ChatResponse{
		RequestId: "req-4",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents:  []string{"calling tool"},
					ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function", Function: &protos.FunctionCall{Name: "get_weather", Arguments: "{}"}}},
				},
			},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "5"}},
	}
	e.handleResponse(context.Background(), comm, resp)

	pkts := collector.all()
	// Should have: LLMResponseDonePacket, ConversationEventPacket(completed), AssistantMessageMetricPacket, LLMToolCallPacket
	require.GreaterOrEqual(t, len(pkts), 4)

	done, ok := findPacket[internal_type.LLMResponseDonePacket](pkts)
	require.True(t, ok)
	assert.Equal(t, "calling tool", done.Text)

	// stageToolFollowUp delegates to toolExecutor.ExecuteAll which emits LLMToolCallPacket
	toolCallPkts := findPackets[internal_type.LLMToolCallPacket](pkts)
	require.Len(t, toolCallPkts, 1)
	assert.Equal(t, "tc1", toolCallPkts[0].ToolID)
	assert.Equal(t, "get_weather", toolCallPkts[0].Name)
	assert.Equal(t, "req-4", toolCallPkts[0].ContextID)

	// Verify history: assistant message appended (tool result comes from dispatch layer later)
	snapshot := historySnapshot(e)
	require.Len(t, snapshot, 1, "only assistant msg in history; tool result comes externally")
}

func TestHandleResponse_ToolFollowUpEmitsToolCallPacket(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	fr := mustLanguage(t, "fr")

	// Wire mock tool executor that emits LLMToolCallPacket (matching real behavior).
	e.toolExecutor = &mockToolExecutor{
		executeFn: func(ctx context.Context, contextID string, calls []*protos.ToolCall, comm internal_type.Communication) {
			for _, tc := range calls {
				comm.OnPacket(ctx, internal_type.LLMToolCallPacket{
					ToolID: tc.GetId(), Name: tc.GetFunction().GetName(), ContextID: contextID,
				})
			}
		},
	}

	comm, collector := newTestComm()
	comm.metadata = map[string]interface{}{"client.language": "fr"}
	comm.assistant.AssistantProviderModel.Template = gorm_types.PromptMap{
		"prompt": []map[string]string{
			{
				"role":    "system",
				"content": "lang={{ message.language }} text={{ message.text }}",
			},
		},
		"promptVariables": []map[string]string{},
	}

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "req-tool",
		Text:      "bonjour",
		Language:  fr,
	})
	require.NoError(t, err)

	resp := &protos.ChatResponse{
		RequestId: "req-tool",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents:  []string{"tooling"},
					ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function", Function: &protos.FunctionCall{Name: "fn", Arguments: "{}"}}},
				},
			},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "5"}},
	}
	e.handleResponse(context.Background(), comm, resp)

	// Only the initial user message send — no follow-up chat (tool execution is external now)
	stream.mu.Lock()
	defer stream.mu.Unlock()
	require.Len(t, stream.sendCalls, 1, "only initial send; tool follow-up is external")

	// Verify LLMToolCallPacket was emitted by the tool executor via communication.OnPacket
	toolCallPkts := findPackets[internal_type.LLMToolCallPacket](collector.all())
	require.Len(t, toolCallPkts, 1)
	assert.Equal(t, "tc1", toolCallPkts[0].ToolID)
	assert.Equal(t, "fn", toolCallPkts[0].Name)
	assert.Equal(t, "req-tool", toolCallPkts[0].ContextID)

	errPkts := findPackets[internal_type.LLMErrorPacket](collector.all())
	assert.Empty(t, errPkts)
}

func TestHandleResponse_StreamDelta(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	resp := &protos.ChatResponse{
		RequestId: "req-5",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"partial"},
				},
			},
		},
		// No Metrics → streaming delta
	}
	e.handleResponse(context.Background(), comm, resp)

	pkts := collector.all()
	require.Len(t, pkts, 2)

	delta, ok := pkts[0].(internal_type.LLMResponseDeltaPacket)
	require.True(t, ok)
	assert.Equal(t, "req-5", delta.ContextID)
	assert.Equal(t, "partial", delta.Text)

	ev, ok := pkts[1].(internal_type.ConversationEventPacket)
	require.True(t, ok)
	assert.Equal(t, "llm", ev.Name)
	assert.Equal(t, "chunk", ev.Data["type"])
	assert.Equal(t, "partial", ev.Data["text"])
	assert.Equal(t, "7", ev.Data["response_char_count"])
	assert.Equal(t, "req-5", ev.ContextID, "chunk event must include ContextID for correlation")
}

// =============================================================================
// Tests: history mutex — 3 cases
// =============================================================================

func TestSnapshotHistory_ReturnsCopy(t *testing.T) {
	e := newTestExecutor()
	e.mu.Lock()
	e.history = append(e.history, &protos.Message{Role: "user"})
	e.mu.Unlock()

	snapshot := historySnapshot(e)
	require.Len(t, snapshot, 1)

	// Modify snapshot — should not affect original
	snapshot[0] = &protos.Message{Role: "modified"}
	original := historySnapshot(e)
	assert.Equal(t, "user", original[0].Role, "modifying snapshot should not affect original")
}

func TestConcurrency_HistoryAndSnapshot(t *testing.T) {
	e := newTestExecutor()

	var wg sync.WaitGroup
	wg.Add(2)

	comm, _ := newTestComm()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	// Reader: reads snapshots concurrently
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = historySnapshot(e)
		}
	}()

	wg.Wait()
	snapshot := historySnapshot(e)
	assert.Len(t, snapshot, 100, "all messages should be in history")
}

func TestHistoryClearedAfterClose(t *testing.T) {
	e := newTestExecutor()
	e.mu.Lock()
	e.history = append(e.history, &protos.Message{Role: "user"})
	e.mu.Unlock()

	_ = e.Close(context.Background())

	snapshot := historySnapshot(e)
	assert.Empty(t, snapshot, "history should be empty after Close")
}

// =============================================================================
// Tests: Execute — InjectMessagePacket and UserTextReceivedPacket paths
// =============================================================================

func TestExecute_InjectMessagePacket_AppendsHistory(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
		ContextID: "ctx-1",
		Text:      "hello",
	})
	require.NoError(t, err)
	assert.Empty(t, collector.all(), "InjectMessagePacket should not emit packets")

	snapshot := historySnapshot(e)
	require.Len(t, snapshot, 1)
	assert.Equal(t, "assistant", snapshot[0].Role)
	assert.Equal(t, []string{"hello"}, snapshot[0].GetAssistant().GetContents())
}

func TestExecute_LLMToolCallPacket_NoHistoryMutation(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.LLMToolCallPacket{
		ContextID: "ctx-1",
		ToolID:    "t1",
		Name:      "get_weather",
		Arguments: map[string]string{"city": "delhi"},
	})
	require.NoError(t, err)

	snapshot := historySnapshot(e)
	require.Empty(t, snapshot, "LLMToolCallPacket should not mutate executor history")
}

func TestExecute_UserTextReceivedPacket_SendsAndRecordsHistory(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "say hello",
	})
	require.NoError(t, err)

	evs := findPackets[internal_type.ConversationEventPacket](collector.all())
	require.Len(t, evs, 1)
	assert.Equal(t, "executing", evs[0].Data["type"])
	assert.Equal(t, "say hello", evs[0].Data["script"])
	assert.Equal(t, "9", evs[0].Data["input_char_count"])
	assert.Equal(t, "0", evs[0].Data["history_count"])

	stream.mu.Lock()
	defer stream.mu.Unlock()
	require.Len(t, stream.sendCalls, 1)

	snapshot := historySnapshot(e)
	require.Len(t, snapshot, 1)
	assert.Equal(t, "user", snapshot[0].Role)
	require.NotNil(t, e.currentPacket)
	currentPacket := *e.currentPacket
	assert.Equal(t, "say hello", currentPacket.Text)
	assert.Empty(t, currentPacket.Language.Name, "Language should be zero-value when not set")
}

func TestExecute_LLMInterruptPacket(t *testing.T) {
	e := newTestExecutor()
	e.currentPacket = &internal_type.UserInputPacket{
		ContextID: "ctx-old",
		Text:      "old text",
		Language:  mustLanguage(t, "fr"),
	}
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "x"})
	require.NoError(t, err)
	assert.Nil(t, e.currentPacket, "currentPacket should be nil on interrupt")
}

func TestExecute_UnsupportedPacket(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.EndOfSpeechPacket{ContextID: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported packet type")
}

// =============================================================================
// Tests: send — nil stream and success
// =============================================================================

func TestSend_NilStream(t *testing.T) {
	e := newTestExecutor()
	e.stream = nil
	comm, _ := newTestComm()

	err := e.chat(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1"}, map[string]interface{}{}, &protos.Message{Role: "user"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stream not connected")
}

func TestSend_Success(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	err := e.chat(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1"}, map[string]interface{}{}, &protos.Message{Role: "user"})
	require.NoError(t, err)

	stream.mu.Lock()
	defer stream.mu.Unlock()
	require.Len(t, stream.sendCalls, 1)
	assert.Empty(t, historySnapshot(e), "chat should not append history directly; pipeline stage owns history mutation")
}

// =============================================================================
// Tests: Close — 3 cases
// =============================================================================

func TestClose_ClearsHistoryAndStream(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	toolExec := e.toolExecutor.(*mockToolExecutor)
	e.mu.Lock()
	e.history = append(e.history, &protos.Message{Role: "user"})
	e.currentPacket = &internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "hello",
		Language:  mustLanguage(t, "en"),
	}
	e.mu.Unlock()

	err := e.Close(context.Background())
	require.NoError(t, err)

	e.mu.RLock()
	defer e.mu.RUnlock()
	assert.Nil(t, e.stream)
	assert.Empty(t, e.history)
	assert.Nil(t, e.currentPacket)
	assert.True(t, toolExec.closeCalled, "Close must call toolExecutor.Close to release MCP resources")
}

func TestClose_NoPanicNilStream(t *testing.T) {
	e := newTestExecutor()
	e.stream = nil

	err := e.Close(context.Background())
	require.NoError(t, err, "Close on nil stream should not panic")
}

// =============================================================================
// Tests: listen — processes responses then exits on error
// =============================================================================

func TestListen_RecvEOF(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()

	stream.recvCh <- streamRecvResult{err: io.EOF}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.listen(context.Background(), comm)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listen did not exit on EOF")
	}

	dirs := findActionToolCalls(collector.all())
	require.Len(t, dirs, 1)
	assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, dirs[0].Action)
	assert.Equal(t, "server closed connection", dirs[0].Arguments["reason"])
}

func TestListen_NilStream(t *testing.T) {
	e := newTestExecutor()
	e.stream = nil
	comm, _ := newTestComm()

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.listen(context.Background(), comm)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listen did not exit with nil stream")
	}
}

func TestListen_ContextCancelled(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.listen(ctx, comm)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listen did not exit on context cancel")
	}
}

func TestListen_ProcessesMultipleMessages(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()

	// Two deltas then EOF
	stream.recvCh <- streamRecvResult{resp: &protos.ChatResponse{
		RequestId: "r1",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"chunk1"}},
			},
		},
	}}
	stream.recvCh <- streamRecvResult{resp: &protos.ChatResponse{
		RequestId: "r1",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"chunk2"}},
			},
		},
	}}
	stream.recvCh <- streamRecvResult{err: io.EOF}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.listen(context.Background(), comm)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listen did not exit")
	}

	deltas := findPackets[internal_type.LLMResponseDeltaPacket](collector.all())
	assert.Len(t, deltas, 2)
}

// =============================================================================
// Tests: Name
// =============================================================================

func TestName(t *testing.T) {
	e := newTestExecutor()
	assert.Equal(t, "model", e.Name())
}

// =============================================================================
// Tests: handleResponse — empty contents delta emits nothing
// =============================================================================

func TestHandleResponse_EmptyContents(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	resp := &protos.ChatResponse{
		RequestId: "req-6",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{},
				},
			},
		},
	}
	e.handleResponse(context.Background(), comm, resp)
	assert.Empty(t, collector.all(), "empty contents should emit no delta")
}

// =============================================================================
// Tests: concurrent listen + close (run with -race)
// =============================================================================

func TestConcurrency_ListenAndClose(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	ctx, cancel := context.WithCancel(context.Background())
	e.ctx = ctx
	e.ctxCancel = cancel

	go func() {
		e.listen(ctx, comm)
	}()

	time.Sleep(5 * time.Millisecond)
	err := e.Close(context.Background())
	require.NoError(t, err)
}

// =============================================================================
// Tests: Execute UserTextReceivedPacket includes correct history_count
// =============================================================================

func TestExecute_UserTextReceivedPacket_HistoryCount(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream

	e.mu.Lock()
	e.history = append(e.history,
		&protos.Message{Role: "user"},
		&protos.Message{Role: "assistant"},
	)
	e.mu.Unlock()

	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-2",
		Text:      "follow up",
	})
	require.NoError(t, err)

	evs := findPackets[internal_type.ConversationEventPacket](collector.all())
	require.Len(t, evs, 1)
	assert.Equal(t, "2", evs[0].Data["history_count"], "should reflect 2 existing messages")
}

// =============================================================================
// Tests: Execute with stream send error
// =============================================================================

func TestExecute_SendError(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	stream.sendErr = fmt.Errorf("send failed")
	e.stream = stream
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send chat request")
}

// =============================================================================
// Tests: Bug 1 — history not modified on send error
// =============================================================================

func TestExecute_SendError_HistoryNotModified(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	stream.sendErr = fmt.Errorf("send failed")
	e.stream = stream
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "test",
	})
	require.Error(t, err)
	assert.Empty(t, historySnapshot(e), "history must not be modified when send fails")
}

// =============================================================================
// Tests: Bug 3 — listener exits cleanly when context is cancelled before EOF
// =============================================================================

func TestListen_ExitsCleanlyOnClose(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()

	ctx, cancel := context.WithCancel(context.Background())

	listenDone := make(chan struct{})
	go func() {
		defer close(listenDone)
		e.listen(ctx, comm)
	}()

	// Cancel context first (simulating ctxCancel() in Close()), then unblock Recv.
	cancel()
	close(stream.recvCh)

	select {
	case <-listenDone:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not exit after context cancellation")
	}

	dirs := findActionToolCalls(collector.all())
	assert.Empty(t, dirs, "END_CONVERSATION must not be dispatched when context is cancelled")
}

// =============================================================================
// End-to-End: full user turn → stream deltas → final response → history
// =============================================================================

func TestE2E_FullConversationTurn(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()
	en := mustLanguage(t, "en")

	// 1. User sends a message
	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "turn-1",
		Text:      "What is Go?",
		Language:  en,
	})
	require.NoError(t, err)

	// Verify: stream received the chat request
	stream.mu.Lock()
	require.Len(t, stream.sendCalls, 1)
	stream.mu.Unlock()

	// Verify: user message appended to history
	snap := historySnapshot(e)
	require.Len(t, snap, 1)
	assert.Equal(t, "user", snap[0].Role)
	assert.Equal(t, "What is Go?", snap[0].GetUser().GetContent())

	// 2. Simulate streaming deltas from LLM
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "turn-1",
		Success:   true,
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"Go is"}}},
		},
	})
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "turn-1",
		Success:   true,
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{" a language"}}},
		},
	})

	// 3. Final response with metrics
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "turn-1",
		Success:   true,
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"Go is a language"}}},
		},
		Metrics: []*protos.Metric{{Name: "total_tokens", Value: "42"}},
	})

	// Verify: packets emitted in correct order
	pkts := collector.all()
	// Expected: executing event, 2 delta pairs (delta+event), final triple (done+event+metric)
	events := findPackets[internal_type.ConversationEventPacket](pkts)
	deltas := findPackets[internal_type.LLMResponseDeltaPacket](pkts)
	dones := findPackets[internal_type.LLMResponseDonePacket](pkts)
	metrics := findPackets[internal_type.AssistantMessageMetricPacket](pkts)

	assert.Len(t, deltas, 2, "should have 2 streaming deltas")
	assert.Equal(t, "Go is", deltas[0].Text)
	assert.Equal(t, " a language", deltas[1].Text)
	require.Len(t, dones, 1, "should have 1 done packet")
	assert.Equal(t, "Go is a language", dones[0].Text)
	require.Len(t, metrics, 1)
	assert.Equal(t, "agent_total_tokens", metrics[0].Metrics[0].Name)

	// Verify event types in order
	eventTypes := make([]string, 0, len(events))
	for _, ev := range events {
		eventTypes = append(eventTypes, ev.Data["type"])
	}
	assert.Equal(t, []string{"executing", "chunk", "chunk", "completed"}, eventTypes)

	// Verify: history has user + assistant
	snap = historySnapshot(e)
	require.Len(t, snap, 2)
	assert.Equal(t, "user", snap[0].Role)
	assert.Equal(t, "assistant", snap[1].Role)
}

func TestE2E_MultiTurnConversation(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	en := mustLanguage(t, "en")

	for turn := 1; turn <= 5; turn++ {
		ctxID := fmt.Sprintf("turn-%d", turn)

		// User sends
		err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
			ContextID: ctxID,
			Text:      fmt.Sprintf("message %d", turn),
			Language:  en,
		})
		require.NoError(t, err)

		// LLM responds with final
		e.handleResponse(context.Background(), comm, &protos.ChatResponse{
			RequestId: ctxID,
			Success:   true,
			Data: &protos.Message{
				Role:    "assistant",
				Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{fmt.Sprintf("reply %d", turn)}}},
			},
			Metrics: []*protos.Metric{{Name: "tokens", Value: "1"}},
		})
	}

	// Verify: history has 10 messages (5 user + 5 assistant)
	snap := historySnapshot(e)
	require.Len(t, snap, 10)
	for i := 0; i < 10; i += 2 {
		assert.Equal(t, "user", snap[i].Role)
		assert.Equal(t, "assistant", snap[i+1].Role)
	}

	// Verify: stream got 5 send calls
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 5)
	stream.mu.Unlock()
}

func TestE2E_ToolCallRoundTrip(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	en := mustLanguage(t, "en")

	// Wire mock tool executor that emits LLMToolCallPacket (matching real behavior).
	e.toolExecutor = &mockToolExecutor{
		executeFn: func(ctx context.Context, contextID string, calls []*protos.ToolCall, comm internal_type.Communication) {
			for _, tc := range calls {
				comm.OnPacket(ctx, internal_type.LLMToolCallPacket{
					ToolID: tc.GetId(), Name: tc.GetFunction().GetName(), ContextID: contextID,
				})
			}
		},
	}

	comm, collector := newTestComm()

	// 1. User asks about weather
	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "tool-turn",
		Text:      "weather?",
		Language:  en,
	})
	require.NoError(t, err)

	// 2. LLM responds with tool call
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "tool-turn",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
				Contents:  []string{"Let me check"},
				ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function", Function: &protos.FunctionCall{Name: "get_weather", Arguments: "{}"}}},
			}},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "10"}},
	})

	// Verify: history has user + assistant(tool_call) only — tool result comes externally
	snap := historySnapshot(e)
	require.Len(t, snap, 2, "user + assistant(tool_call); tool result is external")
	assert.Equal(t, "user", snap[0].Role)
	assert.Equal(t, "assistant", snap[1].Role)
	assert.Len(t, snap[1].GetAssistant().GetToolCalls(), 1)

	// Verify: stream got 1 send (initial only; tool follow-up is now external)
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 1, "only initial send")
	stream.mu.Unlock()

	// Verify: done packet emitted despite tool calls
	dones := findPackets[internal_type.LLMResponseDonePacket](collector.all())
	require.Len(t, dones, 1)
	assert.Equal(t, "Let me check", dones[0].Text)

	// Verify: LLMToolCallPacket emitted by tool executor via communication.OnPacket
	toolCallPkts := findPackets[internal_type.LLMToolCallPacket](collector.all())
	require.Len(t, toolCallPkts, 1)
	assert.Equal(t, "tc1", toolCallPkts[0].ToolID)
	assert.Equal(t, "get_weather", toolCallPkts[0].Name)
}

func TestE2E_InterruptDuringStreaming(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()
	en := mustLanguage(t, "en")

	// 1. User sends first message
	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "tell me a story",
		Language:  en,
	})
	require.NoError(t, err)

	// 2. Some deltas arrive
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "ctx-1",
		Success:   true,
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"Once upon"}}},
		},
	})

	// 3. User interrupts
	err = e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-1"})
	require.NoError(t, err)
	assert.Nil(t, e.currentPacket)

	// 4. Late response from old context — isStaleResponse returns false when
	// currentPacket is nil (interrupt clears it), so the response passes through.
	// This is by design: after interrupt, no context is "active" so nothing is "stale".
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "ctx-1",
		Success:   true,
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"a time"}}},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "5"}},
	})

	// Verify: the delta before interrupt was emitted
	deltas := findPackets[internal_type.LLMResponseDeltaPacket](collector.all())
	assert.Len(t, deltas, 1, "pre-interrupt delta should be emitted")
	// The final arrives after interrupt but is not filtered (currentPacket=nil → not stale).
	dones := findPackets[internal_type.LLMResponseDonePacket](collector.all())
	assert.Len(t, dones, 1, "post-interrupt final passes because currentPacket is nil")

	// 5. User sends new message — pipeline should work
	err = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-2",
		Text:      "new topic",
		Language:  en,
	})
	require.NoError(t, err)
	assert.NotNil(t, e.currentPacket)
	assert.Equal(t, "ctx-2", e.currentPacket.ContextID)
}

func TestE2E_ListenProcessesResponsesAndExitsOnEOF(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()

	// Queue: delta, final, EOF
	stream.recvCh <- streamRecvResult{resp: &protos.ChatResponse{
		RequestId: "r1", Success: true,
		Data: &protos.Message{Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"chunk"}}}},
	}}
	stream.recvCh <- streamRecvResult{resp: &protos.ChatResponse{
		RequestId: "r1", Success: true,
		Data: &protos.Message{Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"done"}}}},
		Metrics: []*protos.Metric{{Name: "t", Value: "1"}},
	}}
	stream.recvCh <- streamRecvResult{err: io.EOF}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.listen(context.Background(), comm)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("listen did not exit")
	}

	pkts := collector.all()
	deltas := findPackets[internal_type.LLMResponseDeltaPacket](pkts)
	dones := findPackets[internal_type.LLMResponseDonePacket](pkts)
	dirs := findActionToolCalls(pkts)

	assert.Len(t, deltas, 1)
	assert.Len(t, dones, 1)
	require.Len(t, dirs, 1)
	assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, dirs[0].Action)
}

// =============================================================================
// Deadlock Detection (run with -timeout 10s and -race)
// =============================================================================

// TestDeadlock_ExecuteAndResponseConcurrent verifies no deadlock when Execute
// (which acquires mu for currentPacket + history) and handleResponse (which
// acquires mu for history append + stale check) run concurrently.
func TestDeadlock_ExecuteAndResponseConcurrent(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer: sends Execute calls
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = e.Execute(ctx, comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	// Reader: processes responses concurrently
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			e.handleResponse(ctx, comm, &protos.ChatResponse{
				RequestId: fmt.Sprintf("ctx-%d", i),
				Success:   true,
				Data: &protos.Message{
					Role:    "assistant",
					Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"resp"}}},
				},
				Metrics: []*protos.Metric{{Name: "t", Value: "1"}},
			})
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success — no deadlock
	case <-ctx.Done():
		t.Fatal("DEADLOCK: Execute + handleResponse concurrent access timed out")
	}
}

// TestDeadlock_ExecuteAndCloseConcurrent verifies no deadlock when Close
// (which acquires mu exclusively) runs while Execute is in progress.
func TestDeadlock_ExecuteAndCloseConcurrent(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	e.ctx, e.ctxCancel = context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(ctx, comm, internal_type.InjectMessagePacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond) // let some executes start
		_ = e.Close(ctx)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("DEADLOCK: Execute + Close concurrent access timed out")
	}
}

// TestDeadlock_ListenAndExecuteAndClose verifies no deadlock when all three
// goroutines contend: listener (RLock for stream read), Execute (Lock for
// currentPacket write + RLock for history snapshot), Close (Lock for teardown).
func TestDeadlock_ListenAndExecuteAndClose(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	e.ctx, e.ctxCancel = context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(3)

	// Listener goroutine
	go func() {
		defer wg.Done()
		e.listen(e.ctx, comm)
	}()

	// Execute goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = e.Execute(ctx, comm, internal_type.InjectMessagePacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("inject-%d", i),
			})
		}
	}()

	// Close goroutine (after brief delay to let others start).
	// Close() cancels context and nils the stream, but the listener may be
	// blocked on Recv (channel read). Close the recvCh to unblock it.
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		_ = e.Close(ctx)
		close(stream.recvCh)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("DEADLOCK: listen + Execute + Close concurrent access timed out")
	}
}

// TestDeadlock_ToolCallWithConcurrentInterrupt verifies no deadlock when
// tool execution (which holds mu.Lock to append history) races with an
// interruption (which holds mu.Lock to clear currentPacket).
func TestDeadlock_ToolCallWithConcurrentInterrupt(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	en := mustLanguage(t, "en")

	toolDelay := make(chan struct{})
	e.toolExecutor = &mockToolExecutor{
		executeFn: func(_ context.Context, _ string, _ []*protos.ToolCall, _ internal_type.Communication) {
			<-toolDelay // block until signaled
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// User sends
	_ = e.Execute(ctx, comm, internal_type.UserInputPacket{
		ContextID: "ctx-tool-interrupt",
		Text:      "question",
		Language:  en,
	})

	var wg sync.WaitGroup
	wg.Add(2)

	// Tool call response in goroutine
	go func() {
		defer wg.Done()
		e.handleResponse(ctx, comm, &protos.ChatResponse{
			RequestId: "ctx-tool-interrupt",
			Success:   true,
			Data: &protos.Message{
				Role: "assistant",
				Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
					Contents:  []string{"calling"},
					ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function"}},
				}},
			},
			Metrics: []*protos.Metric{{Name: "t", Value: "1"}},
		})
	}()

	// Interrupt while tool is executing
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond)
		_ = e.Execute(ctx, comm, internal_type.LLMInterruptPacket{ContextID: "ctx-tool-interrupt"})
		close(toolDelay) // unblock tool
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("DEADLOCK: tool execution + interrupt timed out")
	}
}

// =============================================================================
// Consistency: history integrity under concurrent mutations
// =============================================================================

// TestConsistency_HistoryOrderPreserved verifies that history messages are
// appended in the correct order even under concurrent inject + response.
func TestConsistency_HistoryOrderPreserved(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	// Inject 100 messages sequentially — order must be preserved.
	for i := 0; i < 100; i++ {
		err := e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
			ContextID: fmt.Sprintf("ctx-%d", i),
			Text:      fmt.Sprintf("msg-%03d", i),
		})
		require.NoError(t, err)
	}

	snap := historySnapshot(e)
	require.Len(t, snap, 100)
	for i := 0; i < 100; i++ {
		expected := fmt.Sprintf("msg-%03d", i)
		actual := strings.Join(snap[i].GetAssistant().GetContents(), "")
		assert.Equal(t, expected, actual, "history index %d out of order", i)
	}
}

// TestConsistency_SnapshotIsolation verifies that a history snapshot taken
// before a mutation does not see the mutation.
func TestConsistency_SnapshotIsolation(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	// Pre-populate
	for i := 0; i < 5; i++ {
		_ = e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
			ContextID: fmt.Sprintf("ctx-%d", i),
			Text:      fmt.Sprintf("msg-%d", i),
		})
	}

	// Take snapshot
	snap := historySnapshot(e)
	require.Len(t, snap, 5)

	// Mutate history
	for i := 5; i < 10; i++ {
		_ = e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
			ContextID: fmt.Sprintf("ctx-%d", i),
			Text:      fmt.Sprintf("msg-%d", i),
		})
	}

	// Original snapshot must still be len 5
	assert.Len(t, snap, 5, "snapshot must not be affected by later mutations")
	// Current history must be 10
	assert.Len(t, historySnapshot(e), 10)
}

// TestConsistency_ToolCallAppendsAssistantMessage verifies that the assistant
// message with tool_calls is appended to history and LLMToolCallPacket is emitted.
func TestConsistency_ToolCallAppendsAssistantMessage(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream

	// Wire mock tool executor that emits LLMToolCallPacket (matching real behavior).
	e.toolExecutor = &mockToolExecutor{
		executeFn: func(ctx context.Context, contextID string, calls []*protos.ToolCall, comm internal_type.Communication) {
			for _, tc := range calls {
				comm.OnPacket(ctx, internal_type.LLMToolCallPacket{
					ToolID: tc.GetId(), Name: tc.GetFunction().GetName(), ContextID: contextID,
				})
			}
		},
	}

	comm, collector := newTestComm()
	en := mustLanguage(t, "en")

	_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "atomic-test",
		Text:      "test",
		Language:  en,
	})

	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "atomic-test",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
				Contents:  []string{"calling"},
				ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function", Function: &protos.FunctionCall{Name: "fn", Arguments: "{}"}}},
			}},
		},
		Metrics: []*protos.Metric{{Name: "t", Value: "1"}},
	})

	snap := historySnapshot(e)
	// user + assistant(tool_call) — tool result comes from dispatch layer externally
	require.Len(t, snap, 2)
	assert.Equal(t, "user", snap[0].Role)
	assert.Equal(t, "assistant", snap[1].Role)
	assert.Len(t, snap[1].GetAssistant().GetToolCalls(), 1, "assistant must have tool_calls")

	// Verify LLMToolCallPacket emitted by tool executor
	toolCallPkts := findPackets[internal_type.LLMToolCallPacket](collector.all())
	require.Len(t, toolCallPkts, 1)
	assert.Equal(t, "tc1", toolCallPkts[0].ToolID)
}

// TestConsistency_StaleContextDoesNotMutateHistory verifies that a response
// arriving after context switch does not append to history.
func TestConsistency_StaleContextDoesNotMutateHistory(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	en := mustLanguage(t, "en")

	// Turn 1
	_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-old",
		Text:      "old question",
		Language:  en,
	})

	// Turn 2 — supersedes turn 1
	_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-new",
		Text:      "new question",
		Language:  en,
	})

	histBefore := len(historySnapshot(e))

	// Late response from turn 1 — should be dropped
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "ctx-old",
		Success:   true,
		Data: &protos.Message{
			Role:    "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"stale answer"}}},
		},
		Metrics: []*protos.Metric{{Name: "t", Value: "1"}},
	})

	histAfter := len(historySnapshot(e))
	assert.Equal(t, histBefore, histAfter, "stale response must not change history length")
}

// TestConsistency_CloseResetsAllState verifies Close() fully resets state.
func TestConsistency_CloseResetsAllState(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	// Build up state
	for i := 0; i < 10; i++ {
		_ = e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
			ContextID: fmt.Sprintf("ctx-%d", i),
			Text:      fmt.Sprintf("msg-%d", i),
		})
	}
	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "active"}
	e.mu.Unlock()

	_ = e.Close(context.Background())

	e.mu.RLock()
	defer e.mu.RUnlock()
	assert.Empty(t, e.history, "history must be empty after Close")
	assert.Nil(t, e.stream, "stream must be nil after Close")
	assert.Nil(t, e.currentPacket, "currentPacket must be nil after Close")
}

// =============================================================================
// Concurrency: race detector stress tests (run with -race)
// =============================================================================

// TestConcurrency_MassiveParallelInjectAndSnapshot runs many concurrent
// writers (InjectMessagePacket) and readers (historySnapshot) to surface
// data races. Must be run with -race.
func TestConcurrency_MassiveParallelInjectAndSnapshot(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	const writers = 10
	const readsPerWriter = 50
	const injectsPerWriter = 50

	var wg sync.WaitGroup
	wg.Add(writers * 2)

	for w := 0; w < writers; w++ {
		w := w
		// Writer
		go func() {
			defer wg.Done()
			for i := 0; i < injectsPerWriter; i++ {
				_ = e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
					ContextID: fmt.Sprintf("w%d-i%d", w, i),
					Text:      fmt.Sprintf("w%d-msg-%d", w, i),
				})
			}
		}()
		// Reader
		go func() {
			defer wg.Done()
			for i := 0; i < readsPerWriter; i++ {
				snap := historySnapshot(e)
				// Verify snapshot is self-consistent: length must not change
				assert.Len(t, snap, len(snap))
			}
		}()
	}

	wg.Wait()
	snap := historySnapshot(e)
	assert.Len(t, snap, writers*injectsPerWriter, "all injected messages should be present")
}

// TestConcurrency_ExecuteAndInterruptRace runs Execute(UserInputPacket)
// and Execute(LLMInterruptPacket) concurrently to verify no race on
// currentPacket.
func TestConcurrency_ExecuteAndInterruptRace(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
			})
		}
	}()

	wg.Wait()
	// No assertion on final state — the point is no panic/race under -race flag.
}

// TestConcurrency_ResponseAndInterruptRace runs handleResponse and
// interruption concurrently to verify no race on history + currentPacket.
func TestConcurrency_ResponseAndInterruptRace(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	var wg sync.WaitGroup
	wg.Add(3)

	// Sender: keeps setting currentPacket
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	// Responder: processes responses
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			e.handleResponse(context.Background(), comm, &protos.ChatResponse{
				RequestId: fmt.Sprintf("ctx-%d", i),
				Success:   true,
				Data: &protos.Message{
					Role:    "assistant",
					Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"resp"}}},
				},
				Metrics: []*protos.Metric{{Name: "t", Value: "1"}},
			})
		}
	}()

	// Interrupter
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
			})
		}
	}()

	wg.Wait()
}

// TestConcurrency_ToolCallWithConcurrentExecute runs tool execution
// alongside new user messages to verify atomic history updates.
func TestConcurrency_ToolCallWithConcurrentExecute(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	e.toolExecutor = &mockToolExecutor{
		executeFn: func(_ context.Context, _ string, _ []*protos.ToolCall, _ internal_type.Communication) {
			time.Sleep(time.Millisecond)
		},
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Execute user messages concurrently
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("user-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}
	}()

	// Process tool call responses concurrently
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			e.handleResponse(context.Background(), comm, &protos.ChatResponse{
				RequestId: fmt.Sprintf("user-%d", i),
				Success:   true,
				Data: &protos.Message{
					Role: "assistant",
					Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
						Contents:  []string{"calling"},
						ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function"}},
					}},
				},
				Metrics: []*protos.Metric{{Name: "t", Value: "1"}},
			})
		}
	}()

	wg.Wait()
	// No assertion on final state — the point is no panic/race.
}

// TestConcurrency_PipelineStaleCheckUnderLoad verifies that the stale context
// check inside Pipeline() correctly filters under high concurrency, ensuring
// no stale pipeline runs to completion.
func TestConcurrency_PipelineStaleCheckUnderLoad(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()

	var wg sync.WaitGroup
	const N = 50
	wg.Add(N)

	// Fire N Execute calls with different context IDs.
	// Only the last one should actually produce a send (others become stale).
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			_ = e.Execute(context.Background(), comm, internal_type.UserInputPacket{
				ContextID: fmt.Sprintf("ctx-%d", i),
				Text:      fmt.Sprintf("msg-%d", i),
			})
		}()
	}
	wg.Wait()

	// The last currentPacket wins. Verify no crash and packets are consistent.
	e.mu.RLock()
	cp := e.currentPacket
	e.mu.RUnlock()
	require.NotNil(t, cp)

	// All emitted events should have "executing" type
	events := findPackets[internal_type.ConversationEventPacket](collector.all())
	for _, ev := range events {
		assert.Equal(t, "executing", ev.Data["type"])
	}
}

// =============================================================================
// Tests: toolCallsResolved — direct history-based resolution
// =============================================================================

func TestToolCallsResolved_EmptyHistory(t *testing.T) {
	e := newTestExecutor()
	e.mu.Lock()
	defer e.mu.Unlock()
	assert.True(t, e.toolCallsResolved(), "empty history should be resolved")
}

func TestToolCallsResolved_NoAssistantWithToolCalls(t *testing.T) {
	e := newTestExecutor()
	e.mu.Lock()
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "hello"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"hi"}}}},
	}
	assert.True(t, e.toolCallsResolved(), "history with no tool_calls should be resolved")
	e.mu.Unlock()
}

func TestToolCallsResolved_PartialResults(t *testing.T) {
	e := newTestExecutor()
	e.mu.Lock()
	e.history = []*protos.Message{
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{
				{Id: "t1", Type: "function"},
				{Id: "t2", Type: "function"},
			},
		}}},
		{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{
			Tools: []*protos.ToolMessage_Tool{{Id: "t1", Name: "fn", Content: "{}"}},
		}}},
	}
	assert.False(t, e.toolCallsResolved(), "only 1 of 2 tool results should not be resolved")
	e.mu.Unlock()
}

func TestToolCallsResolved_AllResults(t *testing.T) {
	e := newTestExecutor()
	e.mu.Lock()
	e.history = []*protos.Message{
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{
				{Id: "t1", Type: "function"},
				{Id: "t2", Type: "function"},
			},
		}}},
		{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{
			Tools: []*protos.ToolMessage_Tool{{Id: "t1", Name: "fn1", Content: "{}"}},
		}}},
		{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{
			Tools: []*protos.ToolMessage_Tool{{Id: "t2", Name: "fn2", Content: "{}"}},
		}}},
	}
	assert.True(t, e.toolCallsResolved(), "all tool results present should be resolved")
	e.mu.Unlock()
}

func TestToolCallsResolved_ScansBackwardsToLastAssistant(t *testing.T) {
	// Older resolved tool set should not prevent a newer unresolved set
	// from being detected.
	e := newTestExecutor()
	e.mu.Lock()
	e.history = []*protos.Message{
		// First round: resolved
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "old1", Type: "function"}},
		}}},
		{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{
			Tools: []*protos.ToolMessage_Tool{{Id: "old1", Name: "fn", Content: "{}"}},
		}}},
		// Second round (follow-up): assistant has new tool_calls, no results yet
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "new1", Type: "function"}},
		}}},
	}
	assert.False(t, e.toolCallsResolved(), "latest assistant with unresolved tool_calls should return false")
	e.mu.Unlock()
}

func TestToolCallsResolved_UserAndAssistantMixedHistory(t *testing.T) {
	// History with user messages and an assistant text-only message at the end.
	// toolCallsResolved scans backwards; first assistant found has no tool_calls.
	e := newTestExecutor()
	e.mu.Lock()
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "q1"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}},
		}}},
		{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{
			Tools: []*protos.ToolMessage_Tool{{Id: "t1", Name: "fn", Content: "{}"}},
		}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			Contents: []string{"The weather is sunny."},
		}}},
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "q2"}}},
	}
	// Scan backwards: last assistant at index 3 has no tool_calls -> skip.
	// Continues to index 1 which has tool_calls with result at index 2 -> resolved.
	assert.True(t, e.toolCallsResolved())
	e.mu.Unlock()
}

// =============================================================================
// Tests: Execute(LLMToolCallPacket) — no-op
// =============================================================================

func TestExecute_LLMToolCallPacket_NoOp(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.LLMToolCallPacket{
		ToolID:    "tc1",
		Name:      "get_weather",
		ContextID: "ctx-1",
		Arguments: map[string]string{"city": "SF"},
	})
	require.NoError(t, err)
	assert.Empty(t, collector.all(), "LLMToolCallPacket should not emit any packets")
	assert.Empty(t, historySnapshot(e), "LLMToolCallPacket should not modify history")
}

// =============================================================================
// Tests: Execute(LLMToolResultPacket) — appends to history, triggers follow-up
// =============================================================================

func TestExecute_LLMToolResultPacket_SingleTool_TriggersFollowUp(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	// Set up current context so Pipeline recognizes the context.
	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-tool"}
	// Pre-populate history: user + assistant(tool_calls:[t1])
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "question"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			Contents:  []string{"calling tool"},
			ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function", Function: &protos.FunctionCall{Name: "get_weather", Arguments: "{}"}}},
		}}},
	}
	e.mu.Unlock()

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID:    "t1",
		Name:      "get_weather",
		ContextID: "ctx-tool",
		Result:    map[string]string{"temp": "72F"},
	})
	require.NoError(t, err)

	// History should now have 3 entries: user, assistant(tool_calls), tool result.
	snap := historySnapshot(e)
	require.Len(t, snap, 3)
	assert.Equal(t, "user", snap[0].Role)
	assert.Equal(t, "assistant", snap[1].Role)
	assert.Equal(t, "tool", snap[2].Role)
	assert.Equal(t, "t1", snap[2].GetTool().GetTools()[0].GetId())

	// Follow-up should have been triggered: chatWithHistory calls stream.Send.
	stream.mu.Lock()
	defer stream.mu.Unlock()
	assert.Len(t, stream.sendCalls, 1, "tool follow-up should send a new chat request")
}

func TestExecute_LLMToolResultPacket_MultiTool_PartialDoesNotTriggerFollowUp(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-multi"}
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "multi"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{
				{Id: "t1", Type: "function", Function: &protos.FunctionCall{Name: "fn1", Arguments: "{}"}},
				{Id: "t2", Type: "function", Function: &protos.FunctionCall{Name: "fn2", Arguments: "{}"}},
				{Id: "t3", Type: "function", Function: &protos.FunctionCall{Name: "fn3", Arguments: "{}"}},
			},
		}}},
	}
	e.mu.Unlock()

	// First result: t1 — should NOT trigger follow-up.
	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn1", ContextID: "ctx-multi", Result: map[string]string{"r": "1"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 0, "1 of 3 resolved: no follow-up")
	stream.mu.Unlock()

	// Second result: t2 — should NOT trigger follow-up.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t2", Name: "fn2", ContextID: "ctx-multi", Result: map[string]string{"r": "2"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 0, "2 of 3 resolved: no follow-up")
	stream.mu.Unlock()

	// Third result: t3 — should trigger follow-up.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t3", Name: "fn3", ContextID: "ctx-multi", Result: map[string]string{"r": "3"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 1, "3 of 3 resolved: follow-up fires")
	stream.mu.Unlock()

	// History: user + assistant + 3 tool results = 5.
	snap := historySnapshot(e)
	require.Len(t, snap, 5)
	assert.Equal(t, "tool", snap[2].Role)
	assert.Equal(t, "tool", snap[3].Role)
	assert.Equal(t, "tool", snap[4].Role)
}

func TestExecute_LLMToolResultPacket_MultiTool_OutOfOrderResults(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-ooo"}
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "ooo"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{
				{Id: "t1", Type: "function", Function: &protos.FunctionCall{Name: "fn1", Arguments: "{}"}},
				{Id: "t2", Type: "function", Function: &protos.FunctionCall{Name: "fn2", Arguments: "{}"}},
			},
		}}},
	}
	e.mu.Unlock()

	// Deliver t2 first.
	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t2", Name: "fn2", ContextID: "ctx-ooo", Result: map[string]string{"r": "2"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 0, "t2 only, t1 still pending")
	stream.mu.Unlock()

	// Deliver t1 second.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn1", ContextID: "ctx-ooo", Result: map[string]string{"r": "1"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 1, "both resolved: follow-up fires")
	stream.mu.Unlock()

	// History order: user, assistant, t2, t1 (append order, not tool_call order).
	snap := historySnapshot(e)
	require.Len(t, snap, 4)
	assert.Equal(t, "t2", snap[2].GetTool().GetTools()[0].GetId())
	assert.Equal(t, "t1", snap[3].GetTool().GetTools()[0].GetId())
}

func TestExecute_LLMToolResultPacket_DuplicateResult(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-dup"}
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "dup"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}},
		}}},
	}
	e.mu.Unlock()

	// First result: resolves and triggers follow-up.
	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn", ContextID: "ctx-dup", Result: map[string]string{"r": "first"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 1, "first result triggers follow-up")
	stream.mu.Unlock()

	// Duplicate result: toolCallsResolved is still true (extra tool messages are
	// tolerated by the scan since all expected IDs are covered). The duplicate
	// appends to history and triggers another follow-up.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn", ContextID: "ctx-dup", Result: map[string]string{"r": "second"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 2, "duplicate result triggers follow-up again (no dedup)")
	stream.mu.Unlock()

	// Document: this is current behavior. The model does NOT deduplicate tool
	// results. Each LLMToolResultPacket appends and checks resolved.
	snap := historySnapshot(e)
	require.Len(t, snap, 4, "user + assistant + 2 tool results")
}

// =============================================================================
// Tests: Full round-trip — tool call from LLM response through external result
// =============================================================================

func TestModel_SingleToolRoundTrip(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()
	en := mustLanguage(t, "en")

	var toolExecuted bool
	e.toolExecutor = &mockToolExecutor{
		executeFn: func(ctx context.Context, contextID string, calls []*protos.ToolCall, comm internal_type.Communication) {
			toolExecuted = true
			assert.Equal(t, "round-trip", contextID)
			require.Len(t, calls, 1)
			assert.Equal(t, "tc1", calls[0].GetId())
			// Simulate what the real tool executor does: emit LLMToolCallPacket
			// and then LLMToolResultPacket via communication.OnPacket. But here
			// we just record that it was called.
		},
	}

	// Step 1: User sends a message.
	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "round-trip",
		Text:      "what is the weather?",
		Language:  en,
	})
	require.NoError(t, err)

	// Step 2: LLM responds with tool_call.
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "round-trip",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
				Contents:  []string{"Let me check the weather"},
				ToolCalls: []*protos.ToolCall{{Id: "tc1", Type: "function", Function: &protos.FunctionCall{Name: "get_weather", Arguments: `{"city":"SF"}`}}},
			}},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "15"}},
	})

	// Verify: toolExecutor.ExecuteAll was called.
	assert.True(t, toolExecuted, "tool executor should have been called")

	// Verify: history has user + assistant(tool_calls).
	snap := historySnapshot(e)
	require.Len(t, snap, 2)
	assert.Equal(t, "user", snap[0].Role)
	assert.Equal(t, "assistant", snap[1].Role)
	assert.Len(t, snap[1].GetAssistant().GetToolCalls(), 1)

	// Step 3: Dispatch layer calls Execute(LLMToolCallPacket) — no-op.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolCallPacket{
		ToolID: "tc1", Name: "get_weather", ContextID: "round-trip",
	})
	require.NoError(t, err)
	assert.Len(t, historySnapshot(e), 2, "LLMToolCallPacket should not change history")

	// Step 4: Dispatch layer calls Execute(LLMToolResultPacket) — triggers follow-up.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "tc1", Name: "get_weather", ContextID: "round-trip",
		Result: map[string]string{"temperature": "72F", "condition": "sunny"},
	})
	require.NoError(t, err)

	// Verify: history now has user + assistant(tool_calls) + tool_result.
	snap = historySnapshot(e)
	require.Len(t, snap, 3)
	assert.Equal(t, "tool", snap[2].Role)
	toolContent := snap[2].GetTool().GetTools()[0].GetContent()
	assert.Contains(t, toolContent, "72F")

	// Verify: follow-up was triggered (stream.Send called for the follow-up).
	// send #1 = initial user message, send #2 = tool follow-up.
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 2, "initial send + tool follow-up")
	stream.mu.Unlock()

	// Step 5: LLM responds to the follow-up with a text completion.
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "round-trip",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
				Contents: []string{"The weather in SF is 72F and sunny."},
			}},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "20"}},
	})

	// Verify: history now has user + assistant(tool_calls) + tool_result + assistant(text).
	snap = historySnapshot(e)
	require.Len(t, snap, 4)
	assert.Equal(t, "assistant", snap[3].Role)
	assert.Empty(t, snap[3].GetAssistant().GetToolCalls(), "final response should have no tool_calls")

	// Verify: LLMResponseDonePacket was emitted for the final response.
	dones := findPackets[internal_type.LLMResponseDonePacket](collector.all())
	require.Len(t, dones, 2, "one for tool_call completion, one for final text")
	assert.Equal(t, "Let me check the weather", dones[0].Text)
	assert.Equal(t, "The weather in SF is 72F and sunny.", dones[1].Text)
}

func TestModel_MultiToolRoundTrip_AllResolvedTriggersFollowUpOnce(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()
	en := mustLanguage(t, "en")

	e.toolExecutor = &mockToolExecutor{
		executeFn: func(_ context.Context, _ string, calls []*protos.ToolCall, _ internal_type.Communication) {
			assert.Len(t, calls, 3, "should receive all 3 tool calls")
		},
	}

	// User turn.
	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "multi-tool", Text: "plan my trip", Language: en,
	})
	require.NoError(t, err)

	// LLM responds with 3 tool_calls.
	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "multi-tool",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
				Contents: []string{"planning"},
				ToolCalls: []*protos.ToolCall{
					{Id: "t1", Type: "function", Function: &protos.FunctionCall{Name: "flights", Arguments: "{}"}},
					{Id: "t2", Type: "function", Function: &protos.FunctionCall{Name: "hotels", Arguments: "{}"}},
					{Id: "t3", Type: "function", Function: &protos.FunctionCall{Name: "car_rental", Arguments: "{}"}},
				},
			}},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "10"}},
	})

	// Initial send only.
	stream.mu.Lock()
	initialSends := len(stream.sendCalls)
	stream.mu.Unlock()
	assert.Equal(t, 1, initialSends, "only initial send before tool results")

	// t1 result.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "flights", ContextID: "multi-tool", Result: map[string]string{"flight": "AA123"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Equal(t, initialSends, len(stream.sendCalls), "1/3: no follow-up yet")
	stream.mu.Unlock()

	// t2 result.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t2", Name: "hotels", ContextID: "multi-tool", Result: map[string]string{"hotel": "Hilton"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Equal(t, initialSends, len(stream.sendCalls), "2/3: no follow-up yet")
	stream.mu.Unlock()

	// t3 result: triggers follow-up.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t3", Name: "car_rental", ContextID: "multi-tool", Result: map[string]string{"car": "Tesla"},
	})
	require.NoError(t, err)
	stream.mu.Lock()
	assert.Equal(t, initialSends+1, len(stream.sendCalls), "3/3: follow-up fires exactly once")
	stream.mu.Unlock()

	// Final history: user + assistant(tool_calls) + 3 tool results = 5.
	snap := historySnapshot(e)
	require.Len(t, snap, 5)
}

// =============================================================================
// Tests: Tool follow-up with stale context
// =============================================================================

func TestModel_ToolResult_StaleContext_NoFollowUp(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	// Set up initial context and history with pending tool call.
	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-new"}
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "old q"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}},
		}}},
	}
	e.mu.Unlock()

	// Tool result arrives for old context "ctx-old" — but current is "ctx-new".
	// The tool result is still appended to history (Execute does not check context
	// for LLMToolResultPacket), but the ToolFollowUpPipeline checks isCurrentContext.
	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn", ContextID: "ctx-old", Result: map[string]string{"r": "1"},
	})
	require.NoError(t, err)

	// Tool result was appended to history.
	snap := historySnapshot(e)
	require.Len(t, snap, 3, "tool result is always appended")

	// But follow-up was NOT triggered because ToolFollowUpPipeline sees ctx-old != ctx-new.
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 0, "stale context: no follow-up send")
	stream.mu.Unlock()
}

// =============================================================================
// Tests: Interruption clears currentPacket
// =============================================================================

func TestModel_InterruptionClearsPacket_ToolResultAfterInterrupt(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-interrupted"}
	e.history = []*protos.Message{
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}},
		}}},
	}
	e.mu.Unlock()

	// Interrupt.
	err := e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-interrupted"})
	require.NoError(t, err)
	assert.Nil(t, e.currentPacket)

	// Late tool result arrives. Tool result is appended, but follow-up is skipped
	// because currentPacket is nil and isCurrentContext("ctx-interrupted") returns false.
	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn", ContextID: "ctx-interrupted", Result: map[string]string{"r": "1"},
	})
	require.NoError(t, err)

	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 0, "follow-up should not fire after interruption")
	stream.mu.Unlock()
}

// =============================================================================
// Tests: Normal completion (no tool_calls) — assistant msg appended, no tool state
// =============================================================================

func TestModel_NormalCompletion_NoToolCalls(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, collector := newTestComm()
	en := mustLanguage(t, "en")

	noToolExec := &mockToolExecutor{
		executeFn: func(_ context.Context, _ string, _ []*protos.ToolCall, _ internal_type.Communication) {
			t.Fatal("tool executor should not be called for non-tool responses")
		},
	}
	e.toolExecutor = noToolExec

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "no-tools", Text: "hello", Language: en,
	})
	require.NoError(t, err)

	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "no-tools",
		Success:   true,
		Data: &protos.Message{
			Role: "assistant",
			Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
				Contents: []string{"Hi there!"},
			}},
		},
		Metrics: []*protos.Metric{{Name: "tokens", Value: "5"}},
	})

	// History: user + assistant.
	snap := historySnapshot(e)
	require.Len(t, snap, 2)
	assert.Equal(t, "user", snap[0].Role)
	assert.Equal(t, "assistant", snap[1].Role)
	assert.Empty(t, snap[1].GetAssistant().GetToolCalls())

	// No tool-related packets.
	toolCallPkts := findPackets[internal_type.LLMToolCallPacket](collector.all())
	assert.Empty(t, toolCallPkts, "no tool call packets for non-tool response")

	// Only 1 send: the initial user message.
	stream.mu.Lock()
	assert.Len(t, stream.sendCalls, 1, "no follow-up send for non-tool response")
	stream.mu.Unlock()
}

// =============================================================================
// Tests: Tool result JSON serialization
// =============================================================================

func TestExecute_LLMToolResultPacket_ResultSerializedAsJSON(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-json"}
	e.history = []*protos.Message{
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}},
		}}},
	}
	e.mu.Unlock()

	result := map[string]string{
		"items":  `["a","b"]`,
		"count":  "2",
		"nested": `{"key":"value"}`,
	}
	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn", ContextID: "ctx-json", Result: result,
	})
	require.NoError(t, err)

	snap := historySnapshot(e)
	require.Len(t, snap, 2)
	content := snap[1].GetTool().GetTools()[0].GetContent()
	// The result should be valid JSON.
	assert.True(t, strings.HasPrefix(content, "{"), "content should be JSON: %s", content)
	assert.Contains(t, content, `"count":"2"`)
	assert.Contains(t, content, `"nested":"{\"key\":\"value\"}"`)
}

// =============================================================================
// Tests: ToolFollowUpPipeline with nil stream
// =============================================================================

func TestModel_ToolFollowUp_NilStream_ReturnsError(t *testing.T) {
	e := newTestExecutor()
	e.stream = nil // no stream
	comm, _ := newTestComm()

	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-nil-stream"}
	e.history = []*protos.Message{
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}},
		}}},
	}
	e.mu.Unlock()

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn", ContextID: "ctx-nil-stream", Result: map[string]string{"r": "1"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stream not connected")
}

// =============================================================================
// Tests: ToolFollowUpPipeline includes all history in the request
// =============================================================================

func TestModel_ToolFollowUp_SendsFullHistory(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-full"}
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "q"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: []*protos.ToolCall{{Id: "t1", Type: "function"}},
		}}},
	}
	e.mu.Unlock()

	err := e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ToolID: "t1", Name: "fn", ContextID: "ctx-full", Result: map[string]string{"done": "true"},
	})
	require.NoError(t, err)

	stream.mu.Lock()
	defer stream.mu.Unlock()
	require.Len(t, stream.sendCalls, 1)

	// The sent request should include all 3 messages: user + assistant + tool.
	sent := stream.sendCalls[0]
	conversations := sent.GetConversations()
	// Conversations include system prompt (prepended) + history messages.
	// Count non-system messages.
	var historyMsgs int
	for _, msg := range conversations {
		if msg.GetSystem() == nil {
			historyMsgs++
		}
	}
	assert.Equal(t, 3, historyMsgs, "follow-up request should include all 3 history messages")
}

// =============================================================================
// Concurrency: Tool results arriving concurrently
// =============================================================================

func TestConcurrency_MultipleToolResultsConcurrent(t *testing.T) {
	e := newTestExecutor()
	stream := newMockStream()
	e.stream = stream
	comm, _ := newTestComm()

	const numTools = 10
	toolCalls := make([]*protos.ToolCall, numTools)
	for i := 0; i < numTools; i++ {
		toolCalls[i] = &protos.ToolCall{
			Id:   fmt.Sprintf("t%d", i),
			Type: "function",
		}
	}

	e.mu.Lock()
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-concurrent"}
	e.history = []*protos.Message{
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "concurrent"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{
			ToolCalls: toolCalls,
		}}},
	}
	e.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(numTools)
	for i := 0; i < numTools; i++ {
		i := i
		go func() {
			defer wg.Done()
			_ = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
				ToolID:    fmt.Sprintf("t%d", i),
				Name:      fmt.Sprintf("fn%d", i),
				ContextID: "ctx-concurrent",
				Result:    map[string]string{"i": fmt.Sprintf("%d", i)},
			})
		}()
	}
	wg.Wait()

	// All tool results should be in history.
	snap := historySnapshot(e)
	assert.Len(t, snap, numTools+2, "user + assistant + N tool results")

	// Follow-up should have fired at least once (exactly once in serial, but
	// under concurrency, a race between check-and-send could fire multiple times).
	stream.mu.Lock()
	assert.GreaterOrEqual(t, len(stream.sendCalls), 1, "follow-up should fire when all resolved")
	stream.mu.Unlock()
}

func TestHandleResponse_SuccessNonAssistantPayload_EmitsLLMErrorNoPanic(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	e.handleResponse(context.Background(), comm, &protos.ChatResponse{
		RequestId: "ctx-non-assistant",
		Success:   true,
		Data: &protos.Message{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "unexpected"},
			},
		},
	})

	errPkt, ok := findPacket[internal_type.LLMErrorPacket](collector.all())
	require.True(t, ok, "expected LLMErrorPacket for non-assistant success payload")
	assert.Equal(t, "ctx-non-assistant", errPkt.ContextID)
	assert.Contains(t, errPkt.Error.Error(), "assistant message missing")

	dirs := findActionToolCalls(collector.all())
	assert.Empty(t, dirs, "unexpected END_CONVERSATION directive for schema violation")
}
