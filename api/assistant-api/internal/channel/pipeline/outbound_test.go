// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"errors"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
)

func makePhoneAssistant() *internal_assistant_entity.Assistant {
	a := &internal_assistant_entity.Assistant{AssistantProviderId: 1}
	a.Id = 42
	a.AssistantPhoneDeployment = &internal_assistant_entity.AssistantPhoneDeployment{
		AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
			TelephonyProvider: "twilio",
			TelephonyOption: []*internal_assistant_entity.AssistantDeploymentTelephonyOption{
				{Metadata: gorm_model.Metadata{Key: "phone", Value: "+10000000000"}},
			},
		},
	}
	return a
}

func TestRunOutbound_HappyPath(t *testing.T) {
	assistant := makePhoneAssistant()

	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnLoadAssistant: func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error) {
			return assistant, nil
		},
		OnCreateConversation: func(ctx context.Context, auth types.SimplePrinciple, callerNumber string, assistantID, assistantProviderID uint64, direction string) (uint64, error) {
			if direction != "outbound" {
				t.Errorf("expected direction=outbound, got %s", direction)
			}
			return 200, nil
		},
		OnSaveCallContext: func(ctx context.Context, auth types.SimplePrinciple, a *internal_assistant_entity.Assistant, conversationID uint64, callInfo *internal_type.CallInfo, provider string) (string, error) {
			return "ctx-out-1", nil
		},
		OnDispatchOutbound: func(ctx context.Context, contextID string) error {
			return nil
		},
	})

	result := d.Run(context.Background(), OutboundRequestedPipeline{
		ID:          "out-1",
		AssistantID: 42,
		ToPhone:     "+19999999999",
	})

	if result.Error != nil {
		t.Fatalf("expected no error, got: %v", result.Error)
	}
	if result.ContextID != "ctx-out-1" {
		t.Errorf("expected contextID=ctx-out-1, got %s", result.ContextID)
	}
	if result.ConversationID != 200 {
		t.Errorf("expected conversationID=200, got %d", result.ConversationID)
	}
}

func TestRunOutbound_MissingLoadAssistant(t *testing.T) {
	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		// OnLoadAssistant nil
	})

	result := d.Run(context.Background(), OutboundRequestedPipeline{
		ID:          "out-no-cb",
		AssistantID: 42,
		ToPhone:     "+1",
	})

	if !errors.Is(result.Error, ErrCallbackNotConfigured) {
		t.Errorf("expected ErrCallbackNotConfigured, got: %v", result.Error)
	}
}

func TestRunOutbound_NoPhoneDeployment(t *testing.T) {
	assistant := &internal_assistant_entity.Assistant{}
	assistant.Id = 42

	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnLoadAssistant: func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error) {
			return assistant, nil
		},
	})

	result := d.Run(context.Background(), OutboundRequestedPipeline{
		ID:          "out-no-phone",
		AssistantID: 42,
		ToPhone:     "+1",
	})

	if result.Error == nil {
		t.Fatal("expected error for missing phone deployment")
	}
}

func TestRunOutbound_DispatchError(t *testing.T) {
	assistant := makePhoneAssistant()
	dispatchErr := errors.New("dial failed")

	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnLoadAssistant: func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error) {
			return assistant, nil
		},
		OnCreateConversation: func(ctx context.Context, auth types.SimplePrinciple, callerNumber string, assistantID, assistantProviderID uint64, direction string) (uint64, error) {
			return 200, nil
		},
		OnSaveCallContext: func(ctx context.Context, auth types.SimplePrinciple, a *internal_assistant_entity.Assistant, conversationID uint64, callInfo *internal_type.CallInfo, provider string) (string, error) {
			return "ctx-out-fail", nil
		},
		OnDispatchOutbound: func(ctx context.Context, contextID string) error {
			return dispatchErr
		},
	})

	ctx := context.Background()
	d.Start(ctx) // needed for async CallFailed pipeline

	result := d.Run(ctx, OutboundRequestedPipeline{
		ID:          "out-fail",
		AssistantID: 42,
		ToPhone:     "+1",
	})

	if !errors.Is(result.Error, dispatchErr) {
		t.Errorf("expected dispatchErr, got: %v", result.Error)
	}
	// Should still return contextID and conversationID even on dispatch error
	if result.ContextID != "ctx-out-fail" {
		t.Errorf("expected contextID=ctx-out-fail, got %s", result.ContextID)
	}
}

func TestRunOutbound_InlineExecution(t *testing.T) {
	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnLoadAssistant: func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error) {
			return nil, errors.New("inline-check")
		},
	})

	result := d.Run(context.Background(), OutboundRequestedPipeline{
		ID:          "inline-out",
		AssistantID: 42,
		ToPhone:     "+1",
	})

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil {
		t.Fatal("expected error")
	}
}
