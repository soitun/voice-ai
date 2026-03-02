// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/protos"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"

	internal_agent_embeddings "github.com/rapidaai/api/assistant-api/internal/agent/embedding"
	internal_agent_executor "github.com/rapidaai/api/assistant-api/internal/agent/executor"
	internal_agent_executor_llm "github.com/rapidaai/api/assistant-api/internal/agent/executor/llm"
	internal_agent_rerankers "github.com/rapidaai/api/assistant-api/internal/agent/reranker"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_telemetry_entity "github.com/rapidaai/api/assistant-api/internal/entity/telemetry"
	observe "github.com/rapidaai/api/assistant-api/internal/observe"
	observe_exporters "github.com/rapidaai/api/assistant-api/internal/observe/exporters"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	internal_knowledge_service "github.com/rapidaai/api/assistant-api/internal/services/knowledge"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/parsers"

	//
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// InteractionState — conversation turn state machine
// =============================================================================

// InteractionState tracks the current LLM/TTS generation turn state.
type InteractionState int

const (
	Unknown     InteractionState = 1
	Interrupt   InteractionState = 6
	Interrupted InteractionState = 7
	LLMGenerating InteractionState = 8
	LLMGenerated  InteractionState = 5
)

func (s InteractionState) String() string {
	switch s {
	case Unknown:
		return "Unknown"
	case LLMGenerated:
		return "LLMGenerated"
	case Interrupt:
		return "Interrupt"
	case Interrupted:
		return "Interrupted"
	case LLMGenerating:
		return "LLMGenerating"
	default:
		return "InvalidState"
	}
}

type genericRequestor struct {
	logger   commons.Logger
	config   *config.AssistantConfig
	source   utils.RapidaSource
	auth     types.SimplePrinciple
	streamer internal_type.Streamer

	// service
	assistantService     internal_services.AssistantService
	conversationService  internal_services.AssistantConversationService
	webhookService       internal_services.AssistantWebhookService
	knowledgeService     internal_services.KnowledgeService
	assistantToolService internal_services.AssistantToolService

	//
	postgres      connectors.PostgresConnector
	opensearch    connectors.OpenSearchConnector
	vectordb      connectors.VectorConnector
	queryEmbedder internal_agent_embeddings.QueryEmbedding
	textReranker  internal_agent_rerankers.TextReranking

	// observe collectors — fan out events and metrics to external APM backends
	events  observe.EventCollector
	metrics observe.MetricCollector

	// integration client
	integrationClient integration_client.IntegrationServiceClient
	vaultClient       web_client.VaultClient
	deploymentClient  endpoint_client.DeploymentServiceClient

	// interaction state — inline replacement for the former Messaging wrapper
	msgMu            sync.RWMutex
	contextID        string
	interactionState InteractionState
	msgMode          type_enums.MessageMode

	// listening
	speechToTextTransformer internal_type.SpeechToTextTransformer

	// audio intelligence
	endOfSpeech internal_type.EndOfSpeech
	vad         internal_type.Vad
	denoiser    internal_type.Denoiser

	// speak
	textToSpeechTransformer internal_type.TextToSpeechTransformer
	textAggregator          internal_type.LLMTextAggregator

	recorder       internal_type.Recorder
	templateParser parsers.StringTemplateParser

	// executor
	assistantExecutor internal_agent_executor.AssistantExecutor

	// states
	assistant             *internal_assistant_entity.Assistant
	assistantConversation *internal_conversation_entity.AssistantConversation
	histories             []internal_type.MessagePacket

	args     map[string]interface{}
	metadata map[string]interface{}
	options  map[string]interface{}

	// experience
	idleTimeoutTimer    *time.Timer
	idleTimeoutDeadline time.Time // when the current idle timer is set to fire
	idleTimeoutCount    uint64
	maxSessionTimer     *time.Timer

	// packet dispatcher channels — critical preempts normal, normal preempts low
	criticalCh chan packetEnvelope // interrupts and directives        (cap 16)
	normalCh   chan packetEnvelope // audio, STT, LLM, TTS pipeline    (cap 256)
	lowCh      chan packetEnvelope // recording, metrics, persistence   (cap 512)
}

