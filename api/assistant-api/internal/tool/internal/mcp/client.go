// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// Protocol constants for MCP transports
const (
	ProtocolSSE            = "sse"             // Traditional SSE transport
	ProtocolStreamableHTTP = "streamable_http" // Streamable HTTP transport (default)
	ProtocolWebSocket      = "websocket"       // WebSocket transport
)

type Config struct {
	// MCP server endpoint
	// Examples:
	//   SSE:             http://localhost:3000/mcp
	//   Streamable HTTP: https://mcp.zapier.com/api/v1/connect?token=xxx
	//   WebSocket:       ws://localhost:3000/mcp or wss://...
	ServerURL string

	// Transport protocol: "sse" (default), "streamable_http", or "websocket"
	// - sse: Traditional Server-Sent Events transport (default)
	// - streamable_http: HTTP-based transport (works with Zapier MCP, etc.)
	// - websocket: WebSocket transport (ws:// or wss://)
	Protocol string

	// HTTP timeout in seconds (default: 60)
	Timeout int

	// Custom headers to include in requests
	Headers map[string]string
}

type ToolResponse struct {
	Success bool           `json:"success"`
	Result  []string       `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

func NewToolResponse(success bool) *ToolResponse {
	return &ToolResponse{
		Success: success,
		Data:    map[string]any{},
	}
}

func (r *ToolResponse) WithResult(result []string) *ToolResponse {
	r.Result = result
	return r
}

func (r *ToolResponse) WithError(err string) *ToolResponse {
	r.Error = err
	r.Success = false
	return r
}

func (r *ToolResponse) WithData(key string, value any) *ToolResponse {
	r.Data[key] = value
	return r
}

func (r *ToolResponse) ToMap() map[string]interface{} {
	out := map[string]interface{}{
		"success": r.Success,
		"status":  "FAIL",
	}
	if r.Success {
		out["status"] = "SUCCESS"
	}
	if r.Result != nil {
		out["result"] = r.Result
	}
	if r.Error != "" {
		out["error"] = r.Error
	}
	if len(r.Data) > 0 {
		out["data"] = r.Data
	}
	return out
}

// -----------------------------------------------------------------------------
// Client - MCP Client using mark3labs/mcp-go
// -----------------------------------------------------------------------------

type Client struct {
	logger    commons.Logger
	client    *client.Client
	tools     map[string]mcp.Tool
	serverURL string
	config    *Config
}

func NewClient(ctx context.Context, logger commons.Logger, opts utils.Option) (*Client, error) {

	// ------------------------------------------------------------------
	// Required option
	// ------------------------------------------------------------------
	serverURL, err := opts.GetString("mcp.server_url")
	if err != nil || serverURL == "" {
		return nil, fmt.Errorf("mcp.server_url is required")
	}

	// ------------------------------------------------------------------
	// Defaults
	// ------------------------------------------------------------------
	config := &Config{
		ServerURL: serverURL,
		Protocol:  ProtocolSSE, // Default to SSE
		Timeout:   60,
		Headers:   map[string]string{},
	}

	// Optional protocol - supports: "sse", "SSE", "streamable_http", "Streamable HTTP"
	if protocol, err := opts.GetString("mcp.protocol"); err == nil && protocol != "" {
		// Normalize protocol value
		normalizedProtocol := strings.ToLower(strings.TrimSpace(protocol))
		normalizedProtocol = strings.ReplaceAll(normalizedProtocol, " ", "_")
		config.Protocol = normalizedProtocol
	}

	// Optional timeout
	if timeout, err := opts.GetString("mcp.timeout"); err == nil && timeout != "" {
		if t, e := strconv.Atoi(timeout); e == nil {
			config.Timeout = t
		}
	}

	// Optional headers - can be JSON string or map
	if headersRaw, ok := opts["mcp.headers"]; ok && headersRaw != nil {
		switch h := headersRaw.(type) {
		case string:
			if h != "" && h != "{}" {
				if err := json.Unmarshal([]byte(h), &config.Headers); err != nil {
					logger.Warnf("Failed to parse mcp.headers JSON string: %v", err)
				}
			}
		case map[string]string:
			for k, v := range h {
				config.Headers[k] = v
			}
		case map[string]interface{}:
			for k, v := range h {
				if s, ok := v.(string); ok {
					config.Headers[k] = s
				}
			}
		}
	}

	// ------------------------------------------------------------------
	// Create HTTP client with timeout
	// ------------------------------------------------------------------
	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	// ------------------------------------------------------------------
	// Create MCP client based on protocol
	// ------------------------------------------------------------------
	var mcpClient *client.Client

	switch config.Protocol {
	case ProtocolStreamableHTTP:
		// Streamable HTTP transport (works with Zapier MCP, etc.)
		// Requires Accept header with both application/json and text/event-stream
		httpHeaders := map[string]string{
			"Accept": "application/json, text/event-stream",
		}
		for k, v := range config.Headers {
			httpHeaders[k] = v
		}

		mcpClient, err = client.NewStreamableHttpClient(config.ServerURL,
			transport.WithHTTPBasicClient(httpClient),
			transport.WithHTTPHeaders(httpHeaders),
		)

	case ProtocolWebSocket:
		// WebSocket transport using custom implementation
		wsTransport := NewWebSocketTransport(config.ServerURL,
			WithWebSocketHeaders(config.Headers),
		)
		mcpClient = client.NewClient(wsTransport)

	case ProtocolSSE:
		// SSE transport for traditional SSE servers
		sseHeaders := map[string]string{
			"Accept": "text/event-stream",
		}
		for k, v := range config.Headers {
			sseHeaders[k] = v
		}

		mcpClient, err = client.NewSSEMCPClient(config.ServerURL,
			transport.WithHTTPClient(httpClient),
			transport.WithHeaders(sseHeaders),
		)

	default:
		// Unknown protocol - default to SSE
		logger.Warnf("Unknown protocol %q, defaulting to SSE", config.Protocol)
		config.Protocol = ProtocolSSE
		sseHeaders := map[string]string{
			"Accept": "text/event-stream",
		}
		for k, v := range config.Headers {
			sseHeaders[k] = v
		}

		mcpClient, err = client.NewSSEMCPClient(config.ServerURL,
			transport.WithHTTPClient(httpClient),
			transport.WithHeaders(sseHeaders),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client for %s: %w", config.ServerURL, err)
	}

	c := &Client{
		logger:    logger,
		client:    mcpClient,
		tools:     make(map[string]mcp.Tool),
		serverURL: config.ServerURL,
		config:    config,
	}

	// ------------------------------------------------------------------
	// Connect and initialize
	// ------------------------------------------------------------------
	if err := c.connect(ctx); err != nil {
		mcpClient.Close()
		return nil, err
	}

	return c, nil
}

// connect establishes connection, initializes the client, and loads available tools
func (c *Client) connect(ctx context.Context) error {
	// Start the transport
	if err := c.client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	// Initialize the MCP session
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "Rapida-MCP-Client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err := c.client.Initialize(ctx, initRequest)
	if err != nil {
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	// Load available tools
	toolsResp, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	for _, tool := range toolsResp.Tools {
		c.tools[tool.Name] = tool
	}

	return nil
}

// ServerURL returns the server URL
func (c *Client) ServerURL() string {
	return c.serverURL
}

// ListTools fetches all available tools from the MCP server
func (c *Client) ListTools(ctx context.Context) ([]*protos.FunctionDefinition, error) {
	definitions := make([]*protos.FunctionDefinition, 0, len(c.tools))
	for _, tool := range c.tools {
		def := c.convertTool(tool)
		definitions = append(definitions, def)
	}

	c.logger.Infof("Found %d tools on MCP server %s", len(definitions), c.serverURL)
	return definitions, nil
}

// RefreshTools reloads tools from the server
func (c *Client) RefreshTools(ctx context.Context) error {
	toolsResp, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	c.tools = make(map[string]mcp.Tool)
	for _, tool := range toolsResp.Tools {
		c.tools[tool.Name] = tool
	}

	return nil
}

// Execute calls an MCP tool and returns the response
func (c *Client) Execute(ctx context.Context, toolName string, args map[string]any) (*ToolResponse, error) {
	if _, exists := c.tools[toolName]; !exists {
		return NewToolResponse(false).WithError(fmt.Sprintf("tool %q not found", toolName)), nil
	}

	request := mcp.CallToolRequest{}
	request.Params.Name = toolName
	request.Params.Arguments = args

	result, err := c.client.CallTool(ctx, request)
	if err != nil {
		return NewToolResponse(false).WithError(err.Error()), nil
	}

	return c.convertResult(result), nil
}

// ExecuteWithTimeout calls an MCP tool with a timeout
func (c *Client) ExecuteWithTimeout(toolName string, args map[string]any, timeout time.Duration) (*ToolResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Execute(ctx, toolName, args)
}

// Ping checks if the connection is alive
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx)
}

// Close closes all active sessions
func (c *Client) Close(ctx context.Context) error {
	return c.client.Close()
}

// GetTool returns a specific tool by name
func (c *Client) GetTool(name string) (mcp.Tool, bool) {
	tool, exists := c.tools[name]
	return tool, exists
}

// convertTool converts MCP Tool to FunctionDefinition
func (c *Client) convertTool(tool mcp.Tool) *protos.FunctionDefinition {
	params := c.convertSchema(tool.InputSchema)

	// Ensure properties map is never nil - protobuf omitempty will skip empty maps,
	// but we need {"type": "object", "properties": {}} for valid JSON schema.
	if len(params.Properties) == 0 {
		params.Properties = make(map[string]*protos.FunctionParameterProperty)
	}

	return &protos.FunctionDefinition{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  params,
	}
}

// convertSchema converts MCP inputSchema to FunctionParameter
// Ensures valid JSON schema format even when no properties exist
func (c *Client) convertSchema(schema mcp.ToolInputSchema) *protos.FunctionParameter {
	params := &protos.FunctionParameter{
		Type:       "object",
		Properties: make(map[string]*protos.FunctionParameterProperty),
		Required:   make([]string, 0),
	}

	if schema.Type != "" {
		params.Type = schema.Type
	}

	// Copy required fields
	params.Required = append(params.Required, schema.Required...)

	// Convert properties
	if schema.Properties != nil {
		for name, prop := range schema.Properties {
			p := &protos.FunctionParameterProperty{}

			// Convert property map to struct
			propMap, ok := prop.(map[string]any)
			if !ok {
				continue
			}

			if t, ok := propMap["type"].(string); ok {
				p.Type = t
			}
			if d, ok := propMap["description"].(string); ok {
				p.Description = d
			}
			if e, ok := propMap["enum"].([]any); ok {
				for _, v := range e {
					if s, ok := v.(string); ok {
						p.Enum = append(p.Enum, s)
					}
				}
			}
			params.Properties[name] = p
		}
	}

	return params
}

// convertResult converts MCP CallToolResult to ToolResponse
func (c *Client) convertResult(result *mcp.CallToolResult) *ToolResponse {
	resp := NewToolResponse(!result.IsError)
	var texts []string
	for _, content := range result.Content {
		switch ct := content.(type) {
		case mcp.TextContent:
			texts = append(texts, ct.Text)
		}
	}
	if result.IsError && len(texts) > 0 {
		resp.WithError(texts[0])
	} else {
		resp.WithResult(texts)
	}

	return resp
}
