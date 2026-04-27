// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_agentkit

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type agentkitExecutor struct {
	logger     commons.Logger
	connection *grpc.ClientConn
	talker     grpc.BidiStreamingClient[protos.TalkInput, protos.TalkOutput]
	mu         sync.RWMutex
	currentID  string
	done       chan struct{}
}

func NewAgentKitAssistantExecutor(logger commons.Logger) *agentkitExecutor {
	return &agentkitExecutor{logger: logger}
}

func (e *agentkitExecutor) Name() string { return "agentkit" }

// =============================================================================
// Initialize / Close
// =============================================================================

func (e *agentkitExecutor) Initialize(ctx context.Context, comm internal_type.Communication, cfg *protos.ConversationInitialization) error {
	start := time.Now()
	provider := comm.Assistant().AssistantProviderAgentkit
	if provider == nil {
		return fmt.Errorf("agentkit provider is not enabled")
	}
	if err := e.connect(ctx, provider); err != nil {
		return err
	}

	e.done = make(chan struct{})
	utils.Go(ctx, func() {
		defer close(e.done)
		e.listen(ctx, comm)
	})

	if err := e.sendInitialization(provider.AssistantId, provider.Id, comm.Conversation().Id, cfg); err != nil {
		return fmt.Errorf("failed to send initialization: %w", err)
	}

	comm.OnPacket(ctx, internal_type.ConversationEventPacket{
		Name: "agentkit",
		Data: map[string]string{
			"type": "agentkit_initialized", "provider": "agentkit",
			"url": provider.Url, "init_ms": fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})
	return nil
}

func (e *agentkitExecutor) Close(ctx context.Context) error {
	e.mu.Lock()
	if e.talker != nil {
		e.talker.CloseSend()
		e.talker = nil
	}
	if e.connection != nil {
		e.connection.Close()
		e.connection = nil
	}
	done := e.done
	e.currentID = ""
	e.mu.Unlock()

	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			e.logger.Errorf("timed out waiting for listener goroutine to exit")
		}
	}
	return nil
}

// =============================================================================
// Execute — maps incoming packets to pipeline types
// =============================================================================

func (e *agentkitExecutor) Execute(ctx context.Context, comm internal_type.Communication, pctk internal_type.Packet) error {
	switch p := pctk.(type) {
	case internal_type.UserInputPacket:
		return e.Run(ctx, comm, UserTurnPipeline{Packet: p})
	case internal_type.UserTextReceivedPacket:
		return e.Run(ctx, comm, UserTextPipeline{Packet: p})
	case internal_type.InjectMessagePacket:
		return e.Run(ctx, comm, InjectMessagePipeline{Packet: p})
	case internal_type.LLMInterruptPacket:
		return e.Run(ctx, comm, InterruptionPipeline{Packet: p})
	default:
		return fmt.Errorf("unsupported packet type: %T", pctk)
	}
}

// =============================================================================
// Run — central pipeline dispatch
// =============================================================================

func (e *agentkitExecutor) Run(ctx context.Context, comm internal_type.Communication, p AgentPipeline) error {
	switch v := p.(type) {
	case UserTurnPipeline:
		return e.handleUserTurn(ctx, comm, v.Packet.ContextID, v.Packet.Text)
	case UserTextPipeline:
		return e.handleUserTurn(ctx, comm, v.Packet.ContextID, v.Packet.Text)
	case InjectMessagePipeline:
		// no-op: external agent manages its own history
		return nil
	case InterruptionPipeline:
		e.handleInterruption()
		return nil
	case ResponsePipeline:
		e.handleResponse(ctx, comm, v.Response)
		return nil
	default:
		return fmt.Errorf("unknown pipeline type: %T", p)
	}
}

// =============================================================================
// Pipeline handlers
// =============================================================================

