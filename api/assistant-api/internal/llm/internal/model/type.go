// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_model

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

// AgentPipeline is a sealed marker interface for all pipeline types.
// Only types in this package can satisfy it (unexported method).
type AgentPipeline interface {
	agentPipeline()
}

type UserTurnPipeline struct {
	Packet internal_type.UserInputPacket
}

type InjectMessagePipeline struct {
	Packet internal_type.InjectMessagePacket
}

type ToolResultPipeline struct {
	Packet internal_type.LLMToolResultPacket
}

type InterruptionPipeline struct {
	Packet internal_type.LLMInterruptPacket
}

type ResponsePipeline struct {
	Response *protos.ChatResponse
}

type ToolFollowUpPipeline struct {
	ContextID string
}

func (UserTurnPipeline) agentPipeline()      {}
func (InjectMessagePipeline) agentPipeline() {}
func (ToolResultPipeline) agentPipeline()    {}
func (InterruptionPipeline) agentPipeline()  {}
func (ResponsePipeline) agentPipeline()      {}
func (ToolFollowUpPipeline) agentPipeline()  {}
