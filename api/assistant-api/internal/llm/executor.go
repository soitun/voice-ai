// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_llm

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
