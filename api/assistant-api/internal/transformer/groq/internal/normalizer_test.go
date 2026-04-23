// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package groq_internal

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroqNormalizer(t *testing.T) {
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	normalizer := NewGroqNormalizer(logger, utils.Option{})
	require.NotNil(t, normalizer)
	_, ok := normalizer.(*groqNormalizer)
	assert.True(t, ok)
}

func TestNormalize_Passthrough(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewGroqNormalizer(logger, utils.Option{})

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple text", "Hello world"},
		{"markdown preserved", "# Header **bold**"},
		{"xml chars preserved", "Tom & Jerry a < b > c"},
		{"whitespace preserved", "Hello    world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			assert.Equal(t, tt.input, result)
		})
	}
}

func TestNormalize_NoSSML(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewGroqNormalizer(logger, utils.Option{})

	result := normalizer.Normalize("Tom & Jerry")
	assert.NotContains(t, result, "&amp;")
	assert.NotContains(t, result, "<break")
}

func BenchmarkNormalize(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewGroqNormalizer(logger, utils.Option{})
	text := "Hello, this is a simple text for TTS processing."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}
