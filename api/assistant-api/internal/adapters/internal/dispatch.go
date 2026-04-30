// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	observe "github.com/rapidaai/api/assistant-api/internal/observe"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// =============================================================================
// OnPacket — enqueue into the priority channel
// =============================================================================

// OnPacket enqueues each packet into the appropriate priority channel.
// Each channel has a dedicated dispatcher goroutine so no tier can stall another.
//
// Priority:
//
//	critical — interrupts, directives                        (preempts everything)
//	input    — inbound audio, denoise, VAD, STT, EOS         (user → system)
//	output   — LLM generation, text aggregation, TTS          (system → user)
//	low      — recording, metrics, persistence, events         (background work)
func (r *genericRequestor) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	for _, p := range pkts {
		e := packetEnvelope{ctx: ctx, pkt: p}
		switch p.(type) {
		// Critical — interrupts, tool lifecycle
		case internal_type.InterruptionDetectedPacket,
			internal_type.TTSInterruptPacket,
			internal_type.LLMInterruptPacket,
			internal_type.STTInterruptPacket,
			internal_type.TurnChangePacket:
			r.criticalCh <- e

		// Input — inbound audio pipeline, VAD, STT, EOS
		case internal_type.UserAudioReceivedPacket,
			internal_type.UserTextReceivedPacket,
			internal_type.DenoiseAudioPacket,
			internal_type.DenoisedAudioPacket,
			internal_type.VadAudioPacket,
			internal_type.VadSpeechActivityPacket,
			internal_type.SpeechToTextPacket,
			internal_type.EndOfSpeechPacket,
			internal_type.InterimEndOfSpeechPacket,
			internal_type.UserInputPacket,
			internal_type.LLMToolResultPacket:
			r.inputCh <- e

		// Output — LLM generation, TTS, outbound pipeline
		case internal_type.LLMResponseDeltaPacket,
			internal_type.LLMResponseDonePacket,
			internal_type.ErrorPacket,
			internal_type.InjectMessagePacket,
			internal_type.StartIdleTimeoutPacket,
			internal_type.StopIdleTimeoutPacket,
			internal_type.TTSTextPacket,
			internal_type.TTSDonePacket,
			internal_type.TextToSpeechAudioPacket,
			internal_type.TextToSpeechEndPacket,
			internal_type.LLMToolCallPacket:
			r.outputCh <- e

		// Low — recording, metrics, persistence, events
		case internal_type.RecordUserAudioPacket,
			internal_type.RecordAssistantAudioPacket,
			internal_type.SaveMessagePacket,
			internal_type.ToolLogCreatePacket,
			internal_type.ToolLogUpdatePacket,
			internal_type.ConversationMetricPacket,
			internal_type.ConversationMetadataPacket,
			internal_type.AssistantMessageMetricPacket,
			internal_type.UserMessageMetricPacket,
			internal_type.UserMessageMetadataPacket,
			internal_type.AssistantMessageMetadataPacket,
			internal_type.ConversationEventPacket:
			r.lowCh <- e

		default:
			r.logger.Warnf("OnPacket: unrouted packet type %T, falling back to inputCh", p)
			r.inputCh <- e
		}
	}
	return nil
}

// =============================================================================
// runDispatcher — dedicated consumer goroutines per priority tier
// =============================================================================

// runCriticalDispatcher processes critical-priority packets (interrupts,
// directives) on a dedicated goroutine so they are never queued behind
// normal or low-priority work.
func (r *genericRequestor) runCriticalDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.drainCriticalChannel()
			return
		case e := <-r.criticalCh:
			r.dispatch(e.ctx, e.pkt)
		}
	}
}

func (r *genericRequestor) drainCriticalChannel() {
	for {
		select {
		case e := <-r.criticalCh:
			r.dispatch(e.ctx, e.pkt)
		default:
			return
		}
	}
}

// runInputDispatcher processes inbound pipeline packets (user audio, denoise,
// VAD, STT, EOS) on a dedicated goroutine. Kept separate from the output
// pipeline so a burst of inbound audio never delays LLM/TTS streaming.
func (r *genericRequestor) runInputDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.drainInputChannel()
			return
		case e := <-r.inputCh:
			r.dispatch(e.ctx, e.pkt)
		}
	}
}

func (r *genericRequestor) drainInputChannel() {
	for {
		select {
		case e := <-r.inputCh:
			r.dispatch(e.ctx, e.pkt)
		default:
			return
		}
	}
}

// runOutputDispatcher processes outbound pipeline packets (LLM generation,
// text aggregation, TTS audio) on a dedicated goroutine. Kept separate from
// the input pipeline so LLM response streaming is never queued behind audio.
func (r *genericRequestor) runOutputDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.drainOutputChannel()
			return
		case e := <-r.outputCh:
			r.dispatch(e.ctx, e.pkt)
		}
	}
}

func (r *genericRequestor) drainOutputChannel() {
	for {
		select {
		case e := <-r.outputCh:
			r.dispatch(e.ctx, e.pkt)
		default:
			return
		}
	}
}

// runLowDispatcher processes low-priority packets (persistence, metrics,
// events, tools) on a dedicated goroutine so DB writes never stall audio
// or LLM processing.
func (r *genericRequestor) runLowDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Drain remaining low packets so completion metrics (STATUS,
			// TIME_TAKEN) enqueued just before Disconnect() are not lost.
			r.drainLowChannel()
			return
		case e := <-r.lowCh:
			r.dispatch(e.ctx, e.pkt)
		}
	}
}

func (r *genericRequestor) drainLowChannel() {
	for {
		select {
		case e := <-r.lowCh:
			r.dispatch(e.ctx, e.pkt)
		default:
			return
		}
	}
}

