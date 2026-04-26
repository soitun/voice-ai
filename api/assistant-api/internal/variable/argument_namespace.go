// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

// ArgumentNamespace exposes session arguments. In Apply consumers it is
// accessed via the "argument." prefix; in Expand consumers it is also
// flattened to the top level so prompt templates can use bare {{key}}.
type ArgumentNamespace struct{}

func (n *ArgumentNamespace) Get(suffix string, src Source, _ ResolveContext) (any, bool) {
	v, ok := src.Arguments()[suffix]
	return v, ok
}

func (n *ArgumentNamespace) Enumerate(src Source, _ ResolveContext) map[string]any {
	in := src.Arguments()
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
