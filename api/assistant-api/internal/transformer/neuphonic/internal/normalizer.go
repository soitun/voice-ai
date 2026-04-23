// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package neuphonic_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// neuphonicNormalizer handles NeuPhonic TTS text preprocessing.
// NeuPhonic does NOT support SSML - only plain text is accepted.
type neuphonicNormalizer struct {
	logger commons.Logger
}

// NewNeuPhonicNormalizer creates a NeuPhonic-specific text normalizer.
func NewNeuPhonicNormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	return &neuphonicNormalizer{
		logger: logger,
	}
}

// Normalize returns text unchanged. NeuPhonic does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *neuphonicNormalizer) Normalize(text string) string {
	return text
}