func NewGenericRequestor(
	ctx context.Context,
	config *config.AssistantConfig,
	logger commons.Logger, source utils.RapidaSource,
	postgres connectors.PostgresConnector, opensearch connectors.OpenSearchConnector,
	redis connectors.RedisConnector, storage storages.Storage, streamer internal_type.Streamer,
) *genericRequestor {
	return &genericRequestor{
		logger:   logger,
		config:   config,
		source:   source,
		streamer: streamer,
		// services
		assistantService:     internal_assistant_service.NewAssistantService(config, logger, postgres, opensearch),
		knowledgeService:     internal_knowledge_service.NewKnowledgeService(config, logger, postgres, storage),
		conversationService:  internal_assistant_service.NewAssistantConversationService(logger, postgres, storage),
		webhookService:       internal_assistant_service.NewAssistantWebhookService(logger, postgres, storage),
		assistantToolService: internal_assistant_service.NewAssistantToolService(logger, postgres, storage),
		templateParser:       parsers.NewPongo2StringTemplateParser(logger),
		//

		postgres:      postgres,
		opensearch:    opensearch,
		vectordb:      opensearch,
		queryEmbedder: internal_agent_embeddings.NewQueryEmbedding(logger, config, redis),
		textReranker:  internal_agent_rerankers.NewTextReranker(logger, config, redis),

		// clients
		integrationClient: integration_client.NewIntegrationServiceClientGRPC(&config.AppConfig, logger, redis),
		deploymentClient:  endpoint_client.NewDeploymentServiceClientGRPC(&config.AppConfig, logger, redis),
		vaultClient:       web_client.NewVaultClientGRPC(&config.AppConfig, logger, redis),

		events:  observe.NewEventCollector(logger, observe.SessionMeta{}),
		metrics: observe.NewMetricCollector(logger, observe.SessionMeta{}),
		contextID:        uuid.NewString(),
		interactionState: Unknown,
		msgMode:          type_enums.TextMode,
		assistantExecutor: internal_agent_executor_llm.NewAssistantExecutor(logger),

		//
		histories: make([]internal_type.MessagePacket, 0),
		metadata:  make(map[string]interface{}),
		args:      make(map[string]interface{}),
		options:   make(map[string]interface{}),

		// dispatcher channels
		criticalCh: make(chan packetEnvelope, 16),
		normalCh:   make(chan packetEnvelope, 256),
		lowCh:      make(chan packetEnvelope, 512),
	}
}

// GetSource implements internal_adapter_requests.Messaging.
func (dm *genericRequestor) Source() utils.RapidaSource {
	return dm.source
}

func (deb *genericRequestor) onCreateMessage(ctx context.Context, msg internal_type.MessagePacket) error {
	deb.histories = append(deb.histories, msg)
	dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	defer cancel()
	_, err := deb.conversationService.CreateConversationMessage(dbCtx, deb.Auth(), deb.Source(), deb.Assistant().Id, deb.Assistant().AssistantProviderId, deb.Conversation().Id, msg.ContextId(), msg.Role(), msg.Content())
	if err != nil {
		deb.logger.Error("unable to create message for the user")
		return err
	}
	return nil
}

func (gr *genericRequestor) GetAssistantConversation(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) (*internal_conversation_entity.AssistantConversation, error) {
	return gr.conversationService.GetConversation(ctx, auth, assistantId, assistantConversationId, &internal_services.GetConversationOption{
		InjectContext:  true,
		InjectArgument: true,
		InjectMetadata: true,
		InjectOption:   true,
		InjectMetric:   false},
	)
}

func (r *genericRequestor) identifier(config *protos.ConversationInitialization) string {
	switch identity := config.GetUserIdentity().(type) {
	case *protos.ConversationInitialization_Phone:
		return identity.Phone.GetPhoneNumber()
	case *protos.ConversationInitialization_Web:
		return identity.Web.GetUserId()
	default:
		return uuid.NewString()
	}
}

func (talking *genericRequestor) BeginConversation(ctx context.Context, assistant *internal_assistant_entity.Assistant, direction type_enums.ConversationDirection, config *protos.ConversationInitialization) (*internal_conversation_entity.AssistantConversation, error) {
	talking.assistant = assistant
	conversation, err := talking.conversationService.CreateConversation(ctx, talking.Auth(), talking.identifier(config), assistant.Id, assistant.AssistantProviderId, direction, talking.Source())
	if err != nil {
		return conversation, err
	}

	if arguments, err := utils.AnyMapToInterfaceMap(config.GetArgs()); err == nil {
		talking.args = arguments
		utils.Go(ctx, func() {
			talking.conversationService.ApplyConversationArgument(ctx, talking.Auth(), assistant.Id, conversation.Id, arguments)
		})
	}
	if options, err := utils.AnyMapToInterfaceMap(config.GetOptions()); err == nil {
		talking.options = options
		utils.Go(ctx, func() {
			talking.conversationService.ApplyConversationOption(ctx, talking.Auth(), assistant.Id, conversation.Id, options)
		})
	}
	if metadata, err := utils.AnyMapToInterfaceMap(config.GetMetadata()); err == nil {
		talking.metadata = metadata
		utils.Go(ctx, func() {
			talking.conversationService.ApplyConversationMetadata(ctx, talking.Auth(), assistant.Id, conversation.Id, types.NewMetadataList(metadata))
		})
	}
	talking.assistantConversation = conversation
	return conversation, err
}

