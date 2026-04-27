// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_mcp

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// newTestOption creates a new utils.Option for testing
func newTestOption() utils.Option {
	return make(utils.Option)
}

// GetZapierURL returns the Zapier MCP URL from environment variable
// Set ZAPIER_MCP_URL environment variable to run integration tests
// func GetZapierURL() string {
// 	return "https://mcp.zapier.com/api/v1/connect?token=YOUR_TOKEN"
// }

// TestNewClient_MissingServerURL tests that NewClient returns error when server URL is missing
func TestNewClient_MissingServerURL(t *testing.T) {
	ctx := context.Background()
	logger, _ := commons.NewApplicationLogger()
	opts := newTestOption()

	_, err := NewClient(ctx, logger, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mcp.server_url is required")
}

// TestZapierMCP_Integration tests the MCP client with actual Zapier MCP server
// This test requires ZAPIER_MCP_URL environment variable to be set
// Example: export ZAPIER_MCP_URL="https://mcp.zapier.com/api/v1/connect?token=YOUR_TOKEN"
// func TestZapierMCP_Integration(t *testing.T) {
// 	zapierURL := GetZapierURL()
// 	if zapierURL == "" {
// 		t.Skip("Skipping Zapier integration test: ZAPIER_MCP_URL not set")
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
// 	defer cancel()

// 	logger, _ := commons.NewApplicationLogger()
// 	opts := newTestOption()
// 	opts["mcp.server_url"] = zapierURL
// 	opts["mcp.protocol"] = "streamable_http"
// 	opts["mcp.timeout"] = "60"

// 	// Create client
// 	client, err := NewClient(ctx, logger, opts)
// 	require.NoError(t, err, "Failed to create MCP client")
// 	defer client.Close()

// 	t.Run("ServerURL", func(t *testing.T) {
// 		assert.Equal(t, zapierURL, client.ServerURL())
// 	})

// 	t.Run("ListTools", func(t *testing.T) {
// 		tools, err := client.ListTools(ctx)
// 		require.NoError(t, err, "Failed to list tools")
// 		assert.NotEmpty(t, tools, "Expected at least one tool")

// 		t.Logf("Found %d tools:", len(tools))
// 		for _, tool := range tools {
// 			t.Logf("  - %s: %s", tool.Name, tool.Description)
// 		}
// 	})

// 	t.Run("GetTool", func(t *testing.T) {
// 		// First list tools to get a tool name
// 		tools, err := client.ListTools(ctx)
// 		require.NoError(t, err)
// 		require.NotEmpty(t, tools)

// 		// Get the first tool
// 		toolName := tools[0].Name
// 		tool, exists := client.GetTool(toolName)
// 		assert.True(t, exists, "Tool should exist")
// 		assert.Equal(t, toolName, tool.Name)
// 	})

// 	t.Run("GetTool_NotFound", func(t *testing.T) {
// 		_, exists := client.GetTool("nonexistent_tool_xyz_123")
// 		assert.False(t, exists, "Tool should not exist")
// 	})

// 	t.Run("Ping", func(t *testing.T) {
// 		err := client.Ping(ctx)
// 		assert.NoError(t, err, "Ping should succeed")
// 	})

// 	t.Run("RefreshTools", func(t *testing.T) {
// 		err := client.RefreshTools(ctx)
// 		assert.NoError(t, err, "RefreshTools should succeed")

// 		tools, err := client.ListTools(ctx)
// 		require.NoError(t, err)
// 		assert.NotEmpty(t, tools, "Should have tools after refresh")
// 	})

// 	t.Run("Execute_ToolNotFound", func(t *testing.T) {
// 		result, err := client.Execute(ctx, "nonexistent_tool_xyz_123", map[string]any{})
// 		require.NoError(t, err) // Execute returns ToolResponse with error, not Go error
// 		assert.False(t, result.Success)
// 		assert.Contains(t, result.Error, "not found")
// 	})
// }

// // TestZapierMCP_ExecuteTool tests executing a specific Zapier tool
// // This test requires ZAPIER_MCP_URL environment variable and a configured Notion integration
// func TestZapierMCP_ExecuteTool(t *testing.T) {
// 	zapierURL := GetZapierURL()
// 	if zapierURL == "" {
// 		t.Skip("Skipping Zapier integration test: ZAPIER_MCP_URL not set")
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
// 	defer cancel()

// 	logger, _ := commons.NewApplicationLogger()
// 	opts := newTestOption()
// 	opts["mcp.server_url"] = zapierURL
// 	opts["mcp.protocol"] = "streamable_http"
// 	opts["mcp.timeout"] = "60"

// 	client, err := NewClient(ctx, logger, opts)
// 	require.NoError(t, err)
// 	defer client.Close()

// 	// Check if add_tools exists (a common Zapier tool)
// 	if tool, exists := client.GetTool("add_tools"); exists {
// 		t.Logf("Found add_tools: %s", tool.Description)

// 		// Note: We don't execute add_tools as it requires user interaction
// 		// This just verifies the tool exists
// 	}

// 	// Check if edit_tools exists
// 	if tool, exists := client.GetTool("edit_tools"); exists {
// 		t.Logf("Found edit_tools: %s", tool.Description)
// 	}
// }

// TestToolResponse tests the ToolResponse helper methods
func TestToolResponse(t *testing.T) {
	t.Run("NewToolResponse_Success", func(t *testing.T) {
		resp := NewToolResponse(true)
		assert.True(t, resp.Success)
		assert.Empty(t, resp.Error)
		assert.NotNil(t, resp.Data)
	})

	t.Run("NewToolResponse_Failure", func(t *testing.T) {
		resp := NewToolResponse(false)
		assert.False(t, resp.Success)
	})

	t.Run("WithResult", func(t *testing.T) {
		resp := NewToolResponse(true).WithResult([]string{"test result"})
		assert.Equal(t, []string{"test result"}, resp.Result)
	})

	t.Run("WithError", func(t *testing.T) {
		resp := NewToolResponse(true).WithError("test error")
		assert.False(t, resp.Success, "WithError should set Success to false")
		assert.Equal(t, "test error", resp.Error)
	})

	t.Run("WithData", func(t *testing.T) {
		resp := NewToolResponse(true).WithData("key1", "value1").WithData("key2", 123)
		assert.Equal(t, "value1", resp.Data["key1"])
		assert.Equal(t, 123, resp.Data["key2"])
	})

	t.Run("ToMap_Success", func(t *testing.T) {
		resp := NewToolResponse(true).
			WithResult([]string{"test result"}).
			WithData("key", "value")

		m := resp.ToMap()
		assert.Equal(t, true, m["success"])
		assert.Equal(t, "SUCCESS", m["status"])
		assert.Equal(t, []string{"test result"}, m["result"])
		assert.NotNil(t, m["data"])
	})

	t.Run("ToMap_Failure", func(t *testing.T) {
		resp := NewToolResponse(false).WithError("error message")

		m := resp.ToMap()
		assert.Equal(t, false, m["success"])
		assert.Equal(t, "FAIL", m["status"])
		assert.Equal(t, "error message", m["error"])
	})
}

// BenchmarkListTools benchmarks the ListTools operation
// func BenchmarkListTools(b *testing.B) {
// 	zapierURL := GetZapierURL()
// 	if zapierURL == "" {
// 		b.Skip("Skipping benchmark: ZAPIER_MCP_URL not set")
// 	}

// 	ctx := context.Background()
// 	logger, _ := commons.NewApplicationLogger()
// 	opts := newTestOption()
// 	opts["mcp.server_url"] = zapierURL
// 	opts["mcp.protocol"] = "streamable_http"

// 	client, err := NewClient(ctx, logger, opts)
// 	if err != nil {
// 		b.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer client.Close()

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		_, err := client.ListTools(ctx)
// 		if err != nil {
// 			b.Fatalf("ListTools failed: %v", err)
// 		}
// 	}
// }

// // BenchmarkConnection benchmarks how long it takes to establish a connection to the MCP server
// func BenchmarkConnection(b *testing.B) {
// 	zapierURL := GetZapierURL()
// 	if zapierURL == "" {
// 		b.Skip("Skipping benchmark: ZAPIER_MCP_URL not set")
// 	}

// 	logger, _ := commons.NewApplicationLogger()

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		ctx := context.Background()
// 		opts := newTestOption()
// 		opts["mcp.server_url"] = zapierURL
// 		opts["mcp.protocol"] = "streamable_http"
// 		opts["mcp.timeout"] = "60"

// 		client, err := NewClient(ctx, logger, opts)
// 		if err != nil {
// 			b.Fatalf("Failed to create client: %v", err)
// 		}
// 		client.Close()
// 	}
// }

// BenchmarkConnectionWithToolsList benchmarks connection + initial tools list fetch
// func BenchmarkConnectionWithToolsList(b *testing.B) {
// 	zapierURL := GetZapierURL()
// 	if zapierURL == "" {
// 		b.Skip("Skipping benchmark: ZAPIER_MCP_URL not set")
// 	}

// 	logger, _ := commons.NewApplicationLogger()

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		ctx := context.Background()
// 		opts := newTestOption()
// 		opts["mcp.server_url"] = zapierURL
// 		opts["mcp.protocol"] = "streamable_http"
// 		opts["mcp.timeout"] = "60"

// 		client, err := NewClient(ctx, logger, opts)
// 		if err != nil {
// 			b.Fatalf("Failed to create client: %v", err)
// 		}

// 		// Also fetch tools as part of the benchmark
// 		_, err = client.ListTools(ctx)
// 		if err != nil {
// 			b.Fatalf("Failed to list tools: %v", err)
// 		}

// 		client.Close()
// 	}
// }

// ExampleNewClient demonstrates how to create an MCP client
func ExampleNewClient() {
	ctx := context.Background()
	logger, _ := commons.NewApplicationLogger()
	opts := newTestOption()

	// Configure the MCP client
	opts["mcp.server_url"] = "https://mcp.zapier.com/api/v1/connect?token=YOUR_TOKEN"
	opts["mcp.protocol"] = "streamable_http"
	opts["mcp.timeout"] = "60"

	// Create the client
	client, err := NewClient(ctx, logger, opts)
	if err != nil {
		panic(err)
	}
	defer client.Close(ctx)

	// List available tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		panic(err)
	}

	for _, tool := range tools {
		println(tool.Name, "-", tool.Description)
	}
}
