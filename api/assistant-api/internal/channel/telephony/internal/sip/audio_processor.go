// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/protos"
)

const (
	audioChSize    = 100
	chunkDuration  = 20 * time.Millisecond
	mulawFrameSize = 160 // 20ms at 8kHz, 1 byte/sample
)

var (
	rapida16kConfig = internal_audio.NewLinear16khzMonoAudioConfig()
	mulaw8kConfig   = internal_audio.NewMulaw8khzMonoAudioConfig()
)

type bridgeState struct {
	outRTP    *sip_infra.RTPHandler
	transcode func([]byte) []byte
}

// AudioProcessor handles all RTP audio for a SIP streamer:
//   - Input: AudioIn → codec normalize → resample 8k→16k → onInputAudio callback
//   - Output: 16k LINEAR16 → resample 8k → codec convert → 20ms pacing → AudioOut
//   - Bridge: forward caller↔target + queue for recording via Talk pipeline
//   - Ringback: generate tone → AudioOut
type AudioProcessor struct {
	resampler  internal_type.AudioResampler
	rtpHandler *sip_infra.RTPHandler
	pushInput  func(internal_type.Stream)

	// Output buffering (AI TTS → RTP)
	outputBuffer   bytes.Buffer
	outputBufferMu sync.Mutex
	flushCh        chan struct{}

	// Bridge state
	bridge           atomic.Pointer[bridgeState]
	bridgeUserCh     chan []byte
	bridgeOperatorCh chan []byte
}

type AudioProcessorConfig struct {
	RTPHandler *sip_infra.RTPHandler
	Resampler  internal_type.AudioResampler
	PushInput  func(internal_type.Stream)
}

func NewAudioProcessor(cfg AudioProcessorConfig) *AudioProcessor {
	return &AudioProcessor{
		resampler:        cfg.Resampler,
		rtpHandler:       cfg.RTPHandler,
		pushInput:        cfg.PushInput,
		flushCh:          make(chan struct{}, 1),
		bridgeUserCh:     make(chan []byte, audioChSize),
		bridgeOperatorCh: make(chan []byte, audioChSize),
	}
}

// ProcessInputAudio normalizes codec and resamples RTP audio to 16kHz LINEAR16.
// Returns the resampled audio, or nil if conversion fails.
func (p *AudioProcessor) ProcessInputAudio(audioData []byte) []byte {
	codec := p.rtpHandler.GetCodec()
	if codec != nil && codec.Name == "PCMA" {
		audioData = internal_audio.AlawToUlaw(audioData)
	}
	resampled, err := p.resampler.Resample(audioData, mulaw8kConfig, rapida16kConfig)
	if err != nil {
		return nil
	}
	return resampled
}

// ProcessOutputAudio resamples 16kHz LINEAR16 to 8kHz µ-law and buffers for pacing.
// Discards audio when a bridge is active — the caller hears the operator, not the AI.
func (p *AudioProcessor) ProcessOutputAudio(audioData []byte) error {
	if p.bridge.Load() != nil {
		return nil
	}
	outData, err := p.resampler.Resample(audioData, rapida16kConfig, mulaw8kConfig)
	if err != nil {
		return err
	}
	codec := p.rtpHandler.GetCodec()
	if codec != nil && codec.Name == "PCMA" {
		outData = internal_audio.UlawToAlaw(outData)
	}
	p.outputBufferMu.Lock()
	p.outputBuffer.Write(outData)
	p.outputBufferMu.Unlock()
	return nil
}

func (p *AudioProcessor) getNextChunk() []byte {
	p.outputBufferMu.Lock()
	defer p.outputBufferMu.Unlock()
	if p.outputBuffer.Len() < mulawFrameSize {
		return nil
	}
	chunk := make([]byte, mulawFrameSize)
	p.outputBuffer.Read(chunk)
	return chunk
}

