import { Metadata } from '@rapidaai/react';
import {
  getToolConditionEntries,
  getToolConditionSource,
  getToolConditionSourceLabel,
  normalizeToolConditionSource,
  withToolConditionEntries,
  withToolConditionSource,
  withNormalizedToolCondition,
  validateToolConditionMetadata,
} from '../condition';

const meta = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

describe('tool condition helpers', () => {
  it('normalizes invalid source to all', () => {
    expect(normalizeToolConditionSource('bad-source')).toBe('all');
    expect(normalizeToolConditionSource('phone')).toBe('phone');
  });

  it('returns default condition entry when metadata is missing', () => {
    expect(getToolConditionEntries([])).toEqual([
      { key: 'source', condition: '=', value: 'all' },
    ]);
  });

  it('upserts normalized condition metadata entry', () => {
    const out = withToolConditionEntries([], [
      { key: 'source', condition: '=', value: 'debugger' },
    ]);
    const raw = out.find(m => m.getKey() === 'tool.condition')?.getValue();
    expect(raw).toContain('"source"');
    expect(raw).toContain('"debugger"');
  });

  it('supports source shortcut helper', () => {
    const out = withToolConditionSource([], 'phone');
    expect(getToolConditionSource(out)).toBe('phone');
    expect(getToolConditionSourceLabel('phone')).toBe('Phone');
  });

  it('uses fallback persisted condition when primary has none', () => {
    const fallback = [
      meta(
        'tool.condition',
        JSON.stringify([{ key: 'source', condition: '=', value: 'debugger' }]),
      ),
    ];
    const out = withNormalizedToolCondition([], fallback);
    expect(getToolConditionSource(out)).toBe('debugger');
  });

  it('validates required condition structure and values', () => {
    expect(validateToolConditionMetadata([])).toBe(
      'Condition must be a valid JSON array.',
    );
    expect(validateToolConditionMetadata([meta('tool.condition', '{}')])).toBe(
      'Condition must be a valid JSON array.',
    );
    expect(
      validateToolConditionMetadata([meta('tool.condition', '[]')]),
    ).toBe('Condition must include at least one entry.');
    expect(
      validateToolConditionMetadata([
        meta(
          'tool.condition',
          JSON.stringify([{ key: 'channel', condition: '=', value: 'phone' }]),
        ),
      ]),
    ).toBe('Condition currently supports only the "source" key.');
    expect(
      validateToolConditionMetadata([
        meta(
          'tool.condition',
          JSON.stringify([{ key: 'source', condition: '!=', value: 'phone' }]),
        ),
      ]),
    ).toBe('Condition operator must be "=".');
    expect(
      validateToolConditionMetadata([
        meta(
          'tool.condition',
          JSON.stringify([{ key: 'source', condition: '=', value: 'sms' }]),
        ),
      ]),
    ).toBe('Condition source must be one of: all, sdk, web_plugin, debugger, phone.');
    expect(
      validateToolConditionMetadata([
        meta(
          'tool.condition',
          JSON.stringify([{ key: 'source', condition: '=', value: 'all' }]),
        ),
      ]),
    ).toBeUndefined();
  });
});
