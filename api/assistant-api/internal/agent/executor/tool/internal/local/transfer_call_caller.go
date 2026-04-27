// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"
	"fmt"
	"time"

	internal_tool "github.com/rapidaai/api/assistant-api/internal/agent/executor/tool/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type PostTransferAction string

const (
	PostTransferActionEndCall  PostTransferAction = "end_call"
	PostTransferActionResumeAI PostTransferAction = "resume_ai"
)

type transferCallCaller struct {
	toolCaller
	transferTo         string
	transferDelay      uint32
	transferMessage    string
	postTransferAction PostTransferAction
}

func (tc *transferCallCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) {
	transferTo := tc.transferTo
	transferMessage := tc.transferMessage
	transferDelay := tc.transferDelay
	postTransferAction := tc.postTransferAction

	if to, ok := args["transfer_to"].(string); ok && to != "" {
		transferTo = to
	}

	if msg, ok := args["transfer_message"].(string); ok && msg != "" {
		transferMessage = msg
	}

	if delay, ok := args["transfer_delay"].(float64); ok {
		transferDelay = uint32(delay)
	}

	if action, ok := args["post_transfer_action"].(string); ok && action != "" {
		postTransferAction = getPostTransferAction(action)
	}

	if transferMessage != "" {
		communication.OnPacket(ctx,
			internal_type.InjectMessagePacket{ContextID: contextID, Text: transferMessage},
		)
	}

	if transferDelay > 0 {
		time.Sleep(time.Duration(transferDelay) * time.Millisecond)
	}

	arguments := map[string]string{
		"transfer_to":          transferTo,
		"message":              transferMessage,
		"transfer_message":     transferMessage,
		"transfer_delay":       fmt.Sprintf("%d", transferDelay),
		"post_transfer_action": string(PostTransferActionEndCall),
	}
	if postTransferAction != "" {
		arguments["post_transfer_action"] = string(postTransferAction)
	}

	communication.OnPacket(ctx,
		internal_type.LLMToolCallPacket{
			ToolID:    toolId,
			Name:      tc.Name(),
			ContextID: contextID,
			Action:    protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
			Arguments: arguments,
		})
}

func NewTransferCallCaller(ctx context.Context, logger commons.Logger, toolOptions *internal_assistant_entity.AssistantTool, communication internal_type.Communication,
) (internal_tool.ToolCaller, error) {
	opts := toolOptions.GetOptions()
	transferTo, err := opts.GetString("tool.transfer_to")
	if err != nil {
		return nil, fmt.Errorf("tool.transfer_to is required: %v", err)
	}
	transferDelay, _ := opts.GetUint32("tool.transfer_delay")
	transferMessage, _ := opts.GetString("tool.transfer_message")
	postTransferActionRaw, _ := opts.GetString("tool.post_transfer_action")
	return &transferCallCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
		transferMessage:    transferMessage,
		transferDelay:      transferDelay,
		transferTo:         transferTo,
		postTransferAction: getPostTransferAction(postTransferActionRaw),
	}, nil
}

func getPostTransferAction(raw string) PostTransferAction {
	switch raw {
	case "end_call":
		return PostTransferActionEndCall
	case "resume_ai":
		return PostTransferActionResumeAI
	default:
		return PostTransferActionEndCall
	}
}
