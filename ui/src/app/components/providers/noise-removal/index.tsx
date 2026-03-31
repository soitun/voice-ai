import { HTMLAttributes, useMemo } from 'react';
import { NoiseCancellation } from '@/providers';
import { Metadata } from '@rapidaai/react';
import { NoiseCancellationConfigComponent } from '@/app/components/providers/noise-removal/provider';
import { Dropdown } from '@carbon/react';
import { Stack } from '@/app/components/carbon/form';

interface NoiseCancellationProviderProps
  extends HTMLAttributes<HTMLDivElement> {
  noiseCancellationProvider?: string;
  onChangeNoiseCancellationProvider: (v: string) => void;
  parameters?: Metadata[];
  onChangeParameter?: (parameters: Metadata[]) => void;
}

export const NoiseCancellationProvider: React.FC<
  NoiseCancellationProviderProps
> = ({
  noiseCancellationProvider,
  onChangeNoiseCancellationProvider,
  parameters,
  onChangeParameter,
}) => {
  const providers = useMemo(() => NoiseCancellation(), []);
  const selectedProvider = providers.find(x => x.code === noiseCancellationProvider) || null;

  return (
    <Stack gap={6}>
      <Dropdown
        id="noise-provider"
        titleText="Background noise provider"
        label="Select noise removal provider"
        items={providers}
        selectedItem={selectedProvider}
        itemToString={(item: any) => item?.name || ''}
        onChange={({ selectedItem }: any) => {
          if (selectedItem) onChangeNoiseCancellationProvider(selectedItem.code);
        }}
      />
      {noiseCancellationProvider && parameters && onChangeParameter && (
        <NoiseCancellationConfigComponent
          provider={noiseCancellationProvider}
          parameters={parameters}
          onChangeParameter={onChangeParameter}
          onChangeProvider={() => {}}
        />
      )}
    </Stack>
  );
};