// =============================================================================
// dispatch — routes a single packet to its handler
// =============================================================================

func (r *genericRequestor) dispatch(ctx context.Context, p internal_type.Packet) {
	switch vl := p.(type) {
	case internal_type.UserTextReceivedPacket:
		r.handleUserText(ctx, vl)
	case internal_type.UserAudioReceivedPacket:
		r.handleUserAudio(ctx, vl)
	case internal_type.DenoiseAudioPacket:
		r.handleDenoise(ctx, vl)
	case internal_type.DenoisedAudioPacket:
		r.handleDenoisedAudio(ctx, vl)
	case internal_type.VadAudioPacket:
		r.handleVadAudio(ctx, vl)
	case internal_type.VadSpeechActivityPacket:
		r.callEndOfSpeech(ctx, vl)
	case internal_type.RecordUserAudioPacket:
		r.handleRecordUserAudio(ctx, vl)
	case internal_type.RecordAssistantAudioPacket:
		r.handleRecordAssistantAudio(ctx, vl)
	case internal_type.SpeechToTextPacket:
		r.handleSpeechToText(ctx, vl)

		// End of speech and normalization
	case internal_type.InterimEndOfSpeechPacket:
		r.handleInterimEndOfSpeech(ctx, vl)
	case internal_type.EndOfSpeechPacket:
		r.handleEndOfSpeech(ctx, vl)
	case internal_type.UserInputPacket:
		r.handleUserInput(ctx, vl)

		// Interruptions
	case internal_type.InterruptionDetectedPacket:
		r.handleInterruption(ctx, vl)
	case internal_type.TTSInterruptPacket:
		r.handleInterruptTTS(ctx, vl)
	case internal_type.LLMInterruptPacket:
		r.handleInterruptLLM(ctx, vl)
	case internal_type.STTInterruptPacket:
		r.handleInterruptSTT(ctx, vl)
	case internal_type.TurnChangePacket:
		r.handleContextChange(ctx, vl)

		// LLM pipeline
	case internal_type.LLMResponseDeltaPacket:
		r.handleLLMDelta(ctx, vl)
	case internal_type.LLMResponseDonePacket:
		r.handleLLMDone(ctx, vl)
	case internal_type.ErrorPacket:
		r.handleErrorPacket(ctx, vl)

		// TTS output
	case internal_type.TTSTextPacket:
		r.handleTTSText(ctx, vl)
	case internal_type.TTSDonePacket:
		r.handleTTSDone(ctx, vl)

		// Static / system-injected
	case internal_type.InjectMessagePacket:
		r.handleInjectMessagePacket(ctx, vl)
	case internal_type.StartIdleTimeoutPacket:
		r.handleStartIdleTimeoutPacket(ctx, vl)
	case internal_type.StopIdleTimeoutPacket:
		r.handleStopIdleTimeoutPacket(ctx, vl)
	case internal_type.TextToSpeechAudioPacket:
		r.handleTTSAudio(ctx, vl)
	case internal_type.TextToSpeechEndPacket:
		r.handleTTSEnd(ctx, vl)
	case internal_type.SaveMessagePacket:
		r.handleSaveMessage(ctx, vl)
	case internal_type.ConversationMetricPacket:
		r.handleConversationMetric(ctx, vl)
	case internal_type.ConversationMetadataPacket:
		r.handleConversationMetadata(ctx, vl)
	case internal_type.UserMessageMetricPacket:
		r.handleUserMessageMetric(ctx, vl)
	case internal_type.AssistantMessageMetricPacket:
		r.handleAssistantMessageMetric(ctx, vl)
	case internal_type.UserMessageMetadataPacket:
		r.handleUserMessageMetadata(ctx, vl)
	case internal_type.AssistantMessageMetadataPacket:
		r.handleAssistantMessageMetadata(ctx, vl)
	case internal_type.ToolLogCreatePacket:
		r.handleToolLogCreate(ctx, vl)
	case internal_type.ToolLogUpdatePacket:
		r.handleToolLogUpdate(ctx, vl)
	case internal_type.LLMToolCallPacket:
		r.handleToolCall(ctx, vl)
	case internal_type.LLMToolResultPacket:
		r.handleToolResult(ctx, vl)
	case internal_type.ConversationEventPacket:
		r.handleConversationEvent(ctx, vl)
	default:
		r.logger.Warnf("unknown packet type received in dispatcher %T", vl)
	}
}

// =============================================================================
// calling end of speech and handling
// =============================================================================

func (talking *genericRequestor) callEndOfSpeech(ctx context.Context, vl internal_type.Packet) error {
	if talking.endOfSpeech != nil {
		utils.Go(ctx, func() {
			if err := talking.endOfSpeech.Analyze(ctx, vl); err != nil {
				talking.logger.Errorf("end of speech analyze error: %v", err)
			}
		})
		return nil
	}
	return errors.New("end of speech analyzer not configured")
}

func (talking *genericRequestor) handleEndOfSpeech(ctx context.Context, vl internal_type.EndOfSpeechPacket) {
	if err := talking.callInputNormalizer(ctx, vl); err != nil {
		talking.OnPacket(ctx, internal_type.UserInputPacket{
			ContextID: vl.ContextID,
			Text:      vl.Speech,
		})
	}
}

func (talking *genericRequestor) handleInterimEndOfSpeech(ctx context.Context, vl internal_type.InterimEndOfSpeechPacket) {
	talking.Notify(ctx, &protos.ConversationUserMessage{
		Id:        talking.GetID(),
		Message:   &protos.ConversationUserMessage_Text{Text: vl.Speech},
		Completed: false,
		Time:      timestamppb.New(time.Now()),
	})
}

// =============================================================================
// Input handlers
// =============================================================================