func (p *AudioProcessor) ClearOutputBuffer() {
	p.outputBufferMu.Lock()
	p.outputBuffer.Reset()
	p.outputBufferMu.Unlock()
	select {
	case p.flushCh <- struct{}{}:
	default:
	}
}

// RunOutputSender paces 20ms audio frames to the RTP handler. Blocks until ctx is cancelled.
func (p *AudioProcessor) RunOutputSender(ctx context.Context) {
	ticker := time.NewTicker(chunkDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.flushCh:
			p.rtpHandler.FlushAudioOut()
		case <-ticker.C:
			chunk := p.getNextChunk()
			if chunk == nil {
				continue
			}
			select {
			case p.rtpHandler.AudioOut() <- chunk:
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}

// =============================================================================
// Bridge
// =============================================================================

func (p *AudioProcessor) IsBridgeActive() bool {
	return p.bridge.Load() != nil
}

func (p *AudioProcessor) SetBridgeTarget(rtp *sip_infra.RTPHandler, inCodec, outCodec *sip_infra.Codec) {
	if rtp == nil {
		return
	}
	bs := &bridgeState{outRTP: rtp}
	if inCodec != nil && outCodec != nil && inCodec.Name != outCodec.Name {
		if inCodec.Name == "PCMA" && outCodec.Name == "PCMU" {
			bs.transcode = internal_audio.AlawToUlaw
		} else if inCodec.Name == "PCMU" && outCodec.Name == "PCMA" {
			bs.transcode = internal_audio.UlawToAlaw
		}
	}
	p.bridge.Store(bs)
}

func (p *AudioProcessor) ClearBridgeTarget() {
	p.bridge.Store(nil)
}

// ForwardUserAudio routes caller audio to the bridge target and queues it for recording.
// Returns true if bridge is active and audio was handled.
func (p *AudioProcessor) ForwardUserAudio(audioData []byte) bool {
	bs := p.bridge.Load()
	if bs == nil {
		return false
	}
	rawAudio := audioData
	if bs.transcode != nil {
		audioData = bs.transcode(audioData)
	}
	select {
	case bs.outRTP.AudioOut() <- audioData:
	default:
	}
	select {
	case p.bridgeUserCh <- rawAudio:
	default:
	}
	return true
}

// PushOperatorAudio queues transfer target audio for recording.
func (p *AudioProcessor) PushOperatorAudio(audio []byte) {
	select {
	case p.bridgeOperatorCh <- audio:
	default:
	}
}

// RunBridgeRecorder resamples queued bridge audio and pushes it into the Talk pipeline.
// Blocks until ctx is cancelled.
func (p *AudioProcessor) RunBridgeRecorder(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case audio := <-p.bridgeUserCh:
			if resampled, err := p.resampler.Resample(audio, mulaw8kConfig, rapida16kConfig); err == nil {
				p.pushInput(&protos.ConversationBridgeUserAudio{Audio: resampled})
			}
		case audio := <-p.bridgeOperatorCh:
			if resampled, err := p.resampler.Resample(audio, mulaw8kConfig, rapida16kConfig); err == nil {
				p.pushInput(&protos.ConversationBridgeOperatorAudio{Audio: resampled})
			}
		}
	}
}

// =============================================================================
// Ringback
// =============================================================================

// PlayRingback generates a standard ringback tone and writes directly to AudioOut.
// Blocks until ctx is cancelled.
func (p *AudioProcessor) PlayRingback(ctx context.Context) {
	codec := p.rtpHandler.GetCodec()
	ticker := time.NewTicker(chunkDuration)
	defer ticker.Stop()

	offset := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var frame []byte
			frame, offset = internal_audio.GenerateRingbackMulawFrame(offset)
			if codec != nil && codec.Name == "PCMA" {
				frame = internal_audio.UlawToAlaw(frame)
			}
			select {
			case p.rtpHandler.AudioOut() <- frame:
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}
