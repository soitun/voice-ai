// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"

	internal_sentence_aggregator "github.com/rapidaai/api/assistant-api/internal/aggregator/text"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_denoiser "github.com/rapidaai/api/assistant-api/internal/denoiser"
	internal_end_of_speech "github.com/rapidaai/api/assistant-api/internal/end_of_speech"
	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_vad "github.com/rapidaai/api/assistant-api/internal/vad"
	"github.com/rapidaai/pkg/utils"
	"golang.org/x/sync/errgroup"
)

// Init initializes the audio talking system for a given assistant persona.
// It sets up both audio input and output transformer.
// This function is typically called at the beginning of a communication session.
func (listening *genericRequestor) initializeSpeechToText(ctx context.Context) error {
	eGroup, ectx := errgroup.WithContext(ctx)
	options := utils.Option{"microphone.eos.timeout": 500}
	// only initialize speech to text if the mode is audio or both
	transformerConfig, _ := listening.GetSpeechToTextTransformer()
	if transformerConfig != nil {
		options = utils.MergeMaps(options, transformerConfig.GetOptions())
		eGroup.Go(func() error {
			credentialId, err := options.GetUint64("rapida.credential_id")
			if err != nil {
				listening.logger.Errorf("unable to find credential from options %+v", err)
				return err
			}
			credential, err := listening.VaultCaller().GetCredential(ectx, listening.Auth(), credentialId)
			if err != nil {
				listening.logger.Errorf("Api call to find credential failed %+v", err)
				return err
			}

			// Use the original session ctx (not errgroup's ectx) so the
			// transformer's stream lifecycle is tied to the session, not
			// the short-lived errgroup that finishes after init.
			atransformer, err := internal_transformer.GetSpeechToTextTransformer(
				ctx,
				listening.logger,
				transformerConfig.AudioProvider,
				credential,
				func(pkt ...internal_type.Packet) error { return listening.OnPacket(ctx, pkt...) },
				options)
			if err != nil {
				listening.logger.Errorf("unable to create input audio transformer with error %v", err)
				return err
			}
			err = atransformer.Initialize()
			if err != nil {
				listening.logger.Errorf("unable to initilize transformer %v", err)
				return err
			}
			listening.speechToTextTransformer = atransformer
			return nil

		})

		eGroup.Go(func() error {
			err := listening.initializeVAD(ctx, options)
			if err != nil {
				listening.logger.Errorf("illegal input audio transformer, check the config and re-init")
			}
			return nil
		})

		eGroup.Go(func() error {
			err := listening.initializeDenoiser(ctx, options)
			if err != nil {
				listening.logger.Errorf("illegal input audio transformer, check the config and re-init")
			}
			return nil
		})

	}
	if err := eGroup.Wait(); err != nil {
		listening.logger.Errorf("illegal init %+v", err)
		return err
	}
	return nil
}

func (listening *genericRequestor) disconnectSpeechToText(ctx context.Context) error {
	if listening.speechToTextTransformer != nil {
		if err := listening.speechToTextTransformer.Close(ctx); err != nil {
			listening.logger.Warnf("cancel all output transformer with error %v", err)
		}
		listening.speechToTextTransformer = nil
	}
	if listening.vad != nil {
		if err := listening.vad.Close(); err != nil {
			listening.logger.Warnf("cancel vad with error %v", err)
		}
		listening.vad = nil
	}
	if listening.denoiser != nil {
		if err := listening.denoiser.Close(); err != nil {
			listening.logger.Warnf("cancel denoiser with error %v", err)
		}
		listening.denoiser = nil
	}
	return nil

}

func (listening *genericRequestor) initializeEndOfSpeech(ctx context.Context) error {
	options := utils.Option{"microphone.eos.timeout": 500}
	transformerConfig, _ := listening.GetSpeechToTextTransformer()
	if transformerConfig != nil {
		options = utils.MergeMaps(options, transformerConfig.GetOptions())
	}
	endOfSpeech, err := internal_end_of_speech.GetEndOfSpeech(ctx,
		listening.logger,
		listening.OnPacket,
		options)
	if err != nil {
		listening.logger.Warnf("unable to initialize text analyzer %+v", err)
		return err
	}
	listening.endOfSpeech = endOfSpeech
	return nil
}

