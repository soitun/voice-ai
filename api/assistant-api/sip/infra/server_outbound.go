// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

// MakeCallOptions holds the typed context for an outbound call.
type MakeCallOptions struct {
	Auth            types.SimplePrinciple
	Assistant       *internal_assistant_entity.Assistant
	ConversationID  uint64
	ContextID       string
	VaultCredential *protos.VaultCredential
}

// MakeCall initiates an outbound SIP call using the DialogClientCache.
// The cache stores the dialog so incoming BYE/re-INVITE are properly routed
// to the correct DialogClientSession via handleBye → dialogClientCache.ReadBye.
func (s *Server) MakeCall(ctx context.Context, cfg *Config, toURI, fromURI string, opts MakeCallOptions) (*Session, error) {
	if s.state.Load() != int32(ServerStateRunning) {
		return nil, fmt.Errorf("SIP server is not running")
	}

	invite, err := s.prepareOutboundInvite(ctx, cfg, toURI, fromURI)
	if err != nil {
		return nil, err
	}

	session, err := NewSession(ctx, &SessionConfig{
		Config:          cfg,
		Direction:       CallDirectionOutbound,
		CallID:          invite.callID,
		Codec:           &CodecPCMU,
		Logger:          s.logger,
		Auth:            opts.Auth,
		Assistant:       opts.Assistant,
		ConversationID:  opts.ConversationID,
		ContextID:       opts.ContextID,
		VaultCredential: opts.VaultCredential,
	})
	if err != nil {
		invite.cleanup()
		return nil, fmt.Errorf("failed to create outbound session: %w", err)
	}

	session.SetLocalRTP(invite.externalIP, invite.localPort)
	session.SetRTPHandler(invite.rtpHandler)
	session.SetDialogClientSession(invite.dialogSession)

	s.registerSession(session, invite.callID)

	// Handle the call lifecycle in background
	go s.handleOutboundDialog(session, invite.rtpHandler, invite.dialogSession)

	return session, nil
}

