// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package revai_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Rev AI Text Normalizer
// =============================================================================

// revaiNormalizer handles Rev AI text preprocessing.
// Rev AI is primarily an STT service, but this normalizer handles any TTS needs.
// Rev AI does NOT support SSML - only plain text is accepted.
type revaiNormalizer struct {
	logger   commons.Logger
	language string
}

// NewRevAINormalizer creates a Rev AI-specific text normalizer.
func NewRevAINormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &revaiNormalizer{
		logger:   logger,
		language: language,
	}
}

// Normalize returns text unchanged. Rev AI does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *revaiNormalizer) Normalize(text string) string {
	return text
}
