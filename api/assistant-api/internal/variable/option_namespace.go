// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

// OptionNamespace exposes the conversation options map.
type OptionNamespace struct{}

func (n *OptionNamespace) Get(suffix string, src Source, _ ResolveContext) (any, bool) {
	v, ok := src.Options()[suffix]
	return v, ok
}

func (n *OptionNamespace) Enumerate(src Source, _ ResolveContext) map[string]any {
	in := src.Options()
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