func (talking *genericRequestor) handleUserText(ctx context.Context, vl internal_type.UserTextReceivedPacket) {
	// Cancel any in-flight LLM/TTS before starting the new text turn.
	// On the first message (state Unknown) Transition(Interrupted) is
	// rejected by the state machine and handleInterruption returns early —
	// which is correct: there is nothing active to interrupt.
	talking.handleInterruption(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: talking.GetID(),
		Source:    internal_type.InterruptionSourceWord,
	})

	vl.ContextID = talking.GetID()
	if err := talking.callEndOfSpeech(ctx, vl); err != nil {
		talking.OnPacket(ctx, internal_type.EndOfSpeechPacket{ContextID: vl.ContextID, Speech: vl.Text})
	}
}

func (talking *genericRequestor) handleUserAudio(ctx context.Context, vl internal_type.UserAudioReceivedPacket) {
	if talking.denoiser != nil && !vl.NoiseReduced {
		talking.OnPacket(ctx, internal_type.DenoiseAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio})
		return
	}
	talking.OnPacket(ctx,
		internal_type.RecordUserAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio},
		internal_type.VadAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio},
	)
	if talking.speechToTextTransformer != nil {
		utils.Go(ctx, func() {
			if err := talking.speechToTextTransformer.Transform(ctx, vl); err != nil {
				talking.logger.Tracef(ctx, "error while transforming input %s and error %s", talking.speechToTextTransformer.Name(), err.Error())
			}
		})
	}
	// Route audio to EOS for audio-based turn detectors (e.g. Pipecat Smart Turn, later other end of speech may need audio with conversation context).
	// Text-based and silence-based EOS implementations ignore this packet type.
	talking.callEndOfSpeech(ctx, vl)
}

// =============================================================================
// Pre-processing handlers
// =============================================================================

func (talking *genericRequestor) handleDenoise(ctx context.Context, vl internal_type.DenoiseAudioPacket) {
	if talking.denoiser != nil {
		if err := talking.denoiser.Denoise(ctx, vl); err != nil {
			talking.logger.Warnf("denoiser returned unexpected error: %+v", err)
		}
	}
}

func (talking *genericRequestor) handleDenoisedAudio(ctx context.Context, vl internal_type.DenoisedAudioPacket) {
	// Re-emit as UserAudioReceivedPacket with NoiseReduced=true so it flows
	// through handleUserAudio's normal fan-out (Record, VAD, STT, EOS).
	// Always mark NoiseReduced=true to prevent re-entering the denoiser,
	// even if the denoiser fell back to the original audio on error.
	talking.OnPacket(ctx, internal_type.UserAudioReceivedPacket{
		ContextID:    vl.ContextID,
		Audio:        vl.Audio,
		NoiseReduced: true,
	})
}

func (talking *genericRequestor) handleVadAudio(ctx context.Context, vl internal_type.VadAudioPacket) {
	if talking.vad != nil {
		utils.Go(ctx, func() {
			if err := talking.vad.Process(ctx, internal_type.UserAudioReceivedPacket{ContextID: vl.ContextID, Audio: vl.Audio}); err != nil {
				talking.logger.Warnf("error while processing with vad %s", err.Error())
			}
		})
	}
}

// =============================================================================
// Recording handlers
// =============================================================================

func (talking *genericRequestor) handleRecordUserAudio(ctx context.Context, vl internal_type.RecordUserAudioPacket) {
	if talking.recorder != nil {
		if err := talking.recorder.Record(ctx, vl); err != nil {
			talking.logger.Errorf("recorder error: %v", err)
		}
	}
}

func (talking *genericRequestor) handleRecordAssistantAudio(ctx context.Context, vl internal_type.RecordAssistantAudioPacket) {
	if talking.recorder != nil {
		if err := talking.recorder.Record(ctx, vl); err != nil {
			talking.logger.Errorf("recorder error: %v", err)
		}
	}
}

// =============================================================================
// Speech detection handlers
// =============================================================================

func (talking *genericRequestor) handleSpeechToText(ctx context.Context, vl internal_type.SpeechToTextPacket) {
	vl.ContextID = talking.GetID()
	if err := talking.callEndOfSpeech(ctx, vl); err != nil {
		if !vl.Interim {
			talking.OnPacket(ctx, internal_type.EndOfSpeechPacket{
				ContextID: vl.ContextID,
				Speech:    vl.Script,
				Speechs:   []internal_type.SpeechToTextPacket{vl},
			})
		}
	}
}

func (talking *genericRequestor) callInputNormalizer(ctx context.Context, vl internal_type.EndOfSpeechPacket) error {
	if talking.inputNormalizer == nil {
		return errors.New("input inputNormalizer not configured")
	}
	if err := talking.inputNormalizer.Normalize(ctx, vl); err != nil {
		talking.logger.Errorf("input inputNormalizer error: %v", err)
		return err
	}
	return nil
}

