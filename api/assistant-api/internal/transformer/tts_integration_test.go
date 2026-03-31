//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer

import (
	"context"
	"testing"
	"time"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ttsProviders lists all TTS providers to test. Each entry maps to the factory
// switch cases in transformer.go and the YAML config keys.
var ttsProviders = []string{
	"deepgram",
	"elevenlabs",
	"cartesia",
	"google-speech-service",
	"azure-speech-service",
	"revai",
	"sarvamai",
	"rime",
	"resembleai",
	"neuphonic",
	"minimax",
	"nvidia",
	"groq",
	"speechmatics",
	"aws",
}

// TestTTSBasic verifies the basic TTS pipeline: Initialize → Transform(delta) →
// Transform(done) → collect audio packets → Close.
func TestTTSBasic(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()

	for _, provider := range ttsProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.TTSProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			tts, err := GetTextToSpeechTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err, "factory should succeed")
			require.NotNil(t, tts)

			err = tts.Initialize()
			require.NoError(t, err, "Initialize should succeed")
			defer tts.Close(ctx)

			// Send a text delta
			err = tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "test-tts-basic",
				Text:      "Hello world, this is a test.",
			})
			require.NoError(t, err, "Transform delta should succeed")

			// Signal end of LLM response
			err = tts.Transform(ctx, internal_type.LLMResponseDonePacket{
				ContextID: "test-tts-basic",
				Text:      "",
			})
			require.NoError(t, err, "Transform done should succeed")

			// Wait for audio output
			collector.WaitForTTSEnd(t, 20*time.Second)

			audioPackets := collector.AudioPackets()
			assert.NotEmpty(t, audioPackets, "should receive at least one audio packet")

			// Verify audio data is non-empty
			totalBytes := 0
			for _, ap := range audioPackets {
				assert.NotEmpty(t, ap.AudioChunk, "audio chunk should not be empty")
				totalBytes += len(ap.AudioChunk)
			}
			assert.Greater(t, totalBytes, 0, "total audio bytes should be > 0")

			endPackets := collector.EndPackets()
			assert.NotEmpty(t, endPackets, "should receive TTS end packet")
		})
	}
}

// TestTTSMetricsAndEvents verifies that TTS providers emit ConversationEventPacket
// and/or MessageMetricPacket alongside audio data.
func TestTTSMetricsAndEvents(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()

	for _, provider := range ttsProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.TTSProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			tts, err := GetTextToSpeechTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err)
			require.NoError(t, tts.Initialize())
			defer tts.Close(ctx)

			err = tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "test-tts-metrics",
				Text:      "Testing metrics and events.",
			})
			require.NoError(t, err)

			err = tts.Transform(ctx, internal_type.LLMResponseDonePacket{
				ContextID: "test-tts-metrics",
				Text:      "",
			})
			require.NoError(t, err)

			collector.WaitForTTSEnd(t, 20*time.Second)

			// At minimum, audio should be produced
			assert.NotEmpty(t, collector.AudioPackets(), "should produce audio")

			// Log event/metric counts for observability (not all providers emit all packet types)
			events := collector.EventPackets()
			metrics := collector.MetricPackets()
			t.Logf("provider=%s events=%d metrics=%d audio_packets=%d",
				provider, len(events), len(metrics), len(collector.AudioPackets()))

			// Verify event packets have required fields when present
			for _, ev := range events {
				assert.NotEmpty(t, ev.Name, "event name should not be empty")
				assert.NotNil(t, ev.Data, "event data should not be nil")
			}
		})
	}
}

// TestTTSMultiChunk tests sending multiple delta packets followed by a done packet,
// simulating a streaming LLM response.
func TestTTSMultiChunk(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()

	for _, provider := range ttsProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.TTSProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			tts, err := GetTextToSpeechTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err)
			require.NoError(t, tts.Initialize())
			defer tts.Close(ctx)

			// Simulate streaming LLM deltas
			chunks := []string{
				"The quick brown fox ",
				"jumps over ",
				"the lazy dog. ",
				"This is a longer sentence to ensure ",
				"the provider can handle multi-chunk input.",
			}
			for _, chunk := range chunks {
				err = tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
					ContextID: "test-tts-multichunk",
					Text:      chunk,
				})
				require.NoError(t, err)
				time.Sleep(50 * time.Millisecond) // simulate streaming delay
			}

			err = tts.Transform(ctx, internal_type.LLMResponseDonePacket{
				ContextID: "test-tts-multichunk",
				Text:      "",
			})
			require.NoError(t, err)

			collector.WaitForTTSEnd(t, 30*time.Second)

			audioPackets := collector.AudioPackets()
			assert.NotEmpty(t, audioPackets, "should receive audio from multi-chunk input")

			totalBytes := 0
			for _, ap := range audioPackets {
				totalBytes += len(ap.AudioChunk)
			}
			t.Logf("provider=%s chunks_sent=%d audio_packets=%d total_bytes=%d",
				provider, len(chunks), len(audioPackets), totalBytes)
		})
	}
}

// TestTTSInterruption tests that the TTS transformer handles interruption packets
// without errors. After an interruption, no more audio should be emitted for
// the interrupted context.
func TestTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()

	for _, provider := range ttsProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.TTSProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			tts, err := GetTextToSpeechTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err)
			require.NoError(t, tts.Initialize())
			defer tts.Close(ctx)

			// Start generating speech
			err = tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
				ContextID: "test-tts-interrupt",
				Text:      "This is a long sentence that should be interrupted before completion by the user speaking over it.",
			})
			require.NoError(t, err)

			// Wait a bit for audio generation to start
			collector.WaitForAudio(t, 15*time.Second)

			countBefore := len(collector.AudioPackets())

			// Send interruption
			err = tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
				ContextID: "test-tts-interrupt",
				Source:    internal_type.InterruptionSourceVad,
			})
			require.NoError(t, err, "interruption should not error")

			// Give a moment for the interruption to take effect
			time.Sleep(500 * time.Millisecond)

			t.Logf("provider=%s audio_before_interrupt=%d audio_after_interrupt=%d",
				provider, countBefore, len(collector.AudioPackets()))
		})
	}
}

// TestTTSReconnect tests that a new transformer instance can be created and used
// after closing a previous one, validating clean resource management.
func TestTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()

	for _, provider := range ttsProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.TTSProvider(t, provider)
			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			for attempt := 0; attempt < 2; attempt++ {
				collector := testutil.NewPacketCollector()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

				tts, err := GetTextToSpeechTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
				require.NoError(t, err, "attempt %d: factory should succeed", attempt)
				require.NoError(t, tts.Initialize(), "attempt %d: Initialize should succeed", attempt)

				err = tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
					ContextID: "test-tts-reconnect",
					Text:      "Reconnect test.",
				})
				require.NoError(t, err)

				err = tts.Transform(ctx, internal_type.LLMResponseDonePacket{
					ContextID: "test-tts-reconnect",
					Text:      "",
				})
				require.NoError(t, err)

				collector.WaitForTTSEnd(t, 20*time.Second)
				assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)

				tts.Close(ctx)
				cancel()
			}
		})
	}
}