func (listening *genericRequestor) initializeInputNormalizer(ctx context.Context) error {
	if err := listening.normalizer.Initialize(ctx, func(pkts ...internal_type.Packet) error {
		return listening.OnPacket(ctx, pkts...)
	}); err != nil {
		return err
	}
	return nil
}

func (listening *genericRequestor) disconnectEndOfSpeech(ctx context.Context) error {
	if listening.endOfSpeech != nil {
		if err := listening.endOfSpeech.Close(); err != nil {
			listening.logger.Warnf("cancel end of speech with error %v", err)
		}
	}
	if listening.normalizer != nil {
		if err := listening.normalizer.Close(ctx); err != nil {
			listening.logger.Warnf("cancel input normalizer with error %v", err)
		}
		listening.normalizer = nil
	}
	return nil
}

func (listening *genericRequestor) initializeDenoiser(ctx context.Context, options utils.Option) error {
	denoise, err := internal_denoiser.GetDenoiser(ctx, listening.logger, internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG,
		func(pctx context.Context, pkt ...internal_type.Packet) error { return listening.OnPacket(pctx, pkt...) },
		options)
	if err != nil {
		listening.logger.Errorf("error wile intializing denoiser %+v", err)
	}
	listening.denoiser = denoise
	return nil
}

func (listening *genericRequestor) initializeVAD(ctx context.Context, options utils.Option,
) error {
	vad, err := internal_vad.GetVAD(ctx, listening.logger, listening.OnPacket, options)
	if err != nil {
		listening.logger.Errorf("error wile intializing vad %+v", err)
		return err
	}
	listening.vad = vad
	return nil
}

func (spk *genericRequestor) initializeTextToSpeech(ctx context.Context) error {
	outputTransformer, _ := spk.GetTextToSpeechTransformer()
	if outputTransformer == nil {
		return nil
	}
	speakerOpts := utils.MergeMaps(outputTransformer.GetOptions())
	eGroup, ectx := errgroup.WithContext(ctx)
	eGroup.Go(func() error {
		credentialId, err := speakerOpts.GetUint64("rapida.credential_id")
		if err != nil {
			spk.logger.Errorf("tts: unable to find credential from options %+v", err)
			return err
		}
		credential, err := spk.VaultCaller().GetCredential(ectx, spk.Auth(), credentialId)
		if err != nil {
			spk.logger.Errorf("tts: api call to find credential failed %+v", err)
			return err
		}
		// Use the session ctx (not errgroup's ectx) so the transformer's stream
		// lifecycle is tied to the session, not the short-lived errgroup.
		atransformer, err := internal_transformer.GetTextToSpeechTransformer(
			ctx, spk.logger,
			outputTransformer.GetName(),
			credential,
			func(pkt ...internal_type.Packet) error { return spk.OnPacket(ctx, pkt...) },
			speakerOpts)
		if err != nil {
			spk.logger.Errorf("tts: unable to create transformer %v", err)
			return err
		}
		if err := atransformer.Initialize(); err != nil {
			spk.logger.Errorf("tts: unable to initialize transformer %v", err)
			return err
		}
		spk.textToSpeechTransformer = atransformer
		return nil
	})
	return eGroup.Wait()
}

func (spk *genericRequestor) disconnectTextToSpeech(ctx context.Context) error {
	if spk.textToSpeechTransformer != nil {
		if err := spk.textToSpeechTransformer.Close(ctx); err != nil {
			spk.logger.Errorf("cancel all output transformer with error %v", err)
		}
		spk.textToSpeechTransformer = nil
	}
	return nil
}

// Initialize the text aggregator for assembling sentences from tokens.
// Aggregated sentences are pushed directly to OnPacket as SpeakTextPacket
// values — no intermediate goroutine or channel needed.
func (spk *genericRequestor) initializeTextAggregator(ctx context.Context) error {
	textAggregator, err := internal_sentence_aggregator.GetLLMTextAggregator(ctx, spk.logger,
		func(pctx context.Context, pkts ...internal_type.Packet) error {
			return spk.OnPacket(pctx, pkts...)
		})
	if err == nil {
		spk.textAggregator = textAggregator
	}
	return nil
}

func (io *genericRequestor) disconnectTextAggregator() error {
	if io.textAggregator != nil {
		io.textAggregator.Close()
	}
	return nil
}