func (talking *genericRequestor) handleUserInput(ctx context.Context, vl internal_type.UserInputPacket) {
	talking.OnPacket(ctx, internal_type.StopIdleTimeoutPacket{
		ContextID: talking.GetID(), ResetCount: true,
	})

	//
	if err := talking.Transition(LLMGenerating); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}

	// Use the CURRENT session context, not the packet's carried context.
	// A word-interrupt on the critical dispatcher may have rotated the
	// context between when the EOS/normalizer pipeline started and now.
	// Using the stale packet context would cause all LLM responses to be
	// discarded as "stale_context" by handleLLMDelta/handleLLMDone.
	contextID := talking.GetID()
	vl.ContextID = contextID

	if err := talking.Notify(ctx, &protos.ConversationUserMessage{
		Id:        contextID,
		Message:   &protos.ConversationUserMessage_Text{Text: vl.Text},
		Completed: true,
		Time:      timestamppb.New(time.Now()),
	}); err != nil {
		talking.logger.Tracef(ctx, "might be returning processing the duplicate message so cut it out.")
		return
	}
	talking.OnPacket(ctx,
		internal_type.SaveMessagePacket{ContextID: contextID, MessageRole: "user", Text: vl.Text},
		internal_type.UserMessageMetadataPacket{ContextID: contextID, Metadata: []*protos.Metadata{
			{
				Key:   "language",
				Value: vl.Language.Name,
			},
			{
				Key:   "language_code",
				Value: vl.Language.ISO639_1,
			}}},
		internal_type.UserMessageMetricPacket{ContextID: contextID, Metrics: []*protos.Metric{{Name: "user_turn", Value: type_enums.CONVERSATION_COMPLETE.String(), Description: "User turn started"}}})

	if talking.assistantExecutor != nil {
		utils.Go(ctx, func() {
			if err := talking.assistantExecutor.Execute(ctx, talking, vl); err != nil {
				talking.OnPacket(ctx, internal_type.LLMErrorPacket{ContextID: contextID, Error: err})
			}
		})
	}
}

// =============================================================================
// Interruption handlers
// =============================================================================

func (talking *genericRequestor) handleInterruption(ctx context.Context, vl internal_type.InterruptionDetectedPacket) {
	if vl.ContextID == "" {
		vl.ContextID = talking.GetID()
	}

	switch vl.Source {
	case internal_type.InterruptionSourceWord:
		talking.OnPacket(ctx, internal_type.StopIdleTimeoutPacket{ContextID: vl.ContextID})
		if err := talking.callEndOfSpeech(ctx, vl); err != nil {
			talking.logger.Errorf("end of speech error: %v", err)
		}
		if err := talking.Transition(Interrupted); err != nil {
			return
		}
		talking.OnPacket(ctx,
			internal_type.RecordAssistantAudioPacket{ContextID: vl.ContextID, Truncate: true},
			internal_type.TTSInterruptPacket{ContextID: vl.ContextID, StartAt: vl.StartAt, EndAt: vl.EndAt},
			internal_type.LLMInterruptPacket{ContextID: vl.ContextID},
		)
		utils.Go(ctx, func() {
			talking.Notify(ctx, &protos.ConversationInterruption{
				Type: protos.ConversationInterruption_INTERRUPTION_TYPE_WORD,
				Time: timestamppb.Now(),
			})
		})

	default:
		if vl.StartAt < 5 {
			return
		}

		// for metrics consistency, emit an STTInterruptPacket on the critical dispatcher so it is processed before any subsequent audio.
		talking.OnPacket(ctx, internal_type.STTInterruptPacket{ContextID: vl.ContextID})

		// Call end of speech to trigger any end-of-speech logic (e.g. finalizing STT results, stopping recording) before transitioning state and emitting interrupts.
		if err := talking.callEndOfSpeech(ctx, vl); err != nil {
			talking.logger.Errorf("end of speech error: %v", err)
		}

		if err := talking.Transition(Interrupt); err != nil {
			return
		}
		// For VAD interruptions we may have already emitted the InterruptionDetectedPacket from the end of speech handler, so check if the source is VAD before emitting another TTSInterruptPacket and LLMInterruptPacket to avoid duplicates.
		utils.Go(ctx, func() {
			talking.Notify(ctx, &protos.ConversationInterruption{
				Type: protos.ConversationInterruption_INTERRUPTION_TYPE_VAD,
				Time: timestamppb.Now(),
			})
		})
	}
}

func (talking *genericRequestor) handleContextChange(ctx context.Context, vl internal_type.TurnChangePacket) {
	if vl.ContextID == "" {
		vl.ContextID = talking.GetID()
	}
	if vl.Time.IsZero() {
		vl.Time = time.Now()
	}

	if talking.speechToTextTransformer != nil {
		if err := talking.speechToTextTransformer.Transform(ctx, vl); err != nil {
			talking.logger.Errorf("stt context-change update failed: %v", err)
		}
	}
	if talking.textToSpeechTransformer != nil {
		if err := talking.textToSpeechTransformer.Transform(ctx, vl); err != nil {
			talking.logger.Errorf("tts context-change update failed: %v", err)
		}
	}

	talking.OnPacket(ctx, internal_type.ConversationEventPacket{
		ContextID: vl.ContextID,
		Name:      "turn",
		Data: map[string]string{
			"type":           "change",
			"old_context_id": vl.PreviousContextID,
			"new_context_id": vl.ContextID,
			"reason":         vl.Reason,
			"source":         vl.Source,
		},
		Time: vl.Time,
	})
}

func (talking *genericRequestor) handleInterruptTTS(ctx context.Context, vl internal_type.TTSInterruptPacket) {
	if talking.textToSpeechTransformer != nil {
		if err := talking.textToSpeechTransformer.Transform(ctx, vl); err != nil {
			talking.logger.Errorf("tts interrupt: %v", err)
		}
	}
}

func (talking *genericRequestor) handleInterruptLLM(ctx context.Context, vl internal_type.LLMInterruptPacket) {
	if talking.assistantExecutor != nil {
		if err := talking.assistantExecutor.Execute(ctx, talking, vl); err != nil {
			talking.logger.Errorf("llm interrupt: %v", err)
		}
	}
}

func (talking *genericRequestor) handleInterruptSTT(ctx context.Context, vl internal_type.STTInterruptPacket) {
	if talking.speechToTextTransformer != nil {
		if err := talking.speechToTextTransformer.Transform(ctx, vl); err != nil {
			talking.logger.Errorf("stt interrupt: %v", err)
		}
	}
}

