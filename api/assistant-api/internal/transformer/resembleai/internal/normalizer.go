// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resembleai_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// resembleaiNormalizer handles ResembleAI TTS text preprocessing.
// ResembleAI does NOT support SSML - only plain text is accepted.
type resembleaiNormalizer struct {
	logger commons.Logger
}

// NewResembleAINormalizer creates a ResembleAI-specific text normalizer.
func NewResembleAINormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	return &resembleaiNormalizer{
		logger: logger,
	}
}

// Normalize returns text unchanged. ResembleAI does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *resembleaiNormalizer) Normalize(text string) string {
	return text
}
