// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"bufio"
	"context"
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	observe "github.com/rapidaai/api/assistant-api/internal/observe"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

const (
	signalChSize  = 64
	setupChSize   = 256
	mediaChSize   = 256
	controlChSize = 512
)

// PipelineResult carries the outcome of a sync pipeline stage back to the caller.
type PipelineResult struct {
	ContextID      string
	ConversationID uint64
	Error          error

	Provider     string
	CallerNumber string
	CallStatus   string
	CallEvent    string
	Extra        map[string]string // provider-specific metadata
}

type callEnvelope struct {
	ctx context.Context
	p   Pipeline
}

// Dispatcher routes channel call lifecycle stages to priority-based goroutines.
//
//	signal  — disconnect, completed, failed
//	setup   — call received, conversation created
//	media   — session connected, initialized, active
//	control — events, metrics
type Dispatcher struct {
	logger commons.Logger

	signalCh  chan callEnvelope
	setupCh   chan callEnvelope
	mediaCh   chan callEnvelope
	controlCh chan callEnvelope

	onReceiveCall             OnReceiveCallFunc
	onLoadAssistant           OnLoadAssistantFunc
	onCreateConversation      OnCreateConversationFunc
	onSaveCallContext         OnSaveCallContextFunc
	onAnswerProvider          OnAnswerProviderFunc
	onDispatchOutbound        OnDispatchOutboundFunc
	onApplyConversationExtras OnApplyConversationExtrasFunc
	onResolveSession          OnResolveSessionFunc
	onCreateStreamer          OnCreateStreamerFunc
	onCreateTalker            OnCreateTalkerFunc
	onRunTalk                 OnRunTalkFunc
	onCreateObserver          OnCreateObserverFunc
	onCreateHooks             OnCreateHooksFunc
	onCompleteSession         OnCompleteSessionFunc
}

// OnReceiveCallFunc parses the provider webhook and returns CallInfo.
type OnReceiveCallFunc func(ctx context.Context, provider string, ginCtx *gin.Context) (*internal_type.CallInfo, error)

// OnLoadAssistantFunc loads the assistant from DB.
type OnLoadAssistantFunc func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error)

// OnCreateConversationFunc creates a conversation and returns conversationID.
type OnCreateConversationFunc func(ctx context.Context, auth types.SimplePrinciple, callerNumber string, assistantID, assistantProviderID uint64, direction string) (conversationID uint64, err error)

// OnSaveCallContextFunc saves the call context to Postgres and returns contextID.
type OnSaveCallContextFunc func(ctx context.Context, auth types.SimplePrinciple, assistant *internal_assistant_entity.Assistant, conversationID uint64, callInfo *internal_type.CallInfo, provider string) (contextID string, err error)

// OnAnswerProviderFunc instructs the provider to answer the call.
type OnAnswerProviderFunc func(ctx context.Context, ginCtx *gin.Context, auth types.SimplePrinciple, provider string, assistantID uint64, callerNumber string, conversationID uint64) error

// OnDispatchOutboundFunc dispatches the outbound call (claim, vault, dial).
type OnDispatchOutboundFunc func(ctx context.Context, contextID string) error

// OnApplyConversationExtrasFunc applies options/arguments/metadata to a conversation.
type OnApplyConversationExtrasFunc func(ctx context.Context, auth types.SimplePrinciple, assistantID, conversationID uint64, opts, args, metadata map[string]interface{}) error

// OnResolveSessionFunc resolves a call context and vault credential from a contextID.
type OnResolveSessionFunc func(ctx context.Context, contextID string) (*callcontext.CallContext, *protos.VaultCredential, error)

// OnCreateStreamerFunc creates a provider-specific streamer.
type OnCreateStreamerFunc func(ctx context.Context, cc *callcontext.CallContext, vc *protos.VaultCredential, ws *websocket.Conn, conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) (internal_type.Streamer, error)

// OnCreateTalkerFunc creates a talker (genericRequestor).
type OnCreateTalkerFunc func(ctx context.Context, streamer internal_type.Streamer) (internal_type.Talking, error)

// OnRunTalkFunc runs talker.Talk (blocking for call duration).
type OnRunTalkFunc func(ctx context.Context, talker internal_type.Talking, auth types.SimplePrinciple) error

// OnCreateObserverFunc creates a ConversationObserver.
type OnCreateObserverFunc func(ctx context.Context, callID string, auth types.SimplePrinciple, assistantID, conversationID uint64) *observe.ConversationObserver

// OnCreateHooksFunc creates ConversationHooks (webhooks + analysis).
type OnCreateHooksFunc func(ctx context.Context, auth types.SimplePrinciple, assistantID, conversationID uint64) *observe.ConversationHooks

// OnCompleteSessionFunc marks a call context as completed.
type OnCompleteSessionFunc func(ctx context.Context, contextID string)

