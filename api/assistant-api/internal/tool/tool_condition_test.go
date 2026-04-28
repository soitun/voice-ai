// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool

import "testing"

func TestEvaluateToolCondition_EmptyCondition_Allows(t *testing.T) {
	matcher := newToolConditionMatcher()
	ok, err := matcher.Evaluate("", "phone-call")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected condition to allow tool")
	}
}

func TestEvaluateToolCondition_InvalidJSON_ReturnsError(t *testing.T) {
	matcher := newToolConditionMatcher()
	_, err := matcher.Evaluate("{invalid", "phone-call")
	if err == nil {
		t.Fatalf("expected parse error for invalid JSON")
	}
}

func TestEvaluateToolCondition_SourceMatch_Phone(t *testing.T) {
	matcher := newToolConditionMatcher()
	ok, err := matcher.Evaluate(`[{"key":"source","condition":"=","value":"phone"}]`, "phone-call")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected phone source to match phone-call")
	}
}

func TestEvaluateToolCondition_SourceMismatch_BlocksTool(t *testing.T) {
	matcher := newToolConditionMatcher()
	ok, err := matcher.Evaluate(`[{"key":"source","condition":"=","value":"sdk"}]`, "phone-call")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ok {
		t.Fatalf("expected source mismatch to block tool")
	}
}

func TestEvaluateToolCondition_SourceAll_AlwaysAllows(t *testing.T) {
	matcher := newToolConditionMatcher()
	ok, err := matcher.Evaluate(`[{"key":"source","condition":"=","value":"all"}]`, "debugger")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected source=all to allow tool")
	}
}

func TestEvaluateToolCondition_SourceMatch_WebPlugin(t *testing.T) {
	matcher := newToolConditionMatcher()
	ok, err := matcher.Evaluate(`[{"key":"source","condition":"=","value":"web_plugin"}]`, "web-plugin")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected web_plugin source to match web-plugin runtime source")
	}
}

func TestEvaluateToolCondition_UnsupportedRule_ReturnsError(t *testing.T) {
	matcher := newToolConditionMatcher()
	_, err := matcher.Evaluate(`[{"key":"region","condition":"=","value":"sg"}]`, "sdk")
	if err == nil {
		t.Fatalf("expected unsupported condition key error")
	}
}

func TestEvaluateToolCondition_UnsupportedOperator_ReturnsError(t *testing.T) {
	matcher := newToolConditionMatcher()
	_, err := matcher.Evaluate(`[{"key":"source","condition":"!=","value":"phone"}]`, "phone-call")
	if err == nil {
		t.Fatalf("expected unsupported operator error")
	}
}