// =============================================================================
// LLM pipeline handlers
// =============================================================================

func (talking *genericRequestor) handleLLMDelta(ctx context.Context, vl internal_type.LLMResponseDeltaPacket) {
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: vl.ContextID,
			Name:      "llm",
			Data:      map[string]string{"type": "discarded", "reason": "stale_context", "current_context": talking.GetID(), "text": vl.Text},
			Time:      time.Now(),
		})
		return
	}
	if err := talking.Transition(LLMGenerating); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}
	if talking.outputNormalizer != nil {
		talking.outputNormalizer.Normalize(ctx, vl)
	} else {
		talking.OnPacket(ctx, internal_type.TTSTextPacket{ContextID: vl.ContextID, Text: vl.Text})
	}
}

func (talking *genericRequestor) handleLLMDone(ctx context.Context, vl internal_type.LLMResponseDonePacket) {
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: vl.ContextID,
			Name:      "llm",
			Data:      map[string]string{"type": "discarded", "reason": "stale_context", "packet": "done", "current_context": talking.GetID(), "text": vl.Text},
			Time:      time.Now(),
		})
		return
	}
	talking.OnPacket(ctx, internal_type.StartIdleTimeoutPacket{ContextID: vl.ContextID})
	if err := talking.Transition(LLMGenerated); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}
	talking.OnPacket(ctx,
		internal_type.SaveMessagePacket{ContextID: vl.ContextID, MessageRole: "assistant", Text: vl.Text},
		internal_type.AssistantMessageMetricPacket{
			ContextID: vl.ContextID,
			Metrics:   []*protos.Metric{{Name: "assistant_turn", Value: type_enums.CONVERSATION_COMPLETE.String(), Description: fmt.Sprintf("LLM response completed")}},
		},
	)
	if talking.outputNormalizer != nil {
		talking.outputNormalizer.Normalize(ctx, vl)
	} else {
		talking.OnPacket(ctx, internal_type.TTSDonePacket{ContextID: vl.ContextID, Text: vl.Text})
	}
}

func (talking *genericRequestor) handleErrorPacket(ctx context.Context, vl internal_type.ErrorPacket) {
	switch vl.(type) {
	case internal_type.LLMErrorPacket:
		talking.OnPacket(ctx,
			internal_type.UserMessageMetricPacket{
				ContextID: vl.ContextId(),
				Metrics: []*protos.Metric{{
					Name:        "llm_error",
					Value:       vl.ErrMessage(),
					Description: "An error occurred during LLM processing"}},
			},
			internal_type.ConversationEventPacket{
				ContextID: vl.ContextId(),
				Name:      "llm",
				Data:      map[string]string{"type": "error", "message": vl.ErrMessage()},
				Time:      time.Now(),
			})
		talking.Transition(LLMGenerated)
	case internal_type.STTErrorPacket:
		talking.OnPacket(ctx,
			internal_type.UserMessageMetricPacket{
				ContextID: vl.ContextId(),
				Metrics: []*protos.Metric{{
					Name:        "stt_error",
					Value:       vl.ErrMessage(),
					Description: "An error occurred during STT processing"}},
			},
			internal_type.ConversationEventPacket{
				ContextID: vl.ContextId(),
				Name:      "stt",
				Data:      map[string]string{"type": "error", "message": vl.ErrMessage()},
				Time:      time.Now(),
			})
	case internal_type.TTSErrorPacket:
		talking.OnPacket(ctx,
			internal_type.UserMessageMetricPacket{
				ContextID: vl.ContextId(),
				Metrics: []*protos.Metric{{
					Name:        "tts_error",
					Value:       vl.ErrMessage(),
					Description: "An error occurred during TTS processing"}},
			},
			internal_type.ConversationEventPacket{
				ContextID: vl.ContextId(),
				Name:      "tts",
				Data:      map[string]string{"type": "error", "message": vl.ErrMessage()},
				Time:      time.Now(),
			})

	}
	if !vl.IsRecoverable() {
		talking.Notify(ctx,
			&protos.ConversationError{
				AssistantConversationId: talking.Conversation().Id,
				Message:                 vl.ErrMessage(),
			},
			&protos.ConversationDisconnection{
				Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_UNSPECIFIED,
			})
		return
	}
	_ = talking.Notify(ctx, &protos.ConversationError{
		AssistantConversationId: talking.Conversation().Id,
		Message:                 vl.ErrMessage(),
	})
}

// =============================================================================
// Static / system-injected handler
// =============================================================================

