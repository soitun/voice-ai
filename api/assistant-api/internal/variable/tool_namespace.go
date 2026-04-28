// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

// ToolNamespace exposes the active tool's name and arguments. The tool
// executor registers this namespace and passes ToolName/ToolArgs in
// ResolveContext.
type ToolNamespace struct{}

func (n *ToolNamespace) Get(suffix string, _ Source, ctx ResolveContext) (any, bool) {
	switch suffix {
	case "name":
		return ctx.ToolName, true
	case "argument":
		return ctx.ToolArgs, true
	}
	return nil, false
}

func (n *ToolNamespace) Enumerate(_ Source, ctx ResolveContext) map[string]any {
	return map[string]any{
		"name":     ctx.ToolName,
		"argument": ctx.ToolArgs,
	}
}
