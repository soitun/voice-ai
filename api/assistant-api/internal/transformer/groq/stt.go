// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	groq_internal "github.com/rapidaai/api/assistant-api/internal/transformer/groq/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type groqSTT struct {
	*groqOption
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

func NewGroqSpeechToText(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.SpeechToTextTransformer, error) {
	groqOpts, err := NewGroqOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("groq-stt: initializing groq failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &groqSTT{
		ctx:        ctx2,
		ctxCancel:  contextCancel,
		onPacket:   onPacket,
		logger:     logger,
		groqOption: groqOpts,
	}, nil
}

func (*groqSTT) Name() string {
	return "groq-speech-to-text"
}

func (st *groqSTT) Initialize() error {
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

func (st *groqSTT) Transform(ctx context.Context, in internal_type.Packet) error {
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

func (st *groqSTT) transcribe(audioData []byte, ctxId string) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		st.logger.Errorf("groq-stt: error creating form file: %v", err)
		return
	}

	// Write WAV header for raw PCM
	wavHeader := createWAVHeader(len(audioData), 16000, 1, 16)
	part.Write(wavHeader)
	part.Write(audioData)

	writer.WriteField("model", st.GetSTTModel())
	writer.WriteField("response_format", "verbose_json")
	writer.WriteField("language", st.GetLanguage())
	writer.Close()

	req, err := http.NewRequestWithContext(st.ctx, "POST", GROQ_STT_URL, &body)
	if err != nil {
		st.logger.Errorf("groq-stt: error creating request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+st.GetKey())
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		st.logger.Errorf("groq-stt: error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		st.logger.Errorf("groq-stt: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		return
	}

	var result groq_internal.GroqTranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		st.logger.Errorf("groq-stt: error decoding response: %v", err)
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

func createWAVHeader(dataSize, sampleRate, numChannels, bitsPerSample int) []byte {
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	totalSize := 36 + dataSize

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	header[4] = byte(totalSize)
	header[5] = byte(totalSize >> 8)
	header[6] = byte(totalSize >> 16)
	header[7] = byte(totalSize >> 24)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	header[16] = 16 // chunk size
	header[20] = 1  // PCM format
	header[22] = byte(numChannels)
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)
	header[32] = byte(blockAlign)
	header[34] = byte(bitsPerSample)
	copy(header[36:40], "data")
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	return header
}

func (st *groqSTT) Close(ctx context.Context) error {
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