// handleInjectMessagePacket speaks a pre-written message (greeting, error, idle timeout).
//
// Both modes follow the same state lifecycle: LLMGenerating → deliver →
// LLMGenerated → start idle timer. This ensures the state machine is in
// LLMGenerated after delivery so subsequent Transition(Interrupted) calls
// (from onIdleTimeout or user interrupt) always succeed and rotate the
// context ID.
//
// Text mode delivers synchronously; audio mode flows through the text
// aggregator and TTS pipeline in a goroutine.
func (talking *genericRequestor) handleInjectMessagePacket(ctx context.Context, vl internal_type.InjectMessagePacket) {
	if err := talking.Transition(LLMGenerating); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}

	// Record injected message in executor history (e.g. for LLM context).
	if talking.assistantExecutor != nil {
		utils.Go(ctx, func() {
			if err := talking.assistantExecutor.Execute(ctx, talking, vl); err != nil {
				talking.logger.Errorf("assistant executor error: %v", err)
			}
		})
	}

	// Use the CURRENT session context — not vl.ContextId(). The inject packet
	// was created before the LLMInterruptPacket rotated the context
	// on the same criticalCh. By the time this handler runs, GetID() returns
	// the post-interrupt context. For welcome messages (no prior interrupt),
	// GetID() returns the original context — both cases are correct.
	contextID := talking.GetID()

	if talking.outputNormalizer != nil {
		// Normalizer path: save/metrics/transition here since handleLLMDone is bypassed.
		talking.OnPacket(ctx,
			internal_type.SaveMessagePacket{ContextID: contextID, MessageRole: "assistant", Text: vl.Text},
			internal_type.AssistantMessageMetricPacket{
				ContextID: contextID,
				Metrics:   []*protos.Metric{{Name: "assistant_turn", Value: type_enums.CONVERSATION_COMPLETE.String(), Description: "Injected message completed"}},
			},
		)
		talking.outputNormalizer.Normalize(ctx, internal_type.InjectMessagePacket{ContextID: contextID, Text: vl.Text})
		if err := talking.Transition(LLMGenerated); err != nil {
			talking.logger.Errorf("messaging transition error: %v", err)
		}
	} else {
		// Fallback: LLMResponseDelta/Done flow through handleLLMDelta/handleLLMDone
		// which handle save, metrics, transition, and idle timer.
		talking.OnPacket(ctx,
			internal_type.LLMResponseDeltaPacket{ContextID: contextID, Text: vl.Text},
			internal_type.LLMResponseDonePacket{ContextID: contextID, Text: vl.Text},
		)
	}
}

// handleStartIdleTimeoutPacket (re)starts the idle timeout timer.
// All timer state lives here — there is no synchronous start path elsewhere.
// Routed on outputCh so producers can order it relative to InjectMessagePacket
// and TTS output packets on the same channel.
func (talking *genericRequestor) handleStartIdleTimeoutPacket(ctx context.Context, vl internal_type.StartIdleTimeoutPacket) {
	if talking.idleTimeoutTimer != nil {
		talking.idleTimeoutTimer.Stop()
	}
	behavior, err := talking.GetBehavior()
	if err != nil {
		return
	}
	if behavior.IdleTimeout == nil || *behavior.IdleTimeout == 0 {
		return
	}

	timeoutDuration := time.Duration(*behavior.IdleTimeout) * time.Second
	talking.idleTimeoutDeadline = time.Now().Add(timeoutDuration)
	talking.idleTimeoutTimer = time.AfterFunc(timeoutDuration, func() {
		if err := talking.onIdleTimeout(ctx); err != nil {
			talking.logger.Errorf("error while handling idle timeout: %v", err)
		}
	})
}

// handleStopIdleTimeoutPacket stops the idle timeout timer.
// ResetCount=true also clears the consecutive idle backoff counter
// (used when the user actively engages, not for system-driven stops).
func (talking *genericRequestor) handleStopIdleTimeoutPacket(ctx context.Context, vl internal_type.StopIdleTimeoutPacket) {
	if talking.idleTimeoutTimer != nil {
		talking.idleTimeoutTimer.Stop()
		talking.idleTimeoutTimer = nil
	}
	talking.idleTimeoutDeadline = time.Time{}

	if vl.ResetCount {
		talking.idleTimeoutCount = 0
	}
}

// =============================================================================
// TTS pipeline handlers
// =============================================================================

func (talking *genericRequestor) handleTTSText(ctx context.Context, vl internal_type.TTSTextPacket) {
	if vl.ContextID != talking.GetID() {
		return
	}
	if talking.textToSpeechTransformer != nil && talking.GetMode().Audio() {
		if err := talking.textToSpeechTransformer.Transform(ctx, vl); err != nil {
			talking.logger.Errorf("tts text: failed to send chunk: %v", err)
		}
	}
	talking.Notify(ctx, &protos.ConversationAssistantMessage{
		Time: timestamppb.Now(), Id: vl.ContextID, Completed: false,
		Message: &protos.ConversationAssistantMessage_Text{Text: vl.Text},
	})
}

func (talking *genericRequestor) handleTTSDone(ctx context.Context, vl internal_type.TTSDonePacket) {
	if vl.ContextID != talking.GetID() {
		return
	}

	if talking.textToSpeechTransformer != nil && talking.GetMode().Audio() {
		if err := talking.textToSpeechTransformer.Transform(ctx, vl); err != nil {
			talking.logger.Errorf("tts done: failed to send final: %v", err)
		}
	}
	talking.Notify(ctx, &protos.ConversationAssistantMessage{
		Time: timestamppb.Now(), Id: vl.ContextID, Completed: true,
		Message: &protos.ConversationAssistantMessage_Text{Text: vl.Text},
	})
}

func (talking *genericRequestor) handleTTSAudio(ctx context.Context, vl internal_type.TextToSpeechAudioPacket) {
	if talking.GetMode().Audio() {
		audioInfo := internal_audio.GetAudioInfo(vl.AudioChunk, internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG)
		talking.extendIdleTimeoutTimer(time.Duration(audioInfo.DurationMs) * time.Millisecond)
	}
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx,
			internal_type.ConversationEventPacket{
				ContextID: vl.ContextID,
				Name:      "tts",
				Data:      map[string]string{"type": "discarded", "reason": "stale_context", "packet": "tts_audio", "current_context": talking.GetID()},
				Time:      time.Now(),
			},
			internal_type.AssistantMessageMetricPacket{
				ContextID: vl.ContextID,
				Metrics:   []*protos.Metric{{Name: "discarded_tts_chunk", Value: "true", Description: fmt.Sprintf("tts end packet discarded due to stale contextID %s", talking.GetID())}},
			})
		return
	}
	if err := talking.Notify(ctx, &protos.ConversationAssistantMessage{
		Time:      timestamppb.Now(),
		Id:        vl.ContextID,
		Message:   &protos.ConversationAssistantMessage_Audio{Audio: vl.AudioChunk},
		Completed: false,
	}); err != nil {
		talking.logger.Tracef(ctx, "error while outputting chunk to the user: %w", err)
	}
	talking.OnPacket(ctx, internal_type.RecordAssistantAudioPacket{ContextID: vl.ContextID, Audio: vl.AudioChunk})
}

