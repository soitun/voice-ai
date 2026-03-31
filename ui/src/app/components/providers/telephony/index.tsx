import { Metadata, VaultCredential } from '@rapidaai/react';
import {
  ConfigureExotelTelephony,
  ValidateExotelTelephonyOptions,
} from '@/app/components/providers/telephony/exotel';
import {
  ConfigureTwilioTelephony,
  ValidateTwilioTelephonyOptions,
} from '@/app/components/providers/telephony/twilio';
import {
  ConfigureVonageTelephony,
  ValidateVonageTelephonyOptions,
} from '@/app/components/providers/telephony/vonage';
import {
  ConfigureSIPTelephony,
  ValidateSIPTelephonyOptions,
} from '@/app/components/providers/telephony/sip';
import {
  ConfigureAsteriskTelephony,
  ValidateAsteriskTelephonyOptions,
} from '@/app/components/providers/telephony/asterisk';
import { CredentialDropdown } from '@/app/components/dropdown/credential-dropdown';
import { useCallback } from 'react';
import { ProviderComponentProps } from '@/app/components/providers';
import { TELEPHONY_PROVIDER } from '@/providers';
import { Dropdown } from '@carbon/react';
import { Stack } from '@/app/components/carbon/form';

export const ValidateTelephonyOptions = (
  provider: string,
  parameters: Metadata[],
): boolean => {
  switch (provider) {
    case 'vonage':
      return ValidateVonageTelephonyOptions(parameters);
    case 'twilio':
      return ValidateTwilioTelephonyOptions(parameters);
    case 'exotel':
      return ValidateExotelTelephonyOptions(parameters);
    case 'sip':
      return ValidateSIPTelephonyOptions(parameters);
    case 'asterisk':
      return ValidateAsteriskTelephonyOptions(parameters);
    default:
      return false;
  }
};

export const ConfigureTelephonyComponent: React.FC<ProviderComponentProps> = ({
  provider,
  parameters,
  onChangeParameter,
}) => {
  switch (provider) {
    case 'exotel':
      return (
        <ConfigureExotelTelephony
          parameters={parameters || []}
          onParameterChange={onChangeParameter}
        />
      );
    case 'vonage':
      return (
        <ConfigureVonageTelephony
          parameters={parameters || []}
          onParameterChange={onChangeParameter}
        />
      );
    case 'twilio':
      return (
        <ConfigureTwilioTelephony
          parameters={parameters || []}
          onParameterChange={onChangeParameter}
        />
      );
    case 'sip':
      return (
        <ConfigureSIPTelephony
          parameters={parameters || []}
          onParameterChange={onChangeParameter}
        />
      );
    case 'asterisk':
      return (
        <ConfigureAsteriskTelephony
          parameters={parameters || []}
          onParameterChange={onChangeParameter}
        />
      );
    default:
      return null;
  }
};

export const TelephonyProvider: React.FC<ProviderComponentProps> = props => {
  const { provider, onChangeParameter, onChangeProvider, parameters } = props;
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

  const selectedProvider = TELEPHONY_PROVIDER.find(x => x.code === provider) || null;

  return (
    <Stack gap={6}>
      <Dropdown
        id="telephony-provider"
        titleText="Telephony provider"
        label="Select telephony provider"
        items={TELEPHONY_PROVIDER}
        selectedItem={selectedProvider}
        itemToString={(item: any) => item?.name || ''}
        onChange={({ selectedItem }: any) => {
          if (selectedItem) onChangeProvider(selectedItem.code);
        }}
        helperText="Choose a telephony provider to handle voice communication for your applications."
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
        <div className="grid grid-cols-3 gap-x-6 gap-y-3">
          <ConfigureTelephonyComponent {...props} />
        </div>
      )}
    </Stack>
  );
};