// handleOutboundDialog processes the outbound dialog lifecycle
func (s *Server) handleOutboundDialog(session *Session, rtpHandler *RTPHandler, dialogSession *sipgo.DialogClientSession) {
	callID := session.GetCallID()

	// Ensure dialog resources are cleaned up when the goroutine exits.
	// sipgo's Close() does NOT send BYE — it only releases internal dialog state.
	defer dialogSession.Close()
	digestURI := dialogSession.InviteRequest.Recipient.Addr()
	s.logger.Debugw("Outbound call waiting for answer with digest auth",
		"call_id", callID,
		"auth_username", session.config.Username,
		"auth_realm", session.config.Realm,
		"digest_uri", digestURI,
		"request_uri", dialogSession.InviteRequest.Recipient.String())
	err := dialogSession.WaitAnswer(session.ctx, sipgo.AnswerOptions{
		Username: session.config.Username,
		Password: session.config.Password,
		OnResponse: func(res *sip.Response) error {
			statusCode := res.StatusCode
			s.logger.Debugw("Outbound call response",
				"call_id", callID,
				"status", statusCode)

			if statusCode == 180 || statusCode == 183 {
				s.setCallState(session, CallStateRinging, "outbound_progress_ringing")
			}

			// Log digest auth challenge details for debugging credential issues
			if statusCode == 401 {
				if wwwAuth := res.GetHeader("WWW-Authenticate"); wwwAuth != nil {
					s.logger.Debugw("SIP 401 challenge received",
						"call_id", callID,
						"www_authenticate", wwwAuth.Value(),
						"auth_username", session.config.Username)
				}
				if authHdr := dialogSession.InviteRequest.GetHeader("Authorization"); authHdr != nil {
					s.logger.Debugw("SIP digest Authorization sent",
						"call_id", callID,
						"has_authorization", true)
				}
			}
			if statusCode == 407 {
				if proxyAuth := res.GetHeader("Proxy-Authenticate"); proxyAuth != nil {
					s.logger.Debugw("SIP 407 challenge received",
						"call_id", callID,
						"proxy_authenticate", proxyAuth.Value(),
						"auth_username", session.config.Username)
				}
				if authHdr := dialogSession.InviteRequest.GetHeader("Proxy-Authorization"); authHdr != nil {
					s.logger.Debugw("SIP digest Proxy-Authorization sent",
						"call_id", callID,
						"has_proxy_authorization", true)
				}
			}

			return nil
		},
	})
	if err != nil {
		// Extract SIP status code from ErrDialogResponse if available
		var dialogErr *sipgo.ErrDialogResponse
		if errors.As(err, &dialogErr) {
			// If 401/407 after auth attempt, it means credentials are wrong
			if dialogErr.Res.StatusCode == 401 || dialogErr.Res.StatusCode == 407 {
				s.logger.Errorw("Outbound call authentication failed — check SIP credentials in vault",
					"call_id", callID,
					"status", dialogErr.Res.StatusCode,
					"reason", dialogErr.Res.Reason,
					"auth_username", session.config.Username,
					"auth_password_set", len(session.config.Password) > 0,
					"digest_uri", dialogSession.InviteRequest.Recipient.Addr(),
					"hint", "Verify sip_username and sip_password in vault match the SIP provider's auth credentials")
			} else {
				s.logger.Warnw("Outbound call rejected by remote",
					"call_id", callID,
					"status", dialogErr.Res.StatusCode,
					"reason", dialogErr.Res.Reason)
			}
		} else {
			s.logger.Warnw("Outbound call failed",
				"call_id", callID,
				"error", err)
		}
		s.setCallState(session, CallStateFailed, "outbound_wait_answer_failed")
		s.notifyError(session, err)
		// Call was never answered — no established dialog, so don't send BYE
		session.ClearOnDisconnect()
		s.beginEnding(session, "outbound_setup_failure")
		session.End()
		// Allow the transaction layer time to send ACK for non-2xx responses
		// before terminating the dialog (prevents retransmission floods)
		time.AfterFunc(2*time.Second, func() {
			dialogSession.Close()
		})
		return
	}

	// Call answered — 200 OK received.
	answerTime := time.Now()
	s.logger.Infow("Outbound call 200 OK received — setting up RTP before ACK",
		"call_id", callID)

	// Step 1: Parse remote SDP from 200 OK (available before ACK)
	// This is where we discover what codec the remote side actually accepted.
	// The initial INVITE offered all SupportedCodecs, but the 200 OK's SDP
	// tells us which one was chosen. We MUST update the RTP handler's codec
	// so outgoing packets use the correct payload type, and update the session
	// so subsequent re-INVITE responses advertise only the negotiated codec.
	var remoteRTPIP string
	var remoteRTPPort int
	if dialogSession.InviteResponse != nil {
		if body := dialogSession.InviteResponse.Body(); len(body) > 0 {
			s.logger.Debugw("Outbound call 200 OK SDP answer (raw)",
				"call_id", callID,
				"sdp_body", string(body))
			sdpInfo, parseErr := s.ParseSDP(body)
			if parseErr == nil && sdpInfo.ConnectionIP != "" && sdpInfo.AudioPort > 0 {
				remoteRTPIP = sdpInfo.ConnectionIP
				remoteRTPPort = sdpInfo.AudioPort
				rtpHandler.SetRemoteAddr(remoteRTPIP, remoteRTPPort)
				session.SetRemoteRTP(remoteRTPIP, remoteRTPPort)

				// Negotiate codec from the answer SDP — the remote side may have
				// chosen PCMA (PT 8) even though we offered PCMU first. If we
				// keep sending PT 0 (PCMU) while the remote expects PT 8 (PCMA),
				// the audio is garbled or the PBX drops the call immediately.
				if sdpInfo.PreferredCodec != nil {
					rtpHandler.SetCodec(sdpInfo.PreferredCodec)
					session.SetNegotiatedCodec(sdpInfo.PreferredCodec.Name, int(sdpInfo.PreferredCodec.ClockRate))
					s.logger.Infow("Outbound call codec negotiated from 200 OK",
						"call_id", callID,
						"codec", sdpInfo.PreferredCodec.Name,
						"payload_type", sdpInfo.PreferredCodec.PayloadType,
						"clock_rate", sdpInfo.PreferredCodec.ClockRate)
				} else {
					s.logger.Warnw("No matching codec in 200 OK SDP, keeping PCMU default",
						"call_id", callID,
						"remote_payload_types", sdpInfo.PayloadTypes)
				}
			} else if parseErr != nil {
				s.logger.Warnw("Failed to parse remote SDP from 200 OK",
					"call_id", callID,
					"error", parseErr)
			}
		} else {
			s.logger.Warnw("No SDP body in 200 OK response", "call_id", callID)
		}
	} else {
		s.logger.Warnw("No InviteResponse available after WaitAnswer", "call_id", callID)
	}

	// Step 2: Start RTP — sends the first silence packet synchronously, then
	// launches sendLoop. This fires BEFORE ACK so Asterisk sees media immediately.
	rtpHandler.Start()

	localIP, localPort := rtpHandler.LocalAddr()
	remoteAddr := rtpHandler.GetRemoteAddr()
	s.logger.Infow("RTP started (pre-ACK)",
		"call_id", callID,
		"local_rtp", fmt.Sprintf("%s:%d", localIP, localPort),
		"remote_rtp", fmt.Sprintf("%s:%d", remoteRTPIP, remoteRTPPort),
		"remote_addr_set", remoteAddr != nil,
		"elapsed_since_200ok_ms", time.Since(answerTime).Milliseconds())

	// Step 3: NOW send ACK — dialog is confirmed, RTP is already flowing.
	if err := dialogSession.Ack(session.ctx); err != nil {
		s.logger.Errorw("Failed to send ACK", "error", err, "call_id", callID)
		s.setCallState(session, CallStateFailed, "outbound_ack_failed")
		s.beginEnding(session, "outbound_ack_failure")
		session.End()
		dialogSession.Close()
		return
	}
	s.logger.Infow("ACK sent (RTP already flowing)",
		"call_id", callID,
		"elapsed_since_200ok_ms", time.Since(answerTime).Milliseconds())

	s.setCallState(session, CallStateConnected, "outbound_ack_sent")

	// Notify invite handler (which starts the conversation — may do DB lookups).
	// RTP silence is already flowing, so Asterisk won't time out during this.
	s.mu.RLock()
	onInvite := s.onInvite
	s.mu.RUnlock()
	if onInvite != nil {
		info := session.GetInfo()
		s.logger.Infow("Starting onInvite handler for outbound call",
			"call_id", callID)
		if err := onInvite(session, info.LocalURI, info.RemoteURI); err != nil {
			s.logger.Errorw("Outbound INVITE handler failed", "error", err, "call_id", callID)
		} else {
			s.logger.Infow("onInvite handler completed",
				"call_id", callID,
				"total_elapsed_ms", time.Since(answerTime).Milliseconds())
		}
	}

	// Wait for the session to end. The session lifecycle is now owned by startCall
	// (launched by onInvite as a synchronous or goroutine call). startCall blocks on
	// talker.Talk() for the call duration. When BYE arrives:
	//   1. handleBye calls session.NotifyBye() — signals startCall via ByeReceived()
	//   2. handleBye fires onBye → sip.go:handleBye cancels startCall's callCtx
	//   3. talker.Talk returns → startCall finishes → session.End() is called
	//
	// For app-initiated hangup:
	//   EndCall → dialog.Bye + session.End() → session context cancelled
	//
	// We wait on session.Context() because that's cancelled by session.End(), which
	// is the definitive signal that the call is fully torn down. We do NOT call
	// session.End() here when dialog context is cancelled — that was the race condition
	// that killed startCall before it could begin.
	//
	s.logger.Debugw("Outbound dialog waiting for session to end", "call_id", callID)
	select {
	case <-session.Context().Done():
		s.logger.Infow("Outbound dialog ending — session ended",
			"call_id", callID,
			"call_duration_ms", time.Since(answerTime).Milliseconds())
	case <-dialogSession.Context().Done():
		// BYE received — dialog context cancelled. Wait for session to end naturally
		// via startCall's teardown, but apply a safety timeout to prevent leaks.
		s.logger.Infow("Outbound dialog — BYE received, waiting for session teardown",
			"call_id", callID,
			"call_duration_ms", time.Since(answerTime).Milliseconds())
		select {
		case <-session.Context().Done():
			s.logger.Debugw("Outbound dialog — session ended after BYE",
				"call_id", callID)
		case <-time.After(30 * time.Second):
			s.logger.Warnw("Outbound dialog — session did not end within 30s after BYE, forcing teardown",
				"call_id", callID)
			if !session.IsEnded() {
				s.beginEnding(session, "outbound_teardown_timeout")
				session.End()
			}
		}
	}

	// Session.End() owns RTP stop and onEnded cleanup.
}

