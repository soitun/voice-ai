// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package parsers

type Parser[In, Out any] interface {
	Parse(u string, argument map[string]interface{}) Out
}

type StringTemplateParser interface {
	Parser[string, string]
}
