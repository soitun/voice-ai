// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

// ResolveContext carries per-call extras that some namespaces need.
// Source carries durable state; ResolveContext carries call-time bindings
// (tool name, raw tool args, observe event).
type ResolveContext struct {
	ToolName string
	ToolArgs map[string]any
	Event    string
}

// Namespace is one prefix's worth of data. Get is used by Apply (flat lookup);
// Enumerate is used by Expand (build sub-tree).
type Namespace interface {
	Get(suffix string, src Source, ctx ResolveContext) (any, bool)
	Enumerate(src Source, ctx ResolveContext) map[string]any
}
