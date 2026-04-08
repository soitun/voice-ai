// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"fmt"

	obs "github.com/rapidaai/api/assistant-api/internal/observe"
)

func (d *Dispatcher) runSession(ctx context.Context, v SessionConnectedPipeline) *PipelineResult {
	d.logger.Infow("Pipeline: SessionConnected", "call_id", v.ID)

	contextID := v.ContextID
	if contextID == "" {
		contextID = v.ID
	}

	if d.onResolveSession == nil {
		return &PipelineResult{Error: ErrCallbackNotConfigured}
	}
	cc, vc, err := d.onResolveSession(ctx, v.ContextID)
	if err != nil {
		d.logger.Error("Pipeline: session resolution failed", "call_id", v.ID, "error", err)
		return &PipelineResult{Error: err}
	}

	if d.onCreateStreamer == nil {
		return &PipelineResult{Error: ErrCallbackNotConfigured}
	}
	streamer, err := d.onCreateStreamer(ctx, cc, vc, v.WebSocket, v.Conn, v.Reader, v.Writer)
	if err != nil {
		d.logger.Error("Pipeline: streamer creation failed", "call_id", v.ID, "error", err)
		return &PipelineResult{Error: err}
	}

	if d.onCreateTalker == nil {
		return &PipelineResult{Error: ErrCallbackNotConfigured}
	}
	talker, err := d.onCreateTalker(ctx, streamer)
	if err != nil {
		d.logger.Error("Pipeline: talker creation failed", "call_id", v.ID, "error", err)
		return &PipelineResult{Error: err}
	}

	auth := cc.ToAuth()

	var observer *obs.ConversationObserver
	if d.onCreateObserver != nil {
		observer = d.onCreateObserver(ctx, contextID, auth, cc.AssistantID, cc.ConversationID)
	}

	var hooks *obs.ConversationHooks
	if d.onCreateHooks != nil {
		hooks = d.onCreateHooks(ctx, auth, cc.AssistantID, cc.ConversationID)
		if hooks != nil {
			hooks.OnBegin(ctx)
		}
	}

	if observer != nil {
		observer.EmitEvent(ctx, obs.ComponentSession, map[string]string{
			obs.DataType:     obs.EventSessionConnected,
			obs.DataProvider: cc.Provider,
		})
		observer.EmitMetadata(ctx, obs.ConversationState(cc.Provider, cc.Direction, cc.CallerNumber, contextID))
	}

	if d.onRunTalk == nil {
		return &PipelineResult{Error: ErrCallbackNotConfigured}
	}
	talkErr := d.onRunTalk(ctx, talker, auth)

	if hooks != nil {
		hooks.OnEnd(ctx)
	}

	reason := "talk_completed"
	if talkErr != nil {
		reason = fmt.Sprintf("talk_error: %v", talkErr)
	}
	if observer != nil {
		observer.EmitEvent(ctx, obs.ComponentSession, map[string]string{
			obs.DataType:   obs.EventCallCompleted,
			obs.DataReason: reason,
		})
		observer.Shutdown(ctx)
	}

	if d.onCompleteSession != nil {
		d.onCompleteSession(ctx, contextID)
	}

	return &PipelineResult{Error: talkErr}
}

func (d *Dispatcher) handleModeSwitch(ctx context.Context, v ModeSwitchPipeline) {
	d.logger.Infow("Pipeline: ModeSwitch", "call_id", v.ID, "from", v.From, "to", v.To)
}