func (talking *genericRequestor) handleTTSEnd(ctx context.Context, vl internal_type.TextToSpeechEndPacket) {
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx,
			internal_type.ConversationEventPacket{
				ContextID: vl.ContextID,
				Name:      "tts",
				Data:      map[string]string{"type": "discarded", "reason": "stale_context", "packet": "tts_end", "current_context": talking.GetID()},
				Time:      time.Now(),
			},
			internal_type.AssistantMessageMetricPacket{
				ContextID: vl.ContextID,
				Metrics:   []*protos.Metric{{Name: "discarded_tts", Value: "true", Description: fmt.Sprintf("tts end packet discarded due to stale contextID %s", talking.GetID())}},
			})
		return
	}
	if err := talking.Notify(ctx, &protos.ConversationAssistantMessage{
		Time:      timestamppb.Now(),
		Id:        vl.ContextID,
		Completed: true,
	}); err != nil {
		talking.logger.Tracef(ctx, "error while outputting chunk to the user: %w", err)
	}
}

// =============================================================================
// Persistence handlers
// =============================================================================

func (talking *genericRequestor) handleSaveMessage(ctx context.Context, vl internal_type.SaveMessagePacket) {
	if err := talking.onAddMessage(ctx, vl); err != nil {
		talking.logger.Errorf("Error in onAddMessage: %v", err)
	}
}

func (talking *genericRequestor) handleConversationMetric(ctx context.Context, vl internal_type.ConversationMetricPacket) {
	if len(vl.Metrics) > 0 {
		_ = talking.Notify(ctx, &protos.ConversationMetric{
			AssistantConversationId: talking.Conversation().Id,
			Metrics:                 vl.Metrics,
		})
		if talking.observer != nil {
			talking.observer.EmitMetric(ctx, vl.Metrics)
		}
	}
}

func (talking *genericRequestor) handleConversationMetadata(ctx context.Context, vl internal_type.ConversationMetadataPacket) {
	if len(vl.Metadata) > 0 {
		if err := talking.onAddMetadata(ctx, vl.Metadata...); err != nil {
			talking.logger.Errorf("Error in onAddMetadata: %v", err)
		}
	}
}

func (talking *genericRequestor) handleAssistantMessageMetric(ctx context.Context, vl internal_type.AssistantMessageMetricPacket) {
	if len(vl.Metrics) > 0 {
		_ = talking.Notify(ctx, &protos.ConversationMetric{
			AssistantConversationId: talking.Conversation().Id,
			Metrics:                 vl.Metrics,
		})
		if err := talking.onAddMessageMetric(ctx, "assistant", vl.ContextID, vl.Metrics); err != nil {
			talking.logger.Errorf("Error in onMessageMetric: %v", err)
		}
		if talking.observer != nil {
			talking.observer.MetricCollectors().Collect(ctx, observe.MessageMetricRecord{
				MessageID:      vl.ContextID,
				ConversationID: fmt.Sprintf("%d", talking.Conversation().Id),
				Metrics:        vl.Metrics,
				Time:           time.Now(),
			})
		}
	}
}

func (talking *genericRequestor) handleUserMessageMetric(ctx context.Context, vl internal_type.UserMessageMetricPacket) {
	if len(vl.Metrics) > 0 {
		_ = talking.Notify(ctx, &protos.ConversationMetric{
			AssistantConversationId: talking.Conversation().Id,
			Metrics:                 vl.Metrics,
		})
		if vl.ContextID == "" {
			vl.ContextID = talking.GetID()
		}
		if err := talking.onAddMessageMetric(ctx, "user", vl.ContextID, vl.Metrics); err != nil {
			talking.logger.Errorf("Error in onMessageMetric: %v", err)
		}
		if talking.observer != nil {
			talking.observer.MetricCollectors().Collect(ctx, observe.MessageMetricRecord{
				MessageID:      vl.ContextID,
				ConversationID: fmt.Sprintf("%d", talking.Conversation().Id),
				Metrics:        vl.Metrics,
				Time:           time.Now(),
			})
		}
	}
}

func (talking *genericRequestor) handleUserMessageMetadata(ctx context.Context, vl internal_type.UserMessageMetadataPacket) {
	if len(vl.Metadata) > 0 {
		_ = talking.Notify(ctx, &protos.ConversationMetadata{
			AssistantConversationId: talking.Conversation().Id,
			Metadata:                vl.Metadata,
		})
		if vl.ContextID == "" {
			vl.ContextID = talking.GetID()
		}
		if err := talking.onAddMessageMetadata(ctx, "user", vl.ContextID, vl.Metadata); err != nil {
			talking.logger.Errorf("Error in onAddMessageMetadata: %v", err)
		}
	}
}

func (talking *genericRequestor) handleAssistantMessageMetadata(ctx context.Context, vl internal_type.AssistantMessageMetadataPacket) {
	if len(vl.Metadata) > 0 {
		_ = talking.Notify(ctx, &protos.ConversationMetadata{
			AssistantConversationId: talking.Conversation().Id,
			Metadata:                vl.Metadata,
		})
		if vl.ContextID == "" {
			vl.ContextID = talking.GetID()
		}
		if err := talking.onAddMessageMetadata(ctx, "assistant", vl.ContextID, vl.Metadata); err != nil {
			talking.logger.Errorf("Error in onAddMessageMetadata: %v", err)
		}
	}
}

