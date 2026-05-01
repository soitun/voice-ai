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
	"strings"
	"sync"
	"sync/atomic"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
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
	onTransferInitiated func(targets []string, message string, postTransferAction string)

	cancelParent context.CancelFunc
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
	_, cancel := context.WithCancel(ctx)

	s := &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(
			logger, cc, vaultCred,
			internal_telephony_base.WithSourceAudioConfig(internal_audio.NewMulaw8khzMonoAudioConfig()),
			internal_telephony_base.WithBaseOption(channel_base.WithInputAudioConfig(internal_audio.NewLinear16khzMonoAudioConfig())),
		),
		cancelParent: cancel,
	}

	go func() {
		select {
		case <-sipSession.ByeReceived():
			s.Logger.Infow("SIP streamer: user BYE received")
		case <-sipSession.Context().Done():
			s.Logger.Infow("SIP streamer: session context cancelled")
		case <-s.Ctx.Done():
			return
		}
		if msg := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
			s.Input(msg)
		}
		s.Close()
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
		PushInput:  s.Input,
	})

	go s.forwardIncomingAudio()
	go s.audio.RunOutputSender(s.Ctx)
	go s.audio.RunBridgeRecorder(s.Ctx)
	s.Input(s.CreateConnectionRequest())

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
		case <-s.Ctx.Done():
			return
		case audioData, ok := <-rtpHandler.AudioIn():
			if !ok {
				return
			}
			if s.audio.ForwardUserAudio(audioData) {
				continue
			}
			// During transfer (ringback playing, bridge not yet set, or teardown race),
			// discard audio instead of sending to the AI pipeline.
			if s.transferring.Load() {
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
				s.Input(audioReq)
			}
		}
	}
}

func (s *Streamer) Context() context.Context {
	return s.Ctx
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
	case *protos.ConversationDisconnection:
		s.Logger.Infow("SIP streamer: Send(ConversationDisconnection)", "type", data.GetType().String())
		if disc := s.Disconnect(data.GetType()); disc != nil {
			s.Input(disc)
		}
		s.endSession()
		s.Close()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			s.PushToolCallResult(data.GetId(), data.GetToolId(), data.GetName(), data.GetAction(), map[string]string{
				"status": "completed",
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			raw := data.GetArgs()["transfer_to"]
			if raw == "" {
				s.Logger.Warnw("Transfer tool call missing 'transfer_to' target")
				s.PushToolCallResult(data.GetId(), data.GetToolId(), data.GetName(), data.GetAction(), map[string]string{
					"status": "failed", "reason": "missing transfer target",
				})
				return nil
			}
			targets := s.SplitTransferTargets(raw)
			message := data.GetArgs()["message"]
			postTransferAction := data.GetArgs()["post_transfer_action"]
			s.mu.RLock()
			if s.session != nil {
				s.session.SetMetadata(sip_infra.MetadataBridgeTransferTarget, strings.Join(targets, commons.SEPARATOR))
				s.session.SetMetadata("tool_id", data.GetToolId())
				s.session.SetMetadata("tool_context_id", data.GetId())
			}
			s.mu.RUnlock()
			s.EnterTransferMode(targets, message, postTransferAction)
			return nil
		}
	}
	return nil
}

// =============================================================================
// Transfer
// =============================================================================

func (s *Streamer) EnterTransferMode(targets []string, message string, postTransferAction string) {
	if !s.transferring.CompareAndSwap(false, true) {
		return
	}

	s.mu.RLock()
	session := s.session
	callback := s.onTransferInitiated
	s.mu.RUnlock()

	if session != nil {
		session.SetState(sip_infra.CallStateTransferring)
	}

	ringbackCtx, ringbackCancel := context.WithCancel(s.Ctx)
	s.mu.Lock()
	s.ringbackCancel = ringbackCancel
	s.mu.Unlock()
	go func() {
		s.audio.WaitOutputDrain(ringbackCtx)
		s.audio.PlayRingback(ringbackCtx)
	}()

	if callback != nil {
		callback(targets, message, postTransferAction)
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

func (s *Streamer) PushToolCallResult(contextID, toolID, toolName string, action protos.ToolCallAction, result map[string]string) {
	s.Input(&protos.ConversationToolCallResult{
		Id:     contextID,
		ToolId: toolID,
		Name:   toolName,
		Action: action,
		Result: result,
	})
}

func (s *Streamer) SetOnTransferInitiated(fn func(targets []string, message string, postTransferAction string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onTransferInitiated = fn
}

func (s *Streamer) endSession() {
	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()
	if session != nil {
		session.End()
	}
}

// =============================================================================
// Lifecycle
// =============================================================================

func (s *Streamer) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}

	s.cancelParent()
	s.BaseStreamer.Cancel()
	s.ResetInputBuffer()

	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()

	if session != nil {
		session.End()
	}

	s.Logger.Infow("SIP streamer closed")
	return nil
}
