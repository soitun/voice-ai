import { TELEMETRY_PROVIDER } from '../index';
import { loadProviderConfig } from '../config-loader';

const EXPECTED_TELEMETRY_CODES = [
  'otlp_http',
  'datadog',
  'xray',
  'google_trace',
  'azure_monitor',
];

const INTERNAL_CODES = ['opensearch', 'logging'];

describe('Telemetry providers registry', () => {
  it('includes only supported telemetry providers', () => {
    const codes = TELEMETRY_PROVIDER.map(p => p.code).sort();

    expect(codes).toEqual([...EXPECTED_TELEMETRY_CODES].sort());
    INTERNAL_CODES.forEach(code => {
      expect(codes).not.toContain(code);
    });
  });

  it('all telemetry providers carry telemetry feature', () => {
    TELEMETRY_PROVIDER.forEach(provider => {
      expect(provider.featureList).toContain('telemetry');
    });
  });

  it('all telemetry providers have a telemetry.json config with endpoint parameter', () => {
    TELEMETRY_PROVIDER.forEach(provider => {
      const config = loadProviderConfig(provider.code);
      expect(config).not.toBeNull();
      expect(config?.telemetry).toBeDefined();
      expect(config?.telemetry?.parameters).toBeDefined();

      const hasEndpoint = config?.telemetry?.parameters.some(
        p => p.key === 'endpoint',
      );
      expect(hasEndpoint).toBe(true);
    });
  });
});
