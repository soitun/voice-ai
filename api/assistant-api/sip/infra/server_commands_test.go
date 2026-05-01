// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"
	"testing"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testServerTx struct {
	done      chan struct{}
	err       error
	responses []*sip.Response
}

func newTestServerTx() *testServerTx {
	return &testServerTx{
		done:      make(chan struct{}),
		responses: make([]*sip.Response, 0, 2),
	}
}

func (t *testServerTx) Terminate() { close(t.done) }
func (t *testServerTx) OnTerminate(f sip.FnTxTerminate) bool {
	if f != nil {
		f("test-tx", t.err)
	}
	return false
}
func (t *testServerTx) Done() <-chan struct{} { return t.done }
func (t *testServerTx) Err() error            { return t.err }
func (t *testServerTx) Respond(res *sip.Response) error {
	t.responses = append(t.responses, res)
	return nil
}
func (t *testServerTx) Acks() <-chan *sip.Request { return nil }
func (t *testServerTx) OnCancel(_ sip.FnTxCancel) bool {
	return false
}

func (t *testServerTx) lastStatus() int {
	if len(t.responses) == 0 {
		return 0
	}
	return t.responses[len(t.responses)-1].StatusCode
}

func newServerForCommandTests(t *testing.T) *Server {
	t.Helper()

	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", Host: "127.0.0.1", Port: 5060},
	}

	return &Server{
		logger:            bridgeTestLogger(),
		listenConfig:      &ListenConfig{Address: "127.0.0.1", Port: 5060, ExternalIP: "127.0.0.1"},
		dialogClientCache: sipgo.NewDialogClientCache(client, contact),
		dialogServerCache: sipgo.NewDialogServerCache(client, contact),
		sessions:          make(map[string]*Session),
		lifecycles:        make(map[string]*CallLifecycle),
		pendingInvites:    make(map[string]*pendingInvite),
		cancelledInvites:  make(map[string]bool),
	}
}

func newSIPRequest(method sip.RequestMethod, callID string) *sip.Request {
	recipient := sip.Uri{Scheme: "sip", User: "bob", Host: "example.com", Port: 5060}
	req := sip.NewRequest(method, recipient)

	params := sip.NewParams()
	params["branch"] = sip.GenerateBranch()
	req.AppendHeader(&sip.ViaHeader{
		ProtocolName:    "SIP",
		ProtocolVersion: "2.0",
		Transport:       "UDP",
		Host:            "127.0.0.1",
		Port:            5060,
		Params:          params,
	})
	req.AppendHeader(&sip.FromHeader{
		DisplayName: "Alice",
		Address: sip.Uri{
			Scheme: "sip",
			User:   "alice",
			Host:   "example.com",
			Port:   5060,
		},
		Params: sip.NewParams(),
	})
	req.AppendHeader(&sip.ToHeader{
		DisplayName: "Bob",
		Address:     recipient,
		Params:      sip.NewParams(),
	})
	ch := sip.CallIDHeader(callID)
	req.AppendHeader(&ch)
	req.AppendHeader(&sip.CSeqHeader{SeqNo: 1, MethodName: method})
	req.SetBody(nil)
	return req
}

func newTestSession(t *testing.T, callID string, dir CallDirection) *Session {
	t.Helper()
	s, err := NewSession(context.Background(), &SessionConfig{
		Config:    bridgeTestConfig(),
		Direction: dir,
		CallID:    callID,
		Codec:     &CodecPCMU,
		Logger:    bridgeTestLogger(),
	})
	require.NoError(t, err)
	return s
}

func TestSIPCommand_INVITE_NoResolver_Rejects500(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.INVITE, "call-invite-1")
	tx := newTestServerTx()

	s.handleInvite(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 500, tx.lastStatus())
}

func TestSIPCommand_ACK_InboundSession_MarksConnected(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.ACK, "call-ack-1")
	tx := newTestServerTx()

	session := newTestSession(t, "call-ack-1", CallDirectionInbound)
	session.SetState(CallStateRinging)
	s.sessions["call-ack-1"] = session

	s.handleAck(req, tx)

	assert.Equal(t, CallStateConnected, session.GetState())
}

func TestSIPCommand_BYE_InboundSession_NotifiesAndEnds(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.BYE, "call-bye-1")
	tx := newTestServerTx()

	session := newTestSession(t, "call-bye-1", CallDirectionInbound)
	session.SetState(CallStateConnected)
	s.sessions["call-bye-1"] = session

	s.handleBye(req, tx)

	assert.True(t, session.IsEnded())
	select {
	case <-session.ByeReceived():
	default:
		t.Fatalf("expected ByeReceived signal")
	}
	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 200, tx.lastStatus())
}

func TestSIPCommand_BYE_UnknownSession_Returns481(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.BYE, "call-bye-unknown")
	tx := newTestServerTx()

	s.handleBye(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
}

