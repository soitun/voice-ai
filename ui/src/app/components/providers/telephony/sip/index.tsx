import { Metadata } from '@rapidaai/react';
import { TextInput } from '@/app/components/carbon/form';

export const ValidateSIPTelephonyOptions = (options: Metadata[]): boolean => {
  const credentialID = options.find(
    opt => opt.getKey() === 'rapida.credential_id',
  );
  if (!credentialID?.getValue()) return false;
  const callerId = options.find(opt => opt.getKey() === 'phone');
  if (!callerId?.getValue()) return false;
  return true;
};

export const ConfigureSIPTelephony: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

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
    onParameterChange(updatedParams);
  };

  return (
    <div className="col-span-2">
      <TextInput
        id="sip-caller-id"
        labelText="Caller ID"
        value={getParamValue('phone')}
        onChange={e => updateParameter('phone', e.target.value)}
        placeholder="e.g., +15551234567"
        helperText="The phone number to display as caller ID for outbound calls."
      />
    </div>
  );
};
