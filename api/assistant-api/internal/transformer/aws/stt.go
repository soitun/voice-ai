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
	"sync"
	"sync/atomic"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type awsSTT struct {
	*awsOption
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

func NewAWSSpeechToText(ctx context.Context, logger commons.Logger, vaultCredential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.SpeechToTextTransformer, error) {
	awsOpts, err := NewAWSOption(logger, vaultCredential, opts)
	if err != nil {
		logger.Errorf("aws-stt: initializing aws failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &awsSTT{
		ctx:       ctx2,
		ctxCancel: contextCancel,
		onPacket:  onPacket,
		logger:    logger,
		awsOption: awsOpts,
	}, nil
}

func (*awsSTT) Name() string {
	return "aws-speech-to-text"
}

func (st *awsSTT) Initialize() error {
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

func (st *awsSTT) Transform(ctx context.Context, in internal_type.Packet) error {
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

func (st *awsSTT) transcribe(audioData []byte, ctxId string) {
	region := st.GetRegion()
	endpoint := fmt.Sprintf("https://transcribe.%s.amazonaws.com", region)

	payload := map[string]interface{}{
		"AudioStream": map[string]interface{}{
			"AudioEvent": map[string]interface{}{
				"AudioChunk": audioData,
			},
		},
		"LanguageCode":         st.GetLanguage(),
		"MediaEncoding":        "pcm",
		"MediaSampleRateHertz": 16000,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		st.logger.Errorf("aws-stt: error marshalling request: %v", err)
		return
	}

	now := time.Now().UTC()
	req, err := http.NewRequestWithContext(st.ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		st.logger.Errorf("aws-stt: error creating request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "com.amazonaws.transcribe.Transcribe.StartTranscriptionJob")

	st.signRequest(req, body, now, region, "transcribe")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		st.logger.Errorf("aws-stt: error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		st.logger.Errorf("aws-stt: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
		return
	}

	var result struct {
		Results struct {
			Transcripts []struct {
				Transcript string `json:"Transcript"`
			} `json:"Transcripts"`
		} `json:"Results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		st.logger.Errorf("aws-stt: error decoding response: %v", err)
		return
	}

	transcript := ""
	if len(result.Results.Transcripts) > 0 {
		transcript = result.Results.Transcripts[0].Transcript
	}

	if transcript != "" {
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
				Script:    transcript,
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

func (st *awsSTT) signRequest(req *http.Request, payload []byte, now time.Time, region, service string) {
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	payloadHash := sha256Hex(payload)
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), req.URL.Host, amzDate)
	signedHeaders := "content-type;host;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		"POST", req.URL.Path, req.URL.RawQuery, canonicalHeaders, signedHeaders, payloadHash)

	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, sha256Hex([]byte(canonicalRequest)))

	signingKey := getSignatureKey(st.GetSecretAccessKey(), dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		st.GetAccessKeyId(), credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func (st *awsSTT) Close(ctx context.Context) error {
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
