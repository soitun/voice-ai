package internal_cohere_callers

import (
	"context"
	"fmt"
	"strings"
	"time"

	cohere "github.com/cohere-ai/cohere-go/v2"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	Cohere
}

func NewLargeLanguageCaller(logger commons.Logger, credential *protos.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		Cohere: NewCohere(logger, credential),
	}
}

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
		llc.logger.Errorf("chat completion unable to get client for cohere: %v", err)
		onError(options.Request.GetRequestId(), err)
		options.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		return err
	}

	chatRequest := llc.GetChatStreamRequest(options)
	chatRequest.Messages = llc.BuildHistory(allMessages)

	options.PreHook(utils.ToJson(chatRequest))
	llc.logger.Benchmark("Cohere.llm.GetChatCompletion.llmRequestPrepare", time.Since(start))

	resp, err := client.V2.ChatStream(ctx, chatRequest)
	if err != nil {
		llc.logger.Errorf("Failed to get chat completions stream: %v", err)
		options.PostHook(map[string]interface{}{
			"result": utils.ToJson(resp),
			"error":  err,
		}, metrics.Build())
		onError(options.Request.GetRequestId(), err)
		return err
	}

	defer resp.Close()
	contents := make([]string, 0)
	toolCalls := make([]*protos.ToolCall, 0)
	var currentToolCall *protos.ToolCall
	var currentContent string
	hasToolCalls := false // Flag to track if response contains tool calls
	for {
		select {
		case <-ctx.Done():
			llc.logger.Infof("Context canceled during stream processing")
			return ctx.Err()
		default:
			rep, _ := resp.Recv()
			switch {
			case rep.MessageStart != nil:
				continue
			case rep.ContentStart != nil:
				if rep.ContentStart.Delta != nil && rep.ContentStart.Delta.Message != nil && rep.ContentStart.Delta.Message.Content != nil {
					if text := rep.ContentStart.Delta.Message.Content.GetText(); text != nil {
						currentContent = *text
					}
				}
			case rep.ContentDelta != nil:
				if rep.ContentDelta.Delta != nil && rep.ContentDelta.Delta.Message != nil && rep.ContentDelta.Delta.Message.Content != nil {
					if text := rep.ContentDelta.Delta.Message.Content.GetText(); text != nil {
						currentContent += *text

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
										Contents: []string{*text},
									},
								},
							}
							if err := onStream(options.Request.GetRequestId(), tokenMsg); err != nil {
								llc.logger.Warnf("error streaming token: %v", err)
							}
						}
					}
				}
			case rep.ContentEnd != nil:
				if currentContent != "" {
					contents = append(contents, currentContent)
					currentContent = ""
				}
			case rep.ToolCallStart != nil:
				hasToolCalls = true
				if rep.ToolCallStart.Delta != nil && rep.ToolCallStart.Delta.Message != nil && rep.ToolCallStart.Delta.Message.ToolCalls != nil {
					tc := rep.ToolCallStart.Delta.Message.ToolCalls
					var name, args string
					if tc.Function.Name != nil {
						name = *tc.Function.Name
					}
					if tc.Function.Arguments != nil {
						args = *tc.Function.Arguments
					}
					currentToolCall = &protos.ToolCall{
						Id:   tc.Id,
						Type: tc.Type(),
						Function: &protos.FunctionCall{
							Name:      name,
							Arguments: args,
						},
					}
				}
			case rep.ToolCallDelta != nil:
				if currentToolCall != nil && rep.ToolCallDelta.Delta != nil && rep.ToolCallDelta.Delta.Message != nil && rep.ToolCallDelta.Delta.Message.ToolCalls != nil {
					if rep.ToolCallDelta.Delta.Message.ToolCalls.Function.Arguments != nil {
						currentToolCall.Function.Arguments += *rep.ToolCallDelta.Delta.Message.ToolCalls.Function.Arguments
					}
				}
			case rep.ToolCallEnd != nil:
				toolCalls = append(toolCalls, currentToolCall)
				currentToolCall = nil

			case rep.MessageEnd != nil:
				metrics.OnAddMetrics(llc.UsageMetrics(rep.MessageEnd.Delta.Usage)...)
				protoMsg := &protos.Message{
					Role: "assistant",
					Message: &protos.Message_Assistant{
						Assistant: &protos.AssistantMessage{
							Contents:  contents,
							ToolCalls: toolCalls,
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
					"result": protoMsg,
				}, metrics.Build())
				return nil
			}
		}
	}
}

