// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"fmt"
	"strings"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type toolCaller struct {
	logger      commons.Logger
	toolOptions *internal_assistant_entity.AssistantTool
}

func (executor *toolCaller) Name() string {
	return executor.toolOptions.Name
}

func (executor *toolCaller) Id() uint64 {
	return executor.toolOptions.Id
}

func (executor *toolCaller) ExecutionMethod() string {
	return executor.toolOptions.ExecutionMethod
}

func (executor *toolCaller) Argumenting(mapping map[string]string, args map[string]interface{}, communication internal_type.Communication) map[string]interface{} {
	arguments := make(map[string]interface{})
	for key, value := range mapping {
		if k, ok := strings.CutPrefix(key, "tool."); ok {
			switch k {
			case "name":
				arguments[value] = executor.Name()
			case "argument":
				arguments[value] = args
			}
		}
		if k, ok := strings.CutPrefix(key, "assistant."); ok {
			switch k {
			case "id":
				arguments[value] = fmt.Sprintf("%d", communication.Assistant().Id)
			case "version":
				arguments[value] = fmt.Sprintf("vrsn_%d", communication.Assistant().AssistantProviderModel.Id)
			}
		}
		if k, ok := strings.CutPrefix(key, "conversation."); ok {
			switch k {
			case "id":
				arguments[value] = fmt.Sprintf("%d", communication.Conversation().Id)
			case "messages":
				arguments[value] = simplifyHistory(communication.GetHistories())
			}
		}
		if k, ok := strings.CutPrefix(key, "argument."); ok {
			if aArg, ok := communication.GetArgs()[k]; ok {
				arguments[value] = aArg
			}
		}
		if k, ok := strings.CutPrefix(key, "metadata."); ok {
			if mtd, ok := communication.GetMetadata()[k]; ok {
				arguments[value] = mtd
			}
		}
		if k, ok := strings.CutPrefix(key, "option."); ok {
			if ot, ok := communication.GetOptions()[k]; ok {
				arguments[value] = ot
			}
		}
		if k, ok := strings.CutPrefix(key, "custom."); ok {
			arguments[k] = value
		}
	}
	return arguments
}

func simplifyHistory(msgs []internal_type.MessagePacket) []map[string]string {
	out := make([]map[string]string, 0, len(msgs))
	for _, msg := range msgs {
		out = append(out, map[string]string{
			"role":    msg.Role(),
			"message": msg.Content(),
		})
	}
	return out
}

func (executor *toolCaller) Definition() (*protos.FunctionDefinition, error) {
	definition := &protos.FunctionDefinition{
		Name:       executor.toolOptions.Name,
		Parameters: &protos.FunctionParameter{},
	}
	if executor.toolOptions.Description != nil && *executor.toolOptions.Description != "" {
		definition.Description = *executor.toolOptions.Description
	}
	if err := utils.Cast(executor.toolOptions.Fields, definition.Parameters); err != nil {
		return nil, err
	}
	return definition, nil
}