func TestSIPCommand_CANCEL_ExistingSession_EndsAndCallbacks(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.CANCEL, "call-cancel-1")
	tx := newTestServerTx()

	session := newTestSession(t, "call-cancel-1", CallDirectionInbound)
	s.registerSession(session, "call-cancel-1")

	cancelCalled := false
	s.onCancel = func(_ *Session) error {
		cancelCalled = true
		return nil
	}

	s.handleCancel(req, tx)

	assert.True(t, cancelCalled)
	assert.True(t, session.IsEnded())
	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 200, tx.lastStatus())
	_, ok := s.sessions["call-cancel-1"]
	assert.False(t, ok)
}

func TestSIPCommand_CANCEL_PendingInvite_Sends200And487(t *testing.T) {
	s := newServerForCommandTests(t)

	inviteReq := newSIPRequest(sip.INVITE, "call-cancel-pending")
	inviteTx := newTestServerTx()
	s.setPendingInvite("call-cancel-pending", inviteReq, inviteTx)

	cancelReq := newSIPRequest(sip.CANCEL, "call-cancel-pending")
	cancelTx := newTestServerTx()

	s.handleCancel(cancelReq, cancelTx)

	require.NotEmpty(t, cancelTx.responses)
	assert.Equal(t, 200, cancelTx.lastStatus())
	require.NotEmpty(t, inviteTx.responses)
	assert.Equal(t, 487, inviteTx.lastStatus())
}

func TestRegisterSession_RemoveHappensOnEndNotDisconnect(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-1", CallDirectionOutbound)

	s.registerSession(session, "call-lifecycle-1")
	session.Disconnect()
	_, exists := s.GetSession("call-lifecycle-1")
	assert.True(t, exists, "session should not be removed on Disconnect")

	session.End()
	_, exists = s.GetSession("call-lifecycle-1")
	assert.False(t, exists, "session should be removed after End")
}

func TestCallLifecycle_InvalidTransitionRejected(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-invalid", CallDirectionInbound)
	s.registerSession(session, "call-lifecycle-invalid")

	require.True(t, s.setCallState(session, CallStateRinging, "test_ringing"))
	require.True(t, s.setCallState(session, CallStateConnected, "test_connected"))
	require.False(t, s.setCallState(session, CallStateRinging, "invalid_back_to_ringing"))
	assert.Equal(t, CallStateConnected, session.GetState())
}

func TestSIPCommand_CANCEL_UnknownSession_Returns481(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.CANCEL, "call-cancel-unknown")
	tx := newTestServerTx()

	s.handleCancel(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
}

func TestSIPCommand_CANCEL_ConnectedSession_Returns481(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.CANCEL, "call-cancel-connected")
	tx := newTestServerTx()

	session := newTestSession(t, "call-cancel-connected", CallDirectionInbound)
	session.SetState(CallStateConnected)
	s.registerSession(session, "call-cancel-connected")

	s.handleCancel(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
	assert.False(t, session.IsEnded())
	_, ok := s.GetSession("call-cancel-connected")
	assert.True(t, ok)
}

func TestSIPCommand_REGISTER_OPTIONS_INFO_Return200(t *testing.T) {
	s := newServerForCommandTests(t)

	cases := []struct {
		name   string
		method sip.RequestMethod
		run    func(*sip.Request, sip.ServerTransaction)
	}{
		{"REGISTER", sip.REGISTER, s.handleRegister},
		{"OPTIONS", sip.OPTIONS, s.handleOptions},
		{"INFO", sip.INFO, s.handleInfo},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := newSIPRequest(tc.method, "call-"+string(tc.method))
			tx := newTestServerTx()
			tc.run(req, tx)
			require.NotEmpty(t, tx.responses)
			assert.Equal(t, 200, tx.lastStatus())
		})
	}
}

func TestSIPCommand_UnknownMethodRouting(t *testing.T) {
	s := newServerForCommandTests(t)

	t.Run("in-dialog unknown returns 200", func(t *testing.T) {
		callID := "call-unknown-in-dialog"
		s.sessions[callID] = newTestSession(t, callID, CallDirectionInbound)

		req := newSIPRequest(sip.RequestMethod("PUBLISH"), callID)
		tx := newTestServerTx()
		s.handleUnknownRequest(req, tx)

		require.NotEmpty(t, tx.responses)
		assert.Equal(t, 200, tx.lastStatus())
	})

	t.Run("out-of-dialog SUBSCRIBE returns 489", func(t *testing.T) {
		req := newSIPRequest(sip.SUBSCRIBE, "call-unknown-subscribe")
		tx := newTestServerTx()
		s.handleUnknownRequest(req, tx)

		require.NotEmpty(t, tx.responses)
		assert.Equal(t, 489, tx.lastStatus())
	})

	t.Run("out-of-dialog other returns 405", func(t *testing.T) {
		req := newSIPRequest(sip.RequestMethod("PUBLISH"), "call-unknown-publish")
		tx := newTestServerTx()
		s.handleUnknownRequest(req, tx)

		require.NotEmpty(t, tx.responses)
		assert.Equal(t, 405, tx.lastStatus())
	})
}
