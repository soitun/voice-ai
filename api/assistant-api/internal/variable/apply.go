// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import "strings"

const customPrefix = "custom."

// Apply walks a flat mapping (sourceKey -> destinationKey) and resolves
// each source key against the registry. Used by tool callers and observe
// webhook payloads.
//
// The "custom." prefix is special-cased: it writes the mapping VALUE under
// the suffix as the destination key. This preserves legacy tool semantics
// where custom.foo -> "bar" produces out["foo"] = "bar".
func (r *Registry) Apply(mapping map[string]string, src Source, ctx ResolveContext) map[string]any {
	out := make(map[string]any, len(mapping))
	for srcKey, dest := range mapping {
		if suffix, ok := strings.CutPrefix(srcKey, customPrefix); ok {
			out[suffix] = dest
			continue
		}
		ns, suffix, ok := r.resolve(srcKey)
		if !ok {
			continue
		}
		v, ok := ns.Get(suffix, src, ctx)
		if !ok {
			continue
		}
		out[dest] = v
	}
	return out
}
