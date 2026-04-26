// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

// Expand builds a nested map of every registered namespace's enumeration.
// Used by the agent prompt builder so prompt templates can reference
// {{system.current_date}}, {{assistant.name}}, etc.
//
// Arguments are additionally flattened at the top level (without the
// "argument." wrapper) so existing prompt templates that use bare {{key}}
// for an argument continue to resolve.
func (r *Registry) Expand(src Source, ctx ResolveContext) map[string]any {
	out := make(map[string]any, len(r.namespaces))
	r.each(func(prefix string, ns Namespace) {
		out[prefix] = ns.Enumerate(src, ctx)
	})
	for k, v := range src.Arguments() {
		out[k] = v
	}
	return out
}