func (e *agentkitExecutor) handleUserTurn(ctx context.Context, comm internal_type.Communication, contextID, text string) error {
	if strings.TrimSpace(contextID) == "" {
		return nil
	}
	e.mu.Lock()
	e.currentID = contextID
	e.mu.Unlock()

	comm.OnPacket(ctx, internal_type.ConversationEventPacket{
		ContextID: contextID, Name: "agentkit", Time: time.Now(),
		Data: map[string]string{
			"type": "executing", "script": text,
			"input_char_count": fmt.Sprintf("%d", len(text)),
		},
	})
	if err := e.send(&protos.TalkInput{
		Request: &protos.TalkInput_Message{
			Message: &protos.ConversationUserMessage{
				Message:   &protos.ConversationUserMessage_Text{Text: text},
				Id:        contextID,
				Completed: true,
				Time:      timestamppb.Now(),
			},
		},
	}); err != nil {
		return err
	}
	return nil
}

func (e *agentkitExecutor) handleInterruption() {
	e.mu.Lock()
	e.currentID = ""
	e.mu.Unlock()
}

func (e *agentkitExecutor) handleResponse(ctx context.Context, comm internal_type.Communication, resp *protos.TalkOutput) {
	switch data := resp.GetData().(type) {
	case *protos.TalkOutput_Initialization:
		comm.OnPacket(ctx, internal_type.ConversationEventPacket{
			Name: "agentkit", Time: time.Now(),
			Data: map[string]string{
				"type":            "initialization_ack",
				"conversation_id": fmt.Sprintf("%d", data.Initialization.GetAssistantConversationId()),
			},
		})

	case *protos.TalkOutput_Interruption:
		if !e.isCurrentContext(data.Interruption.GetId()) {
			return
		}
		comm.OnPacket(ctx,
			internal_type.InterruptionDetectedPacket{ContextID: data.Interruption.Id, Source: internal_type.InterruptionSourceWord},
			internal_type.ConversationEventPacket{
				ContextID: data.Interruption.Id, Name: "agentkit", Time: time.Now(),
				Data: map[string]string{"type": "interruption", "source": "word"},
			},
		)

	case *protos.TalkOutput_Assistant:
		if !e.isCurrentContext(data.Assistant.GetId()) {
			return
		}
		contextID := data.Assistant.GetId()
		switch msg := data.Assistant.GetMessage().(type) {
		case *protos.ConversationAssistantMessage_Text:
			if data.Assistant.GetCompleted() {
				comm.OnPacket(ctx,
					internal_type.LLMResponseDonePacket{ContextID: contextID, Text: msg.Text},
					internal_type.ConversationEventPacket{
						ContextID: contextID, Name: "agentkit", Time: time.Now(),
						Data: map[string]string{
							"type": "completed", "text": msg.Text,
							"response_char_count": fmt.Sprintf("%d", len(msg.Text)),
						},
					},
				)
			} else {
				comm.OnPacket(ctx,
					internal_type.LLMResponseDeltaPacket{ContextID: contextID, Text: msg.Text},
					internal_type.ConversationEventPacket{
						ContextID: contextID, Name: "agentkit", Time: time.Now(),
						Data: map[string]string{
							"type": "chunk", "text": msg.Text,
							"response_char_count": fmt.Sprintf("%d", len(msg.Text)),
						},
					},
				)
			}
		}

	case *protos.TalkOutput_ToolCall:
		tc := data.ToolCall
		if !e.isCurrentContext(tc.GetId()) {
			return
		}
		comm.OnPacket(ctx, internal_type.LLMToolCallPacket{
			ContextID: tc.GetId(), ToolID: tc.GetToolId(),
			Name: tc.GetName(), Action: tc.GetAction(), Arguments: tc.GetArgs(),
		})

	case *protos.TalkOutput_ToolCallResult:
		tr := data.ToolCallResult
		if !e.isCurrentContext(tr.GetId()) {
			return
		}
		comm.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: tr.GetId(), Name: "tool", Time: time.Now(),
			Data: map[string]string{
				"type": "tool_result", "tool_id": tr.GetToolId(),
				"name": tr.GetName(), "action": tr.GetAction().String(),
			},
		})

	case *protos.TalkOutput_Error:
		errMsg := data.Error.GetErrorMessage()
		e.logger.Errorf("AgentKit agent error: code=%d message=%s", data.Error.GetErrorCode(), errMsg)
		comm.OnPacket(ctx,
			internal_type.LLMErrorPacket{
				Error: fmt.Errorf("agentkit error %d: %s", data.Error.GetErrorCode(), errMsg),
				Type:  internal_type.LLMSystemPanic,
			},
			internal_type.ConversationEventPacket{
				Name: "agentkit", Time: time.Now(),
				Data: map[string]string{
					"type": "error", "error": errMsg,
					"code": fmt.Sprintf("%d", data.Error.GetErrorCode()),
				},
			},
			internal_type.LLMToolCallPacket{
				ContextID: e.currentContextID(),
				Name:      "end_conversation",
				Action:    protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
				Arguments: map[string]string{"reason": errMsg},
			},
		)
	}
}