func (talking *genericRequestor) ResumeConversation(ctx context.Context, assistant *internal_assistant_entity.Assistant, config *protos.ConversationInitialization) (*internal_conversation_entity.AssistantConversation, error) {
	talking.assistant = assistant
	conversation, err := talking.GetAssistantConversation(ctx, talking.Auth(), assistant.Id, config.GetAssistantConversationId())
	if err != nil {
		talking.logger.Errorf("failed to get assistant conversation: %+v", err)
		return nil, err
	}
	if conversation == nil {
		talking.logger.Errorf("conversation not found: %d", config.GetAssistantConversationId())
		return nil, fmt.Errorf("conversation not found: %d", config.GetAssistantConversationId())
	}
	talking.assistantConversation = conversation
	talking.args = conversation.GetArguments()
	talking.options = conversation.GetOptions()
	talking.metadata = conversation.GetMetadatas()
	return conversation, nil
}

func (talking *genericRequestor) IntegrationCaller() integration_client.IntegrationServiceClient {
	return talking.integrationClient

}

func (talking *genericRequestor) VaultCaller() web_client.VaultClient {
	return talking.vaultClient
}

func (talking *genericRequestor) DeploymentCaller() endpoint_client.DeploymentServiceClient {
	return talking.deploymentClient
}

func (talking *genericRequestor) GetKnowledge(ctx context.Context, knowledgeId uint64) (*internal_knowledge_gorm.Knowledge, error) {
	return talking.knowledgeService.Get(ctx, talking.auth, knowledgeId)
}

func (gr *genericRequestor) GetArgs() map[string]interface{} {
	return gr.args
}

func (gr *genericRequestor) GetOptions() utils.Option {
	return gr.options
}

func (dm *genericRequestor) GetHistories() []internal_type.MessagePacket {
	return dm.histories
}

func (gr *genericRequestor) CreateConversationRecording(ctx context.Context, user, assistant []byte) error {
	dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	defer cancel()
	if _, err := gr.conversationService.CreateConversationRecording(dbCtx, gr.auth, gr.assistant.Id, gr.assistantConversation.Id, user, assistant); err != nil {
		gr.logger.Errorf("unable to create recording for the conversation id %d with error : %v", err)
		return err
	}
	return nil
}

// =============================================================================
// Interaction state methods — inline replacement for the former Messaging wrapper
// =============================================================================

// GetID returns the current interaction context UUID.
// Rotates to a new UUID each time an Interrupted transition fires.
func (r *genericRequestor) GetID() string {
	r.msgMu.RLock()
	defer r.msgMu.RUnlock()
	return r.contextID
}

// GetMode returns the current stream mode (text or audio).
func (r *genericRequestor) GetMode() type_enums.MessageMode {
	return r.msgMode
}

// SwitchMode sets the stream mode.
func (r *genericRequestor) SwitchMode(mm type_enums.MessageMode) {
	r.msgMu.Lock()
	defer r.msgMu.Unlock()
	r.msgMode = mm
}

// Transition advances the interaction state machine.
func (r *genericRequestor) Transition(newState InteractionState) error {
	r.msgMu.Lock()
	defer r.msgMu.Unlock()
	switch newState {
	case Unknown:
		return fmt.Errorf("Transition: invalid transition: cannot transition to Unknown state")
	case Interrupt:
		if r.interactionState == Interrupted || r.interactionState == Interrupt {
			return fmt.Errorf("Transition: invalid transition: agent can't interrupt multiple times")
		}
	case Interrupted:
		if r.interactionState == Interrupted {
			return fmt.Errorf("Transition: invalid transition: agent can't interrupted multiple times")
		}
		r.contextID = uuid.NewString()
	}
	r.interactionState = newState
	return nil
}

