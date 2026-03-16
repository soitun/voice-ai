// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_speechmatics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	speechmatics_internal "github.com/rapidaai/api/assistant-api/internal/transformer/speechmatics/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type speechmaticsSTT struct {
	*speechmaticsOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu            sync.Mutex
	writeMu       sync.Mutex // serializes all WebSocket writes
	contextId     string
	startedAtNano atomic.Int64

	logger     commons.Logger
	connection *websocket.Conn
	onPacket   func(pkt ...internal_type.Packet) error
}

func NewSpeechmaticsSpeechToText(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.SpeechToTextTransformer, error) {
	smOpts, err := NewSpeechmaticsOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("speechmatics-stt: initializing speechmatics failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &speechmaticsSTT{
		ctx:                ctx2,
		ctxCancel:          contextCancel,
		onPacket:           onPacket,
		logger:             logger,
		speechmaticsOption: smOpts,
	}, nil
}

func (*speechmaticsSTT) Name() string {
	return "speechmatics-speech-to-text"
}

func (st *speechmaticsSTT) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+st.GetKey())
	conn, resp, err := websocket.DefaultDialer.Dial(SPEECHMATICS_STT_URL, header)
	if err != nil {
		st.logger.Errorf("speechmatics-stt: error while connecting %s with response %v", err, resp)
		return err
	}

	st.mu.Lock()
	st.connection = conn
	st.mu.Unlock()

	transcriptionConfig := map[string]interface{}{
		"language":        st.GetLanguage(),
		"operating_point": "enhanced",
		"enable_partials": true,
		"max_delay":       2.0,
	}
	if op, err := st.mdlOpts.GetString("listen.operating_point"); err == nil && op != "" {
		transcriptionConfig["operating_point"] = op
	}

	startMsg := map[string]interface{}{
		"message": "StartRecognition",
		"audio_format": map[string]interface{}{
			"type":        "raw",
			"encoding":    "pcm_s16le",
			"sample_rate": 16000,
		},
		"transcription_config": transcriptionConfig,
	}

	st.writeMu.Lock()
	err = conn.WriteJSON(startMsg)
	st.writeMu.Unlock()
	if err != nil {
		st.logger.Errorf("speechmatics-stt: error sending start recognition: %v", err)
		return err
	}

	// Speechmatics requires the client to wait for RecognitionStarted before sending audio.
	if err := st.waitForRecognitionStarted(conn); err != nil {
		st.logger.Errorf("speechmatics-stt: error waiting for RecognitionStarted: %v", err)
		return err
	}

	go st.readLoop(conn)
	st.onPacket(internal_type.ConversationEventPacket{
		Name: "stt",
		Data: map[string]string{
			"type":     "initialized",
			"provider": st.Name(),
			"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})
	return nil
}

// waitForRecognitionStarted reads messages from the WebSocket until it receives
// a RecognitionStarted message or an error. This must be called before the
// readLoop goroutine starts and before any audio is sent.
func (st *speechmaticsSTT) waitForRecognitionStarted(conn *websocket.Conn) error {
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetReadDeadline(time.Time{}) // clear deadline for readLoop

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("speechmatics-stt: failed reading RecognitionStarted: %w", err)
		}
		var response speechmatics_internal.SpeechmaticsSTTResponse
		if err := json.Unmarshal(msg, &response); err != nil {
			return fmt.Errorf("speechmatics-stt: failed parsing RecognitionStarted: %w", err)
		}
		if response.Message == "RecognitionStarted" {
			st.logger.Debugf("speechmatics-stt: RecognitionStarted received")
			return nil
		}
		if response.Message == "Error" {
			return fmt.Errorf("speechmatics-stt: server error during init: %s", string(msg))
		}
		st.logger.Debugf("speechmatics-stt: ignoring pre-start message: %s", response.Message)
	}
}

