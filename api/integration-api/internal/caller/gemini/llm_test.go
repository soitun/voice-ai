// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_gemini_callers

import (
	"encoding/json"
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func newTestCaller() *largeLanguageCaller {
	return &largeLanguageCaller{
		Google: Google{logger: newTestLogger()},
	}
}

func TestBuildHistory_UserMessage(t *testing.T) {
	caller := newTestCaller()
	msgs := []*protos.Message{
		{
			Role:    "user",
			Message: &protos.Message_User{User: &protos.UserMessage{Content: "Hello"}},
		},
	}

	instruction, history, lastPart := caller.buildHistory(msgs)
	require.NotNil(t, instruction)
	assert.Equal(t, "user", instruction.Role)
	assert.Empty(t, history)
	assert.Equal(t, "Hello", lastPart.Text)
}

func TestBuildHistory_SystemMessage(t *testing.T) {
	caller := newTestCaller()
	msgs := []*protos.Message{
		{
			Role:    "system",
			Message: &protos.Message_System{System: &protos.SystemMessage{Content: "Be helpful"}},
		},
	}

	instruction, history, _ := caller.buildHistory(msgs)
	require.NotNil(t, instruction)
	assert.Equal(t, "", instruction.Role)
	assert.Empty(t, history)
	assert.Equal(t, "Be helpful", instruction.Parts[0].Text)
}

func TestBuildHistory_AssistantWithContent(t *testing.T) {
	caller := newTestCaller()
	msgs := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Hello!", "More"},
				},
			},
		},
	}

	instruction, history, _ := caller.buildHistory(msgs)
	require.NotNil(t, instruction)
	assert.Equal(t, "model", instruction.Role)
	assert.Empty(t, history)
	require.Len(t, instruction.Parts, 2)
	assert.Equal(t, "Hello!", instruction.Parts[0].Text)
	assert.Equal(t, "More", instruction.Parts[1].Text)
}

func TestBuildHistory_AssistantWithToolCall(t *testing.T) {
	caller := newTestCaller()
	msgs := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Let me check"},
					ToolCalls: []*protos.ToolCall{
						{
							Id:   "call_1",
							Type: "function",
							Function: &protos.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"city":"NYC"}`,
							},
						},
					},
				},
			},
		},
	}

	instruction, _, _ := caller.buildHistory(msgs)
	require.NotNil(t, instruction)
	assert.Equal(t, "model", instruction.Role)
	require.Len(t, instruction.Parts, 2)
	assert.Equal(t, "Let me check", instruction.Parts[0].Text)
	assert.NotNil(t, instruction.Parts[1].FunctionCall)
	assert.Equal(t, "get_weather", instruction.Parts[1].FunctionCall.Name)
}

func TestBuildHistory_ToolMessage(t *testing.T) {
	caller := newTestCaller()
	toolResult := map[string]interface{}{"temp": float64(72)}
	resultJSON, _ := json.Marshal(toolResult)

	msgs := []*protos.Message{
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{
						{Name: "get_weather", Id: "call_1", Content: string(resultJSON)},
					},
				},
			},
		},
	}

	instruction, _, _ := caller.buildHistory(msgs)
	require.NotNil(t, instruction)
	assert.Equal(t, "user", instruction.Role)
	require.Len(t, instruction.Parts, 1)
	assert.NotNil(t, instruction.Parts[0].FunctionResponse)
	assert.Equal(t, "get_weather", instruction.Parts[0].FunctionResponse.Name)
	assert.Equal(t, float64(72), instruction.Parts[0].FunctionResponse.Response["temp"])
}

func TestBuildHistory_MixedMessages(t *testing.T) {
	caller := newTestCaller()
	msgs := []*protos.Message{
		{Role: "system", Message: &protos.Message_System{System: &protos.SystemMessage{Content: "Be brief"}}},
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "Hi"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"Hello"}}}},
	}

	instruction, history, lastPart := caller.buildHistory(msgs)
	require.NotNil(t, instruction)
	assert.Equal(t, "Be brief", instruction.Parts[0].Text)
	assert.Len(t, history, 2)
	assert.Equal(t, "user", history[0].Role)
	assert.Equal(t, "model", history[1].Role)
	assert.Equal(t, "Hello", lastPart.Text)
}

func TestBuildHistory_EmptyMessages(t *testing.T) {
	caller := newTestCaller()
	instruction, history, lastPart := caller.buildHistory([]*protos.Message{})
	assert.Nil(t, instruction)
	assert.Empty(t, history)
	assert.Equal(t, "", lastPart.Text)
}

func TestBuildHistory_InvalidToolJSON(t *testing.T) {
	caller := newTestCaller()
	msgs := []*protos.Message{
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{
						{Name: "fn", Id: "call_1", Content: "invalid json {{{"},
					},
				},
			},
		},
	}

	instruction, _, _ := caller.buildHistory(msgs)
	require.NotNil(t, instruction)
	assert.NotNil(t, instruction.Parts[0].FunctionResponse)
	assert.Equal(t, 0, len(instruction.Parts[0].FunctionResponse.Response))
}

func TestBuildHistory_ModelRole(t *testing.T) {
	caller := newTestCaller()
	msgs := []*protos.Message{
		{
			Role: "model",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{Contents: []string{"Response"}},
			},
		},
	}

	instruction, _, _ := caller.buildHistory(msgs)
	require.NotNil(t, instruction)
	assert.Equal(t, "model", instruction.Role)
	assert.Equal(t, "Response", instruction.Parts[0].Text)
}

func mustAnyValue(t *testing.T, input interface{}) *anypb.Any {
	t.Helper()
	v, err := structpb.NewValue(input)
	require.NoError(t, err)
	a, err := anypb.New(v)
	require.NoError(t, err)
	return a
}

func TestGetContentConfig_AcceptsLegacyGeminiKeys(t *testing.T) {
	caller := newTestCaller()
	opts := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name":              mustAnyValue(t, "gemini/gemini-2.5-flash"),
				"model.max_output_tokens": mustAnyValue(t, float64(1234)),
				"model.stop_sequences":    mustAnyValue(t, "END,STOP"),
			},
		},
	}

	model, config := caller.getContentConfig(opts)
	assert.Equal(t, "gemini/gemini-2.5-flash", model)
	assert.Equal(t, int32(1234), config.MaxOutputTokens)
	assert.Equal(t, []string{"END", "STOP"}, config.StopSequences)
}

func TestGetContentConfig_MapsThinkingBudget(t *testing.T) {
	caller := newTestCaller()
	opts := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name":            mustAnyValue(t, "gemini/gemini-2.5-flash"),
				"model.thinking_budget": mustAnyValue(t, float64(1200)),
			},
		},
	}

	model, config := caller.getContentConfig(opts)
	assert.Equal(t, "gemini/gemini-2.5-flash", model)
	require.NotNil(t, config.ThinkingConfig)
	require.NotNil(t, config.ThinkingConfig.ThinkingBudget)
	assert.Equal(t, int32(1200), *config.ThinkingConfig.ThinkingBudget)
}
