// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor_tool

import (
	"context"
	"encoding/json"
	"errors"

	internal_agent_executor "github.com/rapidaai/api/assistant-api/internal/agent/executor"
	internal_tool "github.com/rapidaai/api/assistant-api/internal/agent/executor/tool/internal"
	internal_tool_local "github.com/rapidaai/api/assistant-api/internal/agent/executor/tool/internal/local"
	internal_tool_mcp "github.com/rapidaai/api/assistant-api/internal/agent/executor/tool/internal/mcp"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"

	"github.com/rapidaai/protos"

	"github.com/rapidaai/pkg/commons"
)

type toolExecutor struct {
	logger                 commons.Logger
	tools                  map[string]internal_tool.ToolCaller
	availableToolFunctions []*protos.FunctionDefinition
	mcpClients             []*internal_tool_mcp.Client
}

func NewToolExecutor(logger commons.Logger) internal_agent_executor.ToolExecutor {
	return &toolExecutor{
		logger:                 logger,
		mcpClients:             make([]*internal_tool_mcp.Client, 0),
		tools:                  make(map[string]internal_tool.ToolCaller),
		availableToolFunctions: make([]*protos.FunctionDefinition, 0),
	}
}

// registerTool safely registers a tool caller and its definition
func (executor *toolExecutor) registerTool(caller internal_tool.ToolCaller, def *protos.FunctionDefinition) {
	executor.tools[caller.Name()] = caller
	executor.availableToolFunctions = append(executor.availableToolFunctions, def)
}

// getTool safely retrieves a tool caller by name
func (executor *toolExecutor) getTool(name string) (internal_tool.ToolCaller, bool) {
	caller, ok := executor.tools[name]
	return caller, ok
}

// initializeLocalTool creates a tool caller for local execution methods
func (executor *toolExecutor) initializeLocalTool(ctx context.Context, logger commons.Logger, toolOpts *internal_assistant_entity.AssistantTool, communication internal_type.Communication) (internal_tool.ToolCaller, error) {
	switch toolOpts.ExecutionMethod {
	case "knowledge_retrieval":
		return internal_tool_local.NewKnowledgeRetrievalToolCaller(ctx, logger, toolOpts, communication)
	case "api_request":
		return internal_tool_local.NewApiRequestToolCaller(ctx, logger, toolOpts, communication)
	case "endpoint_request":
		return internal_tool_local.NewEndpointToolCaller(ctx, logger, toolOpts, communication)
	case "end_of_conversation":
		return internal_tool_local.NewEndOfConversationCaller(ctx, logger, toolOpts, communication)
	case "transfer_call":
		return internal_tool_local.NewTransferCallCaller(ctx, logger, toolOpts, communication)
	default:
		return nil, errors.New("illegal tool action provided")
	}
}

// initializeTools initializes all tools (local + MCP) for the assistant
func (executor *toolExecutor) initializeTools(ctx context.Context, tools []*internal_assistant_entity.AssistantTool, communication internal_type.Communication) {
	for _, tool := range tools {
		switch tool.ExecutionMethod {
		case "mcp":
			client, err := internal_tool_mcp.NewClient(ctx, executor.logger, tool.GetOptions())
			if err != nil {
				executor.logger.Errorf("Failed to create MCP client for tool %s: %v", tool.Name, err)
				continue
			}
			executor.mcpClients = append(executor.mcpClients, client)
			definitions, err := client.ListTools(ctx)
			if err != nil {
				executor.logger.Errorf("Failed to list tools from MCP server for %s: %v", tool.Name, err)
				continue
			}
			for i, def := range definitions {
				caller := internal_tool_mcp.NewMCPToolCaller(executor.logger, client, tool.Id+uint64(i), def.Name, def)
				executor.registerTool(caller, def)
			}
		default:
			caller, err := executor.initializeLocalTool(ctx, executor.logger, tool, communication)
			if err != nil {
				executor.logger.Errorf("Failed to initialize local tool %s: %v", tool.Name, err)
				continue
			}

			def, err := caller.Definition()
			if err != nil {
				executor.logger.Errorf("Failed to get definition for tool %s: %v", tool.Name, err)
				continue
			}

			executor.registerTool(caller, def)
		}

	}
}

// Initialize sets up all tools (local + MCP) for the assistant
func (executor *toolExecutor) Initialize(ctx context.Context, communication internal_type.Communication) error {
	executor.initializeTools(ctx, communication.Assistant().AssistantTools, communication)
	return nil
}

func (executor *toolExecutor) GetFunctionDefinitions() []*protos.FunctionDefinition {
	return executor.availableToolFunctions
}

func (executor *toolExecutor) ExecuteAll(ctx context.Context, contextID string, calls []*protos.ToolCall, communication internal_type.Communication) {
	for _, call := range calls {
		funC, ok := executor.getTool(call.GetFunction().GetName())
		if !ok {
			executor.logger.Errorf("No tool found for function: %s", call.GetFunction().GetName())
			continue
		}
		funC.Call(ctx, contextID, call.GetId(), executor.parseArgument(call.GetFunction().GetArguments()), communication)
	}
}

func (executor *toolExecutor) parseArgument(arguments string) map[string]interface{} {
	var argMap map[string]interface{}
	err := json.Unmarshal([]byte(arguments), &argMap)
	if err != nil {
		return map[string]interface{}{"raw": arguments}
	} else {
		return argMap
	}
}

func (executor *toolExecutor) Close(ctx context.Context) error {
	for _, client := range executor.mcpClients {
		if err := client.Close(ctx); err != nil {
			executor.logger.Errorf("failed to close MCP client: %v", err)
		}
	}
	return nil
}
