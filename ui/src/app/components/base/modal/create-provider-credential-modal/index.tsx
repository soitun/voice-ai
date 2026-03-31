import { useEffect, useState } from 'react';
import {
  ConnectionConfig,
  CreateProviderCredentialRequest,
} from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { CreateProviderKey } from '@rapidaai/react';
import { ErrorMessage } from '@/app/components/form/error-message';
import { useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { ModalProps } from '@/app/components/base/modal';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/carbon/modal';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { Stack, TextInput, TextArea } from '@/app/components/carbon/form';
import { Dropdown } from '@carbon/react';
import { connectionConfig } from '@/configs';
import { useProviderContext } from '@/context/provider-context';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { INTEGRATION_PROVIDER, RapidaProvider } from '@/providers';

interface CreateProviderCredentialDialogProps extends ModalProps {
  currentProvider?: string | null;
}

export function CreateProviderCredentialDialog(
  props: CreateProviderCredentialDialogProps,
) {
  const { authId, projectId, token } = useCurrentCredential();
  const [provider, setProvider] = useState<RapidaProvider | null>(null);
  const providerCtx = useProviderContext();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [error, setError] = useState('');
  const [keyName, setKeyName] = useState('');
  const [config, setConfig] = useState<Record<string, string>>({});

  useEffect(() => {
    setProvider(
      INTEGRATION_PROVIDER.slice()
        .reverse()
        .find(x => x.code === props.currentProvider) || null,
    );
  }, [props.currentProvider]);

  const handleConfigChange = (name: string, value: string) => {
    setConfig(prev => ({ ...prev, [name]: value }));
  };

  const validateAndSubmit = () => {
    if (!provider) {
      setError('Please select the provider which you want to create the key.');
      return;
    }
    if (!keyName.trim()) {
      setError('Please provide a valid key name for the credential.');
      return;
    }
    const missingFields = provider.configurations?.filter(
      configOption => !config[configOption.name]?.trim(),
    );
    if (missingFields && missingFields.length > 0) {
      setError(
        `Please fill out the following fields: ${missingFields
          .map(field => field.label)
          .join(', ')}`,
      );
      return;
    }

    showLoader();
    const requestObject = new CreateProviderCredentialRequest();
    requestObject.setProvider(provider.code);
    requestObject.setCredential(Struct.fromJavaScript(config));
    requestObject.setName(keyName);

    CreateProviderKey(
      connectionConfig,
      requestObject,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    )
      .then(cpkr => {
        hideLoader();
        if (cpkr?.getSuccess()) {
          toast.success(
            'Provider credential have been successfully added to the vault.',
          );
          providerCtx.reloadProviderCredentials();
          props.setModalOpen(false);
          setError('');
          setKeyName('');
          setConfig({});
        } else {
          let errorMessage = cpkr?.getError();
          setError(
            errorMessage?.getHumanmessage() ??
              'Unable to process your request. Please try again later.',
          );
        }
      })
      .catch(() => {
        hideLoader();
        toast.error(
          'Unable to create provider credential, please try again later.',
        );
      });
  };

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="sm"
      selectorPrimaryFocus="#credential-key-name"
      preventCloseOnClickOutside
    >
      <ModalHeader
        label="Credentials"
        title="Create provider credential"
        onClose={() => props.setModalOpen(false)}
      />
      <ModalBody hasForm>
        <Stack gap={6}>
          <Dropdown
            id="credential-provider"
            titleText="Select your provider"
            label="Select the provider"
            items={INTEGRATION_PROVIDER}
            selectedItem={provider}
            itemToString={(item: RapidaProvider | null) => item?.name || ''}
            onChange={({ selectedItem }) => {
              setError('');
              setKeyName('');
              setConfig({});
              setProvider(selectedItem || null);
            }}
          />
          <TextInput
            id="credential-key-name"
            labelText="Key Name"
            placeholder="Assign a unique name to this provider key"
            value={keyName}
            required
            onChange={e => setKeyName(e.target.value)}
          />
          {provider &&
            provider.configurations?.map((x, idx) =>
              x.type === 'text' ? (
                <TextArea
                  key={idx}
                  id={`config-${x.name}`}
                  labelText={x.label}
                  placeholder={x.label}
                  value={config[x.name] || ''}
                  required
                  onChange={e => handleConfigChange(x.name, e.target.value)}
                />
              ) : (
                <TextInput
                  key={idx}
                  id={`config-${x.name}`}
                  labelText={x.label}
                  placeholder={x.label}
                  value={config[x.name] || ''}
                  required
                  onChange={e => handleConfigChange(x.name, e.target.value)}
                />
              ),
            )}
          <ErrorMessage message={error} />
        </Stack>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => props.setModalOpen(false)}>
          Cancel
        </SecondaryButton>
        <PrimaryButton size="lg" onClick={validateAndSubmit} isLoading={loading}>
          Configure
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
