// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import "fmt"

// AssistantNamespace exposes the active assistant: id, version, name,
// language, description.
type AssistantNamespace struct{}

func (n *AssistantNamespace) Get(suffix string, src Source, _ ResolveContext) (any, bool) {
	v, ok := n.fields(src.Assistant())[suffix]
	return v, ok
}

func (n *AssistantNamespace) Enumerate(src Source, _ ResolveContext) map[string]any {
	return n.fields(src.Assistant())
}

func (n *AssistantNamespace) fields(a *AssistantInfo) map[string]any {
	if a == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":          fmt.Sprintf("%d", a.ID),
		"version":     fmt.Sprintf("vrsn_%d", a.VersionID),
		"name":        a.Name,
		"language":    a.Language,
		"description": a.Description,
	}
}