// =============================================================================
// Tool handlers
// =============================================================================

func (talking *genericRequestor) handleToolCall(ctx context.Context, vl internal_type.LLMToolCallPacket) {
	talking.logger.Debugf("tool-call-> %+v", vl)
	// Notify client + emit event (fast, stays on critical)
	talking.OnPacket(ctx, internal_type.ConversationEventPacket{
		ContextID: vl.ContextID,
		Name:      observe.ComponentTool,
		Data:      map[string]string{observe.DataType: observe.EventToolCallStarted, "name": vl.Name, "id": vl.ToolID, "action": vl.Action.String()},
		Time:      time.Now(),
	})

	if msg, ok := vl.Arguments["message"]; ok && msg != "" {
		talking.OnPacket(ctx,
			internal_type.TTSInterruptPacket{ContextID: vl.ContextID},
			internal_type.InjectMessagePacket{ContextID: vl.ContextID, Text: msg})
	}

	talking.Notify(ctx, &protos.ConversationToolCall{
		Id: vl.ContextID, ToolId: vl.ToolID, Name: vl.Name,
		Action: vl.Action, Args: vl.Arguments, Time: timestamppb.Now(),
	})

	if vl.Action != protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED {
		// Stop idle timer via packet so it is ordered AFTER any InjectMessagePacket
		// enqueued above on outputCh — otherwise inject's auto-restart races us.
		talking.OnPacket(ctx, internal_type.StopIdleTimeoutPacket{
			ContextID: talking.GetID(), ResetCount: true,
		})
		if talking.maxSessionTimer != nil {
			talking.maxSessionTimer.Stop()
		}
	}

	// DB write → lowCh (non-blocking)
	req, _ := json.Marshal(vl)
	talking.OnPacket(ctx, internal_type.ToolLogCreatePacket{
		ContextID: vl.ContextID, ToolID: vl.ToolID, Name: vl.Name, Request: req,
	})

	// Executor → async goroutine
	if talking.assistantExecutor != nil {
		utils.Go(ctx, func() {
			if err := talking.assistantExecutor.Execute(ctx, talking, vl); err != nil {
				talking.logger.Errorf("assistant executor error: %v", err)
			}
		})
	}
}

func (talking *genericRequestor) handleToolResult(ctx context.Context, vl internal_type.LLMToolResultPacket) {
	talking.logger.Debugf("tool-call-> %+v", vl)
	res, _ := json.Marshal(vl)

	// for tool call first persist the tool then do anything else, this ensures that even if the process crashes after this point we have a record of the tool call and its result
	talking.OnPacket(ctx,
		internal_type.ToolLogUpdatePacket{
			ContextID: vl.ContextID, ToolID: vl.ToolID, Response: res,
		})

	switch vl.Action {
	case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
		talking.Notify(ctx, &protos.ConversationDisconnection{
			Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_TOOL,
		})
		return
	case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
		if vl.Result["next_action"] == "end_call" {
			talking.Notify(ctx, &protos.ConversationDisconnection{
				Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_TOOL,
			})
			return
		}
	}

	talking.OnPacket(
		ctx,
		internal_type.TTSInterruptPacket{ContextID: vl.ContextID},
		internal_type.StartIdleTimeoutPacket{ContextID: vl.ContextID},
		internal_type.ConversationEventPacket{
			ContextID: vl.ContextID,
			Name:      observe.ComponentTool,
			Data:      map[string]string{observe.DataType: observe.EventToolCallCompleted, "name": vl.Name, "id": vl.ToolID},
			Time:      time.Now(),
		},
	)
	if talking.assistantExecutor != nil {
		utils.Go(ctx, func() {
			if err := talking.assistantExecutor.Execute(ctx, talking, vl); err != nil {
				talking.logger.Errorf("tool result processing failed: %v", err)
			}
		})
	}
}

func (talking *genericRequestor) handleToolLogCreate(ctx context.Context, vl internal_type.ToolLogCreatePacket) {
	if err := talking.CreateToolLog(ctx, vl.ContextID, vl.ToolID, vl.Name, type_enums.RECORD_IN_PROGRESS, vl.Request); err != nil {
		talking.logger.Errorf("error logging tool call start: %v", err)
	}
}

func (talking *genericRequestor) handleToolLogUpdate(ctx context.Context, vl internal_type.ToolLogUpdatePacket) {
	if err := talking.UpdateToolLog(ctx, vl.ToolID, type_enums.RECORD_COMPLETE, vl.Response); err != nil {
		talking.logger.Errorf("error logging tool call result: %v", err)
	}
}

// =============================================================================
// Observability handler
// =============================================================================

// handleConversationEvent forwards a ConversationEventPacket to the debugger
// via the gRPC stream. If the packet's ContextID is empty (e.g. emitted by an
// STT callback that doesn't hold the session context) the current interaction
// context ID from r.GetID() is used.
func (talking *genericRequestor) handleConversationEvent(ctx context.Context, vl internal_type.ConversationEventPacket) {
	contextID := vl.ContextID
	if contextID == "" {
		contextID = talking.GetID()
	}
	if vl.Time.IsZero() {
		vl.Time = time.Now()
	}
	_ = talking.Notify(ctx, &protos.ConversationEvent{
		Id:   contextID,
		Name: vl.Name,
		Data: vl.Data,
		Time: timestamppb.New(vl.Time),
	})
	if talking.observer != nil {
		talking.observer.EventCollectors().Collect(ctx, observe.EventRecord{
			ConversationID: talking.observer.Meta().AssistantConversationID,
			MessageID:      contextID,
			Name:           vl.Name,
			Data:           vl.Data,
			Time:           vl.Time,
		})
	}
}
