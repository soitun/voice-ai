// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_nvidia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type nvidiaTTS struct {
	*nvidiaOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	contextId      string
	ttsConnectedAt time.Time
	textBuffer     strings.Builder

	ttsStartedAt  time.Time
	ttsMetricSent bool

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewNvidiaTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	nvidiaOpts, err := NewNvidiaOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("nvidia-tts: initializing nvidia failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &nvidiaTTS{
		ctx:          ctx2,
		ctxCancel:    contextCancel,
		onPacket:     onPacket,
		logger:       logger,
		nvidiaOption: nvidiaOpts,
	}, nil
}

func (ct *nvidiaTTS) Initialize() error {
	start := time.Now()
	ct.mu.Lock()
	if ct.ttsConnectedAt.IsZero() {
		ct.ttsConnectedAt = time.Now()
	}
	ct.mu.Unlock()
	ct.onPacket(internal_type.ConversationEventPacket{
		Name: "tts",
		Data: map[string]string{
			"type":     "initialized",
			"provider": ct.Name(),
			"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})
	return nil
}

func (*nvidiaTTS) Name() string {
	return "nvidia-text-to-speech"
}

func (t *nvidiaTTS) flush() {
	t.mu.Lock()
	text := t.textBuffer.String()
	t.textBuffer.Reset()
	ctxId := t.contextId
	t.mu.Unlock()

	if text == "" || ctxId == "" {
		return
	}

	go t.streamHTTPTTS(text, ctxId)
}

func (t *nvidiaTTS) streamHTTPTTS(text string, ctxId string) {
	apiURL := fmt.Sprintf("https://api.nvcf.nvidia.com/v2/nvcf/pexec/functions/%s", t.GetFunctionId())

	payload := map[string]interface{}{
		"text":          text,
		"voice_name":    t.GetVoice(),
		"language_code": t.GetLanguage(),
		"encoding":      "LINEAR_PCM",
		"sample_rate":   16000,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.logger.Errorf("nvidia-tts: error marshalling request: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(t.ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		t.logger.Errorf("nvidia-tts: error creating request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+t.GetKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("NVCF-INPUT-ASSET-REFERENCES", t.GetFunctionId())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.logger.Errorf("nvidia-tts: error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.logger.Errorf("nvidia-tts: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		return
	}

	buf := make([]byte, 4096)
	firstChunk := true
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}
		n, err := resp.Body.Read(buf)
		if n > 0 {
			audioChunk := make([]byte, n)
			copy(audioChunk, buf[:n])

			if firstChunk {
				firstChunk = false
				t.mu.Lock()
				startedAt := t.ttsStartedAt
				metricSent := t.ttsMetricSent
				if !metricSent && !startedAt.IsZero() {
					t.ttsMetricSent = true
				}
				t.mu.Unlock()
				if !metricSent && !startedAt.IsZero() {
					t.onPacket(internal_type.AssistantMessageMetricPacket{
						ContextID: ctxId,
						Metrics: []*protos.Metric{{
							Name:  "tts_latency_ms",
							Value: fmt.Sprintf("%d", time.Since(startedAt).Milliseconds()),
						}},
					})
				}
			}

			t.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: ctxId, AudioChunk: audioChunk})
		}
		if err != nil {
			if err != io.EOF {
				t.logger.Errorf("nvidia-tts: error reading response body: %v", err)
			}
			break
		}
	}

	t.onPacket(
		internal_type.TextToSpeechEndPacket{ContextID: ctxId},
		internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{"type": "completed"},
			Time: time.Now(),
		},
	)
}

func (t *nvidiaTTS) Transform(ctx context.Context, in internal_type.LLMPacket) error {
	t.mu.Lock()
	currentCtx := t.contextId
	if in.ContextId() != t.contextId {
		t.contextId = in.ContextId()
		t.ttsStartedAt = time.Time{}
		t.ttsMetricSent = false
		t.textBuffer.Reset()
	}
	t.mu.Unlock()

	switch input := in.(type) {
	case internal_type.InterruptionDetectedPacket:
		if currentCtx != "" {
			t.mu.Lock()
			t.ttsStartedAt = time.Time{}
			t.ttsMetricSent = false
			t.textBuffer.Reset()
			t.mu.Unlock()
			t.onPacket(internal_type.ConversationEventPacket{
				Name: "tts",
				Data: map[string]string{"type": "interrupted"},
				Time: time.Now(),
			})
		}
		return nil
	case internal_type.LLMResponseDeltaPacket:
		t.mu.Lock()
		if t.ttsStartedAt.IsZero() {
			t.ttsStartedAt = time.Now()
		}
		t.textBuffer.WriteString(input.Text)
		t.mu.Unlock()
		t.onPacket(internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{
				"type": "speaking",
				"text": input.Text,
			},
			Time: time.Now(),
		})
	case internal_type.LLMResponseDonePacket:
		t.flush()
		return nil
	default:
		return fmt.Errorf("nvidia-tts: unsupported input type %T", in)
	}
	return nil
}

func (t *nvidiaTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.mu.Lock()
	ctxID := t.contextId
	connectedAt := t.ttsConnectedAt
	t.ttsConnectedAt = time.Time{}
	t.mu.Unlock()

	if !connectedAt.IsZero() {
		t.onPacket(
			internal_type.ConversationEventPacket{
				ContextID: ctxID,
				Name:      "tts",
				Data: map[string]string{
					"type":     "closed",
					"provider": t.Name(),
				},
				Time: time.Now(),
			},
			internal_type.ConversationMetricPacket{
				ContextID: 0,
				Metrics: []*protos.Metric{{
					Name:        type_enums.CONVERSATION_TTS_DURATION.String(),
					Value:       fmt.Sprintf("%d", time.Since(connectedAt).Nanoseconds()),
					Description: "Total TTS connection duration in nanoseconds",
				}},
			},
		)
	}
	return nil
}