func (llc *largeLanguageCaller) BuildHistory(allMessages []*protos.Message) cohere.ChatMessages {
	msgHistory := make([]*cohere.ChatMessageV2, 0)
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case "chatbot":
		case "assistant":
			if assistant := cntn.GetAssistant(); assistant != nil {
				_msg := &cohere.ChatMessageV2{
					Role:      "assistant",
					Assistant: &cohere.AssistantMessage{},
				}

				if len(assistant.GetContents()) > 0 {
					_msg.Assistant.Content = &cohere.AssistantMessageV2Content{
						String: strings.Join(assistant.GetContents(), ""),
					}
				}
				if len(assistant.GetToolCalls()) > 0 {
					fctCall := make([]*cohere.ToolCallV2, 0)
					err := utils.Cast(assistant.GetToolCalls(), &fctCall)
					if err != nil {
						llc.logger.Errorf("unable to cast the to function tool call %v", err)
					}
					_msg.Assistant.ToolCalls = fctCall
				}
				msgHistory = append(msgHistory, _msg)
			}
		case "system":
			if system := cntn.GetSystem(); system != nil {
				msgHistory = append(msgHistory, &cohere.ChatMessageV2{
					Role: "system",
					System: &cohere.SystemMessageV2{
						Content: &cohere.SystemMessageV2Content{
							String: system.GetContent(),
						},
					},
				})
			}
		case "user":
			if user := cntn.GetUser(); user != nil {
				msgHistory = append(msgHistory, &cohere.ChatMessageV2{
					Role: "user",
					User: &cohere.UserMessageV2{Content: &cohere.UserMessageV2Content{
						String: user.GetContent(),
					}},
				})
			}
		case "tool":
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					msgHistory = append(msgHistory,
						&cohere.ChatMessageV2{
							Role: "tool",
							Tool: &cohere.ToolMessageV2{
								ToolCallId: t.GetId(),
								Content: &cohere.ToolMessageV2Content{
									String: t.GetContent(),
								}},
						})
				}
			}
		default:
			llc.logger.Warnf("Unknown role: %s and everytihgn", cntn.String())
			continue
		}
	}
	return msgHistory
}
func (llc *largeLanguageCaller) GetChatRequest(opts *internal_callers.ChatCompletionOptions) *cohere.V2ChatRequest {
	options := &cohere.V2ChatRequest{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = mn
			}
		case "model.max_tokens":
			if mt, err := utils.AnyToInt(value); err == nil {
				options.MaxTokens = utils.Ptr(mt)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.P = utils.Ptr(topP)
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat64(value); err == nil {
				options.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat64(value); err == nil {
				options.PresencePenalty = utils.Ptr(pp)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.StopSequences = strings.Split(stopStr, ",")
			}
		}
	}
	return options
}
func (llc *largeLanguageCaller) GetChatStreamRequest(opts *internal_callers.ChatCompletionOptions) *cohere.V2ChatStreamRequest {
	options := &cohere.V2ChatStreamRequest{}
	if len(opts.ToolDefinitions) > 0 {
		options.Tools = make([]*cohere.ToolV2, len(opts.ToolDefinitions))
		for idx, tl := range opts.ToolDefinitions {
			fn := &cohere.ToolV2Function{
				Name: tl.Function.Name,
			}
			if tl.Function.Parameters != nil {
				fn.Parameters = tl.Function.Parameters.ToMap()
			}
			if tl.Function.Description != "" {
				fn.Description = &tl.Function.Description
			}
			options.Tools[idx] = &cohere.ToolV2{
				Function: fn,
			}
		}
	}

	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = mn
			}
		case "model.max_tokens":
			if mt, err := utils.AnyToInt(value); err == nil {
				options.MaxTokens = utils.Ptr(mt)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.P = utils.Ptr(topP)
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat64(value); err == nil {
				options.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat64(value); err == nil {
				options.PresencePenalty = utils.Ptr(pp)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.StopSequences = strings.Split(stopStr, ",")
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				switch format["type"].(string) {
				case "text":
					options.ResponseFormat = &cohere.ResponseFormatV2{
						Type: "text",
					}
				case "json_object":
					if schemaData, ok := format["json_schema"].(map[string]interface{}); ok {
						options.ResponseFormat = &cohere.ResponseFormatV2{
							Type: "json_object",
							JsonObject: &cohere.JsonResponseFormatV2{
								JsonSchema: schemaData,
							},
						}
					}
				}
			}
		}
	}
	return options
}

func (llc *largeLanguageCaller) GetChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	llc.logger.Debugf("chat completion for cohere")
	//
	// Working with chat completion with vision
	//
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("chat completion unable to get client for cohere %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	metrics.OnStart()
	// single minute timeout and cancellable by the client as context will get cancel
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	chatRequest := llc.GetChatRequest(options)
	chatRequest.Messages = llc.BuildHistory(allMessages)

	options.PreHook(utils.ToJson(*chatRequest))
	resp, err := client.V2.Chat(
		ctx,
		chatRequest,
	)
	if err != nil {
		llc.logger.Errorf("chat completion unable to get client for cohere %v", err)
		options.PostHook(map[string]interface{}{
			"error":  err,
			"result": resp,
		}, metrics.OnFailure().Build())
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()

	// // call when you are done
	options.PostHook(map[string]interface{}{
		"result": resp,
	}, metrics.Build())

	contents := make([]string, 0)
	toolCalls := make([]*protos.ToolCall, 0)

	for _, msg := range resp.GetMessage().GetContent() {
		contents = append(contents, msg.Text.Text)
	}

	for _, tl := range resp.GetMessage().GetToolCalls() {
		var name, args string
		if n := tl.GetFunction().GetName(); n != nil {
			name = *n
		}
		if a := tl.GetFunction().GetArguments(); a != nil {
			args = *a
		}
		toolCalls = append(toolCalls, &protos.ToolCall{
			Id:   tl.GetId(),
			Type: tl.Type(),
			Function: &protos.FunctionCall{
				Name:      name,
				Arguments: args,
			},
		})
	}

	message := &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents:  contents,
				ToolCalls: toolCalls,
			},
		},
	}

	// Add usage metrics from response
	if resp.Usage != nil {
		metrics.OnAddMetrics(llc.UsageMetrics(resp.Usage)...)
	}

	return message, metrics.Build(), nil
}
