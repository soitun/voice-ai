// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package nvidia_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// nvidiaNormalizer handles Nvidia TTS text preprocessing.
// Nvidia does NOT support SSML - only plain text is accepted.
type nvidiaNormalizer struct {
	logger commons.Logger
}

// NewNvidiaNormalizer creates an Nvidia-specific text normalizer.
func NewNvidiaNormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	return &nvidiaNormalizer{
		logger: logger,
	}
}

// Normalize returns text unchanged. Nvidia does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *nvidiaNormalizer) Normalize(text string) string {
	return text
}
