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
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// WebSocketTransport implements the transport.Interface for WebSocket connections
type WebSocketTransport struct {
	url                 string
	conn                *websocket.Conn
	dialer              *websocket.Dialer
	headers             http.Header
	sessionID           string
	requestID           atomic.Int64
	notificationHandler func(notification mcp.JSONRPCNotification)
	pendingRequests     map[mcp.RequestId]chan *transport.JSONRPCResponse
	mu                  sync.RWMutex
	closed              atomic.Bool
	readCtx             context.Context
	readCancel          context.CancelFunc
}

// WebSocketOption is a function that configures the WebSocket transport
type WebSocketOption func(*WebSocketTransport)

// WithWebSocketDialer sets a custom websocket dialer
func WithWebSocketDialer(dialer *websocket.Dialer) WebSocketOption {
	return func(t *WebSocketTransport) {
		t.dialer = dialer
	}
}

// WithWebSocketHeaders sets custom headers for the websocket connection
func WithWebSocketHeaders(headers map[string]string) WebSocketOption {
	return func(t *WebSocketTransport) {
		for k, v := range headers {
			t.headers.Set(k, v)
		}
	}
}

// NewWebSocketTransport creates a new WebSocket transport
func NewWebSocketTransport(url string, opts ...WebSocketOption) *WebSocketTransport {
	t := &WebSocketTransport{
		url:             url,
		headers:         make(http.Header),
		sessionID:       uuid.New().String(),
		pendingRequests: make(map[mcp.RequestId]chan *transport.JSONRPCResponse),
		dialer: &websocket.Dialer{
			HandshakeTimeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Start establishes the WebSocket connection
func (t *WebSocketTransport) Start(ctx context.Context) error {
	conn, _, err := t.dialer.DialContext(ctx, t.url, t.headers)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	t.conn = conn
	t.readCtx, t.readCancel = context.WithCancel(context.Background())

	// Start reading messages in a goroutine
	go t.readLoop()

	return nil
}

// readLoop continuously reads messages from the WebSocket connection.
// Exits on context cancellation, close, or persistent read errors.
func (t *WebSocketTransport) readLoop() {
	consecutiveErrors := 0
	for {
		select {
		case <-t.readCtx.Done():
			return
		default:
			_, message, err := t.conn.ReadMessage()
			if err != nil {
				if t.closed.Load() {
					return
				}
				consecutiveErrors++
				if consecutiveErrors > 10 {
					return
				}
				time.Sleep(time.Duration(consecutiveErrors*100) * time.Millisecond)
				continue
			}
			consecutiveErrors = 0
			t.handleMessage(message)
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (t *WebSocketTransport) handleMessage(message []byte) {
	// Try to parse as response first (has ID field)
	var response transport.JSONRPCResponse
	if err := json.Unmarshal(message, &response); err == nil {
		// Check if this is a response (has result or error)
		if response.Result != nil || response.Error != nil {
			t.mu.RLock()
			ch, exists := t.pendingRequests[response.ID]
			t.mu.RUnlock()

			if exists {
				ch <- &response
				return
			}
		}
	}

	// Try to parse as notification
	var notification mcp.JSONRPCNotification
	if err := json.Unmarshal(message, &notification); err == nil && notification.Method != "" {
		t.mu.RLock()
		handler := t.notificationHandler
		t.mu.RUnlock()

		if handler != nil {
			handler(notification)
		}
	}
}

// SendRequest sends a JSON-RPC request and waits for the response
func (t *WebSocketTransport) SendRequest(ctx context.Context, request transport.JSONRPCRequest) (*transport.JSONRPCResponse, error) {
	if t.conn == nil {
		return nil, fmt.Errorf("websocket connection not established")
	}

	// Create response channel
	responseCh := make(chan *transport.JSONRPCResponse, 1)

	t.mu.Lock()
	t.pendingRequests[request.ID] = responseCh
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.pendingRequests, request.ID)
		t.mu.Unlock()
	}()

	// Marshal and send request
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	t.mu.Lock()
	err = t.conn.WriteMessage(websocket.TextMessage, data)
	t.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response or timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-responseCh:
		return response, nil
	}
}

// SendNotification sends a JSON-RPC notification (no response expected)
func (t *WebSocketTransport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	if t.conn == nil {
		return fmt.Errorf("websocket connection not established")
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	t.mu.Lock()
	err = t.conn.WriteMessage(websocket.TextMessage, data)
	t.mu.Unlock()
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

// SetNotificationHandler sets the handler for incoming notifications
func (t *WebSocketTransport) SetNotificationHandler(handler func(notification mcp.JSONRPCNotification)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.notificationHandler = handler
}

// Close closes the WebSocket connection
func (t *WebSocketTransport) Close() error {
	if t.closed.Swap(true) {
		return nil // Already closed
	}

	if t.readCancel != nil {
		t.readCancel()
	}

	t.mu.Lock()
	conn := t.conn
	t.conn = nil
	t.mu.Unlock()
	if conn != nil {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return conn.Close()
	}

	return nil
}

// GetSessionId returns the session ID
func (t *WebSocketTransport) GetSessionId() string {
	return t.sessionID
}
