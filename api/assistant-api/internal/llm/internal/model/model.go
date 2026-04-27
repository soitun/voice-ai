// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	internal_agent_tool "github.com/rapidaai/api/assistant-api/internal/tool"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/variable"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/parsers"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type modelAssistantExecutor struct {
	logger             commons.Logger
	toolExecutor       internal_agent_tool.ToolExecutor
	providerCredential *protos.VaultCredential
	inputBuilder       integration_client_builders.InputChatBuilder
	history            *ConversationHistory
	stream             grpc.BidiStreamingClient[protos.ChatRequest, protos.ChatResponse]

	mu            sync.RWMutex
	currentPacket *internal_type.UserInputPacket

	ctx       context.Context
	ctxCancel context.CancelFunc
}

func NewModelAssistantExecutor(logger commons.Logger) *modelAssistantExecutor {
	return &modelAssistantExecutor{
		logger:       logger,
		inputBuilder: integration_client_builders.NewChatInputBuilder(logger),
		toolExecutor: internal_agent_tool.NewToolExecutor(logger),
		history:      NewConversationHistory(),
	}
}

func (e *modelAssistantExecutor) Name() string { return "model" }

// =============================================================================
// Initialize / Close
// =============================================================================

func (e *modelAssistantExecutor) Initialize(ctx context.Context, communication internal_type.Communication, cfg *protos.ConversationInitialization) error {
	start := time.Now()
	g, gCtx := errgroup.WithContext(ctx)
	var providerCredential *protos.VaultCredential

	g.Go(func() error {
		credentialID, err := communication.Assistant().AssistantProviderModel.GetOptions().GetUint64("rapida.credential_id")
		if err != nil {
			return fmt.Errorf("failed to get credential ID: %w", err)
		}
		cred, err := communication.VaultCaller().GetCredential(gCtx, communication.Auth(), credentialID)
		if err != nil {
			return fmt.Errorf("failed to get provider credential: %w", err)
		}
		providerCredential = cred
		return nil
	})
	g.Go(func() error {
		return e.toolExecutor.Initialize(gCtx, communication)
	})
	if err := g.Wait(); err != nil {
		return err
	}

	e.providerCredential = providerCredential
	stream, err := communication.IntegrationCaller().StreamChat(
		ctx, communication.Auth(),
		communication.Assistant().AssistantProviderModel.ModelProviderName,
	)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	e.stream = stream
	e.ctx, e.ctxCancel = context.WithCancel(ctx)
	utils.Go(e.ctx, func() { e.listen(e.ctx, communication) })

	llmData := communication.Assistant().AssistantProviderModel.GetOptions().ToStringMap()
	llmData["type"] = "llm_initialized"
	llmData["provider"] = communication.Assistant().AssistantProviderModel.ModelProviderName
	llmData["init_ms"] = fmt.Sprintf("%d", time.Since(start).Milliseconds())
	communication.OnPacket(ctx, internal_type.ConversationEventPacket{Name: "llm", Data: llmData, Time: time.Now()})
	return nil
}

func (e *modelAssistantExecutor) Close(ctx context.Context) error {
	if e.ctxCancel != nil {
		e.ctxCancel()
	}
	e.mu.Lock()
	if e.stream != nil {
		e.stream.CloseSend()
		e.stream = nil
	}
	e.currentPacket = nil
	e.mu.Unlock()
	e.history.Reset()
	if e.toolExecutor != nil {
		if err := e.toolExecutor.Close(ctx); err != nil {
			e.logger.Errorf("error closing tool executor: %v", err)
		}
	}
	return nil
}

// =============================================================================
// Execute — maps incoming packets to pipeline types
// =============================================================================

func (e *modelAssistantExecutor) Execute(ctx context.Context, communication internal_type.Communication, pctk internal_type.Packet) error {
	switch p := pctk.(type) {
	case internal_type.UserInputPacket:
		if supersededCtx := e.history.SupersedePending(); supersededCtx != "" {
			communication.OnPacket(ctx, internal_type.ConversationEventPacket{
				ContextID: supersededCtx, Name: "tool", Time: time.Now(),
				Data: map[string]string{"type": "tool_block_superseded", "reason": "user_interrupted"},
			})
		}
		e.mu.Lock()
		e.currentPacket = &p
		e.mu.Unlock()
		e.Run(ctx, communication, UserTurnPipeline{Packet: p})

	case internal_type.InjectMessagePacket:
		e.Run(ctx, communication, InjectMessagePipeline{Packet: p})

	case internal_type.LLMToolCallPacket:
		// no-op: dispatch handles logging/notification

	case internal_type.LLMToolResultPacket:
		e.Run(ctx, communication, ToolResultPipeline{Packet: p})

	case internal_type.LLMInterruptPacket:
		e.Run(ctx, communication, InterruptionPipeline{Packet: p})

	default:
		e.logger.Errorf("unsupported packet type: %T", pctk)
	}
	return nil
}

