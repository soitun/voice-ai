// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor_tool

import (
	"encoding/json"
	"fmt"
	"strings"
)

type toolConditionEntry struct {
	Key       string `json:"key"`
	Condition string `json:"condition"`
	Value     string `json:"value"`
}

type toolConditionMatcher struct{}

var allowedConditionSources = map[string]struct{}{
	"all":        {},
	"sdk":        {},
	"web_plugin": {},
	"debugger":   {},
	"phone":      {},
}

func newToolConditionMatcher() *toolConditionMatcher {
	return &toolConditionMatcher{}
}

func (m *toolConditionMatcher) Evaluate(raw string, source string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true, nil
	}

	var entries []toolConditionEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return false, fmt.Errorf("failed to parse condition JSON: %w", err)
	}

	if len(entries) == 0 {
		return false, fmt.Errorf("condition must include at least one entry")
	}

	normalizedSource := m.normalizeSourceToken(source)

	for _, entry := range entries {
		if err := m.validateEntry(entry); err != nil {
			return false, err
		}
		expected := m.normalizeSourceToken(entry.Value)
		if expected == "all" {
			continue
		}
		if expected != normalizedSource {
			return false, nil
		}
	}
	return true, nil
}

func (m *toolConditionMatcher) validateEntry(entry toolConditionEntry) error {
	key := strings.TrimSpace(entry.Key)
	if key != "source" {
		return fmt.Errorf("unsupported condition key: %s", key)
	}

	condition := strings.TrimSpace(entry.Condition)
	if condition != "=" {
		return fmt.Errorf("unsupported condition operator for source: %s", condition)
	}

	normalizedValue := m.normalizeSourceToken(entry.Value)
	if _, ok := allowedConditionSources[normalizedValue]; !ok {
		return fmt.Errorf("unsupported condition source value: %s", entry.Value)
	}

	return nil
}

func (m *toolConditionMatcher) normalizeSourceToken(raw string) string {
	v := strings.TrimSpace(strings.ToLower(raw))
	v = strings.ReplaceAll(v, "-", "_")

	switch v {
	case "webplugin":
		return "web_plugin"
	case "web_plugin":
		return "web_plugin"
	case "phonecall":
		return "phone"
	case "phone_call":
		return "phone"
	case "phone":
		return "phone"
	default:
		return v
	}
}
