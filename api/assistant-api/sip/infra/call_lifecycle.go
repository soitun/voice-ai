package sip_infra

import (
	"fmt"
	"sync"

	"github.com/rapidaai/pkg/commons"
)

// CallLifecycle is the single owner of call-state transitions inside sip/infra.
// It validates transitions and emits structured transition logs.
type CallLifecycle struct {
	mu     sync.Mutex
	callID string
	state  CallState
	logger commons.Logger
}

func newCallLifecycle(callID string, initial CallState, logger commons.Logger) *CallLifecycle {
	return &CallLifecycle{
		callID: callID,
		state:  initial,
		logger: logger,
	}
}

func (c *CallLifecycle) State() CallState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *CallLifecycle) Transition(next CallState, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == next {
		return nil
	}
	if !lifecycleTransitionAllowed(c.state, next) {
		return fmt.Errorf("invalid lifecycle transition: %s -> %s", c.state, next)
	}

	prev := c.state
	c.state = next
	if c.logger != nil {
		c.logger.Infow("Call lifecycle transition",
			"call_id", c.callID,
			"from", prev,
			"to", next,
			"from_phase", lifecyclePhase(prev),
			"to_phase", lifecyclePhase(next),
			"reason", reason)
	}
	return nil
}

func lifecycleTransitionAllowed(from, to CallState) bool {
	if from == to {
		return true
	}
	switch from {
	case CallStateInitializing:
		return to == CallStateRinging || to == CallStateConnected || to == CallStateEnding || to == CallStateFailed
	case CallStateRinging:
		return to == CallStateConnected || to == CallStateEnding || to == CallStateFailed
	case CallStateConnected:
		return to == CallStateOnHold || to == CallStateTransferring || to == CallStateBridgeConnected || to == CallStateEnding || to == CallStateFailed
	case CallStateOnHold:
		return to == CallStateConnected || to == CallStateEnding || to == CallStateFailed
	case CallStateTransferring:
		return to == CallStateConnected || to == CallStateBridgeConnected || to == CallStateEnding || to == CallStateFailed
	case CallStateBridgeConnected:
		return to == CallStateConnected || to == CallStateEnding || to == CallStateFailed
	case CallStateEnding:
		return to == CallStateEnded || to == CallStateFailed
	case CallStateFailed:
		return to == CallStateEnding || to == CallStateEnded
	default:
		return false
	}
}

func lifecyclePhase(state CallState) string {
	switch state {
	case CallStateInitializing:
		return "inviting"
	case CallStateRinging:
		return "ringing"
	case CallStateTransferring:
		return "transferring"
	case CallStateBridgeConnected:
		return "bridged"
	case CallStateEnding:
		return "ending"
	case CallStateEnded:
		return "ended"
	default:
		return string(state)
	}
}