// =============================================================================
// Run — central pipeline dispatch
// =============================================================================

func (e *modelAssistantExecutor) Run(ctx context.Context, communication internal_type.Communication, p AgentPipeline) {
	switch v := p.(type) {
	case UserTurnPipeline:
		e.handleUserTurn(ctx, communication, v.Packet)
	case InjectMessagePipeline:
		e.history.AppendInjected(v.Packet.Text)
	case ToolResultPipeline:
		e.handleToolResult(ctx, communication, v.Packet)
	case InterruptionPipeline:
		e.handleInterruption()
	case ResponsePipeline:
		e.handleResponse(ctx, communication, v.Response)
	case ToolFollowUpPipeline:
		e.handleToolFollowUp(ctx, communication, v.ContextID)
	default:
		e.logger.Errorf("unknown pipeline type: %T", p)
	}
}

// =============================================================================
// Pipeline handlers
// =============================================================================

func (e *modelAssistantExecutor) handleUserTurn(ctx context.Context, communication internal_type.Communication, p internal_type.UserInputPacket) {
	snapshot := e.history.Snapshot()
	promptArgs := e.buildPromptArgs(communication, p)

	if err := e.validateHistorySequence(snapshot); err != nil {
		communication.OnPacket(ctx, internal_type.LLMErrorPacket{ContextID: p.ContextID, Error: fmt.Errorf("history integrity: %w", err)})
		return
	}

	communication.OnPacket(ctx, internal_type.ConversationEventPacket{
		ContextID: p.ContextID, Name: "llm", Time: time.Now(),
		Data: map[string]string{
			"type": "executing", "script": p.Text,
			"input_char_count": fmt.Sprintf("%d", len(p.Text)),
			"history_count":    fmt.Sprintf("%d", len(snapshot)),
		},
	})

	userMsg := &protos.Message{
		Role:    "user",
		Message: &protos.Message_User{User: &protos.UserMessage{Content: p.Text}},
	}
	if err := e.sendChat(communication, p.ContextID, promptArgs, append(snapshot, userMsg)...); err != nil {
		communication.OnPacket(ctx, internal_type.LLMErrorPacket{ContextID: p.ContextID, Error: err})
		return
	}
	e.history.AppendUser(p.Text)
}

func (e *modelAssistantExecutor) handleToolResult(ctx context.Context, communication internal_type.Communication, p internal_type.LLMToolResultPacket) {
	resultJSON, _ := json.Marshal(p.Result)
	accepted, resolved := e.history.AcceptToolResult(p.ContextID, p.ToolID, p.Name, string(resultJSON))
	if !accepted {
		pendingCtx := e.history.PendingContextID()
		reason := "no_pending_block"
		data := map[string]string{"type": "tool_result_ignored", "reason": reason, "tool_id": p.ToolID}
		if pendingCtx != "" {
			reason = "context_or_id_mismatch"
			data["reason"] = reason
			data["pending_context"] = pendingCtx
		}
		communication.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: p.ContextID, Name: "tool", Time: time.Now(), Data: data,
		})
		return
	}
	if !resolved {
		return
	}

	contextID, followUp := e.history.FlushToolBlock()
	if !followUp {
		communication.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: contextID, Name: "tool", Time: time.Now(),
			Data: map[string]string{"type": "tool_block_discarded", "reason": "superseded"},
		})
		return
	}
	e.Run(ctx, communication, ToolFollowUpPipeline{ContextID: contextID})
}

func (e *modelAssistantExecutor) handleInterruption() {
	e.history.SupersedePending()
}

func (e *modelAssistantExecutor) handleResponse(ctx context.Context, communication internal_type.Communication, resp *protos.ChatResponse) {
	if e.isStaleResponse(resp.GetRequestId()) {
		return
	}
	contextID := resp.GetRequestId()

	if !resp.GetSuccess() && resp.GetError() != nil {
		errMsg := resp.GetError().GetErrorMessage()
		communication.OnPacket(ctx,
			internal_type.LLMErrorPacket{ContextID: contextID, Error: errors.New(errMsg)},
			internal_type.ConversationEventPacket{ContextID: contextID, Name: "llm", Data: map[string]string{"type": "error", "error": errMsg}, Time: time.Now()},
		)
		return
	}

	output := resp.GetData()
	if output == nil || output.GetAssistant() == nil {
		return
	}

	if len(resp.GetMetrics()) == 0 {
		e.onStreamingChunk(ctx, communication, contextID, output)
		return
	}
	e.onCompletion(ctx, communication, contextID, resp.GetFinishReason(), output, resp.GetMetrics())
}

