import { RAPIDA_RESERVED_RUNTIME_VARIABLES } from '@/utils/prompt-reserved-variables';

export type PromptVariableSuggestion = {
  key: string;
  label: string;
  description: string;
  insertText: string;
};

// Detect unfinished template variables like "{{" or "{{assistant."
const TEMPLATE_TRIGGER_REGEX = /\{\{\s*([a-zA-Z0-9_.]*)$/;

export const extractPromptVariableQuery = (linePrefix: string): string | null => {
  const match = linePrefix.match(TEMPLATE_TRIGGER_REGEX);
  if (!match) return null;
  return match[1] || '';
};

export const getPromptVariableSuggestions = (
  linePrefix: string,
): PromptVariableSuggestion[] => {
  const query = extractPromptVariableQuery(linePrefix);
  if (query === null) return [];

  const normalizedQuery = query.toLowerCase();
  return RAPIDA_RESERVED_RUNTIME_VARIABLES.filter(item =>
    item.key.toLowerCase().startsWith(normalizedQuery),
  ).map(item => ({
    key: item.key,
    label: item.variable,
    description: item.runtimeValue,
    insertText: `${item.key}}}`,
  }));
};
