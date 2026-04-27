// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_tool "github.com/rapidaai/api/assistant-api/internal/tool/internal"
	internal_tool_local "github.com/rapidaai/api/assistant-api/internal/tool/internal/local"
	internal_tool_mcp "github.com/rapidaai/api/assistant-api/internal/tool/internal/mcp"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"

	"github.com/rapidaai/protos"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type toolExecutor struct {
	logger                 commons.Logger
	tools                  map[string]internal_tool.ToolCaller
	availableToolFunctions []*protos.FunctionDefinition
	mcpClients             []*internal_tool_mcp.Client
	conditionMatcher       *toolConditionMatcher
}

type toolRegistration struct {
	caller internal_tool.ToolCaller
	def    *protos.FunctionDefinition
}

func NewToolExecutor(logger commons.Logger) ToolExecutor {
	return &toolExecutor{
		logger:                 logger,
		mcpClients:             make([]*internal_tool_mcp.Client, 0),
		tools:                  make(map[string]internal_tool.ToolCaller),
		availableToolFunctions: make([]*protos.FunctionDefinition, 0),
		conditionMatcher:       newToolConditionMatcher(),
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

func (executor *toolExecutor) discoverTools(communication internal_type.Communication) []*internal_assistant_entity.AssistantTool {
	if communication.Assistant() == nil {
		return nil
	}
	return communication.Assistant().AssistantTools
}

func (executor *toolExecutor) filterToolsByCondition(
	tools []*internal_assistant_entity.AssistantTool,
	communication internal_type.Communication,
) []*internal_assistant_entity.AssistantTool {
	if executor.conditionMatcher == nil {
		executor.conditionMatcher = newToolConditionMatcher()
	}
	filtered := make([]*internal_assistant_entity.AssistantTool, 0, len(tools))
	for _, tool := range tools {
		opts := tool.GetOptions()
		rawCondition, err := opts.GetString("tool.condition")
		if err != nil || rawCondition == "" {
			filtered = append(filtered, tool)
			continue
		}

		allowed, evalErr := executor.conditionMatcher.Evaluate(rawCondition, string(communication.GetSource()))
		if evalErr != nil {
			executor.logger.Warnf("invalid tool.condition for tool %s, excluding tool: %v", tool.Name, evalErr)
			continue
		}
		if allowed {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

func (executor *toolExecutor) buildLocalToolRegistration(
	ctx context.Context,
	tool *internal_assistant_entity.AssistantTool,
	communication internal_type.Communication,
) ([]toolRegistration, error) {
	caller, err := executor.initializeLocalTool(ctx, executor.logger, tool, communication)
	if err != nil {
		return nil, err
	}
	def, err := caller.Definition()
	if err != nil {
		return nil, err
	}
	return []toolRegistration{{caller: caller, def: def}}, nil
}

func (executor *toolExecutor) buildMCPToolRegistrations(
	ctx context.Context,
	tool *internal_assistant_entity.AssistantTool,
) ([]toolRegistration, error) {
	client, err := internal_tool_mcp.NewClient(ctx, executor.logger, tool.GetOptions())
	if err != nil {
		return nil, err
	}
	executor.mcpClients = append(executor.mcpClients, client)

	definitions, err := client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	registrations := make([]toolRegistration, 0, len(definitions))
	for i, def := range definitions {
		caller := internal_tool_mcp.NewMCPToolCaller(executor.logger, client, tool.Id+uint64(i), def.Name, def)
		registrations = append(registrations, toolRegistration{caller: caller, def: def})
	}
	return registrations, nil
}

func (executor *toolExecutor) buildToolRegistrations(
	ctx context.Context,
	tool *internal_assistant_entity.AssistantTool,
	communication internal_type.Communication,
) ([]toolRegistration, error) {
	switch tool.ExecutionMethod {
	case "mcp":
		return executor.buildMCPToolRegistrations(ctx, tool)
	default:
		return executor.buildLocalToolRegistration(ctx, tool, communication)
	}
}

func (executor *toolExecutor) registerToolDefinitions(registrations []toolRegistration) {
	for _, registration := range registrations {
		executor.registerTool(registration.caller, registration.def)
	}
}

func (executor *toolExecutor) executeTools(ctx context.Context, contextID string, calls []*protos.ToolCall, communication internal_type.Communication) {
	for _, call := range calls {
		funC, ok := executor.getTool(call.GetFunction().GetName())
		if !ok {
			executor.logger.Errorf("No tool found for function: %s", call.GetFunction().GetName())
			continue
		}
		funC.Call(ctx, contextID, call.GetId(), executor.parseArgument(call.GetFunction().GetArguments()), communication)
	}
}

// Run dispatches strongly-typed tool pipeline stages.
func (executor *toolExecutor) Run(ctx context.Context, pipeline ToolPipeline) error {
	switch v := pipeline.(type) {
	case DiscoverToolsPipeline:
		tools := executor.discoverTools(v.Communication)
		return executor.Run(ctx, FilterToolsPipeline{
			Tools:         tools,
			Communication: v.Communication,
		})
	case FilterToolsPipeline:
		filtered := executor.filterToolsByCondition(v.Tools, v.Communication)
		return executor.Run(ctx, BuildAndRegisterToolsPipeline{
			Tools:         filtered,
			Communication: v.Communication,
		})
	case BuildAndRegisterToolsPipeline:
		for _, tool := range v.Tools {
			registrations, err := executor.buildToolRegistrations(ctx, tool, v.Communication)
			if err != nil {
				executor.logger.Errorf("Failed to initialize tool %s: %v", tool.Name, err)
				continue
			}
			executor.registerToolDefinitions(registrations)
		}
		return nil
	case ExecuteToolsPipeline:
		executor.executeTools(ctx, v.ContextID, v.Calls, v.Communication)
		return nil
	default:
		return fmt.Errorf("unknown tool pipeline type: %T", pipeline)
	}
}

func (executor *toolExecutor) initializeToolPipeline(
	ctx context.Context,
	communication internal_type.Communication,
) {
	if err := executor.Run(ctx, DiscoverToolsPipeline{Communication: communication}); err != nil {
		executor.logger.Errorf("tool initialize pipeline failed: %v", err)
	}
}

// Initialize sets up all tools (local + MCP) for the assistant
func (executor *toolExecutor) Initialize(ctx context.Context, communication internal_type.Communication) error {
	executor.initializeToolPipeline(ctx, communication)
	return nil
}

func (executor *toolExecutor) GetFunctionDefinitions() []*protos.FunctionDefinition {
	return executor.availableToolFunctions
}

func (executor *toolExecutor) ExecuteAll(ctx context.Context, contextID string, calls []*protos.ToolCall, communication internal_type.Communication) {
	utils.Go(ctx, func() {
		if err := executor.Run(ctx, ExecuteToolsPipeline{
			ContextID:     contextID,
			Calls:         calls,
			Communication: communication,
		}); err != nil {
			executor.logger.Errorf("tool execute pipeline failed: %v", err)
		}
	})
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
