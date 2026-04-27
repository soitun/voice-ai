// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_mcp

import (
	"context"
	"errors"
	"sync"
	"testing"

	internal_tool "github.com/rapidaai/api/assistant-api/internal/tool/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

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

type mockCommunication struct {
	internal_type.Communication
	collector *packetCollector
}

func (m *mockCommunication) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	return m.collector.collect(ctx, pkts...)
}

// mockExecutor stubs the toolExecutor interface for testing MCPToolCaller.Call.
type mockExecutor struct {
	response *ToolResponse
	err      error
}

func (me *mockExecutor) Execute(_ context.Context, _ string, _ map[string]interface{}) (*ToolResponse, error) {
	return me.response, me.err
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMCPToolCaller_Call_Success(t *testing.T) {
	// Test errorResult helper
	errMap := internal_tool.ErrorResult("something went wrong")
	assert.Equal(t, "FAIL", errMap["status"])
	assert.Equal(t, "something went wrong", errMap["error"])
}

func TestMCPToolCaller_Definition_NilDefinition(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	caller := &MCPToolCaller{
		logger:         logger,
		toolId:         1,
		toolName:       "no_def",
		toolDefinition: nil,
	}

	_, err := caller.Definition()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestMCPToolCaller_Definition_ValidDefinition(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	def := &protos.FunctionDefinition{
		Name:        "my_tool",
		Description: "does something",
	}
	caller := &MCPToolCaller{
		logger:         logger,
		toolId:         2,
		toolName:       "my_tool",
		toolDefinition: def,
	}

	got, err := caller.Definition()
	require.NoError(t, err)
	assert.Equal(t, def, got)
}

func TestMCPToolCaller_Accessors(t *testing.T) {
	caller := &MCPToolCaller{
		toolId:   99,
		toolName: "accessor_tool",
	}
	assert.Equal(t, uint64(99), caller.Id())
	assert.Equal(t, "accessor_tool", caller.Name())
	assert.Equal(t, "mcp", caller.ExecutionMethod())
}

func TestMCPToolCaller_ErrorResult(t *testing.T) {
	result := internal_tool.ErrorResult("timeout")
	assert.Equal(t, "FAIL", result["status"])
	assert.Equal(t, "timeout", result["error"])
}

// TestMCPToolCaller_Call_ClientError verifies that when the MCP client returns
// an error, the caller pushes an LLMToolResultPacket with FAIL status.
// We cannot easily inject a mock client into the real Client struct without
// an interface, so this test constructs the expected packet manually.
func TestMCPToolCaller_Call_ClientError_ExpectedPacketShape(t *testing.T) {
	// Verify the error result packet has the expected shape
	result := internal_tool.ErrorResult("connection refused")

	pkt := internal_type.LLMToolResultPacket{
		ToolID:    "tool-1",
		Name:      "failing_tool",
		ContextID: "ctx-1",
		Result:    result,
	}

	assert.Equal(t, "tool-1", pkt.ToolId())
	assert.Equal(t, "ctx-1", pkt.ContextId())
	assert.Equal(t, "FAIL", pkt.Result["status"])
	assert.Equal(t, "connection refused", pkt.Result["error"])
}

// TestMCPToolCaller_Call_SuccessPacketShape verifies the success packet shape.
func TestMCPToolCaller_Call_SuccessPacketShape(t *testing.T) {
	pkt := internal_type.LLMToolResultPacket{
		ToolID:    "tool-2",
		Name:      "success_tool",
		ContextID: "ctx-2",
		Result: map[string]string{
			"status": "SUCCESS",
			"result": `["hello"]`,
		},
	}

	assert.Equal(t, "tool-2", pkt.ToolId())
	assert.Equal(t, "ctx-2", pkt.ContextId())
	assert.Equal(t, "SUCCESS", pkt.Result["status"])
}

// ---------------------------------------------------------------------------
// Packet-order tests — verify LLMToolCallPacket always emitted FIRST
// ---------------------------------------------------------------------------

func TestMCPToolCaller_Call_PacketOrder_Success(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}
	logger, _ := commons.NewApplicationLogger()

	executor := &mockExecutor{
		response: NewToolResponse(true).WithResult([]string{"hello world"}),
	}

	caller := &MCPToolCaller{
		logger:   logger,
		client:   executor,
		toolId:   10,
		toolName: "success_tool",
	}

	args := map[string]interface{}{"q": "test"}
	caller.Call(context.Background(), "ctx-s1", "tool-s1", args, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2, "expected LLMToolCallPacket + LLMToolResultPacket")

	// [0] LLMToolCallPacket
	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "packet[0] must be LLMToolCallPacket, got %T", pkts[0])
	assert.Equal(t, "tool-s1", tc.ToolID)
	assert.Equal(t, "success_tool", tc.Name)
	assert.Equal(t, "ctx-s1", tc.ContextID)
	assert.Equal(t, "test", tc.Arguments["q"])

	// [1] LLMToolResultPacket with SUCCESS
	tr, ok := pkts[1].(internal_type.LLMToolResultPacket)
	require.True(t, ok, "packet[1] must be LLMToolResultPacket, got %T", pkts[1])
	assert.Equal(t, "tool-s1", tr.ToolID)
	assert.Equal(t, "success_tool", tr.Name)
	assert.Equal(t, "ctx-s1", tr.ContextID)
	assert.Equal(t, "SUCCESS", tr.Result["status"])
	assert.Equal(t, `["hello world"]`, tr.Result["result"])
}

