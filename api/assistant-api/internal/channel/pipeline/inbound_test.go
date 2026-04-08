// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
)

func newInboundTestDispatcher(cfg *DispatcherConfig) *Dispatcher {
	if cfg.Logger == nil {
		cfg.Logger = newTestLogger()
	}
	return NewDispatcher(cfg)
}

func fakeGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestRunInboundCall_HappyPath(t *testing.T) {
	ginCtx := fakeGinContext()
	assistant := &internal_assistant_entity.Assistant{AssistantProviderId: 1}
	assistant.Id = 42

	d := newInboundTestDispatcher(&DispatcherConfig{
		OnReceiveCall: func(ctx context.Context, provider string, c *gin.Context) (*internal_type.CallInfo, error) {
			return &internal_type.CallInfo{CallerNumber: "+1234567890", StatusInfo: internal_type.StatusInfo{}}, nil
		},
		OnLoadAssistant: func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error) {
			return assistant, nil
		},
		OnCreateConversation: func(ctx context.Context, auth types.SimplePrinciple, callerNumber string, assistantID, assistantProviderID uint64, direction string) (uint64, error) {
			if direction != "inbound" {
				t.Errorf("expected direction=inbound, got %s", direction)
			}
			return 100, nil
		},
		OnSaveCallContext: func(ctx context.Context, auth types.SimplePrinciple, a *internal_assistant_entity.Assistant, conversationID uint64, callInfo *internal_type.CallInfo, provider string) (string, error) {
			return "ctx-abc", nil
		},
		OnAnswerProvider: func(ctx context.Context, c *gin.Context, auth types.SimplePrinciple, provider string, assistantID uint64, callerNumber string, conversationID uint64) error {
			// Verify contextId was set on gin context
			val, exists := c.Get("contextId")
			if !exists {
				t.Error("contextId not set on gin context before onAnswerProvider")
			}
			if val != "ctx-abc" {
				t.Errorf("expected contextId=ctx-abc, got %v", val)
			}
			return nil
		},
	})

	result := d.Run(context.Background(), CallReceivedPipeline{
		ID:         "call-1",
		Provider:   "twilio",
		AssistantID: 42,
		GinContext:  ginCtx,
	})

	if result.Error != nil {
		t.Fatalf("expected no error, got: %v", result.Error)
	}
	if result.ContextID != "ctx-abc" {
		t.Errorf("expected contextID=ctx-abc, got %s", result.ContextID)
	}
	if result.ConversationID != 100 {
		t.Errorf("expected conversationID=100, got %d", result.ConversationID)
	}
}

func TestRunInboundCall_NilCallInfo(t *testing.T) {
	d := newInboundTestDispatcher(&DispatcherConfig{
		OnReceiveCall: func(ctx context.Context, provider string, c *gin.Context) (*internal_type.CallInfo, error) {
			return nil, nil // status callback, no action needed
		},
	})

	result := d.Run(context.Background(), CallReceivedPipeline{
		ID:         "status-cb",
		Provider:   "twilio",
		GinContext:  fakeGinContext(),
	})

	if result.Error != nil {
		t.Fatalf("expected nil error for nil callInfo, got: %v", result.Error)
	}
}

func TestRunInboundCall_ReceiveCallError(t *testing.T) {
	webhookErr := errors.New("bad webhook")
	d := newInboundTestDispatcher(&DispatcherConfig{
		OnReceiveCall: func(ctx context.Context, provider string, c *gin.Context) (*internal_type.CallInfo, error) {
			return nil, webhookErr
		},
	})

	result := d.Run(context.Background(), CallReceivedPipeline{
		ID:         "bad-wh",
		Provider:   "twilio",
		GinContext:  fakeGinContext(),
	})

	if !errors.Is(result.Error, webhookErr) {
		t.Errorf("expected webhookErr, got: %v", result.Error)
	}
}

func TestRunInboundCall_MissingReceiveCallCallback(t *testing.T) {
	d := newInboundTestDispatcher(&DispatcherConfig{
		// OnReceiveCall nil
	})

	result := d.Run(context.Background(), CallReceivedPipeline{
		ID:         "no-cb",
		Provider:   "twilio",
		GinContext:  fakeGinContext(),
	})

	if !errors.Is(result.Error, ErrCallbackNotConfigured) {
		t.Errorf("expected ErrCallbackNotConfigured, got: %v", result.Error)
	}
}

func TestRunInboundCall_LoadAssistantError(t *testing.T) {
	loadErr := errors.New("assistant not found")
	d := newInboundTestDispatcher(&DispatcherConfig{
		OnReceiveCall: func(ctx context.Context, provider string, c *gin.Context) (*internal_type.CallInfo, error) {
			return &internal_type.CallInfo{CallerNumber: "+1"}, nil
		},
		OnLoadAssistant: func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error) {
			return nil, loadErr
		},
	})

	ctx := context.Background()
	d.Start(ctx) // needed for async CallFailed pipeline

	result := d.Run(ctx, CallReceivedPipeline{
		ID:         "bad-asst",
		Provider:   "twilio",
		GinContext:  fakeGinContext(),
	})

	if !errors.Is(result.Error, loadErr) {
		t.Errorf("expected loadErr, got: %v", result.Error)
	}
}

func TestRunInboundCall_SaveCallContextError(t *testing.T) {
	saveErr := errors.New("db error")
	assistant := &internal_assistant_entity.Assistant{AssistantProviderId: 1}
	assistant.Id = 42

	d := newInboundTestDispatcher(&DispatcherConfig{
		OnReceiveCall: func(ctx context.Context, provider string, c *gin.Context) (*internal_type.CallInfo, error) {
			return &internal_type.CallInfo{CallerNumber: "+1"}, nil
		},
		OnLoadAssistant: func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error) {
			return assistant, nil
		},
		OnCreateConversation: func(ctx context.Context, auth types.SimplePrinciple, callerNumber string, assistantID, assistantProviderID uint64, direction string) (uint64, error) {
			return 100, nil
		},
		OnSaveCallContext: func(ctx context.Context, auth types.SimplePrinciple, a *internal_assistant_entity.Assistant, conversationID uint64, callInfo *internal_type.CallInfo, provider string) (string, error) {
			return "", saveErr
		},
	})

	ctx := context.Background()
	d.Start(ctx)

	result := d.Run(ctx, CallReceivedPipeline{
		ID:         "save-fail",
		Provider:   "twilio",
		GinContext:  fakeGinContext(),
	})

	if !errors.Is(result.Error, saveErr) {
		t.Errorf("expected saveErr, got: %v", result.Error)
	}
}

func TestRunInboundCall_InlineExecution(t *testing.T) {
	d := newInboundTestDispatcher(&DispatcherConfig{
		OnReceiveCall: func(ctx context.Context, provider string, c *gin.Context) (*internal_type.CallInfo, error) {
			return nil, errors.New("inline-check")
		},
	})

	result := d.Run(context.Background(), CallReceivedPipeline{
		ID:         "inline",
		Provider:   "twilio",
		GinContext:  fakeGinContext(),
	})

	// Result must be available synchronously.
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil || result.Error.Error() != "inline-check" {
		t.Fatalf("expected inline-check error, got: %v", result.Error)
	}
}
