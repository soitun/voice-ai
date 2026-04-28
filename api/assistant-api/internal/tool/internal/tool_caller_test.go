// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorResult_StatusFail(t *testing.T) {
	r := ErrorResult("something broke")
	assert.Equal(t, "FAIL", r["status"])
	assert.Equal(t, "something broke", r["error"])
}

func TestErrorResult_ResultJSON(t *testing.T) {
	r := ErrorResult("bad input")
	json := r.Result()
	assert.Contains(t, json, `"status":"FAIL"`)
	assert.Contains(t, json, `"error":"bad input"`)
}

func TestResult_Success(t *testing.T) {
	r := Result("ok", true)
	assert.Equal(t, "SUCCESS", r["status"])
	assert.Equal(t, "ok", r["data"])
}

func TestResult_Failure(t *testing.T) {
	r := Result("not found", false)
	assert.Equal(t, "FAIL", r["status"])
	assert.Equal(t, "not found", r["error"])
}

func TestJustResult_PassesThrough(t *testing.T) {
	data := map[string]interface{}{"key": "value", "count": 42}
	r := JustResult(data)
	assert.Equal(t, "value", r["key"])
	assert.Equal(t, "42", r["count"])
}

func TestJustResult_NestedObject(t *testing.T) {
	data := map[string]interface{}{
		"status": "SUCCESS",
		"result": []string{"hello", "world"},
	}
	r := JustResult(data)
	assert.Equal(t, "SUCCESS", r["status"])
	assert.Equal(t, `["hello","world"]`, r["result"])
}

func TestToolCallResult_Result_MarshalError(t *testing.T) {
	// An empty ToolCallResult should still marshal to valid JSON.
	r := ToolCallResult{}
	json := r.Result()
	assert.Equal(t, "{}", json)
}
