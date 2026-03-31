import { Metadata } from '@rapidaai/react';
import { TextInput } from '@/app/components/carbon/form';

export const ValidateAsteriskTelephonyOptions = (
  options: Metadata[],
): boolean => {
  const credentialID = options.find(
    opt => opt.getKey() === 'rapida.credential_id',
  );
  if (!credentialID?.getValue()) return false;
  const context = options.find(opt => opt.getKey() === 'context');
  if (!context?.getValue()) return false;
  const extension = options.find(opt => opt.getKey() === 'extension');
  if (!extension?.getValue()) return false;
  const callerId = options.find(opt => opt.getKey() === 'phone');
  if (!callerId?.getValue()) return false;
  return true;
};

export const ConfigureAsteriskTelephony: React.FC<{
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
    <>
      <div className="col-span-1">
        <TextInput
          id="asterisk-context"
          labelText="Context"
          value={getParamValue('context')}
          onChange={e => updateParameter('context', e.target.value)}
          placeholder="e.g., internal"
          helperText="Dialplan context for routing calls."
        />
      </div>
      <div className="col-span-1">
        <TextInput
          id="asterisk-extension"
          labelText="Extension"
          value={getParamValue('extension')}
          onChange={e => updateParameter('extension', e.target.value)}
          placeholder="e.g., 1002"
          helperText="Extension number for this assistant."
        />
      </div>
      <div className="col-span-1">
        <TextInput
          id="asterisk-caller-id"
          labelText="Caller ID"
          value={getParamValue('phone')}
          onChange={e => updateParameter('phone', e.target.value)}
          placeholder="e.g., +15559876543"
          helperText="Caller ID for outbound calls."
        />
      </div>
    </>
  );
};
