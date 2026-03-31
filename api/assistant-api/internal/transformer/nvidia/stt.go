// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_nvidia

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type nvidiaSTT struct {
	*nvidiaOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	contextId      string
	sttConnectedAt time.Time
	audioBuffer    bytes.Buffer
	startedAtNano  atomic.Int64

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewNvidiaSpeechToText(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.SpeechToTextTransformer, error) {
	nvidiaOpts, err := NewNvidiaOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("nvidia-stt: initializing nvidia failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &nvidiaSTT{
		ctx:          ctx2,
		ctxCancel:    contextCancel,
		onPacket:     onPacket,
		logger:       logger,
		nvidiaOption: nvidiaOpts,
	}, nil
}

func (*nvidiaSTT) Name() string {
	return "nvidia-speech-to-text"
}

func (st *nvidiaSTT) Initialize() error {
	start := time.Now()
	st.mu.Lock()
	st.sttConnectedAt = time.Now()
	ctxID := st.contextId
	st.mu.Unlock()
	st.onPacket(internal_type.ConversationEventPacket{
		ContextID: ctxID,
		Name:      "stt",
		Data: map[string]string{
			"type":     "initialized",
			"provider": st.Name(),
			"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})
	return nil
}

func (st *nvidiaSTT) Transform(ctx context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		st.mu.Lock()
		st.contextId = pkt.ContextID
		st.mu.Unlock()
		return nil
	case internal_type.InterruptionDetectedPacket:
		if pkt.Source == internal_type.InterruptionSourceVad {
			st.startedAtNano.Store(time.Now().UnixNano())
		}
		return nil
	case internal_type.UserAudioReceivedPacket:
		st.mu.Lock()
		st.audioBuffer.Write(pkt.Audio)
		audioData := make([]byte, st.audioBuffer.Len())
		copy(audioData, st.audioBuffer.Bytes())
		st.audioBuffer.Reset()
		ctxId := st.contextId
		st.mu.Unlock()

		go st.transcribe(audioData, ctxId)
		return nil
	default:
		return nil
	}
}

func (st *nvidiaSTT) transcribe(audioData []byte, ctxId string) {
	apiURL := fmt.Sprintf("https://api.nvcf.nvidia.com/v2/nvcf/pexec/functions/%s", st.GetFunctionId())

	payload := map[string]interface{}{
		"audio":         base64.StdEncoding.EncodeToString(audioData),
		"encoding":      "LINEAR_PCM",
		"sample_rate":   16000,
		"language_code": st.GetLanguage(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		st.logger.Errorf("nvidia-stt: error marshalling request: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(st.ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		st.logger.Errorf("nvidia-stt: error creating request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+st.GetKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("NVCF-INPUT-ASSET-REFERENCES", st.GetFunctionId())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		st.logger.Errorf("nvidia-stt: error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		st.logger.Errorf("nvidia-stt: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		return
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		st.logger.Errorf("nvidia-stt: error decoding response: %v", err)
		return
	}

	if result.Text != "" {
		startedNano := st.startedAtNano.Swap(0)
		if startedNano > 0 {
			st.onPacket(internal_type.UserMessageMetricPacket{
				ContextID: ctxId,
				Metrics: []*protos.Metric{{
					Name:  "stt_latency_ms",
					Value: fmt.Sprintf("%d", (time.Now().UnixNano()-startedNano)/int64(time.Millisecond)),
				}},
			})
		}

		st.onPacket(
			internal_type.InterruptionDetectedPacket{ContextID: ctxId, Source: "word"},
			internal_type.SpeechToTextPacket{
				ContextID: ctxId,
				Script:    result.Text,
				Interim:   false,
			},
			internal_type.ConversationEventPacket{
				ContextID: ctxId,
				Name:      "stt",
				Data:      map[string]string{"type": "completed"},
				Time:      time.Now(),
			},
		)
	}
}

func (st *nvidiaSTT) Close(ctx context.Context) error {
	st.ctxCancel()
	st.mu.Lock()
	ctxID := st.contextId
	connectedAt := st.sttConnectedAt
	st.sttConnectedAt = time.Time{}
	st.mu.Unlock()

	if !connectedAt.IsZero() {
		st.onPacket(
			internal_type.ConversationEventPacket{
				ContextID: ctxID,
				Name:      "stt",
				Data: map[string]string{
					"type":     "closed",
					"provider": st.Name(),
				},
				Time: time.Now(),
			},
			internal_type.ConversationMetricPacket{
				ContextID: 0,
				Metrics: []*protos.Metric{{
					Name:        type_enums.CONVERSATION_STT_DURATION.String(),
					Value:       fmt.Sprintf("%d", time.Since(connectedAt).Nanoseconds()),
					Description: "Total STT connection duration in nanoseconds",
				}},
			},
		)
	}
	return nil
}