// outboundInvite holds the result of prepareOutboundInvite — the allocated
// resources and dialog needed to complete or clean up an outbound call.
type outboundInvite struct {
	rtpHandler    *RTPHandler
	rtpPort       int
	localPort     int
	externalIP    string
	callID        string
	dialogSession *sipgo.DialogClientSession

	server *Server // back-reference for cleanup
}

// cleanup releases all resources allocated during prepareOutboundInvite.
// Safe to call on error paths before the session takes ownership.
func (o *outboundInvite) cleanup() {
	if o.rtpHandler != nil {
		o.rtpHandler.Stop()
	}
	if o.server != nil && o.rtpPort > 0 {
		o.server.rtpAllocator.Release(o.rtpPort)
	}
	if o.dialogSession != nil {
		time.AfterFunc(2*time.Second, func() { o.dialogSession.Close() })
	}
}

// prepareOutboundInvite allocates RTP, builds SDP + INVITE headers, and sends
// the INVITE via the dialog client cache. Returns the allocated resources in an
// outboundInvite struct. The caller must call invite.cleanup() on error, or
// transfer ownership of the resources to a Session on success.
//
// Shared between MakeCall and MakeBridgeCall.
func (s *Server) prepareOutboundInvite(ctx context.Context, cfg *Config, toURI, fromURI string) (*outboundInvite, error) {
	rtpPort, err := s.rtpAllocator.Allocate()
	if err != nil {
		return nil, fmt.Errorf("no RTP ports available: %w", err)
	}

	rtpBindIP := s.listenConfig.GetBindAddress()
	rtpHandler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:     rtpBindIP,
		LocalPort:   rtpPort,
		PayloadType: CodecPCMU.PayloadType,
		ClockRate:   CodecPCMU.ClockRate,
		Logger:      s.logger,
	})
	if err != nil {
		s.rtpAllocator.Release(rtpPort)
		return nil, fmt.Errorf("failed to create RTP handler: %w", err)
	}

	_, localPort := rtpHandler.LocalAddr()
	externalIP := s.listenConfig.GetExternalIP()
	sdpBody := s.GenerateSDP(DefaultSDPConfig(externalIP, localPort))

	scheme := "sip"
	if cfg.Transport == TransportTLS {
		scheme = "sips"
	}

	recipient := sip.Uri{
		Scheme: scheme,
		Host:   cfg.Server,
		Port:   cfg.Port,
		User:   toURI,
	}
	if cfg.Transport == TransportTLS || cfg.Transport == TransportTCP {
		if recipient.UriParams == nil {
			recipient.UriParams = sip.NewParams()
		}
		recipient.UriParams.Add("transport", string(cfg.Transport))
	}

	fromDomain := cfg.Domain
	if fromDomain == "" {
		fromDomain = cfg.Server
	}
	fromUser := strings.TrimSpace(fromURI)
	if cfg.CallerID != "" {
		fromUser = cfg.CallerID
	}
	if fromUser == "" {
		fromUser = cfg.Username
	}
	if fromUser == "" {
		rtpHandler.Stop()
		s.rtpAllocator.Release(rtpPort)
		return nil, fmt.Errorf("SIP From user is empty: fromPhone, caller_id, or sip_username must be set")
	}

	fromHDR := &sip.FromHeader{
		DisplayName: fromURI,
		Address: sip.Uri{
			Scheme: scheme,
			User:   fromUser,
			Host:   fromDomain,
		},
		Params: sip.NewParams(),
	}
	fromHDR.Params.Add("tag", sip.GenerateTagN(16))

	inviteHeaders := []sip.Header{fromHDR}
	if callerID := strings.TrimSpace(fromURI); callerID != "" {
		pai := sip.NewHeader("P-Asserted-Identity", fmt.Sprintf("<%s:%s@%s>", scheme, callerID, fromDomain))
		inviteHeaders = append(inviteHeaders, pai)
	}
	for name, value := range cfg.CustomHeaders {
		inviteHeaders = append(inviteHeaders, sip.NewHeader(name, value))
	}

	dialogSession, err := s.dialogClientCache.Invite(ctx, recipient, []byte(sdpBody), inviteHeaders...)
	if err != nil {
		rtpHandler.Stop()
		s.rtpAllocator.Release(rtpPort)
		return nil, fmt.Errorf("failed to send INVITE: %w", err)
	}

	callID := dialogSession.InviteRequest.CallID().Value()

	return &outboundInvite{
		rtpHandler:    rtpHandler,
		rtpPort:       rtpPort,
		localPort:     localPort,
		externalIP:    externalIP,
		callID:        callID,
		dialogSession: dialogSession,
		server:        s,
	}, nil
}

