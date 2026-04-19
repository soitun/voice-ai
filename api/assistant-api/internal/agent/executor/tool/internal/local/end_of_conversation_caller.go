// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"

	internal_tool "github.com/rapidaai/api/assistant-api/internal/agent/executor/tool/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type endOfConversationCaller struct {
	toolCaller
}

func (t *endOfConversationCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) {
	communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
		ToolID: toolId, Name: t.Name(), ContextID: contextID, Arguments: args,
	})
	communication.OnPacket(ctx,
		internal_type.DirectivePacket{
			Directive: protos.ConversationDirective_END_CONVERSATION,
			Arguments: map[string]interface{}{
				"tool_id": toolId,
			},
			ContextID: contextID,
		},
	)
}

func NewEndOfConversationCaller(ctx context.Context, logger commons.Logger, toolOptions *internal_assistant_entity.AssistantTool, communication internal_type.Communication,
) (internal_tool.ToolCaller, error) {
	return &endOfConversationCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
	}, nil
}
