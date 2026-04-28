// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool

import (
	"context"
	"encoding/json"
	"fmt"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"

	"github.com/rapidaai/protos"
)

type ToolCallResult map[string]string

func Result(msg string, success bool) ToolCallResult {
	if success {
		return map[string]string{"data": msg, "status": "SUCCESS"}
	} else {
		return map[string]string{"error": msg, "status": "FAIL"}
	}
}

// JustResult converts an arbitrary map to ToolCallResult by serializing
// non-string values to their JSON representation. This keeps callers that
// pass map[string]interface{} (e.g. from API responses) working while
// producing the map[string]string that LLMToolResultPacket now requires.
func JustResult(data map[string]interface{}) ToolCallResult {
	out := make(ToolCallResult, len(data))
	for k, v := range data {
		switch val := v.(type) {
		case string:
			out[k] = val
		default:
			b, err := json.Marshal(val)
			if err != nil {
				out[k] = fmt.Sprintf("%v", val)
			} else {
				out[k] = string(b)
			}
		}
	}
	return out
}

// ErrorResult creates an error result map
func ErrorResult(errorMsg string) ToolCallResult {
	return ToolCallResult{
		"status": "FAIL",
		"error":  errorMsg,
	}
}

func (rt ToolCallResult) Result() string {
	bytes, err := json.Marshal(rt)
	if err != nil {
		return `{"error":"failed to marshal result","success":false,"status":"FAIL"}`
	}

	return string(bytes)
}

// StringifyArgs converts map[string]interface{} to map[string]string for
// LLMToolCallPacket.Arguments. Non-string values are formatted with %v.
func StringifyArgs(args map[string]interface{}) map[string]string {
	if args == nil {
		return nil
	}
	out := make(map[string]string, len(args))
	for k, v := range args {
		switch val := v.(type) {
		case string:
			out[k] = val
		default:
			out[k] = fmt.Sprintf("%v", val)
		}
	}
	return out
}

// ToolCaller defines the contract for invoking a tool/function.
// Call executes the tool — the tool pushes its result (ToolResultPacket)
// or LLMToolCallPacket with Action via communication.OnPacket.
type ToolCaller interface {
	Id() uint64
	Name() string
	Definition() (*protos.FunctionDefinition, error)
	ExecutionMethod() string
	Call(ctx context.Context, contextID string, toolId string, args map[string]interface{}, communication internal_type.Communication)
}