// =============================================================================
// Context state
// =============================================================================

func (e *agentkitExecutor) isCurrentContext(id string) bool {
	clean := strings.TrimSpace(id)
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.currentID == "" {
		return true
	}
	if clean == "" {
		return true
	}
	return clean == e.currentID
}

func (e *agentkitExecutor) currentContextID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentID
}

// =============================================================================
// Stream I/O
// =============================================================================

func (e *agentkitExecutor) send(req *protos.TalkInput) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.talker == nil {
		return fmt.Errorf("not connected")
	}
	return e.talker.Send(req)
}

func (e *agentkitExecutor) sendInitialization(assistantId, assistantProviderID, conversationID uint64, cfg *protos.ConversationInitialization) error {
	return e.send(&protos.TalkInput{
		Request: &protos.TalkInput_Initialization{
			Initialization: &protos.ConversationInitialization{
				AssistantConversationId: conversationID,
				Assistant: &protos.AssistantDefinition{
					AssistantId: assistantId,
					Version:     utils.GetVersionString(assistantProviderID),
				},
				Args: cfg.GetArgs(), Metadata: cfg.GetMetadata(),
				Options: cfg.GetOptions(), StreamMode: cfg.GetStreamMode(),
				UserIdentity: cfg.GetUserIdentity(), Time: timestamppb.Now(),
			},
		},
	})
}

func (e *agentkitExecutor) listen(ctx context.Context, comm internal_type.Communication) {
	for {
		if ctx.Err() != nil {
			return
		}

		e.mu.RLock()
		talker := e.talker
		e.mu.RUnlock()
		if talker == nil {
			return
		}

		resp, err := talker.Recv()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			reason := e.streamErrorReason(err)
			comm.OnPacket(ctx, internal_type.LLMToolCallPacket{
				ContextID: e.currentContextID(),
				Name:      "end_conversation",
				Action:    protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
				Arguments: map[string]string{"reason": reason},
			})
			return
		}
		_ = e.Run(ctx, comm, ResponsePipeline{Response: resp})
	}
}

func (e *agentkitExecutor) streamErrorReason(err error) string {
	switch {
	case errors.Is(err, io.EOF):
		return "server closed connection"
	case status.Code(err) == codes.Canceled:
		return "connection canceled"
	case status.Code(err) == codes.Unavailable:
		return "server unavailable"
	default:
		return err.Error()
	}
}

// =============================================================================
// Connection
// =============================================================================

func (e *agentkitExecutor) connect(ctx context.Context, provider *internal_assistant_entity.AssistantProviderAgentkit) error {
	opts := []grpc.DialOption{grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt64), grpc.MaxCallSendMsgSize(math.MaxInt64))}
	if provider.Certificate != "" {
		creds, err := e.buildTLSCredentials(provider.Certificate)
		if err != nil {
			return fmt.Errorf("TLS credentials failed: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(provider.Url, opts...)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	e.connection = conn

	streamCtx := ctx
	if len(provider.Metadata) > 0 {
		streamCtx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string(provider.Metadata)))
	}
	talker, err := protos.NewAgentKitClient(conn).Talk(streamCtx)
	if err != nil {
		return fmt.Errorf("stream start failed: %w", err)
	}
	e.talker = talker
	return nil
}

func (e *agentkitExecutor) buildTLSCredentials(certPEM string) (credentials.TransportCredentials, error) {
	if certPEM == "insecure" || certPEM == "skip-verify" {
		e.logger.Warnf("Using insecure TLS (skipping certificate verification)")
		return credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}), nil
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(certPEM)) {
		return nil, fmt.Errorf("invalid certificate: failed to parse PEM")
	}
	return credentials.NewTLS(&tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}), nil
}