// loadTelemetryProviders fetches enabled telemetry provider configurations
// (with their options) for the current assistant from the database.
func (r *genericRequestor) loadTelemetryProviders(ctx context.Context) ([]*internal_telemetry_entity.AssistantTelemetryProvider, error) {
	var providers []*internal_telemetry_entity.AssistantTelemetryProvider
	err := r.postgres.DB(ctx).
		Preload("Options").
		Where("assistant_id = ? AND enabled = true", r.assistant.Id).
		Find(&providers).Error
	return providers, err
}

// initializeCollectors builds EventCollector and MetricCollector from the
// assistant's telemetry provider configuration stored in the database.
// Connection details come from the provider's Options key-value pairs.
// Collectors default to no-op when no providers are configured.
func (r *genericRequestor) initializeCollectors(ctx context.Context) {
	providers, err := r.loadTelemetryProviders(ctx)
	if err != nil {
		r.logger.Errorf("observe: failed to load telemetry providers: %v", err)
	}

	var projectID, orgID uint64
	if pid := r.auth.GetCurrentProjectId(); pid != nil {
		projectID = *pid
	}
	if oid := r.auth.GetCurrentOrganizationId(); oid != nil {
		orgID = *oid
	}

	meta := observe.SessionMeta{
		AssistantID:             r.assistant.Id,
		AssistantConversationID: r.assistantConversation.Id,
		ProjectID:               projectID,
		OrganizationID:          orgID,
	}

	var eventExporters []observe.EventExporter
	var metricExporters []observe.MetricExporter

	// OpenSearch is the platform's own telemetry store — register only when reachable.
	if r.opensearch != nil && r.opensearch.IsConnected(ctx) {
		exp := observe_exporters.NewOpenSearchExporter(r.logger, &r.config.AppConfig, r.opensearch)
		eventExporters = append(eventExporters, exp)
		metricExporters = append(metricExporters, exp)
	}

	for _, p := range providers {
		opts := p.GetOptions()
		switch p.ProviderType {
		case "otlp_http", "otlp_grpc":
			cfg := observe_exporters.OTLPConfigFromOptions(opts, p.ProviderType)
			if cfg.Endpoint == "" {
				r.logger.Errorf("observe: OTLP provider %d has no endpoint in options", p.Id)
				continue
			}
			exp, err := observe_exporters.NewOTLPExporter(ctx, cfg)
			if err != nil {
				r.logger.Errorf("observe: OTLP exporter creation failed for provider %d: %v", p.Id, err)
				continue
			}
			eventExporters = append(eventExporters, exp)
			metricExporters = append(metricExporters, exp)

		case "xray":
			exp, err := observe_exporters.NewXRayExporter(ctx, opts)
			if err != nil {
				r.logger.Errorf("observe: X-Ray exporter creation failed for provider %d: %v", p.Id, err)
				continue
			}
			eventExporters = append(eventExporters, exp)
			metricExporters = append(metricExporters, exp)

		case "google_trace":
			exp, err := observe_exporters.NewGoogleTraceExporter(ctx, opts)
			if err != nil {
				r.logger.Errorf("observe: Google Cloud Trace exporter creation failed for provider %d: %v", p.Id, err)
				continue
			}
			eventExporters = append(eventExporters, exp)
			metricExporters = append(metricExporters, exp)

		case "azure_monitor":
			exp, err := observe_exporters.NewAzureMonitorExporter(ctx, opts)
			if err != nil {
				r.logger.Errorf("observe: Azure Monitor exporter creation failed for provider %d: %v", p.Id, err)
				continue
			}
			eventExporters = append(eventExporters, exp)
			metricExporters = append(metricExporters, exp)

		case "datadog":
			exp, err := observe_exporters.NewDatadogExporter(ctx, opts)
			if err != nil {
				r.logger.Errorf("observe: Datadog exporter creation failed for provider %d: %v", p.Id, err)
				continue
			}
			eventExporters = append(eventExporters, exp)
			metricExporters = append(metricExporters, exp)

		case "logging":
			exp := observe_exporters.NewLoggingExporter(r.logger)
			eventExporters = append(eventExporters, exp)
			metricExporters = append(metricExporters, exp)
		}
	}

	r.events = observe.NewEventCollector(r.logger, meta, eventExporters...)
	r.metrics = observe.NewMetricCollector(r.logger, meta, metricExporters...)
}

// shutdownCollectors waits for in-flight exports and shuts down all exporters.
// Uses a background context so shutdown completes even if the session context
// is already cancelled at disconnect time.
func (r *genericRequestor) shutdownCollectors(_ context.Context) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.events.Shutdown(shutdownCtx)
	r.metrics.Shutdown(shutdownCtx)
}
