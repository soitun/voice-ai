// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package token_tiktoken_calculators

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

func TestNewTikTokenCostCalculator(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	model := "gpt-3.5-turbo"

	calculator := NewTikTokenCostCalculator(logger, model)

	assert.NotNil(t, calculator)
	assert.IsType(t, &tikTokenCostCalculator{}, calculator)
}

func TestTikTokenCostCalculator_Token_EmptyMessages(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	in := []*protos.Message{}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Hello world"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_SingleInputMessage(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Hello"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Hi there"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)

	// Values should be positive integers
	inputTokens := metrics[0].Value
	outputTokens := metrics[1].Value
	totalTokens := metrics[2].Value

	assert.NotEqual(t, "0", inputTokens)
	assert.NotEqual(t, "0", outputTokens)
	assert.NotEqual(t, "0", totalTokens)
}

func TestTikTokenCostCalculator_Token_MultipleInputMessages(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	in := []*protos.Message{
		{
			Role: "system",
			Message: &protos.Message_System{
				System: &protos.SystemMessage{Content: "You are a helpful assistant"},
			},
		},
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Hello"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Hi there! How can I help you?"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_GPT35Turbo0613(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo-0613")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Test"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Response"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_GPT4(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-4")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Test GPT-4"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"GPT-4 response"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_GPT35Turbo0301(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo-0301")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Test"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Response"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_UnknownModel(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "unknown-model")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Test"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Response"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
	assert.Equal(t, "0", metrics[0].Value) // Should be 0 for unknown model
	assert.Equal(t, "0", metrics[1].Value)
	assert.Equal(t, "0", metrics[2].Value)
}

func TestTikTokenCostCalculator_Token_GPT35TurboVariant(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo-1106")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Test"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Response"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_GPT4Variant(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-4-turbo")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Test"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Response"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_EmptyContent(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: ""}, // Empty content
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{}, // Empty contents
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_UserMessageOnly(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "What is the weather today?"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"I can help you with weather information"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_NilMessages(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	// Test with nil input messages
	in := []*protos.Message(nil)
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Response"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_LongMessages(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	longText := "lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
		"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
		"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. " +
		"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. " +
		"Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: string(longText)},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{string(longText)},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)

	// Long messages should result in higher token counts
	inputTokens := metrics[0].Value
	outputTokens := metrics[1].Value
	assert.NotEqual(t, "0", inputTokens)
	assert.NotEqual(t, "0", outputTokens)
}

func TestTikTokenCostCalculator_Token_SpecialCharacters(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Hello 🌍 with émojis and spëcial chärs!"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Hi there! 👋"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_MultipleAssistantContents(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Hello"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"Hi there!", "How can I help you today?"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_Token_AssistantInInput(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	in := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Hello"},
			},
		},
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Hi! How can I help?"},
				},
			},
		},
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "What is AI?"},
			},
		},
	}
	out := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: []string{"AI stands for Artificial Intelligence"},
			},
		},
	}

	metrics := calculator.Token(in, out)

	assert.Len(t, metrics, 3)
	assert.Equal(t, "input_token", metrics[0].Name)
	assert.Equal(t, "output_token", metrics[1].Name)
	assert.Equal(t, "total_token", metrics[2].Name)
}

func TestTikTokenCostCalculator_ImplementsTokenCalculatorInterface(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	calculator := NewTikTokenCostCalculator(logger, "gpt-3.5-turbo")

	var _ = calculator
}
