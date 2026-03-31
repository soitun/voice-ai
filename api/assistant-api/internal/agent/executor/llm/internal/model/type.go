// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_model

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

// PipelineType is a sealed marker interface constraining what can enter Pipeline().
// Only types in this package can satisfy it (unexported method + internal package).
type PipelineType interface {
	pipelineMarker()
}

// PrepareHistoryPipeline snapshots current history and creates the user message.
type PrepareHistoryPipeline struct {
	Packet internal_type.NormalizedUserTextPacket
}

func (PrepareHistoryPipeline) pipelineMarker() {}

// ArgumentationPipeline carries per-request state through context enrichment.
// The four buildX methods (assistant, conversation, message, session) are called
// sequentially to merge prompt context before routing to the output stage.
type ArgumentationPipeline struct {
	Packet       internal_type.NormalizedUserTextPacket
	UserMessage  *protos.Message
	History      []*protos.Message
	PromptArgs   map[string]interface{}
	ToolFollowUp bool
}

func (ArgumentationPipeline) pipelineMarker() {}

// LLMRequestPipeline emits the "executing" event, sends the chat request, and
// appends the user message to local history.
type LLMRequestPipeline struct {
	Packet      internal_type.NormalizedUserTextPacket
	UserMessage *protos.Message
	History     []*protos.Message
	PromptArgs  map[string]interface{}
}

func (LLMRequestPipeline) pipelineMarker() {}

// ToolFollowUpPipeline sends a follow-up chat request using the current history
// (which already includes tool call results).
type ToolFollowUpPipeline struct {
	ContextID  string
	PromptArgs map[string]interface{}
}

func (ToolFollowUpPipeline) pipelineMarker() {}

// LocalHistoryPipeline appends a message to local in-memory history.
type LocalHistoryPipeline struct {
	Message *protos.Message
}

func (LocalHistoryPipeline) pipelineMarker() {}

// LLMResponsePipeline is the typed response state flowing through validation,
// emission, and tool follow-up stages.
type LLMResponsePipeline struct {
	Response *protos.ChatResponse

	Output  *protos.Message
	Metrics []*protos.Metric
}

func (LLMResponsePipeline) pipelineMarker() {}
