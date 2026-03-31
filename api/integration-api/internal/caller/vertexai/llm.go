// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_callers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	VertexAi
}

func NewLargeLanguageCaller(logger commons.Logger, credential *protos.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		VertexAi: vertexai(logger, credential),
	}
}

// StreamChatCompletion streams the chat response piece-by-piece from the AI model.
// Parameters:
// - ctx: Context for managing request lifecycle.
// - allMessages: List of all previous messages exchanged in the chat.
// - options: Configuration options for the chat completion request.
// - onStream: Callback function to handle streaming chunks of type `*protos.Message`.
// - onMetrics: Callback to collect metrics for processing completion.
// - onError: Callback to handle errors encountered during processing.
// Returns:
// - error: Any errors encountered during processing.

func (llc *largeLanguageCaller) StreamChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
	onStream func(string, *protos.Message) error,
	onMetrics func(string, *protos.Message, []*protos.Metric) error,
	onError func(string, error),
) error {
	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	var firstTokenTime *time.Time
	client, err := llc.GetClient()
	if err != nil {
		options.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		onError(options.Request.GetRequestId(), err)
		return err
	}

	// Setting up timeout for streaming
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	instruction, history, current := llc.buildHistory(allMessages)
	model, config := llc.getGenerationConfig(options)
	config.SystemInstruction = instruction
	chat, err := client.Chats.Create(ctx,
		model,
		config,
		history,
	)
	if err != nil {
		options.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		onError(options.Request.GetRequestId(), err)
		return err
	}

	options.PreHook(llc.toSimplifiedJson(model, config, history, current))
	contents := make([]string, 0)
	toolCalls := make([]*protos.ToolCall, 0)
	contentBuilders := make([]strings.Builder, 0)
	hasToolCalls := false // Flag to track if response contains tool calls
	accumlator := &GoogleChatCompletionAccumulator{}
	for resp, err := range chat.SendMessageStream(ctx, current) {
		if err != nil {
			options.PostHook(map[string]interface{}{
				"result": utils.ToJson(resp),
				"error":  err,
			}, metrics.OnFailure().Build())
			onError(options.Request.GetRequestId(), err)
			return err
		}
		accumlator.AddChunk(resp)
		for _, choice := range resp.Candidates {
			if choice.Content != nil {
				for _, part := range choice.Content.Parts {
					if part.FunctionCall != nil {
						hasToolCalls = true
						for len(toolCalls) <= int(choice.Index) {
							toolCalls = append(toolCalls, nil)
						}
						argsJSON, err := json.Marshal(part.FunctionCall.Args)
						if err != nil {
							llc.logger.Errorf("Error marshaling function args: %v", err)
							argsJSON = []byte("{}")
						}
						toolCalls[int(choice.Index)] = &protos.ToolCall{
							Id:   part.FunctionCall.ID,
							Type: "function",
							Function: &protos.FunctionCall{
								Name:      part.FunctionCall.Name,
								Arguments: string(argsJSON),
							},
						}
					}
					if part.Text != "" {
						for len(contentBuilders) <= int(choice.Index) {
							contentBuilders = append(contentBuilders, strings.Builder{})
						}
						contentBuilders[int(choice.Index)].WriteString(part.Text)

						// Stream in real-time when no tool calls
						if !hasToolCalls {
							if firstTokenTime == nil {
								now := time.Now()
								firstTokenTime = &now
							}
							tokenMsg := &protos.Message{
								Role: "assistant",
								Message: &protos.Message_Assistant{
									Assistant: &protos.AssistantMessage{
										Contents: []string{part.Text},
									},
								},
							}
							if err := onStream(options.Request.GetRequestId(), tokenMsg); err != nil {
								llc.logger.Warnf("error streaming token: %v", err)
							}
						}
					}
				}
			}
		}
	}

	// Build contents from builders
	for _, builder := range contentBuilders {
		contents = append(contents, builder.String())
	}

	// Filter nil tool calls
	filteredToolCalls := make([]*protos.ToolCall, 0)
	for _, tc := range toolCalls {
		if tc != nil {
			filteredToolCalls = append(filteredToolCalls, tc)
		}
	}

	metrics.OnAddMetrics(llc.UsageMetrics(accumlator.UsageMetadata)...)
	protoMsg := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents:  contents,
				ToolCalls: filteredToolCalls,
			},
		},
	}

	// Add first token time metric if tokens were streamed
	if firstTokenTime != nil {
		metrics.OnAddMetrics(&protos.Metric{
			Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
			Value:       fmt.Sprintf("%d", firstTokenTime.Sub(start)),
			Description: "Time to receive first token from LLM",
		})
	}
	// Update time taken and status
	metrics.OnSuccess()
	// Send metrics with complete message
	onMetrics(options.Request.GetRequestId(), protoMsg, metrics.Build())

	// Call PostHook after metrics for each message end
	options.PostHook(map[string]interface{}{
		"result": accumlator,
	}, metrics.Build())
	return nil
}

