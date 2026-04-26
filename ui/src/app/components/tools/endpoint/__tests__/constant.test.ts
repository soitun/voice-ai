import { Metadata } from '@rapidaai/react';
import {
  GetEndpointDefaultOptions,
  ValidateEndpointDefaultOptions,
} from '../constant';

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

describe('endpoint default options and validation', () => {
  it('builds required keys with sensible defaults', () => {
    const options = GetEndpointDefaultOptions([]);
    const byKey = Object.fromEntries(
      options.map(option => [option.getKey(), option.getValue()]),
    );

    expect(byKey['tool.endpoint_id']).toBeUndefined();
    expect(byKey['tool.parameters']).toBe('{"tool.argument":"argument"}');
  });

  it('preserves existing values when provided', () => {
    const options = GetEndpointDefaultOptions([
      createMetadata('tool.endpoint_id', 'endpoint-123'),
      createMetadata('tool.parameters', '{"tool.argument":"customer_id"}'),
    ]);

    const byKey = Object.fromEntries(
      options.map(option => [option.getKey(), option.getValue()]),
    );
    expect(byKey['tool.endpoint_id']).toBe('endpoint-123');
    expect(byKey['tool.parameters']).toContain('tool.argument');
  });

  it('fails validation for missing endpoint id', () => {
    const error = ValidateEndpointDefaultOptions([
      createMetadata('tool.parameters', '{"tool.argument":"customer_id"}'),
    ]);

    expect(error).toBe(
      'Missing required metadata keys: tool.endpoint_id.',
    );
  });

  it('fails validation for invalid parameters json', () => {
    const error = ValidateEndpointDefaultOptions([
      createMetadata('tool.endpoint_id', 'endpoint-123'),
      createMetadata('tool.parameters', '{invalid-json'),
    ]);

    expect(error).toBe('Please provide a valid JSON string for parameters.');
  });

  it('fails validation for duplicate mapped parameter values', () => {
    const error = ValidateEndpointDefaultOptions([
      createMetadata('tool.endpoint_id', 'endpoint-123'),
      createMetadata(
        'tool.parameters',
        '{"tool.argument":"dup","conversation.messages":"dup"}',
      ),
    ]);

    expect(error).toBe('Parameter values must be unique.');
  });

  it('passes validation for a valid endpoint config', () => {
    const error = ValidateEndpointDefaultOptions([
      createMetadata('tool.endpoint_id', 'endpoint-123'),
      createMetadata(
        'tool.parameters',
        '{"tool.argument":"item_id","assistant.prompt":"prompt"}',
      ),
    ]);

    expect(error).toBeUndefined();
  });
});
