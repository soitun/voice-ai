// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package minimax_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// minimaxNormalizer handles MiniMax TTS text preprocessing.
// MiniMax does NOT support SSML - only plain text is accepted.
type minimaxNormalizer struct {
	logger commons.Logger
}

// NewMiniMaxNormalizer creates a MiniMax-specific text normalizer.
func NewMiniMaxNormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	return &minimaxNormalizer{
		logger: logger,
	}
}

// Normalize returns text unchanged. MiniMax does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *minimaxNormalizer) Normalize(text string) string {
	return text
}