// DispatcherConfig holds dependencies for creating a channel dispatcher.
type DispatcherConfig struct {
	Logger                    commons.Logger
	OnReceiveCall             OnReceiveCallFunc
	OnLoadAssistant           OnLoadAssistantFunc
	OnCreateConversation      OnCreateConversationFunc
	OnSaveCallContext         OnSaveCallContextFunc
	OnAnswerProvider          OnAnswerProviderFunc
	OnDispatchOutbound        OnDispatchOutboundFunc
	OnApplyConversationExtras OnApplyConversationExtrasFunc
	OnResolveSession          OnResolveSessionFunc
	OnCreateStreamer          OnCreateStreamerFunc
	OnCreateTalker            OnCreateTalkerFunc
	OnRunTalk                 OnRunTalkFunc
	OnCreateObserver          OnCreateObserverFunc
	OnCreateHooks             OnCreateHooksFunc
	OnCompleteSession         OnCompleteSessionFunc
}

func NewDispatcher(cfg *DispatcherConfig) *Dispatcher {
	return &Dispatcher{
		logger:                    cfg.Logger,
		onReceiveCall:             cfg.OnReceiveCall,
		onLoadAssistant:           cfg.OnLoadAssistant,
		onCreateConversation:      cfg.OnCreateConversation,
		onSaveCallContext:         cfg.OnSaveCallContext,
		onAnswerProvider:          cfg.OnAnswerProvider,
		onDispatchOutbound:        cfg.OnDispatchOutbound,
		onApplyConversationExtras: cfg.OnApplyConversationExtras,
		onResolveSession:          cfg.OnResolveSession,
		onCreateStreamer:          cfg.OnCreateStreamer,
		onCreateTalker:            cfg.OnCreateTalker,
		onRunTalk:                 cfg.OnRunTalk,
		onCreateObserver:          cfg.OnCreateObserver,
		onCreateHooks:             cfg.OnCreateHooks,
		onCompleteSession:         cfg.OnCompleteSession,
		signalCh:                  make(chan callEnvelope, signalChSize),
		setupCh:                   make(chan callEnvelope, setupChSize),
		mediaCh:                   make(chan callEnvelope, mediaChSize),
		controlCh:                 make(chan callEnvelope, controlChSize),
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	go d.runDispatcher(ctx, d.signalCh)
	go d.runDispatcher(ctx, d.setupCh)
	go d.runDispatcher(ctx, d.mediaCh)
	go d.runDispatcher(ctx, d.controlCh)
	d.logger.Infow("Channel pipeline dispatcher started")
}

// OnPipeline enqueues a pipeline stage asynchronously (fire-and-forget).
func (d *Dispatcher) OnPipeline(ctx context.Context, stages ...Pipeline) {
	for _, s := range stages {
		d.enqueue(ctx, s)
	}
}

// Run executes a sync pipeline stage inline on the caller's goroutine.
// For CallReceived, SessionConnected, and OutboundRequested the handler runs
// sequentially without channels or goroutines. All other stage types are
// forwarded to OnPipeline (async fire-and-forget) and an empty result is returned.
func (d *Dispatcher) Run(ctx context.Context, stage Pipeline) *PipelineResult {
	switch v := stage.(type) {
	case CallReceivedPipeline:
		return d.runInboundCall(ctx, v)
	case SessionConnectedPipeline:
		return d.runSession(ctx, v)
	case OutboundRequestedPipeline:
		return d.runOutbound(ctx, v)
	default:
		d.OnPipeline(ctx, stage)
		return &PipelineResult{}
	}
}

func (d *Dispatcher) enqueue(ctx context.Context, s Pipeline) {
	e := callEnvelope{ctx: ctx, p: s}
	switch s.(type) {
	case DisconnectRequestedPipeline, CallCompletedPipeline, CallFailedPipeline:
		d.signalCh <- e
	case ModeSwitchPipeline:
		d.mediaCh <- e
	case EventEmittedPipeline, MetricEmittedPipeline:
		d.controlCh <- e
	default:
		d.logger.Warnw("OnPipeline: unrouted type", "type", fmt.Sprintf("%T", s))
		d.controlCh <- e
	}
}

func (d *Dispatcher) runDispatcher(ctx context.Context, ch chan callEnvelope) {
	for {
		select {
		case <-ctx.Done():
			d.drain(ch)
			return
		case e := <-ch:
			d.dispatch(e)
		}
	}
}

func (d *Dispatcher) drain(ch chan callEnvelope) {
	for {
		select {
		case e := <-ch:
			d.dispatch(e)
		default:
			return
		}
	}
}

func (d *Dispatcher) dispatch(e callEnvelope) {
	ctx := e.ctx

	switch v := e.p.(type) {
	case DisconnectRequestedPipeline:
		d.handleDisconnectRequested(ctx, v)
	case CallCompletedPipeline:
		d.handleCallCompleted(ctx, v)
	case CallFailedPipeline:
		d.handleCallFailed(ctx, v)
	case ModeSwitchPipeline:
		d.handleModeSwitch(ctx, v)
	case EventEmittedPipeline:
		d.handleEventEmitted(ctx, v)
	case MetricEmittedPipeline:
		d.handleMetricEmitted(ctx, v)
	default:
		d.logger.Warnw("dispatch: unknown pipeline type", "type", fmt.Sprintf("%T", e.p))
	}
}

