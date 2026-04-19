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
	"github.com/rapidaai/pkg/clients/rest"
	"github.com/rapidaai/pkg/commons"
)

type apiRequestToolCaller struct {
	toolCaller
	apiRequestHeader    map[string]string
	apiRequestParameter map[string]string
	apiMethod           string
	apiEndpoint         string
}

func (t *apiRequestToolCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) {
	communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
		ToolID: toolId, Name: t.Name(), ContextID: contextID, Arguments: args,
	})

	client := rest.NewRestClientWithConfig(t.apiEndpoint, t.apiRequestHeader, 15)
	body := t.Argumenting(t.apiRequestParameter, args, communication)

	output, err := t.execute(ctx, client, body)
	if err != nil {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.ErrorResult("Unable to get result"),
		})
		return
	}

	v, err := output.ToMap()
	if err != nil {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.ErrorResult("Unable to get result"),
		})
		return
	}

	communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
		ToolID: toolId, Name: t.Name(), ContextID: contextID,
		Result: internal_tool.JustResult(v),
	})
}

func (t *apiRequestToolCaller) execute(ctx context.Context, client *rest.RestClient, body map[string]interface{}) (*rest.APIResponse, error) {
	switch t.apiMethod {
	case "POST":
		return client.Post(ctx, "", body, t.apiRequestHeader)
	case "PUT":
		return client.Put(ctx, "", body, t.apiRequestHeader)
	case "PATCH":
		return client.Patch(ctx, "", body, t.apiRequestHeader)
	default:
		return client.Get(ctx, "", body, t.apiRequestHeader)
	}
}

func NewApiRequestToolCaller(ctx context.Context, logger commons.Logger, toolOptions *internal_assistant_entity.AssistantTool, communication internal_type.Communication) (internal_tool.ToolCaller, error) {
	opts := toolOptions.GetOptions()
	endpoint, err := opts.GetString("tool.endpoint")
	if err != nil {
		return nil, fmt.Errorf("tool.endpoint is required: %v", err)
	}
	method, err := opts.GetString("tool.method")
	if err != nil {
		return nil, fmt.Errorf("tool.method is required: %v", err)
	}
	parameters, err := opts.GetStringMap("tool.parameters")
	if err != nil {
		return nil, fmt.Errorf("tool.parameters is required: %v", err)
	}
	headers, err := opts.GetStringMap("tool.headers")
	if err != nil {
		logger.Infof("ignoring headers for api requests.")
	}
	return &apiRequestToolCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
		apiRequestHeader:    headers,
		apiRequestParameter: parameters,
		apiEndpoint:         endpoint,
		apiMethod:           method,
	}, nil
}
