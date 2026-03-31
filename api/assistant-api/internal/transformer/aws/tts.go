// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_aws

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

type awsTTS struct {
	*awsOption
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

func NewAWSTextToSpeech(ctx context.Context, logger commons.Logger, vaultCredential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	awsOpts, err := NewAWSOption(logger, vaultCredential, opts)
	if err != nil {
		logger.Errorf("aws-tts: initializing aws failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &awsTTS{
		ctx:       ctx2,
		ctxCancel: contextCancel,
		onPacket:  onPacket,
		logger:    logger,
		awsOption: awsOpts,
	}, nil
}

func (ct *awsTTS) Initialize() error {
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

func (*awsTTS) Name() string {
	return "aws-text-to-speech"
}

func (t *awsTTS) flush() {
	t.mu.Lock()
	text := t.textBuffer.String()
	t.textBuffer.Reset()
	ctxId := t.contextId
	t.mu.Unlock()

	if text == "" || ctxId == "" {
		return
	}

	go t.synthesize(text, ctxId)
}

func (t *awsTTS) synthesize(text string, ctxId string) {
	region := t.GetRegion()
	endpoint := fmt.Sprintf("https://polly.%s.amazonaws.com/v1/speech", region)

	payload := map[string]interface{}{
		"Engine":       t.GetEngine(),
		"LanguageCode": t.GetLanguage(),
		"OutputFormat": "pcm",
		"SampleRate":   "16000",
		"Text":         text,
		"VoiceId":      t.GetVoice(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.logger.Errorf("aws-tts: error marshalling request: %v", err)
		return
	}

	now := time.Now().UTC()
	req, err := http.NewRequestWithContext(t.ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		t.logger.Errorf("aws-tts: error creating request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	t.signPollyRequest(req, body, now, region)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.logger.Errorf("aws-tts: error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.logger.Errorf("aws-tts: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
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
				t.logger.Errorf("aws-tts: error reading response body: %v", err)
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

func (t *awsTTS) signPollyRequest(req *http.Request, payload []byte, now time.Time, region string) {
	service := "polly"
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	payloadHash := ttsSha256Hex(payload)
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), req.URL.Host, amzDate)
	signedHeaders := "content-type;host;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		"POST", req.URL.Path, req.URL.RawQuery, canonicalHeaders, signedHeaders, payloadHash)

	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, ttsSha256Hex([]byte(canonicalRequest)))

	signingKey := ttsGetSignatureKey(t.GetSecretAccessKey(), dateStamp, region, service)
	signature := hex.EncodeToString(ttsHmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		t.GetAccessKeyId(), credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func ttsSha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func ttsHmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func ttsGetSignatureKey(secret, dateStamp, region, service string) []byte {
	kDate := ttsHmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := ttsHmacSHA256(kDate, []byte(region))
	kService := ttsHmacSHA256(kRegion, []byte(service))
	return ttsHmacSHA256(kService, []byte("aws4_request"))
}

func (t *awsTTS) Transform(ctx context.Context, in internal_type.LLMPacket) error {
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
		return fmt.Errorf("aws-tts: unsupported input type %T", in)
	}
	return nil
}

func (t *awsTTS) Close(ctx context.Context) error {
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
