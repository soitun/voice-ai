// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_google

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	speech "cloud.google.com/go/speech/apiv2"
	"cloud.google.com/go/speech/apiv2/speechpb"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type googleSpeechToText struct {
	*googleOption
	mu sync.Mutex

	logger commons.Logger

	client        *speech.Client
	stream        speechpb.Speech_StreamingRecognizeClient
	streamFactory func(ctx context.Context) (speechpb.Speech_StreamingRecognizeClient, error)
	onPacket      func(pkt ...internal_type.Packet) error

	// context management
	ctx       context.Context
	ctxCancel context.CancelFunc

	// observability: time when speech started
	startedAt      time.Time
	contextId      string
	sttConnectedAt time.Time
}

// Name implements internal_transformer.SpeechToTextTransformer.
func (g *googleSpeechToText) Name() string {
	return "google-speech-to-text"
}

func NewGoogleSpeechToText(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.SpeechToTextTransformer, error) {
	start := time.Now()
	googleOption, err := NewGoogleOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("google-stt: Error while GoogleOption err: %v", err)
		return nil, err
	}
	client, err := speech.NewClient(ctx, googleOption.GetSpeechToTextClientOptions()...)

	if err != nil {
		logger.Errorf("google-stt: Error creating Google client: %v", err)
		return nil, err
	}

	xctx, contextCancel := context.WithCancel(ctx)
	// Context for callback management
	logger.Benchmark("google.NewGoogleSpeechToText", time.Since(start))
	g := &googleSpeechToText{
		ctx:          xctx,
		ctxCancel:    contextCancel,
		logger:       logger,
		client:       client,
		googleOption: googleOption,
		onPacket:     onPacket,
	}
	g.streamFactory = func(ctx context.Context) (speechpb.Speech_StreamingRecognizeClient, error) {
		return client.StreamingRecognize(ctx)
	}
	return g, nil
}

// Transform implements internal_transformer.SpeechToTextTransformer.
func (google *googleSpeechToText) Transform(c context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		google.mu.Lock()
		google.contextId = pkt.ContextID
		google.mu.Unlock()
		return nil
	case internal_type.InterruptionDetectedPacket:
		google.mu.Lock()
		if pkt.Source == internal_type.InterruptionSourceVad && google.startedAt.IsZero() {
			google.startedAt = time.Now()
		}
		google.mu.Unlock()
		return nil
	case internal_type.UserAudioReceivedPacket:
		google.mu.Lock()
		strm := google.stream
		google.mu.Unlock()

		// If the stream was lost (e.g. Google timed out waiting for audio during
		// slow boot, or reinit failed), re-establish it transparently.
		if strm == nil {
			google.logger.Infof("google-stt: stream not available, re-initializing")
			google.mu.Lock()
			if err := google.initializeStreamLocked(); err != nil {
				google.mu.Unlock()
				return fmt.Errorf("google-stt: re-initialize failed: %w", err)
			}
			strm = google.stream
			google.mu.Unlock()
			if strm == nil {
				return fmt.Errorf("google-stt: stream not initialized after re-initialize")
			}
		}

		return strm.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_Audio{
				Audio: pkt.Audio,
			},
		})
	default:
		return nil
	}
}

