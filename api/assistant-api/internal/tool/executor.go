// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool

import (
	"context"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

type ToolExecutor interface {

	// init tool executor
	//  get all the tools that is required for the assistant and intialize or do the dirty work that
	// optimize the execution or etc
	Initialize(ctx context.Context, communication internal_type.Communication) error
	/**
	 * GetFunctionDefinitions retrieves function definitions based on the provided communication.
	 *
	 * This method is responsible for returning a slice of FunctionDefinition pointers
	 * that represent the available functions or tools based on the given communication context.
	 *
	 * @param com The communication object containing context for function definition retrieval.
	 * @return A slice of FunctionDefinition pointers representing available functions or tools.
	 */
	GetFunctionDefinitions() []*protos.FunctionDefinition

	// ExecuteAll resolves and executes each tool call. Each tool pushes its
	// own packets (LLMToolCallPacket, LLMToolResultPacket)
	// via communication.OnPacket. Execution is concurrent per tool.
	ExecuteAll(ctx context.Context, contextID string, calls []*protos.ToolCall, communication internal_type.Communication)

	// clean up resources
	Close(ctx context.Context) error
}
