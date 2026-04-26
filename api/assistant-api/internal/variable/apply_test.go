// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import (
	"testing"
)

func TestApply_DefaultNamespaces(t *testing.T) {
	src := newFixtureSource()
	r := NewDefaultRegistry()

	mapping := map[string]string{
		"assistant.id":            "agent_id",
		"conversation.id":         "conv_id",
		"argument.foo":            "foo_out",
		"system.current_date":     "today",
		"client.direction":        "dir",
		"option.max_tokens":       "max",
		"unknown.namespace":       "ignored",
		"system.does_not_exist":   "skipped",
	}

	out := r.Apply(mapping, src, ResolveContext{})

	want := map[string]any{
		"agent_id": "42",
		"conv_id":  "100",
		"foo_out":  "bar",
		"today":    "2026-04-26",
		"dir":      "outbound",
		"max":      "1024",
	}
	if len(out) != len(want) {
		t.Errorf("Apply size = %d, want %d (%v)", len(out), len(want), out)
	}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("Apply[%s] = %v, want %v", k, out[k], v)
		}
	}
	if _, ok := out["ignored"]; ok {
		t.Errorf("unknown namespace should not write output")
	}
	if _, ok := out["skipped"]; ok {
		t.Errorf("missing suffix should not write output")
	}
}

func TestApply_CustomPrefix_WritesMappingValueUnderSuffix(t *testing.T) {
	src := newFixtureSource()
	r := NewDefaultRegistry()

	mapping := map[string]string{
		"custom.literal_key": "literal_value",
	}

	out := r.Apply(mapping, src, ResolveContext{})
	if out["literal_key"] != "literal_value" {
		t.Errorf("custom.* should write mapping value under suffix; got %v", out)
	}
	if _, ok := out["literal_value"]; ok {
		t.Errorf("custom.* should not write under destination key")
	}
}

func TestApply_ToolNamespace_RegisteredByCaller(t *testing.T) {
	src := newFixtureSource()
	r := NewDefaultRegistry().With("tool", &ToolNamespace{})

	mapping := map[string]string{
		"tool.name":     "tool_name",
		"tool.argument": "tool_arg",
	}
	ctx := ResolveContext{ToolName: "search", ToolArgs: map[string]any{"q": "hi"}}

	out := r.Apply(mapping, src, ctx)
	if out["tool_name"] != "search" {
		t.Errorf("tool_name = %v", out["tool_name"])
	}
	gotArg, ok := out["tool_arg"].(map[string]any)
	if !ok || gotArg["q"] != "hi" {
		t.Errorf("tool_arg = %v", out["tool_arg"])
	}
}

func TestApply_EmptyMapping(t *testing.T) {
	r := NewDefaultRegistry()
	out := r.Apply(nil, newFixtureSource(), ResolveContext{})
	if len(out) != 0 {
		t.Errorf("nil mapping should produce empty output, got %v", out)
	}
}

func TestApply_KeyWithoutDot(t *testing.T) {
	r := NewDefaultRegistry()
	out := r.Apply(map[string]string{"justaword": "dest"}, newFixtureSource(), ResolveContext{})
	if len(out) != 0 {
		t.Errorf("dotless key should be skipped, got %v", out)
	}
}
