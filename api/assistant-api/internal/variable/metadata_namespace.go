// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import "strings"

// MetadataNamespace exposes the full conversation metadata map.
type MetadataNamespace struct{}

func (n *MetadataNamespace) Get(suffix string, src Source, _ ResolveContext) (any, bool) {
	v, ok := src.Metadata()[suffix]
	return v, ok
}

func (n *MetadataNamespace) Enumerate(src Source, _ ResolveContext) map[string]any {
	in := src.Metadata()
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// MetadataPrefixNamespace projects metadata keys that share a prefix as a
// dedicated namespace. Used for "client." -> client.* and "analysis." ->
// analysis.* views over the metadata map.
type MetadataPrefixNamespace struct {
	Prefix string
}

func (n *MetadataPrefixNamespace) Get(suffix string, src Source, _ ResolveContext) (any, bool) {
	v, ok := src.Metadata()[n.Prefix+suffix]
	return v, ok
}

func (n *MetadataPrefixNamespace) Enumerate(src Source, _ ResolveContext) map[string]any {
	out := map[string]any{}
	for k, v := range src.Metadata() {
		if rest, ok := strings.CutPrefix(k, n.Prefix); ok {
			out[rest] = v
		}
	}
	return out
}
