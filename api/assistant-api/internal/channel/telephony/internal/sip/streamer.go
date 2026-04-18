// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	rapida_utils "github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/anypb"
)

type Streamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	mu     sync.RWMutex
	closed atomic.Bool

	session    *sip_infra.Session
	rtpHandler *sip_infra.RTPHandler
	audio      *AudioProcessor

	transferring        atomic.Bool
	ringbackCancel      context.CancelFunc
	onTransferInitiated func(target string)

	ctx    context.Context
	cancel context.CancelFunc
}

func NewStreamer(ctx context.Context,
	logger commons.Logger,
	sipSession *sip_infra.Session,
	cc *callcontext.CallContext,
	vaultCred *protos.VaultCredential,
) (internal_type.Streamer, error) {
	if sipSession == nil {
		return nil, fmt.Errorf("SIP session is required — standalone server mode is not supported")
	}
	streamerCtx, cancel := context.WithCancel(ctx)

	s := &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(
			logger, cc, vaultCred,
			internal_telephony_base.WithSourceAudioConfig(internal_audio.NewMulaw8khzMonoAudioConfig()),
		),
		ctx:    streamerCtx,
		cancel: cancel,
	}

	go func() {
		<-streamerCtx.Done()
		s.BaseStreamer.Cancel()
	}()

	rtpHandler := sipSession.GetRTPHandler()
	if rtpHandler == nil {
		cancel()
		return nil, sip_infra.NewSIPError("NewStreamer", sipSession.GetCallID(), "session has no RTP handler", sip_infra.ErrRTPNotInitialized)
	}

	s.session = sipSession
	s.rtpHandler = rtpHandler
	s.audio = NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtpHandler,
		Resampler:  s.Resampler(),
		PushInput:  s.PushInput,
	})

	go s.forwardIncomingAudio()
	go s.audio.RunOutputSender(streamerCtx)
	go s.audio.RunBridgeRecorder(streamerCtx)
	s.PushInput(s.CreateConnectionRequest())

	localIP, localPort := rtpHandler.LocalAddr()
	codecName := "PCMU"
	if negotiated := sipSession.GetNegotiatedCodec(); negotiated != nil {
		codecName = negotiated.Name
	}
	logger.Infow("SIP streamer created",
		"call_id", sipSession.GetCallID(),
		"codec", codecName,
		"rtp_port", localPort,
		"local_ip", localIP)

	return s, nil
}

func (s *Streamer) forwardIncomingAudio() {
	s.mu.RLock()
	rtpHandler := s.rtpHandler
	s.mu.RUnlock()
	if rtpHandler == nil {
		return
	}
	bufferThreshold := s.InputBufferThreshold()
	for {
		select {
		case <-s.ctx.Done():
			return
		case audioData, ok := <-rtpHandler.AudioIn():
			if !ok {
				return
			}
			if s.audio.ForwardUserAudio(audioData) {
				continue
			}
			resampled := s.audio.ProcessInputAudio(audioData)
			if resampled == nil {
				continue
			}
			var audioReq *protos.ConversationUserMessage
			s.WithInputBuffer(func(buf *bytes.Buffer) {
				buf.Write(resampled)
				if buf.Len() >= bufferThreshold {
					data := make([]byte, buf.Len())
					copy(data, buf.Bytes())
					buf.Reset()
					audioReq = &protos.ConversationUserMessage{
						Message: &protos.ConversationUserMessage_Audio{Audio: data},
					}
				}
			})
			if audioReq != nil {
				s.PushInput(audioReq)
			}
		}
	}
}

func (s *Streamer) Context() context.Context {
	return s.ctx
}

func (s *Streamer) Send(response internal_type.Stream) error {
	if s.closed.Load() {
		return sip_infra.ErrSessionClosed
	}
	switch data := response.(type) {
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			return s.audio.ProcessOutputAudio(content.Audio)
		}
	case *protos.ConversationInterruption:
		if data.Type == protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
			s.audio.ClearOutputBuffer()
		}
	case *protos.ConversationDirective:
		switch data.GetType() {
		case protos.ConversationDirective_END_CONVERSATION:
			return s.Close()
		case protos.ConversationDirective_TRANSFER_CONVERSATION:
			to := s.extractTransferTarget(data.GetArgs())
			if to == "" {
				s.Logger.Warnw("Transfer directive missing 'to' target")
				return nil
			}
			s.mu.RLock()
			if s.session != nil {
				s.session.SetMetadata(sip_infra.MetadataBridgeTransferTarget, to)
			}
			s.mu.RUnlock()
			s.EnterTransferMode(to)
			return nil
		}
	}
	return nil
}

// =============================================================================
// Transfer
// =============================================================================

func (s *Streamer) EnterTransferMode(target string) {
	if !s.transferring.CompareAndSwap(false, true) {
		return
	}
	s.audio.ClearOutputBuffer()

	s.mu.RLock()
	session := s.session
	callback := s.onTransferInitiated
	s.mu.RUnlock()

	if session != nil {
		session.SetState(sip_infra.CallStateTransferring)
	}

	ringbackCtx, ringbackCancel := context.WithCancel(s.ctx)
	s.mu.Lock()
	s.ringbackCancel = ringbackCancel
	s.mu.Unlock()
	go s.audio.PlayRingback(ringbackCtx)

	if callback != nil {
		callback(target)
	}
}

func (s *Streamer) ExitTransferMode() {
	if !s.transferring.Load() {
		return
	}

	s.mu.RLock()
	cancelFn := s.ringbackCancel
	session := s.session
	s.mu.RUnlock()

	if cancelFn != nil {
		cancelFn()
	}
	if session != nil {
		session.SetState(sip_infra.CallStateConnected)
	}

	s.audio.ClearBridgeTarget()
	s.transferring.Store(false)
	s.Logger.Infow("Transfer mode: exited, AI resuming")
}

func (s *Streamer) StopRingback() {
	s.mu.RLock()
	cancelFn := s.ringbackCancel
	s.mu.RUnlock()
	if cancelFn != nil {
		cancelFn()
	}
	s.audio.ClearOutputBuffer()
}

func (s *Streamer) SetBridgeOutRTP(rtp *sip_infra.RTPHandler) {
	s.mu.RLock()
	inCodec := s.rtpHandler.GetCodec()
	s.mu.RUnlock()
	var outCodec *sip_infra.Codec
	if rtp != nil {
		outCodec = rtp.GetCodec()
	}
	s.audio.SetBridgeTarget(rtp, inCodec, outCodec)
}

func (s *Streamer) ClearBridgeTarget() {
	s.audio.ClearBridgeTarget()
}

func (s *Streamer) PushBridgeOperatorAudio(audio []byte) {
	s.audio.PushOperatorAudio(audio)
}

func (s *Streamer) SetOnTransferInitiated(fn func(target string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onTransferInitiated = fn
}

// =============================================================================
// Lifecycle
// =============================================================================

func (s *Streamer) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}

	s.cancel()
	s.BaseStreamer.Cancel()
	s.ResetInputBuffer()

	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()

	if s.transferring.Load() {
		return nil
	}

	if session != nil {
		session.End()
	}

	s.Logger.Infow("SIP streamer closed")
	return nil
}

func (s *Streamer) extractTransferTarget(args map[string]*anypb.Any) string {
	if args == nil {
		return ""
	}
	iface, err := rapida_utils.AnyMapToInterfaceMap(args)
	if err != nil {
		return ""
	}
	if to, ok := iface["to"].(string); ok {
		return to
	}
	return ""
}