func (e *modelAssistantExecutor) handleToolFollowUp(ctx context.Context, communication internal_type.Communication, contextID string) {
	snapshot := e.history.Snapshot()

	e.mu.RLock()
	stream := e.stream
	e.mu.RUnlock()
	if stream == nil {
		e.logger.Errorf("stream not connected for tool follow-up")
		return
	}
	if err := e.validateHistorySequence(snapshot); err != nil {
		e.logger.Errorf("history integrity failed, blocking tool follow-up: %v", err)
		return
	}
	promptArgs := e.buildBasePromptArgs(communication)
	if err := stream.Send(e.buildChatRequest(communication, contextID, promptArgs, snapshot...)); err != nil {
		e.logger.Errorf("tool follow-up send failed: %v", err)
	}
}

// =============================================================================
// Stream I/O
// =============================================================================

func (e *modelAssistantExecutor) sendChat(
	communication internal_type.Communication,
	contextID string,
	promptArgs map[string]interface{},
	messages ...*protos.Message,
) error {
	e.mu.RLock()
	stream := e.stream
	e.mu.RUnlock()
	if stream == nil {
		return fmt.Errorf("stream not connected")
	}
	return stream.Send(e.buildChatRequest(communication, contextID, promptArgs, messages...))
}

func (e *modelAssistantExecutor) listen(ctx context.Context, communication internal_type.Communication) {
	for {
		e.mu.RLock()
		stream := e.stream
		e.mu.RUnlock()
		if stream == nil {
			return
		}
		resp, err := stream.Recv()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			communication.OnPacket(ctx, internal_type.LLMErrorPacket{
				ContextID: e.currentContextID(),
				Error:     err,
				Type:      internal_type.LLMSystemPanic,
			})
			return
		}
		e.Run(ctx, communication, ResponsePipeline{Response: resp})
	}
}

// =============================================================================
// Response sub-handlers
// =============================================================================

func (e *modelAssistantExecutor) onStreamingChunk(ctx context.Context, communication internal_type.Communication, contextID string, output *protos.Message) {
	text := strings.Join(output.GetAssistant().GetContents(), "")
	communication.OnPacket(ctx,
		internal_type.LLMResponseDeltaPacket{ContextID: contextID, Text: text},
		internal_type.ConversationEventPacket{ContextID: contextID, Name: "llm", Time: time.Now(),
			Data: map[string]string{"type": "chunk", "text": text, "response_char_count": fmt.Sprintf("%d", len(text))},
		},
	)
}

func (e *modelAssistantExecutor) onCompletion(ctx context.Context, communication internal_type.Communication, contextID, finishReason string, output *protos.Message, metrics []*protos.Metric) {
	assistant := output.GetAssistant()
	responseText := strings.Join(assistant.GetContents(), "")
	toolCalls := assistant.GetToolCalls()

	supersededCtx := e.history.AppendAssistant(contextID, output)
	if supersededCtx != "" {
		e.logger.Errorf("new tool block while previous unresolved (context=%s), superseding", supersededCtx)
		communication.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: supersededCtx, Name: "tool", Time: time.Now(),
			Data: map[string]string{"type": "tool_block_superseded", "reason": "new_tool_block"},
		})
	}

	communication.OnPacket(ctx,
		internal_type.LLMResponseDonePacket{ContextID: contextID, Text: responseText},
		internal_type.ConversationEventPacket{ContextID: contextID, Name: "llm", Time: time.Now(),
			Data: map[string]string{
				"type": "completed", "text": responseText,
				"response_char_count": fmt.Sprintf("%d", len(responseText)),
				"finish_reason":       finishReason,
			},
		},
		internal_type.AssistantMessageMetricPacket{ContextID: contextID, Metrics: e.buildCompletionMetrics(metrics)},
	)

	if len(toolCalls) > 0 {
		e.toolExecutor.ExecuteAll(ctx, contextID, toolCalls, communication)
	}
}

// =============================================================================
// Context state
// =============================================================================

func (e *modelAssistantExecutor) currentContextID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.currentPacket == nil {
		return ""
	}
	return e.currentPacket.ContextID
}

func (e *modelAssistantExecutor) isStaleResponse(requestID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.currentPacket == nil {
		return true
	}
	return requestID != e.currentPacket.ContextId()
}

// =============================================================================
// Metrics
// =============================================================================