// answeredCall holds the result of waitForAnswer — the RTP handler with remote
// address set, codec negotiated, RTP started, and ACK sent.
type answeredCall struct {
	rtpHandler      *RTPHandler
	negotiatedCodec *Codec
}

// waitForAnswer blocks until the outbound dialog receives a 200 OK, parses
// the SDP answer, starts RTP, and sends ACK. Returns the answered call state.
//
// Shared between handleOutboundDialog and MakeBridgeCall.
func (s *Server) waitForAnswer(ctx context.Context, invite *outboundInvite, cfg *Config) (*answeredCall, error) {
	err := invite.dialogSession.WaitAnswer(ctx, sipgo.AnswerOptions{
		Username: cfg.Username,
		Password: cfg.Password,
		OnResponse: func(res *sip.Response) error {
			s.logger.Debugw("Outbound call response",
				"call_id", invite.callID, "status", res.StatusCode)
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	// 200 OK — parse SDP, start RTP before ACK (Asterisk needs media immediately)
	var negotiatedCodec *Codec
	if invite.dialogSession.InviteResponse != nil {
		if body := invite.dialogSession.InviteResponse.Body(); len(body) > 0 {
			sdpInfo, parseErr := s.ParseSDP(body)
			if parseErr == nil && sdpInfo.ConnectionIP != "" && sdpInfo.AudioPort > 0 {
				invite.rtpHandler.SetRemoteAddr(sdpInfo.ConnectionIP, sdpInfo.AudioPort)
				if sdpInfo.PreferredCodec != nil {
					invite.rtpHandler.SetCodec(sdpInfo.PreferredCodec)
					negotiatedCodec = sdpInfo.PreferredCodec
				}
			} else if parseErr != nil {
				s.logger.Warnw("Failed to parse SDP from 200 OK",
					"call_id", invite.callID, "error", parseErr)
			}
		} else {
			s.logger.Warnw("No SDP body in 200 OK response", "call_id", invite.callID)
		}
	} else {
		s.logger.Warnw("No InviteResponse available after WaitAnswer", "call_id", invite.callID)
	}

	invite.rtpHandler.Start()

	ackCtx, ackCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ackCancel()
	if err := invite.dialogSession.Ack(ackCtx); err != nil {
		invite.rtpHandler.Stop()
		return nil, fmt.Errorf("failed to send ACK: %w", err)
	}

	return &answeredCall{
		rtpHandler:      invite.rtpHandler,
		negotiatedCodec: negotiatedCodec,
	}, nil
}
