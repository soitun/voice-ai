// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"
	"fmt"

	internal_tool "github.com/rapidaai/api/assistant-api/internal/agent/executor/tool/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type transferCallCaller struct {
	toolCaller
}

func (tc *transferCallCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) internal_tool.ToolCallResult {
	to, _ := args["to"].(string)
	if to == "" {
		return internal_tool.Result("Missing 'to' parameter — provide a phone number or SIP URI to transfer to.", false)
	}
	communication.OnPacket(ctx, internal_type.DirectivePacket{
		Directive: protos.ConversationDirective_TRANSFER_CONVERSATION,
		Arguments: args,
		ContextID: contextID,
	})
	return internal_tool.Result(fmt.Sprintf("Call transferred to %s.", to), true)
}

func NewTransferCallCaller(ctx context.Context, logger commons.Logger, toolOptions *internal_assistant_entity.AssistantTool, communication internal_type.Communication,
) (internal_tool.ToolCaller, error) {
	return &transferCallCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
	}, nil
}
