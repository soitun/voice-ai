// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"
	"sync"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// packetCollector is a thread-safe collector for packets pushed via OnPacket.
type packetCollector struct {
	mu   sync.Mutex
	pkts []internal_type.Packet
}

func (c *packetCollector) collect(_ context.Context, pkts ...internal_type.Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pkts = append(c.pkts, pkts...)
	return nil
}

func (c *packetCollector) all() []internal_type.Packet {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]internal_type.Packet, len(c.pkts))
	copy(out, c.pkts)
	return out
}

// mockCommunication implements just the Communication methods we need.
type mockCommunication struct {
	internal_type.Communication // embed nil for unimplemented methods
	collector                   *packetCollector
}

func (m *mockCommunication) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	return m.collector.collect(ctx, pkts...)
}

func TestEndOfConversationCaller_Call_EmitsDirectivePacket(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	caller := &endOfConversationCaller{toolCaller: toolCaller{toolOptions: &internal_assistant_entity.AssistantTool{Name: "end_of_conversation"}}}
	args := map[string]interface{}{"reason": "user_requested"}

	caller.Call(context.Background(), "ctx-123", "tool-456", args, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2, "expected LLMToolCallPacket + DirectivePacket")

	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "first packet should be LLMToolCallPacket, got %T", pkts[0])
	assert.Equal(t, "tool-456", tc.ToolID)
	assert.Equal(t, "ctx-123", tc.ContextID)

	dp, ok := pkts[1].(internal_type.DirectivePacket)
	require.True(t, ok, "expected DirectivePacket, got %T", pkts[0])
	assert.Equal(t, protos.ConversationDirective_END_CONVERSATION, dp.Directive)
	assert.Equal(t, "ctx-123", dp.ContextID)
	assert.Equal(t, "tool-456", dp.Arguments["tool_id"])
}

func TestEndOfConversationCaller_Call_NoToolResultPacket(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	caller := &endOfConversationCaller{toolCaller: toolCaller{toolOptions: &internal_assistant_entity.AssistantTool{Name: "end_of_conversation"}}}
	args := map[string]interface{}{}

	caller.Call(context.Background(), "ctx-1", "tool-1", args, comm)

	pkts := collector.all()
	for _, p := range pkts {
		_, isToolResult := p.(internal_type.LLMToolResultPacket)
		assert.False(t, isToolResult, "end_of_conversation should not emit LLMToolResultPacket")
	}
}

func TestEndOfConversationCaller_Call_PacketOrder(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	caller := &endOfConversationCaller{toolCaller: toolCaller{toolOptions: &internal_assistant_entity.AssistantTool{Name: "end_of_conversation"}}}
	args := map[string]interface{}{"reason": "done"}

	caller.Call(context.Background(), "ctx-ord-1", "tool-ord-1", args, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2, "expected exactly 2 packets: LLMToolCallPacket then DirectivePacket")

	// [0] must be LLMToolCallPacket
	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "packet[0] must be LLMToolCallPacket, got %T", pkts[0])
	assert.Equal(t, "tool-ord-1", tc.ToolID)
	assert.Equal(t, "end_of_conversation", tc.Name)
	assert.Equal(t, "ctx-ord-1", tc.ContextID)
	assert.Equal(t, "done", tc.Arguments["reason"])

	// [1] must be DirectivePacket
	dp, ok := pkts[1].(internal_type.DirectivePacket)
	require.True(t, ok, "packet[1] must be DirectivePacket, got %T", pkts[1])
	assert.Equal(t, protos.ConversationDirective_END_CONVERSATION, dp.Directive)
	assert.Equal(t, "ctx-ord-1", dp.ContextID)
	assert.Equal(t, "tool-ord-1", dp.Arguments["tool_id"])

	// Verify no LLMToolResultPacket anywhere
	for i, p := range pkts {
		_, isResult := p.(internal_type.LLMToolResultPacket)
		assert.False(t, isResult, "packet[%d] should not be LLMToolResultPacket", i)
	}
}

func TestEndOfConversationCaller_Call_PacketOrder_EmptyArgs(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}

	caller := &endOfConversationCaller{toolCaller: toolCaller{toolOptions: &internal_assistant_entity.AssistantTool{Name: "end_of_conversation"}}}

	caller.Call(context.Background(), "ctx-empty", "tool-empty", map[string]interface{}{}, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2)

	_, isCall := pkts[0].(internal_type.LLMToolCallPacket)
	assert.True(t, isCall, "packet[0] must be LLMToolCallPacket even with empty args")

	_, isDirective := pkts[1].(internal_type.DirectivePacket)
	assert.True(t, isDirective, "packet[1] must be DirectivePacket even with empty args")
}
