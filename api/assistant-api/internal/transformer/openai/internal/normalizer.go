// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_openai

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// OpenAI Text Normalizer
// =============================================================================

// openaiNormalizer handles OpenAI TTS text preprocessing.
// OpenAI TTS does NOT support SSML - only plain text is accepted.
type openaiNormalizer struct {
	logger   commons.Logger
	language string
}

// NewOpenAINormalizer creates an OpenAI-specific text normalizer.
func NewOpenAINormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &openaiNormalizer{
		logger:   logger,
		language: language,
	}
}

// Normalize returns text unchanged. OpenAI TTS does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *openaiNormalizer) Normalize(text string) string {
	return text
}
