import { Metadata } from '@rapidaai/react';
import {
  GetAPIRequestDefaultOptions,
  ValidateAPIRequestDefaultOptions,
} from '../constant';

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

describe('api-request default options and validation', () => {
  it('builds required keys with sensible defaults', () => {
    const options = GetAPIRequestDefaultOptions([]);
    const byKey = Object.fromEntries(
      options.map(option => [option.getKey(), option.getValue()]),
    );

    expect(byKey['tool.method']).toBe('POST');
    expect(byKey['tool.endpoint']).toBeUndefined();
    expect(byKey['tool.headers']).toBe('{}');
    expect(byKey['tool.parameters']).toBe('{"tool.argument":"argument"}');
  });

  it('preserves existing values when provided', () => {
    const options = GetAPIRequestDefaultOptions([
      createMetadata('tool.method', 'PATCH'),
      createMetadata('tool.endpoint', 'https://api.example.com/items'),
      createMetadata('tool.headers', '{"Authorization":"Bearer token"}'),
      createMetadata('tool.parameters', '{"tool.argument":"id"}'),
    ]);

    const byKey = Object.fromEntries(
      options.map(option => [option.getKey(), option.getValue()]),
    );
    expect(byKey['tool.method']).toBe('PATCH');
    expect(byKey['tool.endpoint']).toBe('https://api.example.com/items');
    expect(byKey['tool.headers']).toContain('Authorization');
    expect(byKey['tool.parameters']).toContain('tool.argument');
  });

  it('fails validation for invalid endpoint url', () => {
    const error = ValidateAPIRequestDefaultOptions([
      createMetadata('tool.method', 'GET'),
      createMetadata('tool.endpoint', 'not-a-url'),
      createMetadata('tool.headers', '{}'),
      createMetadata('tool.parameters', '{"tool.argument":"id"}'),
    ]);

    expect(error).toBe('Please provide a valid URL for the endpoint.');
  });

  it('fails validation for invalid headers json', () => {
    const error = ValidateAPIRequestDefaultOptions([
      createMetadata('tool.method', 'GET'),
      createMetadata('tool.endpoint', 'https://api.example.com/items'),
      createMetadata('tool.headers', '{invalid-json'),
      createMetadata('tool.parameters', '{"tool.argument":"id"}'),
    ]);

    expect(error).toBe('Please provide valid JSON for headers.');
  });

  it('fails validation for duplicate mapped parameter values', () => {
    const error = ValidateAPIRequestDefaultOptions([
      createMetadata('tool.method', 'GET'),
      createMetadata('tool.endpoint', 'https://api.example.com/items'),
      createMetadata('tool.headers', '{}'),
      createMetadata(
        'tool.parameters',
        '{"tool.argument":"dup","conversation.messages":"dup"}',
      ),
    ]);

    expect(error).toBe('Parameter values must be unique.');
  });

  it('passes validation for a valid api request config', () => {
    const error = ValidateAPIRequestDefaultOptions([
      createMetadata('tool.method', 'POST'),
      createMetadata('tool.endpoint', 'https://api.example.com/items'),
      createMetadata('tool.headers', '{"Authorization":"Bearer token"}'),
      createMetadata(
        'tool.parameters',
        '{"tool.argument":"item_id","assistant.prompt":"prompt"}',
      ),
    ]);

    expect(error).toBeUndefined();
  });
});