// recvLoop reads responses from the gRPC stream for the lifetime of the STT session.
// It exits when the stream ends (EOF, cancellation, or error).
func (g *googleSpeechToText) recvLoop(stream speechpb.Speech_StreamingRecognizeClient) {
	for {
		select {
		case <-g.ctx.Done():
			return
		default:
		}

		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return
			}
			g.logger.Errorf("google-stt: recv error: %v", err)

			// Acquire lock and reinitialize the stream immediately
			g.mu.Lock()
			g.stream = nil
			if reinitErr := g.initializeStreamLocked(); reinitErr != nil {
				g.mu.Unlock()
				g.logger.Errorf("google-stt: re-initialize failed: %v", reinitErr)
				g.onPacket(internal_type.ConversationEventPacket{
					ContextID: g.contextId,
					Name:      "stt",
					Data:      map[string]string{"type": "error", "error": err.Error()},
					Time:      time.Now(),
				})
				return
			}
			g.mu.Unlock()
			g.logger.Infof("google-stt: stream re-initialized after error")
			// New recvLoop was started by initializeStreamLocked, exit this one
			return
		}
		if resp == nil {
			continue
		}

		for _, result := range resp.Results {
			if len(result.Alternatives) == 0 {
				continue
			}
			g.mu.Lock()
			ctxID := g.contextId
			g.mu.Unlock()
			alt := result.Alternatives[0]
			if len(alt.GetTranscript()) == 0 {
				continue
			}
			confStr := fmt.Sprintf("%.4f", float64(alt.GetConfidence()))
			transcript := alt.GetTranscript()

			if result.GetIsFinal() {
				if v, err := g.mdlOpts.GetFloat64("listen.threshold"); err == nil {
					if alt.GetConfidence() < float32(v) {
						g.onPacket(
							internal_type.ConversationEventPacket{
								ContextID: ctxID,
								Name:      "stt",
								Data: map[string]string{
									"type":       "low_confidence",
									"script":     transcript,
									"confidence": confStr,
									"threshold":  fmt.Sprintf("%.4f", v),
								},
								Time: time.Now(),
							},
						)
						continue
					}
				}
				now := time.Now()
				var latencyMs int64
				g.mu.Lock()
				if !g.startedAt.IsZero() {
					latencyMs = now.Sub(g.startedAt).Milliseconds()
					g.startedAt = time.Time{}
				}
				g.mu.Unlock()
				g.onPacket(
					internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
					internal_type.SpeechToTextPacket{
						ContextID:  ctxID,
						Script:     transcript,
						Confidence: float64(alt.GetConfidence()),
						Language:   result.GetLanguageCode(),
						Interim:    false,
					},
					internal_type.ConversationEventPacket{
						ContextID: ctxID,
						Name:      "stt",
						Data: map[string]string{
							"type":       "completed",
							"script":     transcript,
							"confidence": confStr,
							"language":   result.GetLanguageCode(),
							"word_count": fmt.Sprintf("%d", len(strings.Fields(transcript))),
							"char_count": fmt.Sprintf("%d", len(transcript)),
						},
						Time: now,
					},
					internal_type.UserMessageMetricPacket{
						ContextID: ctxID,
						Metrics:   []*protos.Metric{{Name: "stt_latency_ms", Value: fmt.Sprintf("%d", latencyMs)}},
					},
				)
			} else {
				g.onPacket(
					internal_type.InterruptionDetectedPacket{ContextID: ctxID, Source: internal_type.InterruptionSourceWord},
					internal_type.SpeechToTextPacket{
						ContextID:  ctxID,
						Script:     transcript,
						Confidence: float64(result.GetStability()),
						Language:   result.GetLanguageCode(),
						Interim:    true,
					},
					internal_type.ConversationEventPacket{
						ContextID: ctxID,
						Name:      "stt",
						Data: map[string]string{
							"type":       "interim",
							"script":     transcript,
							"confidence": confStr,
						},
						Time: time.Now(),
					},
				)
			}
		}
	}
}

func (google *googleSpeechToText) Initialize() error {
	start := time.Now()
	google.mu.Lock()
	google.sttConnectedAt = time.Now()
	err := google.initializeStreamLocked()
	google.mu.Unlock()
	if err != nil {
		return err
	}
	google.onPacket(internal_type.ConversationEventPacket{
		ContextID: google.contextId,
		Name:      "stt",
		Data: map[string]string{
			"type":     "initialized",
			"provider": google.Name(),
			"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})
	return nil
}

// initializeStreamLocked opens a new StreamingRecognize gRPC stream, sends the
// config, and starts recvLoop. Caller MUST hold google.mu.
func (google *googleSpeechToText) initializeStreamLocked() error {
	stream, err := google.streamFactory(google.ctx)
	if err != nil {
		google.logger.Errorf("google-stt: error creating google-stt stream: %v", err)
		return err
	}

	if google.stream != nil {
		_ = google.stream.CloseSend()
	}
	google.stream = stream

	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		Recognizer: google.GetRecognizer(),
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: google.SpeechToTextOptions(),
		},
	}); err != nil {
		google.logger.Errorf("google-stt: error sending config: %v", err)
		google.stream = nil
		return err
	}

	go google.recvLoop(stream)
	google.logger.Debugf("google-stt: connection established")
	return nil
}

func (g *googleSpeechToText) Close(ctx context.Context) error {
	g.ctxCancel()

	g.mu.Lock()
	ctxID := g.contextId
	connectedAt := g.sttConnectedAt
	g.sttConnectedAt = time.Time{}

	var combinedErr error
	if g.stream != nil {
		if err := g.stream.CloseSend(); err != nil {
			combinedErr = fmt.Errorf("error closing StreamClient: %v", err)
			g.logger.Errorf(combinedErr.Error())
		}
	}

	if g.client != nil {
		if err := g.client.Close(); err != nil {
			// Log the error if closure fails.
			combinedErr = fmt.Errorf("error closing Client: %v", err)
			g.logger.Errorf(combinedErr.Error())
		}
	}
	g.mu.Unlock()

	if !connectedAt.IsZero() {
		g.onPacket(
			internal_type.ConversationEventPacket{
				ContextID: ctxID,
				Name:      "stt",
				Data: map[string]string{
					"type":     "closed",
					"provider": g.Name(),
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
	return combinedErr
}
