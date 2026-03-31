import { ProviderComponentProps } from '@/app/components/providers';
import { EndOfSpeech } from '@/providers';
import { EndOfSpeechConfigComponent } from '@/app/components/providers/end-of-speech/provider';
import { useMemo } from 'react';
import { Dropdown } from '@carbon/react';
import { Stack } from '@/app/components/carbon/form';

export const EndOfSpeechProvider: React.FC<ProviderComponentProps> = props => {
  const { provider, onChangeProvider } = props;
  const providers = useMemo(() => EndOfSpeech(), []);
  const selectedProvider = providers.find(x => x.code === provider) || null;

  return (
    <Stack gap={6}>
      <Dropdown
        id="eos-provider"
        titleText="End-of-speech provider"
        label="Select end of speech provider"
        items={providers}
        selectedItem={selectedProvider}
        itemToString={(item: any) => item?.name || ''}
        onChange={({ selectedItem }: any) => {
          if (selectedItem) onChangeProvider(selectedItem.code);
        }}
      />
      {provider && <EndOfSpeechConfigComponent {...props} />}
    </Stack>
  );
};
