// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferCallCaller_Call_EmitsDirective(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	caller := &transferCallCaller{toolCaller: toolCaller{toolOptions: &internal_assistant_entity.AssistantTool{Name: "transfer_call"}}, transferTo: "+15551234567"}
	args := map[string]interface{}{"extra": "data"}

	caller.Call(context.Background(), "ctx-100", "tool-200", args, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2, "expected LLMToolCallPacket + DirectivePacket")

	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "first packet should be LLMToolCallPacket, got %T", pkts[0])
	assert.Equal(t, "tool-200", tc.ToolID)
	assert.Equal(t, "ctx-100", tc.ContextID)

	dp, ok := pkts[1].(internal_type.DirectivePacket)
	require.True(t, ok, "expected DirectivePacket, got %T", pkts[0])
	assert.Equal(t, protos.ConversationDirective_TRANSFER_CONVERSATION, dp.Directive)
	assert.Equal(t, "ctx-100", dp.ContextID)
	assert.Equal(t, "+15551234567", dp.Arguments["to"])
	assert.Equal(t, "tool-200", dp.Arguments["tool_id"])
	assert.Equal(t, "ctx-100", dp.Arguments["context_id"])
}

func TestTransferCallCaller_Call_NoToolResultPacket(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	caller := &transferCallCaller{toolCaller: toolCaller{toolOptions: &internal_assistant_entity.AssistantTool{Name: "transfer_call"}}, transferTo: "sip:agent@example.com"}
	args := map[string]interface{}{}

	caller.Call(context.Background(), "ctx-1", "tool-1", args, comm)

	pkts := collector.all()
	for _, p := range pkts {
		_, isToolResult := p.(internal_type.LLMToolResultPacket)
		assert.False(t, isToolResult, "transfer_call should not emit LLMToolResultPacket")
	}
}

func TestTransferCallCaller_Call_PacketOrder(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	caller := &transferCallCaller{
		toolCaller: toolCaller{toolOptions: &internal_assistant_entity.AssistantTool{Name: "transfer_call"}},
		transferTo: "+15559876543",
	}
	args := map[string]interface{}{"reason": "escalation"}

	caller.Call(context.Background(), "ctx-ord-2", "tool-ord-2", args, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2, "expected exactly 2 packets: LLMToolCallPacket then DirectivePacket")

	// [0] must be LLMToolCallPacket
	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "packet[0] must be LLMToolCallPacket, got %T", pkts[0])
	assert.Equal(t, "tool-ord-2", tc.ToolID)
	assert.Equal(t, "transfer_call", tc.Name)
	assert.Equal(t, "ctx-ord-2", tc.ContextID)
	assert.Equal(t, "escalation", tc.Arguments["reason"])

	// [1] must be DirectivePacket with TRANSFER_CONVERSATION directive
	dp, ok := pkts[1].(internal_type.DirectivePacket)
	require.True(t, ok, "packet[1] must be DirectivePacket, got %T", pkts[1])
	assert.Equal(t, protos.ConversationDirective_TRANSFER_CONVERSATION, dp.Directive)
	assert.Equal(t, "ctx-ord-2", dp.ContextID)
	assert.Equal(t, "+15559876543", dp.Arguments["to"])
	assert.Equal(t, "tool-ord-2", dp.Arguments["tool_id"])
	assert.Equal(t, "ctx-ord-2", dp.Arguments["context_id"])

	// Verify no LLMToolResultPacket anywhere
	for i, p := range pkts {
		_, isResult := p.(internal_type.LLMToolResultPacket)
		assert.False(t, isResult, "packet[%d] should not be LLMToolResultPacket", i)
	}
}

func TestTransferCallCaller_Call_PacketOrder_SIPTarget(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	caller := &transferCallCaller{
		toolCaller: toolCaller{toolOptions: &internal_assistant_entity.AssistantTool{Name: "transfer_call"}},
		transferTo: "sip:support@pbx.example.com",
	}
	args := map[string]interface{}{}

	caller.Call(context.Background(), "ctx-sip", "tool-sip", args, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2)

	// [0] LLMToolCallPacket
	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "packet[0] must be LLMToolCallPacket")
	assert.Equal(t, "tool-sip", tc.ToolID)

	// [1] DirectivePacket with SIP URI in "to"
	dp, ok := pkts[1].(internal_type.DirectivePacket)
	require.True(t, ok, "packet[1] must be DirectivePacket")
	assert.Equal(t, protos.ConversationDirective_TRANSFER_CONVERSATION, dp.Directive)
	assert.Equal(t, "sip:support@pbx.example.com", dp.Arguments["to"])
}
