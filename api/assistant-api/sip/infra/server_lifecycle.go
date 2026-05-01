package sip_infra

import (
	"context"
	"fmt"
	"time"

	"github.com/emiago/sipgo/sip"
)

// GetSession returns the session for a call ID.
func (s *Server) GetSession(callID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[callID]
	return session, exists
}

// EndCall terminates a call by ending the owning session.
func (s *Server) EndCall(session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	s.beginEnding(session, "end_call")
	session.End()
	return nil
}

// sendBye sends SIP BYE to the remote party via the active dialog session.
func (s *Server) sendBye(session *Session) {
	callID := session.GetCallID()

	if ds := session.GetDialogClientSession(); ds != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ds.Bye(ctx); err != nil {
			s.logger.Warnw("Failed to send BYE for outbound call",
				"call_id", callID, "error", err)
		} else {
			s.logger.Infow("Sent BYE for outbound call", "call_id", callID)
		}
	}

	if ds := session.GetDialogServerSession(); ds != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ds.Bye(ctx); err != nil {
			s.logger.Warnw("Failed to send BYE for inbound call",
				"call_id", callID, "error", err)
		} else {
			s.logger.Infow("Sent BYE for inbound call", "call_id", callID)
		}
	}
}

// removeSession removes a session from memory and releases its RTP port.
func (s *Server) removeSession(callID string) {
	s.mu.Lock()
	session, exists := s.sessions[callID]
	if exists {
		delete(s.sessions, callID)
		s.sessionCount.Add(-1)
	}
	delete(s.lifecycles, callID)
	s.mu.Unlock()

	if exists && session != nil {
		if port := session.GetRTPLocalPort(); port > 0 {
			s.rtpAllocator.Release(port)
		}
	}
}

func (s *Server) getOrCreateLifecycle(session *Session) *CallLifecycle {
	if session == nil {
		return nil
	}
	callID := session.GetCallID()
	current := session.GetState()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lifecycles == nil {
		s.lifecycles = make(map[string]*CallLifecycle)
	}
	if lc, ok := s.lifecycles[callID]; ok && lc != nil {
		return lc
	}
	lc := newCallLifecycle(callID, current, s.logger)
	s.lifecycles[callID] = lc
	return lc
}

func (s *Server) transitionLifecycle(session *Session, next CallState, reason string) bool {
	lc := s.getOrCreateLifecycle(session)
	if lc == nil {
		return false
	}
	if err := lc.Transition(next, reason); err != nil {
		s.logger.Warnw("Call lifecycle transition rejected",
			"call_id", session.GetCallID(),
			"from", lc.State(),
			"to", next,
			"reason", reason,
			"error", err)
		return false
	}
	return true
}

func (s *Server) setCallState(session *Session, next CallState, reason string) bool {
	if session == nil {
		return false
	}
	if !s.transitionLifecycle(session, next, reason) {
		return false
	}
	session.SetState(next)
	return true
}

func (s *Server) beginEnding(session *Session, reason string) {
	if session == nil {
		return
	}
	_ = s.transitionLifecycle(session, CallStateEnding, reason)
}

func (s *Server) setPendingInvite(callID string, req *sip.Request, tx sip.ServerTransaction) {
	s.mu.Lock()
	if s.pendingInvites == nil {
		s.pendingInvites = make(map[string]*pendingInvite)
	}
	s.pendingInvites[callID] = &pendingInvite{req: req, tx: tx}
	s.mu.Unlock()
}

func (s *Server) clearPendingInvite(callID string) {
	s.mu.Lock()
	delete(s.pendingInvites, callID)
	s.mu.Unlock()
}

func (s *Server) terminatePendingInvite(callID string, status int) bool {
	s.mu.Lock()
	pending, ok := s.pendingInvites[callID]
	if ok {
		delete(s.pendingInvites, callID)
	}
	s.mu.Unlock()

	if !ok || pending == nil || pending.req == nil || pending.tx == nil {
		return false
	}
	s.sendResponse(pending.tx, pending.req, status)
	return true
}

func (s *Server) markInviteCancelled(callID string) {
	s.mu.Lock()
	if s.cancelledInvites == nil {
		s.cancelledInvites = make(map[string]bool)
	}
	s.cancelledInvites[callID] = true
	s.mu.Unlock()
}

func (s *Server) isInviteCancelled(callID string) bool {
	s.mu.RLock()
	cancelled := s.cancelledInvites[callID]
	s.mu.RUnlock()
	return cancelled
}

func (s *Server) clearInviteCancelled(callID string) {
	s.mu.Lock()
	delete(s.cancelledInvites, callID)
	s.mu.Unlock()
}

// notifyError notifies the configured error handler.
func (s *Server) notifyError(session *Session, err error) {
	s.mu.RLock()
	onError := s.onError
	s.mu.RUnlock()

	if onError != nil {
		onError(session, err)
	}
}

// registerSession registers a session and installs disconnect cleanup.
func (s *Server) registerSession(session *Session, callID string) {
	initialState := session.GetState()
	lifecycle := newCallLifecycle(callID, initialState, s.logger)

	session.SetOnDisconnect(func(sess *Session) {
		s.sendBye(sess)
	})
	session.SetOnEnded(func(sess *Session) {
		_ = s.transitionLifecycle(sess, CallStateEnded, "session_end")
		s.removeSession(callID)
	})
	s.mu.Lock()
	if s.lifecycles == nil {
		s.lifecycles = make(map[string]*CallLifecycle)
	}
	s.sessions[callID] = session
	s.lifecycles[callID] = lifecycle
	s.sessionCount.Add(1)
	s.mu.Unlock()
}
