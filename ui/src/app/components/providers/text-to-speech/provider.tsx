import { Metadata } from '@rapidaai/react';
import { FC } from 'react';
import { loadProviderConfig } from '@/providers/config-loader';
import { getDefaultsFromConfig, validateFromConfig } from '@/providers/config-defaults';
import { ConfigRenderer } from '@/app/components/providers/config-renderer';
import { ProviderComponentProps } from '@/app/components/providers';

export const GetDefaultSpeakerConfig = (
  existing: Metadata[] = [],
): Metadata[] => {
  const defaultConfig = [
    {
      key: 'speaker.conjunction.boundaries',
      value: '',
    },
    {
      key: 'speaker.conjunction.break',
      value: '240',
    },
    {
      key: 'speaker.pronunciation.dictionaries',
      value: '',
    },
  ];

  const result = [...existing];
  defaultConfig.forEach(item => {
    if (!existing.some(e => e.getKey() === item.key)) {
      const metadata = new Metadata();
      metadata.setKey(item.key);
      metadata.setValue(item.value);
      result.push(metadata);
    }
  });
  return result;
};

export const GetDefaultTextToSpeechIfInvalid = (
  provider: string,
  parameters: Metadata[],
): Metadata[] => {
  const config = loadProviderConfig(provider);
  if (config?.tts) return getDefaultsFromConfig(config, 'tts', parameters, provider);
  return parameters;
};

export const ValidateTextToSpeechIfInvalid = (
  provider: string,
  parameters: Metadata[],
  providerCredentialIds?: string[],
): string | undefined => {
  const config = loadProviderConfig(provider);
  if (!config?.tts) return undefined;

  const validationError = validateFromConfig(config, 'tts', provider, parameters);
  if (validationError) return validationError;

  if (!providerCredentialIds) return undefined;

  const credentialID = parameters.find(
    opt => opt.getKey() === 'rapida.credential_id',
  )?.getValue();
  if (!credentialID) {
    return `Please provide a valid ${provider} credential.`;
  }
  if (!providerCredentialIds.includes(credentialID)) {
    return `Please select a valid ${provider} credential.`;
  }

  return undefined;
};

export const TextToSpeechConfigComponent: FC<ProviderComponentProps> = ({
  provider,
  parameters,
  onChangeParameter,
}) => {
  const config = loadProviderConfig(provider);
  if (!config?.tts) return null;
  return (
    <ConfigRenderer
      provider={provider}
      category="tts"
      config={config.tts}
      parameters={parameters}
      onParameterChange={onChangeParameter}
    />
  );
};