// GetChatCompletion performs a synchronous request for a single chat completion response.
// Parameters:
// - ctx: Context for managing request lifecycle.
// - allMessages: List of all previous messages exchanged in the chat.
// - options: Configuration options for the chat completion request.
// Returns:
// - *protos.Message: Constructed message response.
// - types.Metrics: Metrics collected during execution.
// - error: Any errors encountered during processing.

func (llc *largeLanguageCaller) GetChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("getting error for chat completion %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	if len(allMessages) == 0 {
		err := errors.New("no messages in the input")
		llc.logger.Errorf("invalid input: %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	//
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	instruction, histories, current := llc.buildHistory(allMessages)
	model, config := llc.getGenerationConfig(options)
	config.SystemInstruction = instruction
	chat, err := client.Chats.Create(ctx,
		model,
		config,
		histories)

	if err != nil {
		llc.logger.Errorf("error creating chat: %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	options.PreHook(llc.toSimplifiedJson(model, config, histories, current))
	resp, err := chat.SendMessage(ctx, current)
	if err != nil {
		llc.logger.Errorf("getting error for chat completion %+v %+v", err, resp)
		metrics.OnFailure()
		options.PostHook(map[string]interface{}{"result": resp, "error": err}, metrics.Build())
		return nil, metrics.Build(), err
	}

	contents := make([]string, len(resp.Candidates))
	metrics.OnSuccess()
	for _, choice := range resp.Candidates {
		if choice.Content != nil {
			buf := strings.Builder{}
			if choice.Content != nil {
				for _, part := range choice.Content.Parts {
					_, _ = buf.WriteString(part.Text)
				}
			}
			contents[choice.Index] = buf.String()
		}
	}
	message := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents: contents,
			},
		},
	}

	// Add usage metrics from response
	if resp.UsageMetadata != nil {
		metrics.OnAddMetrics(llc.UsageMetrics(resp.UsageMetadata)...)
	}

	options.PostHook(map[string]interface{}{"result": resp}, metrics.Build())
	return message, metrics.Build(), nil
}

// buildHistory constructs the historical context for the chat session.
// Parameters:
// - allMessages: List of all previous messages exchanged in the chat.
// Returns:
// - *genai.Content: System instructions extracted.
// - []*genai.Content: Conversation history sent as input.
// - genai.Part: Last conversation part extracted.

func (llc *largeLanguageCaller) buildHistory(allMessages []*protos.Message) (*genai.Content, []*genai.Content, genai.Part) {
	history := make([]*genai.Content, 0)
	for _, msg := range allMessages {
		switch msg.GetRole() {
		case "user":
			if u := msg.GetUser(); u != nil {
				history = append(history, &genai.Content{
					Role:  "user",
					Parts: []*genai.Part{&genai.Part{Text: u.GetContent()}},
				})
			}
		case "model", "assistant":
			content := &genai.Content{
				Role:  "model",
				Parts: make([]*genai.Part, 0),
			}
			if assistant := msg.GetAssistant(); assistant != nil {
				// Add text contents
				for _, ct := range assistant.GetContents() {
					content.Parts = append(content.Parts, &genai.Part{Text: ct})
				}
				// Add tool calls
				for _, tc := range assistant.GetToolCalls() {
					var argumentMap map[string]any
					if err := json.Unmarshal([]byte(tc.GetFunction().GetArguments()), &argumentMap); err != nil {
						argumentMap = make(map[string]any)
					}
					content.Parts = append(content.Parts, &genai.Part{
						FunctionCall: &genai.FunctionCall{
							ID:   tc.GetId(),
							Args: argumentMap,
							Name: tc.GetFunction().GetName(),
						},
					})
				}
			}
			history = append(history, content)
		case "system":
			if system := msg.GetSystem(); system != nil {
				sysContent := &genai.Content{
					Parts: []*genai.Part{&genai.Part{Text: system.GetContent()}},
				}
				history = append(history, sysContent)
			}
		case "tool":
			if tool := msg.GetTool(); tool != nil {
				content := &genai.Content{
					Role:  "user",
					Parts: make([]*genai.Part, 0),
				}
				for _, t := range tool.GetTools() {
					var responseMap map[string]any
					if err := json.Unmarshal([]byte(t.GetContent()), &responseMap); err != nil {
						responseMap = make(map[string]any)
					}
					content.Parts = append(content.Parts, &genai.Part{
						FunctionResponse: &genai.FunctionResponse{
							Name:     t.GetName(),
							ID:       t.GetId(),
							Response: responseMap,
						},
					})
				}
				history = append(history, content)
			}
		default:
			llc.logger.Warnf("Unknown role: %s", msg.GetRole())
			continue
		}
	}

	var lastPart genai.Part
	if len(history) > 0 && len(history[len(history)-1].Parts) > 0 {
		lastPart = *history[len(history)-1].Parts[0]
	} else {
		lastPart = genai.Part{} // or some default value
	}

	if len(history) == 0 {
		return nil, history, lastPart
	}
	return history[0], history[1:], lastPart
}

