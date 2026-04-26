import { Metadata } from '@rapidaai/react';
import {
  GetMCPDefaultOptions,
  ValidateMCPDefaultOptions,
} from '../constant';

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

describe('mcp default options and validation', () => {
  it('builds required and optional keys with defaults', () => {
    const options = GetMCPDefaultOptions([]);
    const byKey = Object.fromEntries(
      options.map(option => [option.getKey(), option.getValue()]),
    );

    expect(byKey['mcp.server_url']).toBeUndefined();
    expect(byKey['mcp.tool_name']).toBeUndefined();
    expect(byKey['mcp.protocol']).toBe('sse');
    expect(byKey['mcp.timeout']).toBe('30');
    expect(byKey['mcp.headers']).toBeUndefined();
  });

  it('preserves existing values when provided', () => {
    const options = GetMCPDefaultOptions([
      createMetadata('mcp.server_url', 'https://mcp.example.com/sse'),
      createMetadata('mcp.tool_name', 'calendar_lookup'),
      createMetadata('mcp.protocol', 'streamable_http'),
      createMetadata('mcp.timeout', '75'),
      createMetadata('mcp.headers', '{"Authorization":"Bearer token"}'),
    ]);

    const byKey = Object.fromEntries(
      options.map(option => [option.getKey(), option.getValue()]),
    );
    expect(byKey['mcp.server_url']).toBe('https://mcp.example.com/sse');
    expect(byKey['mcp.tool_name']).toBe('calendar_lookup');
    expect(byKey['mcp.protocol']).toBe('streamable_http');
    expect(byKey['mcp.timeout']).toBe('75');
    expect(byKey['mcp.headers']).toContain('Authorization');
  });

  it('fails validation when required server_url key is missing', () => {
    const error = ValidateMCPDefaultOptions([
      createMetadata('mcp.protocol', 'sse'),
      createMetadata('mcp.timeout', '30'),
    ]);

    expect(error).toBe('Missing required configuration: mcp.server_url');
  });

  it('fails validation for invalid server url', () => {
    const error = ValidateMCPDefaultOptions([
      createMetadata('mcp.server_url', 'bad-url'),
      createMetadata('mcp.protocol', 'sse'),
      createMetadata('mcp.timeout', '30'),
    ]);

    expect(error).toBe('Invalid MCP Server URL format.');
  });

  it('fails validation for unsupported protocol', () => {
    const error = ValidateMCPDefaultOptions([
      createMetadata('mcp.server_url', 'https://mcp.example.com'),
      createMetadata('mcp.protocol', 'grpc'),
      createMetadata('mcp.timeout', '30'),
    ]);

    expect(error).toBe(
      'Protocol must be "sse", "websocket", or "streamable_http".',
    );
  });

  it('fails validation for timeout out of range', () => {
    const error = ValidateMCPDefaultOptions([
      createMetadata('mcp.server_url', 'https://mcp.example.com'),
      createMetadata('mcp.protocol', 'sse'),
      createMetadata('mcp.timeout', '999'),
    ]);

    expect(error).toBe('Timeout must be a number between 1 and 300 seconds.');
  });

  it('fails validation for invalid headers json', () => {
    const error = ValidateMCPDefaultOptions([
      createMetadata('mcp.server_url', 'https://mcp.example.com'),
      createMetadata('mcp.protocol', 'sse'),
      createMetadata('mcp.timeout', '30'),
      createMetadata('mcp.headers', '{invalid-json'),
    ]);

    expect(error).toBe('Invalid JSON format for headers.');
  });

  it('passes validation for a valid mcp config', () => {
    const error = ValidateMCPDefaultOptions([
      createMetadata('mcp.server_url', 'wss://mcp.example.com/ws'),
      createMetadata('mcp.tool_name', 'crm_search'),
      createMetadata('mcp.protocol', 'websocket'),
      createMetadata('mcp.timeout', '45'),
      createMetadata('mcp.headers', '{"x-api-key":"k1"}'),
    ]);

    expect(error).toBeUndefined();
  });
});
