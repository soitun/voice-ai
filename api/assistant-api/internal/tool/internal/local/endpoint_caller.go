// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"
	"encoding/json"
	"fmt"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_tool "github.com/rapidaai/api/assistant-api/internal/tool/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	endpoint_client_builders "github.com/rapidaai/pkg/clients/endpoint/builders"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type endpointToolCaller struct {
	toolCaller
	endpointId         uint64
	endpointParameters map[string]string
	inputBuilder       endpoint_client_builders.InputInvokeBuilder
}

func NewEndpointToolCaller(
	ctx context.Context,
	logger commons.Logger,
	toolOptions *internal_assistant_entity.AssistantTool,
	communication internal_type.Communication,
) (internal_tool.ToolCaller, error) {
	opts := toolOptions.GetOptions()
	endpointID, err := opts.GetUint64("tool.endpoint_id")
	if err != nil {
		return nil, fmt.Errorf("tool.endpoint_id is not a valid number: %v", err)
	}
	parameters, err := opts.GetStringMap("tool.parameters")
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool.parameters: %v", err)
	}

	return &endpointToolCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
		endpointId:         endpointID,
		endpointParameters: parameters,
		inputBuilder:       endpoint_client_builders.NewInputInvokeBuilder(logger),
	}, nil
}

func (t *endpointToolCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) {
	communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
		ToolID: toolId, Name: t.Name(), ContextID: contextID, Arguments: internal_tool.StringifyArgs(args),
	})

	body := t.Argumenting(t.endpointParameters, args, communication)
	ivk, err := communication.DeploymentCaller().Invoke(
		ctx,
		communication.Auth(),
		t.inputBuilder.Invoke(&protos.EndpointDefinition{EndpointId: t.endpointId, Version: "latest"}, t.inputBuilder.Arguments(body, nil), t.inputBuilder.Metadata(map[string]interface{}{"message_id": contextID}, nil), nil),
	)
	if err != nil {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.ErrorResult("Failed to resolve"),
		})
		return
	}
	if !ivk.GetSuccess() {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.ErrorResult("Failed to resolve"),
		})
		return
	}
	data := ivk.GetData()
	if len(data) == 0 {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.ErrorResult("Failed to resolve"),
		})
		return
	}
	var contentData map[string]interface{}
	if err := json.Unmarshal([]byte(data[0]), &contentData); err != nil {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.Result(data[0], true),
		})
	} else {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.JustResult(contentData),
		})
	}
}
