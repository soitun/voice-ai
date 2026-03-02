// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_type

import (
	"context"

	"github.com/rapidaai/protos"
)

// TalkInput defines the interface for incoming conversation messages from clients.
// It represents messages that can be sent to the assistant during a conversation stream,
// including initialization parameters, configuration updates, user messages, and metadata.
type Stream interface {
	// GetInitialization returns the conversation initialization message if present.
	// Contains initial setup parameters for the conversation stream.
	ProtoMessage()
}

// Streamer defines a bidirectional streaming interface for real-time conversation with the assistant.
// It manages the lifecycle of a conversation stream, allowing clients to send input messages
// and receive output responses asynchronously. The stream persists until explicitly closed
// or an error occurs.
type Streamer interface {
	// Context returns the context associated with this stream.
	// The context can be used to manage cancellation, timeouts, and deadlines.
	Context() context.Context

	// Recv receives the next output message from the stream.
	// It blocks until a message is available, the stream is closed, or an error occurs.
	// Returns the received message and any error encountered. If the stream is closed,
	// it should return (nil, io.EOF).
	Recv() (Stream, error)

	// Send sends an input message to the stream.
	// It returns an error if the send operation fails (e.g., stream closed, network error).
	Send(Stream) error

	// NotifyMode signals that the session is ready for the given stream mode.
	// For AUDIO mode this triggers transport setup (e.g. WebRTC handshake).
	// Called by the Talk loop after Connect() completes so that audio
	// transport is only started once STT/TTS are initialized.
	// Implementations that do not need mode-specific transport setup
	// (e.g. telephony, plain gRPC) should no-op.
	NotifyMode(mode protos.StreamMode)
}
