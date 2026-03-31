// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_callers

import (
	"testing"

	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestFunctionParameterToMap_DefaultsToObjectAndProperties(t *testing.T) {
	fp := &FunctionParameter{}

	got := fp.ToMap()

	assert.Equal(t, "object", got["type"])
	props, ok := got["properties"].(map[string]interface{})
	require.True(t, ok)
	assert.Empty(t, props)
	_, hasRequired := got["required"]
	assert.False(t, hasRequired)
}

func TestFunctionParameterToMap_WithPropertiesAndRequired(t *testing.T) {
	fp := &FunctionParameter{
		Type:     "object",
		Required: []string{"city"},
		Properties: map[string]FunctionParameterProperty{
			"city": {
				Type:        "string",
				Description: "City name",
			},
		},
	}

	got := fp.ToMap()

	assert.Equal(t, "object", got["type"])
	assert.Equal(t, []string{"city"}, got["required"])

	props, ok := got["properties"].(map[string]interface{})
	require.True(t, ok)
	city, ok := props["city"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", city["type"])
	assert.Equal(t, "City name", city["description"])
}

func TestFunctionParameterPropertyToMap_OnlyIncludesSetFields(t *testing.T) {
	first := "first"
	second := "second"
	fpp := &FunctionParameterProperty{
		Type:        "array",
		Description: "An array field",
		Enum:        []*string{&first, &second},
		Items: map[string]interface{}{
			"type": "string",
		},
	}

	got := fpp.ToMap()

	assert.Equal(t, "array", got["type"])
	assert.Equal(t, "An array field", got["description"])
	assert.Equal(t, []*string{&first, &second}, got["enum"])
	assert.Equal(t, map[string]interface{}{"type": "string"}, got["items"])
}

func TestNewChatOptions_SetsOptionsAndCastsToolDefinitions(t *testing.T) {
	preCalled := false
	postCalled := false
	preHook := func(map[string]interface{}) {
		preCalled = true
	}
	postHook := func(map[string]interface{}, []*protos.Metric) {
		postCalled = true
	}

	anyVal, err := anypb.New(&emptypb.Empty{})
	require.NoError(t, err)

	request := &protos.ChatRequest{
		ModelParameters: map[string]*anypb.Any{
			"model.name": anyVal,
		},
		ToolDefinitions: []*protos.ToolDefinition{
			{
				Type: "function",
				FunctionDefinition: &protos.FunctionDefinition{
					Name:        "weather",
					Description: "Get weather",
					Parameters: &protos.FunctionParameter{
						Type:     "object",
						Required: []string{"city"},
						Properties: map[string]*protos.FunctionParameterProperty{
							"city": {
								Type:        "string",
								Description: "City",
							},
						},
					},
				},
			},
		},
	}

	opts := NewChatOptions(123, request, preHook, postHook)

	require.NotNil(t, opts)
	assert.Equal(t, uint64(123), opts.RequestId)
	assert.Equal(t, request, opts.Request)
	assert.Equal(t, request.ModelParameters, opts.ModelParameter)
	require.Len(t, opts.ToolDefinitions, 1)
	assert.Equal(t, "function", opts.ToolDefinitions[0].Type)
	require.NotNil(t, opts.ToolDefinitions[0].Function)
	assert.Equal(t, "weather", opts.ToolDefinitions[0].Function.Name)
	require.NotNil(t, opts.ToolDefinitions[0].Function.Parameters)
	assert.Equal(t, "object", opts.ToolDefinitions[0].Function.Parameters.Type)
	assert.Contains(t, opts.ToolDefinitions[0].Function.Parameters.Properties, "city")

	opts.PreHook(nil)
	opts.PostHook(nil, nil)
	assert.True(t, preCalled)
	assert.True(t, postCalled)
}

func TestNewEmbeddingOptions_SetsExpectedFields(t *testing.T) {
	anyVal, err := anypb.New(&emptypb.Empty{})
	require.NoError(t, err)

	request := &protos.EmbeddingRequest{
		ModelParameters: map[string]*anypb.Any{
			"model.id": anyVal,
		},
	}

	opts := NewEmbeddingOptions(55, request, func(map[string]interface{}) {}, func(map[string]interface{}, []*protos.Metric) {})

	require.NotNil(t, opts)
	assert.Equal(t, uint64(55), opts.RequestId)
	assert.Equal(t, request.ModelParameters, opts.ModelParameter)
	require.NotNil(t, opts.PreHook)
	require.NotNil(t, opts.PostHook)
}

func TestNewRerankerOptions_SetsExpectedFields(t *testing.T) {
	anyVal, err := anypb.New(&emptypb.Empty{})
	require.NoError(t, err)

	request := &protos.RerankingRequest{
		ModelParameters: map[string]*anypb.Any{
			"model.id": anyVal,
		},
	}

	opts := NewRerankerOptions(89, request, func(map[string]interface{}) {}, func(map[string]interface{}, []*protos.Metric) {})

	require.NotNil(t, opts)
	assert.Equal(t, uint64(89), opts.RequestId)
	assert.Equal(t, request.ModelParameters, opts.ModelParameter)
	require.NotNil(t, opts.PreHook)
	require.NotNil(t, opts.PostHook)
}