// readLoop owns the WebSocket connection for the lifetime of the STT session.
// It exits when the connection closes — intentionally (Close) or unexpectedly (drop).
func (st *speechmaticsSTT) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-st.ctx.Done():
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			st.mu.Lock()
			intentional := st.connection == nil // set to nil before conn.Close() on intentional paths
			if !intentional {
				st.connection = nil // unintentional drop
			}
			st.mu.Unlock()
			if !intentional {
				st.logger.Errorf("speechmatics-stt: connection lost: %v", err)
			}
			return
		}

		var response speechmatics_internal.SpeechmaticsSTTResponse
		if err := json.Unmarshal(msg, &response); err != nil {
			st.logger.Errorf("speechmatics-stt: error parsing response: %v", err)
			continue
		}

		st.mu.Lock()
		ctxId := st.contextId
		st.mu.Unlock()

		switch response.Message {
		case "AddPartialTranscript":
			transcript := response.Metadata.Transcript
			if transcript != "" && ctxId != "" {
				st.onPacket(
					internal_type.InterruptionPacket{ContextID: ctxId, Source: "word"},
					internal_type.SpeechToTextPacket{
						ContextID: ctxId,
						Script:    transcript,
						Interim:   true,
					},
					internal_type.ConversationEventPacket{
						Name: "stt",
						Data: map[string]string{"type": "interim"},
						Time: time.Now(),
					},
				)
			}
		case "AddTranscript":
			transcript := response.Metadata.Transcript
			if transcript != "" && ctxId != "" {
				startedNano := st.startedAtNano.Load()
				if startedNano > 0 {
					st.onPacket(internal_type.MessageMetricPacket{
						ContextID: ctxId,
						Metrics: []*protos.Metric{{
							Name:  "stt_latency_ms",
							Value: fmt.Sprintf("%d", (time.Now().UnixNano()-startedNano)/int64(time.Millisecond)),
						}},
					})
					st.startedAtNano.Store(0)
				}
				st.onPacket(
					internal_type.InterruptionPacket{ContextID: ctxId, Source: "word"},
					internal_type.SpeechToTextPacket{
						ContextID: ctxId,
						Script:    transcript,
						Interim:   false,
					},
					internal_type.ConversationEventPacket{
						Name: "stt",
						Data: map[string]string{"type": "completed"},
						Time: time.Now(),
					},
				)
			}
		case "Error":
			st.logger.Errorf("speechmatics-stt: server error: %s", string(msg))
			st.onPacket(internal_type.ConversationEventPacket{
				Name: "stt",
				Data: map[string]string{"type": "error"},
				Time: time.Now(),
			})
		case "AudioAdded", "EndOfTranscript", "Info":
			// Acknowledged — no action needed.
		default:
			st.logger.Debugf("speechmatics-stt: unhandled message type: %s", response.Message)
		}
	}
}

func (st *speechmaticsSTT) Transform(ctx context.Context, in internal_type.UserAudioPacket) error {
	st.startedAtNano.CompareAndSwap(0, time.Now().UnixNano())

	st.mu.Lock()
	st.contextId = in.ContextID
	conn := st.connection
	st.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("speechmatics-stt: websocket connection is not initialized")
	}

	st.writeMu.Lock()
	err := conn.WriteMessage(websocket.BinaryMessage, in.Audio)
	st.writeMu.Unlock()
	if err != nil {
		st.logger.Errorf("speechmatics-stt: error sending audio: %v", err)
		return err
	}
	return nil
}

func (st *speechmaticsSTT) Close(ctx context.Context) error {
	st.ctxCancel()
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.connection != nil {
		conn := st.connection
		st.connection = nil // mark before Close so readLoop sees intentional

		// Send EndOfStream so the server flushes any pending transcripts.
		st.writeMu.Lock()
		_ = conn.WriteJSON(map[string]interface{}{
			"message":     "EndOfStream",
			"last_seq_no": 0,
		})
		st.writeMu.Unlock()

		conn.Close()
	}
	return nil
}
