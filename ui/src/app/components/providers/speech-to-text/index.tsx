import { CredentialDropdown } from '@/app/components/dropdown/credential-dropdown';
import { ProviderComponentProps } from '@/app/components/providers';
import { SpeechToTextConfigComponent } from '@/app/components/providers/speech-to-text/provider';
import { SPEECH_TO_TEXT_PROVIDER } from '@/providers';
import { Metadata, VaultCredential } from '@rapidaai/react';
import { useCallback } from 'react';
import { Dropdown } from '@carbon/react';
import { Stack } from '@/app/components/carbon/form';

export const SpeechToTextProvider: React.FC<ProviderComponentProps> = props => {
  const { parameters, provider, onChangeParameter, onChangeProvider } = props;

  const getParamValue = useCallback(
    (key: string) =>
      parameters?.find(p => p.getKey() === key)?.getValue() ?? '',
    [parameters],
  );

  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...(parameters || [])];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }
    onChangeParameter(updatedParams);
  };

  const selectedProvider = SPEECH_TO_TEXT_PROVIDER.find(x => x.code === provider) || null;

  return (
    <Stack gap={6}>
      <Dropdown
        id="stt-provider"
        titleText="Provider"
        label="Select voice input provider"
        items={SPEECH_TO_TEXT_PROVIDER}
        selectedItem={selectedProvider}
        itemToString={(item: any) => item?.name || ''}
        onChange={({ selectedItem }: any) => {
          if (selectedItem) onChangeProvider(selectedItem.code);
        }}
      />
      {provider && (
        <CredentialDropdown
          onChangeCredential={(c: VaultCredential) => {
            updateParameter('rapida.credential_id', c.getId());
          }}
          currentCredential={getParamValue('rapida.credential_id')}
          provider={provider}
        />
      )}
      {provider && (
        <div className="grid grid-cols-3 gap-6">
          <SpeechToTextConfigComponent {...props} />
        </div>
      )}
    </Stack>
  );
};
