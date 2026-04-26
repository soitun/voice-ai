// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor_tool

import (
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

// ToolPipeline is a sealed marker interface for tool executor pipeline stages.
// Only types in this package can satisfy it.
type ToolPipeline interface {
	toolPipeline()
}

type DiscoverToolsPipeline struct {
	Communication internal_type.Communication
}

type FilterToolsPipeline struct {
	Tools         []*internal_assistant_entity.AssistantTool
	Communication internal_type.Communication
}

type BuildAndRegisterToolsPipeline struct {
	Tools         []*internal_assistant_entity.AssistantTool
	Communication internal_type.Communication
}

type ExecuteToolsPipeline struct {
	ContextID     string
	Calls         []*protos.ToolCall
	Communication internal_type.Communication
}

func (DiscoverToolsPipeline) toolPipeline()            {}
func (FilterToolsPipeline) toolPipeline()              {}
func (BuildAndRegisterToolsPipeline) toolPipeline()    {}
func (ExecuteToolsPipeline) toolPipeline()             {}