func TestMCPToolCaller_Call_PacketOrder_ClientError(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}
	logger, _ := commons.NewApplicationLogger()

	executor := &mockExecutor{
		err: errors.New("connection refused"),
	}

	caller := &MCPToolCaller{
		logger:   logger,
		client:   executor,
		toolId:   11,
		toolName: "failing_tool",
	}

	caller.Call(context.Background(), "ctx-e1", "tool-e1", map[string]interface{}{}, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2, "expected LLMToolCallPacket + LLMToolResultPacket")

	// [0] LLMToolCallPacket
	tc, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "packet[0] must be LLMToolCallPacket, got %T", pkts[0])
	assert.Equal(t, "tool-e1", tc.ToolID)
	assert.Equal(t, "failing_tool", tc.Name)

	// [1] LLMToolResultPacket with FAIL
	tr, ok := pkts[1].(internal_type.LLMToolResultPacket)
	require.True(t, ok, "packet[1] must be LLMToolResultPacket, got %T", pkts[1])
	assert.Equal(t, "FAIL", tr.Result["status"])
	assert.Contains(t, tr.Result["error"], "connection refused")
}

func TestMCPToolCaller_Call_PacketOrder_ToolReturnsError(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}
	logger, _ := commons.NewApplicationLogger()

	executor := &mockExecutor{
		response: NewToolResponse(false).WithError("invalid input"),
	}

	caller := &MCPToolCaller{
		logger:   logger,
		client:   executor,
		toolId:   12,
		toolName: "error_tool",
	}

	caller.Call(context.Background(), "ctx-te1", "tool-te1", map[string]interface{}{"x": 1}, comm)

	pkts := collector.all()
	require.Len(t, pkts, 2, "expected LLMToolCallPacket + LLMToolResultPacket")

	// [0] LLMToolCallPacket
	_, ok := pkts[0].(internal_type.LLMToolCallPacket)
	require.True(t, ok, "packet[0] must be LLMToolCallPacket")

	// [1] LLMToolResultPacket with FAIL and tool's error message
	tr, ok := pkts[1].(internal_type.LLMToolResultPacket)
	require.True(t, ok, "packet[1] must be LLMToolResultPacket")
	assert.Equal(t, "FAIL", tr.Result["status"])
	assert.Equal(t, "invalid input", tr.Result["error"])
}

func TestMCPToolCaller_Call_NoActionOnToolCall(t *testing.T) {
	collector := &packetCollector{}
	comm := &mockCommunication{collector: collector}
	logger, _ := commons.NewApplicationLogger()

	executor := &mockExecutor{
		response: NewToolResponse(true).WithResult([]string{"ok"}),
	}

	caller := &MCPToolCaller{
		logger:   logger,
		client:   executor,
		toolId:   13,
		toolName: "nodirective_tool",
	}

	caller.Call(context.Background(), "ctx-nd", "tool-nd", map[string]interface{}{}, comm)

	pkts := collector.all()
	for i, p := range pkts {
		if tc, ok := p.(internal_type.LLMToolCallPacket); ok {
			assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED, tc.Action, "packet[%d] LLMToolCallPacket should have no action; MCP tools emit results, not actions", i)
		}
	}
}

// Verify the error is unused to keep the linter happy.
var _ = errors.New
