// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_input_normalizers

import (
	"context"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
)

const (
	LanguageAttributeISO6391 = "language.iso639_1"
	LanguageAttributeISO6392 = "language.iso639_2"
	LanguageAttributeName    = "language.name"
	LanguageAttributeSource  = "language.source"
)

type NormalizerPipeline interface {
	normalizerPipeline()
}

type InputPipeline struct {
	ContextID string
	Speech    string
	Speechs   []internal_type.SpeechToTextPacket
}

type DetectLanguagePipeline struct {
	ContextID string
	Speech    string
	Speechs   []internal_type.SpeechToTextPacket
}

type OutputPipeline struct {
	ContextID string
	Speech    string
	Language  types.Language
}

func (InputPipeline) normalizerPipeline()          {}
func (DetectLanguagePipeline) normalizerPipeline() {}
func (OutputPipeline) normalizerPipeline()         {}

type InputNormalizer interface {
	Initialize(ctx context.Context, onPacket func(...internal_type.Packet) error) error
	Normalize(ctx context.Context, packets ...internal_type.Packet) error
	Close(ctx context.Context) error
}