// buildFunctionParameter converts internal function parameter definitions into Vertex AI schemas.
// Parameters:
// - fp: Internal representation of function parameters.
// Returns:
// - *genai.Schema: Converted schema in Vertex AI format.
func (llc *largeLanguageCaller) buildFunctionParameter(fp *internal_callers.FunctionParameter) *genai.Schema {
	schema := &genai.Schema{
		Type:       genai.Type(fp.Type),
		Properties: make(map[string]*genai.Schema),
	}
	if fp.Required != nil {
		schema.Required = fp.Required
	}
	for key, prop := range fp.Properties {
		schema.Properties[key] = llc.buildFunctionProperties(&prop)
	}
	return schema
}

// buildFunctionProperties maps function property details to Vertex AI schema properties.
// Parameters:
// - fpp: Function parameter property details.
// Returns:
// - *genai.Schema: Converted schema in Vertex AI format.
func (llc *largeLanguageCaller) buildFunctionProperties(fpp *internal_callers.FunctionParameterProperty) *genai.Schema {
	schema := &genai.Schema{
		Type:        genai.Type(fpp.Type),
		Description: fpp.Description,
	}
	if fpp.Description != "" {
		schema.Description = fpp.Description
	}
	if fpp.Enum != nil {
		schema.Enum = make([]string, len(fpp.Enum))
		for i, v := range fpp.Enum {
			if v != nil {
				schema.Enum[i] = *v
			}
		}
	}
	if fpp.Items != nil {
		schema.Items = &genai.Schema{
			Type: genai.Type(fpp.Items["type"].(string)),
		}
	}
	return schema
}

// getGenerationConfig converts user-provided options into Vertex AI configuration.
// Parameters:
// - opts: User-defined chat completion options.
// Returns:
// - mdl: Model name used for the request.
// - config: Generated configuration for the request.
func (llc *largeLanguageCaller) getGenerationConfig(
	opts *internal_callers.ChatCompletionOptions,
) (mdl string, config *genai.GenerateContentConfig) {
	config = &genai.GenerateContentConfig{}
	if len(opts.ToolDefinitions) > 0 {
		fd := make([]*genai.FunctionDeclaration, len(opts.ToolDefinitions))
		for idx, tl := range opts.ToolDefinitions {
			switch tl.Type {
			case "function":
				fn := tl.Function
				if fn != nil {
					funcDef := &genai.FunctionDeclaration{
						Name:        fn.Name,
						Description: fn.Description,
					}
					if fn.Parameters != nil {
						funcDef.Parameters = llc.buildFunctionParameter(fn.Parameters)
					}
					fd[idx] = funcDef
				}
			}
		}

		config.Tools = []*genai.Tool{{
			FunctionDeclarations: fd,
		}}
	}

	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				mdl = modelName
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat32(value); err == nil {
				config.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat32(value); err == nil {
				config.TopP = utils.Ptr(topP)
			}
		case "model.top_k":
			if topK, err := utils.AnyToFloat32(value); err == nil {
				config.TopK = utils.Ptr(topK)
			}
		case "model.max_completion_tokens":
			if maxTokens, err := utils.AnyToInt64(value); err == nil {
				config.MaxOutputTokens = int32(maxTokens)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				config.StopSequences = strings.Split(stopStr, ",")
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat32(value); err == nil {
				config.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat32(value); err == nil {
				config.PresencePenalty = utils.Ptr(pp)
			}
		case "model.seed":
			if seed, err := utils.AnyToInt32(value); err == nil {
				config.Seed = utils.Ptr(seed)
			}

		case "model.thinking":
			if format, err := utils.AnyToJSON(value); err == nil {
				config.ThinkingConfig = &genai.ThinkingConfig{}
				if enabled, ok := format["include_thoughts"].(bool); ok && enabled {
					config.ThinkingConfig.IncludeThoughts = enabled

					if budgetTokens, ok := format["thinking_budget"].(int32); ok {
						config.ThinkingConfig.ThinkingBudget = utils.Ptr(int32(budgetTokens))
					}
				}
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				switch format["response_mime_type"].(string) {
				case "text/x.enum":
					if schemaData, ok := format["response_schema"].(map[string]interface{}); ok {
						config.ResponseMIMEType = "text/x.enum"
						config.ResponseJsonSchema = schemaData
					}
				case "application/json":
					if schemaData, ok := format["response_schema"].(map[string]interface{}); ok {
						config.ResponseMIMEType = "application/json"
						config.ResponseJsonSchema = schemaData
					}
				}
			}
		}
	}
	return mdl, config
}

// toSimplifiedJson simplifies the JSON representation of the final request payload.
// Parameters:
// - model: Model name used for the request.
// - cfg: Configuration settings used.
// - history: Historical context provided.
// - ct: Current conversation part.
// Returns:
// - map[string]interface{}: Simplified JSON payload.
func (llc *largeLanguageCaller) toSimplifiedJson(model string, cfg *genai.GenerateContentConfig, history []*genai.Content, ct genai.Part) map[string]interface{} {
	wt := struct {
		Config               *genai.GenerateContentConfig
		Current              genai.Part
		Model                string
		ComprehensiveHistory []*genai.Content
	}{
		Model:                model,
		Config:               cfg,
		Current:              ct,
		ComprehensiveHistory: history,
	}
	return utils.ToJson(wt)
}
