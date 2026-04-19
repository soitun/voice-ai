// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor

import (
	"context"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

/*
AssistantExecutor and its related interfaces define the contract for executing
assistant-related actions in the system. These interfaces are crucial for
implementing various modes of interaction with the assistant, such as text-based
chat and voice communication.

AssistantMessageExecutor handles text-based chat interactions. It defines a Chat
method that processes messaging requests and returns any errors encountered during
the chat process.

AssistantTalkExecutor is responsible for voice-based interactions. Its Talk method
takes care of processing talking requests and handles any errors that may occur
during the voice interaction.

AssistantExecutor combines both text and voice capabilities, allowing for a more
versatile assistant that can handle multiple modes of communication. By embedding
both AssistantMessageExecutor and AssistantTalkExecutor, it ensures that any
implementing type can handle both chat and talk functionalities.

These interfaces provide a clean separation of concerns and allow for easy
extension of the assistant's capabilities in the future. They also promote
loose coupling between the assistant's implementation and the rest of the system,
making it easier to maintain and evolve the codebase over time.
*/

type AssistantExecutor interface {

	// Initialize sets up all fields after creation
	Initialize(ctx context.Context, communication internal_type.Communication, cfg *protos.ConversationInitialization) error

	// name
	Name() string

	// Execute processes an incoming packet
	Execute(ctx context.Context, communication internal_type.Communication, pctk internal_type.Packet) error

	// disconnect
	Close(ctx context.Context) error
}

/**
 * ToolExecutor is an interface that defines methods for executing tools and retrieving function definitions.
 *
 * This interface provides a contract for implementing tool execution functionality,
 * allowing for flexible and extensible tool management within the system.
 */

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
	// own packets (LLMToolCallPacket, LLMToolResultPacket, DirectivePacket)
	// via communication.OnPacket. Execution is concurrent per tool.
	ExecuteAll(ctx context.Context, contextID string, calls []*protos.ToolCall, communication internal_type.Communication)

	// clean up resources
	Close(ctx context.Context) error
}