func (e *modelAssistantExecutor) buildCompletionMetrics(providerMetrics []*protos.Metric) []*protos.Metric {
	out := make([]*protos.Metric, 0, len(providerMetrics)+1)
	for _, m := range providerMetrics {
		out = append(out, &protos.Metric{
			Name: "agent_" + m.GetName(), Value: m.GetValue(), Description: m.GetDescription(),
		})
		if m.GetName() == "time_to_first_token" {
			if ns, err := strconv.ParseInt(m.GetValue(), 10, 64); err == nil {
				out = append(out, &protos.Metric{
					Name: "llm_latency_ms", Value: fmt.Sprintf("%d", ns/int64(time.Millisecond)),
				})
			}
		}
	}
	return out
}

// =============================================================================
// Prompt argumentation
// =============================================================================

func (e *modelAssistantExecutor) buildPromptArgs(communication internal_type.Communication, p internal_type.UserInputPacket) map[string]interface{} {
	return utils.MergeMaps(e.buildBasePromptArgs(communication), map[string]interface{}{"message": map[string]interface{}{
		"text": p.Text, "language_code": p.Language.ISO639_1, "language": p.Language.Name,
	}})
}

// buildBasePromptArgs builds the nested prompt-argument map consumed by the
// LLM template engine. Resolution is delegated to the shared variable
// resolver — see api/assistant-api/internal/variable. The message.* sub-tree
// is per-message and stays here; buildPromptArgs overlays it on top.
func (e *modelAssistantExecutor) buildBasePromptArgs(communication internal_type.Communication) map[string]interface{} {
	registry := variable.NewDefaultRegistry()
	src := variable.NewCommunicationSource(communication)
	out := registry.Expand(src, variable.ResolveContext{})
	out["message"] = map[string]interface{}{"language": "English"}
	return out
}

// =============================================================================
// Chat request builder
// =============================================================================

func (e *modelAssistantExecutor) buildChatRequest(communication internal_type.Communication, contextID string, promptArgs map[string]interface{}, messages ...*protos.Message) *protos.ChatRequest {
	assistant := communication.Assistant()
	template := assistant.AssistantProviderModel.Template.GetTextChatCompleteTemplate()
	defaultArgs := parsers.CanonicalizePromptArguments(e.inputBuilder.PromptArguments(template.Variables))
	runtimeArgs := parsers.CanonicalizePromptArguments(promptArgs)
	systemMessages := e.inputBuilder.Message(template.Prompt, utils.MergeMaps(defaultArgs, runtimeArgs))
	req := e.inputBuilder.Chat(
		contextID,
		&protos.Credential{Id: e.providerCredential.GetId(), Value: e.providerCredential.GetValue()},
		e.inputBuilder.Options(utils.MergeMaps(assistant.AssistantProviderModel.GetOptions(), communication.GetOptions()), nil),
		e.toolExecutor.GetFunctionDefinitions(),
		map[string]string{
			"assistant_id":                fmt.Sprintf("%d", assistant.Id),
			"message_id":                  contextID,
			"assistant_provider_model_id": fmt.Sprintf("%d", assistant.AssistantProviderModel.Id),
		},
		append(systemMessages, messages...)...,
	)
	req.ProviderName = strings.ToLower(assistant.AssistantProviderModel.ModelProviderName)
	return req
}

// =============================================================================
// History validation
// =============================================================================

func (e *modelAssistantExecutor) validateHistorySequence(messages []*protos.Message) error {
	for i, msg := range messages {
		if ast := msg.GetAssistant(); ast != nil && len(ast.GetToolCalls()) > 0 {
			if i+1 >= len(messages) || messages[i+1].GetTool() == nil {
				return fmt.Errorf("history: assistant tool_call at %d not followed by tool response", i)
			}
			if err := e.validateToolIDMatch(ast.GetToolCalls(), messages[i+1].GetTool().GetTools(), i); err != nil {
				return err
			}
		}
		if tool := msg.GetTool(); tool != nil {
			if i == 0 {
				return fmt.Errorf("history: orphan tool response at %d", i)
			}
			prev := messages[i-1].GetAssistant()
			if prev == nil || len(prev.GetToolCalls()) == 0 {
				return fmt.Errorf("history: orphan tool response at %d", i)
			}
		}
	}
	return nil
}

func (e *modelAssistantExecutor) validateToolIDMatch(calls []*protos.ToolCall, tools []*protos.ToolMessage_Tool, idx int) error {
	expected := make(map[string]struct{}, len(calls))
	for _, c := range calls {
		if id := strings.TrimSpace(c.GetId()); id != "" {
			expected[id] = struct{}{}
		}
	}
	for _, t := range tools {
		id := strings.TrimSpace(t.GetId())
		if _, ok := expected[id]; !ok {
			return fmt.Errorf("history: orphan tool result %q at assistant %d", id, idx)
		}
		delete(expected, id)
	}
	for id := range expected {
		return fmt.Errorf("history: missing tool result for %q at assistant %d", id, idx)
	}
	return nil
}
