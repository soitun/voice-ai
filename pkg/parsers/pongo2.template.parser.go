// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package parsers

import (
	"sort"
	"strings"

	"github.com/flosch/pongo2/v6"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type pongo2TemplateParser struct {
	logger commons.Logger
}

type pongo2StringTemplateParser struct {
	pongo2TemplateParser
}

type pongo2MessageTemplateParser struct {
	pongo2TemplateParser
}

func NewPongo2StringTemplateParser(logger commons.Logger) StringTemplateParser {
	return &pongo2StringTemplateParser{
		pongo2TemplateParser: pongo2TemplateParser{logger: logger},
	}
}

func (stp *pongo2StringTemplateParser) Parse(template string, argument map[string]interface{}) string {
	tpl, err := pongo2.FromString(template)
	if err != nil {
		stp.logger.Errorf("error while parsing the template with pongo2: %v", err)
		return template
	}
	formattedTemplate, err := tpl.Execute(pongo2.Context(CanonicalizePromptArguments(utils.NormalizeInterface(argument))))
	if err != nil {
		stp.logger.Errorf("error while executing the template with pongo2: %v", err)
		return template
	}
	return formattedTemplate
}

// canonicalizePromptArguments expands dotted top-level keys (for example
// "message.language") into nested maps so pongo2 receives valid identifiers.
// It processes plain keys first, then dotted keys in lexical order so
// conflicts resolve deterministically.
func CanonicalizePromptArguments(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return map[string]interface{}{}
	}

	out := make(map[string]interface{}, len(in))
	dottedKeys := make([]string, 0)

	for key, value := range in {
		if strings.Contains(key, ".") {
			dottedKeys = append(dottedKeys, key)
			continue
		}

		if nested, ok := value.(map[string]interface{}); ok {
			value = CanonicalizePromptArguments(nested)
		}
		out[key] = value
	}

	sort.Strings(dottedKeys)
	for _, key := range dottedKeys {
		value := in[key]
		if nested, ok := value.(map[string]interface{}); ok {
			value = CanonicalizePromptArguments(nested)
		}
		setNestedPromptArgument(out, strings.Split(key, "."), value)
	}

	return out
}

func setNestedPromptArgument(target map[string]interface{}, parts []string, value interface{}) {
	current := target
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			return
		}

		if i == len(parts)-1 {
			if _, exists := current[part]; exists {
				return
			}
			current[part] = value
			return
		}

		next, exists := current[part]
		if !exists {
			child := make(map[string]interface{})
			current[part] = child
			current = child
			continue
		}

		child, ok := next.(map[string]interface{})
		if !ok {
			child = make(map[string]interface{})
			current[part] = child
		}
		current = child
	}
}
