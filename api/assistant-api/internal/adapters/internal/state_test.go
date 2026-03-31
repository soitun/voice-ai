package adapter_internal

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_telemetry_entity "github.com/rapidaai/api/assistant-api/internal/entity/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTelemetryProvider_AssistantNotInitialized(t *testing.T) {
	gr := &genericRequestor{}

	providers, err := gr.GetTelemetryProvider(context.Background())
	require.Error(t, err)
	assert.Nil(t, providers)
}

func TestGetTelemetryProvider_NoProvidersConfigured(t *testing.T) {
	gr := &genericRequestor{
		assistant: &internal_assistant_entity.Assistant{},
	}

	providers, err := gr.GetTelemetryProvider(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, providers)
	assert.Len(t, providers, 0)
}

func TestGetTelemetryProvider_ReturnsConfiguredProviders(t *testing.T) {
	expected := []*internal_telemetry_entity.AssistantTelemetryProvider{
		{ProviderType: "logging", Enabled: true},
	}
	gr := &genericRequestor{
		assistant: &internal_assistant_entity.Assistant{
			AssistantTelemetryProviders: expected,
		},
	}

	providers, err := gr.GetTelemetryProvider(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expected, providers)
}
