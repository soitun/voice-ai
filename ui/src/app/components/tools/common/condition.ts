import { Metadata } from '@rapidaai/react';
import { getOptionValue } from './utils';

export const TOOL_CONDITION_JSON_KEY = 'tool.condition';
export const TOOL_CONDITION_OPERATOR_SYMBOL = '=';

export const TOOL_CONDITION_SOURCES = [
  'all',
  'app',
  'debugger',
  'phone',
] as const;

export type ToolConditionSource = (typeof TOOL_CONDITION_SOURCES)[number];
export interface ToolConditionEntry {
  key: string;
  condition: string;
  value: string;
}

export const TOOL_CONDITION_SOURCE_OPTIONS: Array<{
  label: string;
  value: ToolConditionSource;
}> = [
  { label: 'All', value: 'all' },
  { label: 'App', value: 'app' },
  { label: 'Debugger', value: 'debugger' },
  { label: 'Phone', value: 'phone' },
];

const upsertMetadata = (
  parameters: Metadata[],
  key: string,
  value: string,
): Metadata[] => {
  const updated = [...parameters];
  const index = updated.findIndex(param => param.getKey() === key);
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  if (index >= 0) {
    updated[index] = metadata;
  } else {
    updated.push(metadata);
  }
  return updated;
};

export const normalizeToolConditionSource = (
  value?: string,
): ToolConditionSource => {
  if (value && TOOL_CONDITION_SOURCES.includes(value as ToolConditionSource)) {
    return value as ToolConditionSource;
  }
  return 'all';
};

const normalizeConditionEntry = (
  raw?: Partial<ToolConditionEntry> | null,
): ToolConditionEntry => ({
  key: raw?.key === 'source' ? 'source' : 'source',
  condition:
    raw?.condition === TOOL_CONDITION_OPERATOR_SYMBOL
      ? TOOL_CONDITION_OPERATOR_SYMBOL
      : TOOL_CONDITION_OPERATOR_SYMBOL,
  value: normalizeToolConditionSource(raw?.value),
});

const defaultConditionEntries = (): ToolConditionEntry[] => [
  {
    key: 'source',
    condition: TOOL_CONDITION_OPERATOR_SYMBOL,
    value: 'all',
  },
];

export const toToolConditionJson = (entries: ToolConditionEntry[]): string =>
  JSON.stringify(entries, null, 2);

const parseToolConditionJson = (
  jsonValue?: string,
): ToolConditionEntry[] | null => {
  if (!jsonValue) return null;
  try {
    const parsed = JSON.parse(jsonValue);
    if (Array.isArray(parsed)) {
      return parsed
        .filter(
          item =>
            typeof item === 'object' && item !== null && !Array.isArray(item),
        )
        .map(item =>
          normalizeConditionEntry(item as Partial<ToolConditionEntry>),
        );
    }

    return null;
  } catch {
    return null;
  }
};

export const getToolConditionEntries = (
  parameters: Metadata[] | null | undefined,
): ToolConditionEntry[] => {
  const params = parameters || [];
  const jsonEntries = parseToolConditionJson(
    getOptionValue(params, TOOL_CONDITION_JSON_KEY),
  );
  if (jsonEntries && jsonEntries.length > 0) {
    return jsonEntries;
  }

  return defaultConditionEntries();
};

export const getToolConditionSource = (
  parameters: Metadata[] | null | undefined,
): ToolConditionSource => {
  const sourceCondition = getToolConditionEntries(parameters).find(
    entry => entry.key === 'source',
  );
  return normalizeToolConditionSource(sourceCondition?.value);
};

export const getToolConditionSourceLabel = (
  source: ToolConditionSource,
): string =>
  TOOL_CONDITION_SOURCE_OPTIONS.find(option => option.value === source)
    ?.label || 'All';

export const withToolConditionSource = (
  parameters: Metadata[],
  source: ToolConditionSource,
): Metadata[] => {
  return withToolConditionEntries(parameters, [
    {
      key: 'source',
      condition: TOOL_CONDITION_OPERATOR_SYMBOL,
      value: source,
    },
  ]);
};

export const withToolConditionEntries = (
  parameters: Metadata[],
  entries: ToolConditionEntry[],
): Metadata[] => {
  const normalizedEntries =
    entries.length > 0
      ? entries.map(entry => normalizeConditionEntry(entry))
      : defaultConditionEntries();
  const sourceEntry =
    normalizedEntries.find(entry => entry.key === 'source') ||
    defaultConditionEntries()[0];
  const conditionJson = toToolConditionJson([
    {
      key: 'source',
      condition: TOOL_CONDITION_OPERATOR_SYMBOL,
      value: normalizeToolConditionSource(sourceEntry.value),
    },
  ]);

  return upsertMetadata(parameters, TOOL_CONDITION_JSON_KEY, conditionJson);
};

export const withNormalizedToolCondition = (
  parameters: Metadata[],
  fallback?: Metadata[],
): Metadata[] => {
  const primary = getToolConditionEntries(parameters);
  const fallbackEntries = getToolConditionEntries(fallback || []);
  return withToolConditionEntries(
    parameters,
    primary.length > 0 ? primary : fallbackEntries,
  );
};

export const validateToolConditionMetadata = (
  parameters: Metadata[],
): string | undefined => {
  const raw = getOptionValue(parameters, TOOL_CONDITION_JSON_KEY);
  if (!raw || raw.trim() === '') {
    return 'Condition must be a valid JSON array.';
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return 'Condition must be a valid JSON array.';
  }

  if (!Array.isArray(parsed)) {
    return 'Condition must be a valid JSON array.';
  }

  if (parsed.length === 0) {
    return 'Condition must include at least one entry.';
  }

  for (const condition of parsed) {
    if (
      typeof condition !== 'object' ||
      condition === null ||
      Array.isArray(condition)
    ) {
      return 'Each condition must be an object with key, condition, and value.';
    }

    const entry = condition as Record<string, unknown>;
    if (
      typeof entry.key !== 'string' ||
      typeof entry.condition !== 'string' ||
      typeof entry.value !== 'string'
    ) {
      return 'Each condition entry must have string key, condition, and value.';
    }

    if (entry.key !== 'source') {
      return 'Condition currently supports only the "source" key.';
    }

    if (entry.condition !== TOOL_CONDITION_OPERATOR_SYMBOL) {
      return 'Condition operator must be "=".';
    }

    if (!TOOL_CONDITION_SOURCES.includes(entry.value as ToolConditionSource)) {
      return `Condition source must be one of: ${TOOL_CONDITION_SOURCES.join(', ')}.`;
    }
  }

  return undefined;
};
