import { Metadata } from '@rapidaai/react';
import { ProviderComponentProps } from '@/app/components/providers';
import { loadProviderConfig } from '@/providers/config-loader';
import { getDefaultsFromConfig } from '@/providers/config-defaults';
import { ConfigRenderer } from '@/app/components/providers/config-renderer';
import { FC } from 'react';

export const GetDefaultTelemetryIfInvalid = (
  provider: string,
  parameters: Metadata[],
) => {
  const config = loadProviderConfig(provider);
  if (!config?.telemetry) return parameters;
  return getDefaultsFromConfig(config, 'telemetry', parameters, provider, {
    includeCredential: true,
  });
};

export const ValidateTelemetry = (
  provider: string,
  parameters: Metadata[],
): string | undefined => {
  const credentialId = parameters.find(
    p => p.getKey() === 'rapida.credential_id',
  );
  if (!credentialId || !credentialId.getValue()) {
    return `Please provide a valid ${provider} credential.`;
  }

  const config = loadProviderConfig(provider);
  if (!config?.telemetry) return undefined;
  return validateTelemetryFromConfig(config.telemetry.parameters, parameters);
};

function validateTelemetryFromConfig(
  paramConfigs: { key: string; label: string; required?: boolean }[],
  options: Metadata[],
): string | undefined {
  for (const param of paramConfigs) {
    const isRequired = param.required !== false;
    const option = options.find(opt => opt.getKey() === param.key);
    const value = option?.getValue() ?? '';
    if (isRequired && !value) {
      return `Please provide a valid value for ${param.label.toLowerCase()}.`;
    }
  }
  return undefined;
}

export const TelemetryConfigComponent: FC<
  Pick<ProviderComponentProps, 'provider' | 'parameters' | 'onChangeParameter'>
> = ({ provider, parameters, onChangeParameter }) => {
  const config = loadProviderConfig(provider);
  if (!config?.telemetry) return null;
  return (
    <ConfigRenderer
      provider={provider}
      category="telemetry"
      config={config.telemetry}
      parameters={parameters}
      onParameterChange={onChangeParameter}
    />
  );
};
