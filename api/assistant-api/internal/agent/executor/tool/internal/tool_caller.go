// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool

import (
	"context"
	"encoding/json"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"

	"github.com/rapidaai/protos"
)

type ToolCallResult map[string]interface{}

func Result(msg string, success bool) ToolCallResult {
	if success {
		return map[string]interface{}{"data": msg, "status": "SUCCESS"}
	} else {
		return map[string]interface{}{"error": msg, "status": "FAIL"}
	}
}

func JustResult(data map[string]interface{}) ToolCallResult {
	return ToolCallResult(data)
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

// ToolCaller defines the contract for invoking a tool/function.
// Call executes the tool — the tool pushes its result (ToolResultPacket)
// or directive (DirectivePacket) via communication.OnPacket.
type ToolCaller interface {
	Id() uint64
	Name() string
	Definition() (*protos.FunctionDefinition, error)
	ExecutionMethod() string
	Call(ctx context.Context, contextID string, toolId string, args map[string]interface{}, communication internal_type.Communication)
}
