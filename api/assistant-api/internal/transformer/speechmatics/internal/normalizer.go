// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package speechmatics_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Speechmatics Text Normalizer
// =============================================================================

// speechmaticsNormalizer handles Speechmatics text preprocessing.
// Speechmatics is primarily an STT service, but this normalizer handles any TTS needs.
// Speechmatics does NOT support SSML - only plain text is accepted.
type speechmaticsNormalizer struct {
	logger   commons.Logger
	language string
}

// NewSpeechmaticsNormalizer creates a Speechmatics-specific text normalizer.
func NewSpeechmaticsNormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &speechmaticsNormalizer{
		logger:   logger,
		language: language,
	}
}

// Normalize returns text unchanged. Speechmatics does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *speechmaticsNormalizer) Normalize(text string) string {
	return text
}
