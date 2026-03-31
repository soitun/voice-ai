// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_type

// SpeechToTextTransformer is an interface for transforming input audio data.
// It extends the Transformers interface, specifying that it transforms
// from []byte (raw audio data) to string (processed audio representation).
type SpeechToTextTransformer interface {
	// The `Name() string` method is defining a function signature for a method that returns a string
	// representing the name of the transformer. This method is expected to provide a human-readable
	// identifier for the transformer implementation, allowing users to easily identify and differentiate
	// between different transformer instances based on their names.
	Name() string

	//
	Transformers[Packet]
}
