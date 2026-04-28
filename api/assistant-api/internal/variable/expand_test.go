// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import (
	"testing"
)

func TestExpand_NestedNamespaces(t *testing.T) {
	src := newFixtureSource()
	r := NewDefaultRegistry()

	out := r.Expand(src, ResolveContext{})

	asst, ok := out["assistant"].(map[string]any)
	if !ok {
		t.Fatalf("assistant subtree missing: %v", out)
	}
	if asst["id"] != "42" {
		t.Errorf("assistant.id = %v", asst["id"])
	}
	if asst["version"] != "vrsn_7" {
		t.Errorf("assistant.version = %v", asst["version"])
	}

	sys, ok := out["system"].(map[string]any)
	if !ok || sys["current_date"] != "2026-04-26" {
		t.Errorf("system subtree wrong: %v", sys)
	}

	conv, ok := out["conversation"].(map[string]any)
	if !ok || conv["identifier"] != "conv-abc" {
		t.Errorf("conversation subtree wrong: %v", conv)
	}

	client, ok := out["client"].(map[string]any)
	if !ok || client["direction"] != "outbound" {
		t.Errorf("client subtree wrong: %v", client)
	}
}

func TestExpand_FlattensArgumentsAtTopLevel(t *testing.T) {
	src := newFixtureSource()
	r := NewDefaultRegistry()
	out := r.Expand(src, ResolveContext{})

	if out["foo"] != "bar" {
		t.Errorf("expected bare {{foo}} = bar, got %v", out["foo"])
	}
	if out["count"] != 3 {
		t.Errorf("expected bare {{count}} = 3, got %v", out["count"])
	}
}

func TestExpand_AllDefaultNamespacesPresent(t *testing.T) {
	src := newFixtureSource()
	r := NewDefaultRegistry()
	out := r.Expand(src, ResolveContext{})

	expected := []string{
		"system", "assistant", "conversation", "session",
		"argument", "metadata", "option", "client", "analysis",
	}
	for _, k := range expected {
		if _, ok := out[k]; !ok {
			t.Errorf("Expand missing namespace %q", k)
		}
	}
}

func TestApply_Expand_ParityOnSharedKeys(t *testing.T) {
	src := newFixtureSource()
	r := NewDefaultRegistry()

	mapping := map[string]string{
		"assistant.id":        "assistant_id",
		"assistant.name":      "assistant_name",
		"conversation.id":     "conversation_id",
		"system.current_date": "current_date",
		"client.direction":    "client_direction",
	}
	flat := r.Apply(mapping, src, ResolveContext{})
	nested := r.Expand(src, ResolveContext{})

	asst := nested["assistant"].(map[string]any)
	conv := nested["conversation"].(map[string]any)
	sys := nested["system"].(map[string]any)
	client := nested["client"].(map[string]any)

	if flat["assistant_id"] != asst["id"] {
		t.Errorf("assistant.id mismatch: Apply=%v Expand=%v", flat["assistant_id"], asst["id"])
	}
	if flat["assistant_name"] != asst["name"] {
		t.Errorf("assistant.name mismatch: Apply=%v Expand=%v", flat["assistant_name"], asst["name"])
	}
	if flat["conversation_id"] != conv["id"] {
		t.Errorf("conversation.id mismatch: Apply=%v Expand=%v", flat["conversation_id"], conv["id"])
	}
	if flat["current_date"] != sys["current_date"] {
		t.Errorf("system.current_date mismatch: Apply=%v Expand=%v", flat["current_date"], sys["current_date"])
	}
	if flat["client_direction"] != client["direction"] {
		t.Errorf("client.direction mismatch: Apply=%v Expand=%v", flat["client_direction"], client["direction"])
	}
}
