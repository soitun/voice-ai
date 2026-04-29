import React, { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  PrimaryButton,
  SecondaryButton,
  TertiaryButton,
} from '@/app/components/carbon/button';
import { TextInput, TextArea, Stack } from '@/app/components/carbon/form';
import { MultiSelect } from '@/app/components/carbon/dropdown';
import { InputGroup } from '@/app/components/input-group';
import {
  ButtonSet,
  Select as CarbonSelect,
  SelectItem,
  NumberInput,
  Checkbox,
  Button,
  Tooltip,
} from '@carbon/react';
import { Add, TrashCan, ArrowRight, Information } from '@carbon/icons-react';
import { Slider } from '@/app/components/form/slider';
import { GetAssistantWebhook, UpdateWebhook } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import { TabForm } from '@/app/components/form/tab-form';

const webhookEvents = [
  {
    id: 'conversation.begin',
    name: 'conversation.begin',
    description: 'Triggered when a new conversation begins.',
  },
  {
    id: 'conversation.completed',
    name: 'conversation.completed',
    description: 'Triggered when a conversation ends successfully.',
  },
  {
    id: 'conversation.failed',
    name: 'conversation.failed',
    description: 'Triggered when a conversation fails.',
  },
];

const renderLabelWithTooltip = (label: string, tooltip: string) => (
  <span className="inline-flex items-center gap-1">
    {label}
    <Tooltip align="right" label={tooltip}>
      <Information size={14} />
    </Tooltip>
  </span>
);

type WebhookParameterType =
  | 'event'
  | 'assistant'
  | 'client'
  | 'conversation'
  | 'argument'
  | 'metadata'
  | 'option'
  | 'analysis';

const getDefaultParameterKey = (type: WebhookParameterType): string => {
  switch (type) {
    case 'event':
      return 'type';
    case 'assistant':
      return 'id';
    case 'client':
      return 'phone';
    case 'conversation':
      return 'messages';
    default:
      return '';
  }
};

export const UpdateAssistantWebhook: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { webhookId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const { loading, showLoader, hideLoader } = useRapidaStore();

  const [activeTab, setActiveTab] = useState('destination');
  const [errorMessage, setErrorMessage] = useState('');

  const [method, setMethod] = useState('POST');
  const [endpoint, setEndpoint] = useState('');
  const [description, setDescription] = useState('');
  const [retryOnStatus, setRetryOnStatus] = useState<string[]>(['50X']);
  const [maxRetries, setMaxRetries] = useState(3);
  const [requestTimeout, setRequestTimeout] = useState(180);
  const [headers, setHeaders] = useState<{ key: string; value: string }[]>([]);
  const [priority, setPriority] = useState<number>(0);
  const [parameters, setParameters] = useState<
    {
      type: WebhookParameterType;
      key: string;
      value: string;
    }[]
  >([]);
  const [events, setEvents] = useState<string[]>([]);

  useEffect(() => {
    showLoader();
    GetAssistantWebhook(
      connectionConfig,
      assistantId,
      webhookId!,
      (err, res) => {
        hideLoader();
        if (err) {
          toast.error('Unable to load webhook, please try again later.');
          return;
        }
        const wb = res?.getData();
        if (wb) {
          setMethod(wb.getHttpmethod());
          setEndpoint(wb.getHttpurl());
          setDescription(wb.getDescription());
          setRetryOnStatus(wb.getRetrystatuscodesList());
          setMaxRetries(wb.getRetrycount());
          setRequestTimeout(wb.getTimeoutsecond());
          setPriority(wb.getExecutionpriority());
          const headersMap = wb.getHttpheadersMap();
          setHeaders(
            Array.from(headersMap.entries()).map(([key, value]) => ({
              key,
              value,
            })),
          );
          const parametersMap = wb.getHttpbodyMap();
          setParameters(
            Array.from(parametersMap.entries()).map(([key, value]) => {
              const [type, paramKey] = key.split('.');
              return {
                type: type as WebhookParameterType,
                key: paramKey,
                value,
              };
            }),
          );
          setEvents(wb.getAssistanteventsList());
        }
      },
      {
        'x-auth-id': authId,
        authorization: token,
        'x-project-id': projectId,
      },
    );
  }, [assistantId, webhookId, authId, token, projectId]);

  const updateParameter = (index: number, field: string, value: string) => {
    setParameters(prevParams =>
      prevParams.map((param, i) => {
        if (i === index) {
          const updatedParam = { ...param, [field]: value };
          if (field === 'type') {
            updatedParam.key = getDefaultParameterKey(
              value as WebhookParameterType,
            );
            updatedParam.value = '';
          }
          return updatedParam;
        }
        return param;
      }),
    );
  };

  const validateDestination = (): boolean => {
    setErrorMessage('');
    if (!endpoint) {
      setErrorMessage('Please provide a server URL for the webhook.');
      return false;
    }
    if (!/^https?:\/\/.+/.test(endpoint)) {
      setErrorMessage('Please provide a valid server URL for the webhook.');
      return false;
    }
    return true;
  };

  const validatePayload = (): boolean => {
    setErrorMessage('');
    if (parameters.length === 0) {
      setErrorMessage(
        'Please provide one or more parameters which can be passed as data to your server.',
      );
      return false;
    }
    const keys = parameters.map(param => `${param.type}.${param.key}`);
    const uniqueKeys = new Set(keys);
    if (keys.length !== uniqueKeys.size) {
      setErrorMessage('Duplicate parameter keys are not allowed.');
      return false;
    }
    const emptyKeysOrValues = parameters.filter(
      param => param.key.trim() === '' || param.value.trim() === '',
    );
    if (emptyKeysOrValues.length > 0) {
      setErrorMessage('Empty parameter keys or values are not allowed.');
      return false;
    }
    const values = parameters.map(param => param.value.trim());
    const uniqueValues = new Set(values);
    if (values.length !== uniqueValues.size) {
      setErrorMessage('Duplicate parameter values are not allowed.');
      return false;
    }
    return true;
  };

  const onSubmit = () => {
    setErrorMessage('');
    if (events.length === 0) {
      setErrorMessage(
        'Please select at least one event when the webhook will get triggered.',
      );
      return;
    }
    showLoader();
    const parameterKeyValuePairs = parameters.map(param => ({
      key: `${param.type}.${param.key}`,
      value: param.value,
    }));
    UpdateWebhook(
      connectionConfig,
      assistantId,
      webhookId!,
      method,
      endpoint,
      headers,
      parameterKeyValuePairs,
      events,
      retryOnStatus,
      maxRetries,
      requestTimeout,
      priority,
      (err, response) => {
        hideLoader();
        if (err) {
          setErrorMessage(
            'Unable to update assistant webhook, please check and try again.',
          );
          return;
        }
        if (response?.getSuccess()) {
          toast.success(`Assistant's webhook updated successfully`);
          navigator.goToAssistantWebhook(assistantId);
        } else {
          if (response?.getError()) {
            const message = response.getError()?.getHumanmessage();
            if (message) {
              setErrorMessage(message);
              return;
            }
          }
          setErrorMessage(
            'Unable to update assistant webhook, please check and try again.',
          );
        }
      },
      {
        'x-auth-id': authId,
        authorization: token,
        'x-project-id': projectId,
      },
      description,
    );
  };

  return (
    <>
      <ConfirmDialogComponent />
      <TabForm
        formHeading="Update all steps to reconfigure your webhook."
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            code: 'destination',
            name: 'Destination',
            description:
              'Configure the HTTP endpoint that will receive webhook events.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => showDialog(navigator.goBack)}
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  onClick={() => {
                    if (validateDestination()) setActiveTab('payload');
                  }}
                >
                  Continue
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="pb-8">
                <InputGroup
                  title={renderLabelWithTooltip(
                    'Destination',
                    'Configure the HTTP destination that receives the webhook request.',
                  )}
                >
                  <Stack gap={6}>
                    <div className="flex gap-2">
                      <div className="w-36 shrink-0">
                        <CarbonSelect
                          id="webhook-method"
                          labelText="Method"
                          value={method}
                          onChange={e => setMethod(e.target.value)}
                        >
                          <SelectItem value="POST" text="POST" />
                          <SelectItem value="PUT" text="PUT" />
                          <SelectItem value="PATCH" text="PATCH" />
                        </CarbonSelect>
                      </div>
                      <div className="flex-1">
                        <TextInput
                          id="webhook-endpoint"
                          labelText="Server URL"
                          value={endpoint}
                          onChange={e => setEndpoint(e.target.value)}
                          placeholder="https://your-domain.com/webhook"
                        />
                      </div>
                    </div>
                    <TextArea
                      id="webhook-description"
                      labelText="Description (Optional)"
                      value={description}
                      onChange={e => setDescription(e.target.value)}
                      placeholder="An optional description of this webhook destination..."
                      rows={2}
                    />
                  </Stack>
                </InputGroup>
              </div>
            ),
          },
          {
            code: 'payload',
            name: 'Payload',
            description:
              'Define the headers and data fields included in each webhook call.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => showDialog(navigator.goBack)}
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  onClick={() => {
                    if (validatePayload()) setActiveTab('events');
                  }}
                >
                  Continue
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="pb-8 flex flex-col">
                <InputGroup
                  childClass="space-y-4"
                  title={renderLabelWithTooltip(
                    `Headers (${headers.length})`,
                    'HTTP headers included with every webhook request.',
                  )}
                >
                  <table className="w-full border-collapse border border-gray-200 dark:border-gray-700 text-sm [&_input]:!border-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none [&_.cds--form-item]:!m-0">
                    <thead>
                      <tr className="bg-gray-50 dark:bg-gray-900">
                        <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/2">
                          Key
                        </th>
                        <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/2">
                          Value
                        </th>
                        <th className="border-b border-gray-200 dark:border-gray-700 w-8" />
                      </tr>
                    </thead>
                    <tbody>
                      {headers.length === 0 && (
                        <tr>
                          <td
                            colSpan={3}
                            className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400"
                          >
                            No headers yet. Click <strong>Add header</strong>{' '}
                            below to add key-value pairs.
                          </td>
                        </tr>
                      )}
                      {headers.map((header, index) => (
                        <tr
                          key={index}
                          className="border-b border-gray-200 dark:border-gray-700 last:border-b-0"
                        >
                          <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                            <TextInput
                              id={`header-key-${index}`}
                              labelText=""
                              hideLabel
                              value={header.key}
                              onChange={e => {
                                const h = [...headers];
                                h[index].key = e.target.value;
                                setHeaders(h);
                              }}
                              placeholder="Key"
                              size="md"
                            />
                          </td>
                          <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                            <TextInput
                              id={`header-val-${index}`}
                              labelText=""
                              hideLabel
                              value={header.value}
                              onChange={e => {
                                const h = [...headers];
                                h[index].value = e.target.value;
                                setHeaders(h);
                              }}
                              placeholder="Value"
                              size="md"
                            />
                          </td>
                          <td className="p-0 text-center">
                            <Button
                              hasIconOnly
                              renderIcon={TrashCan}
                              iconDescription="Remove"
                              kind="danger--ghost"
                              size="sm"
                              onClick={() =>
                                setHeaders(
                                  headers.filter((_, i) => i !== index),
                                )
                              }
                            />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  <TertiaryButton
                    size="md"
                    renderIcon={Add}
                    onClick={() =>
                      setHeaders([...headers, { key: '', value: '' }])
                    }
                    className="!w-full !max-w-none"
                  >
                    Add header
                  </TertiaryButton>
                </InputGroup>

                <InputGroup
                  title={renderLabelWithTooltip(
                    `Payload Mapping (${parameters.length})`,
                    'Map assistant, client, event, and conversation values into the webhook request body.',
                  )}
                  childClass="space-y-4"
                >
                  <table className="w-full border-collapse border border-gray-200 dark:border-gray-700 text-sm [&_input]:!border-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none [&_.cds--select-input]:!border-none [&_.cds--form-item]:!m-0">
                    <thead>
                      <tr className="bg-gray-50 dark:bg-gray-900">
                        <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-[140px]">
                          Type
                        </th>
                        <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-[140px]">
                          Key
                        </th>
                        <th className="border-b border-r border-gray-200 dark:border-gray-700 w-8" />
                        <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700">
                          Value
                        </th>
                        <th className="border-b border-gray-200 dark:border-gray-700 w-8" />
                      </tr>
                    </thead>
                    <tbody>
                      {parameters.map((params, index) => (
                        <tr
                          key={index}
                          className="border-b border-gray-200 dark:border-gray-700 last:border-b-0"
                        >
                          <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                            <CarbonSelect
                              id={`param-type-${index}`}
                              labelText=""
                              hideLabel
                              value={params.type}
                              onChange={e =>
                                updateParameter(index, 'type', e.target.value)
                              }
                              size="md"
                            >
                              <SelectItem value="event" text="Event" />
                              <SelectItem value="assistant" text="Assistant" />
                              <SelectItem value="client" text="Client" />
                              <SelectItem
                                value="conversation"
                                text="Conversation"
                              />
                              <SelectItem value="argument" text="Argument" />
                              <SelectItem value="metadata" text="Metadata" />
                              <SelectItem value="option" text="Option" />
                              <SelectItem value="analysis" text="Analysis" />
                            </CarbonSelect>
                          </td>
                          <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                            <TypeKeySelector
                              type={params.type}
                              value={params.key}
                              onChange={newKey =>
                                updateParameter(index, 'key', newKey)
                              }
                            />
                          </td>
                          <td className="border-r border-gray-200 dark:border-gray-700 p-0 text-center text-gray-400">
                            <ArrowRight className="w-4 h-4 mx-auto" />
                          </td>
                          <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                            <TextInput
                              id={`param-val-${index}`}
                              labelText=""
                              hideLabel
                              value={params.value}
                              onChange={e =>
                                updateParameter(index, 'value', e.target.value)
                              }
                              placeholder="Value"
                              size="md"
                            />
                          </td>
                          <td className="p-0 text-center">
                            <Button
                              hasIconOnly
                              renderIcon={TrashCan}
                              iconDescription="Remove"
                              kind="danger--ghost"
                              size="sm"
                              onClick={() =>
                                setParameters(
                                  parameters.filter((_, i) => i !== index),
                                )
                              }
                            />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  <TertiaryButton
                    size="md"
                    renderIcon={Add}
                    onClick={() =>
                      setParameters([
                        ...parameters,
                        { type: 'assistant', key: 'id', value: '' },
                      ])
                    }
                    className="!w-full !max-w-none"
                  >
                    Add parameter
                  </TertiaryButton>
                </InputGroup>
              </div>
            ),
          },
          {
            code: 'events',
            name: 'Events & Settings',
            description:
              'Choose which events trigger the webhook and configure retry behavior.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => showDialog(navigator.goBack)}
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton size="lg" isLoading={loading} onClick={onSubmit}>
                  Update webhook
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="pb-8 flex flex-col">
                <InputGroup
                  title={renderLabelWithTooltip(
                    'Events',
                    'Choose which assistant lifecycle events trigger this webhook.',
                  )}
                  childClass="space-y-4"
                >
                  <MultiSelect
                    id="webhook-events"
                    titleText="Select events"
                    label="Select events"
                    items={webhookEvents}
                    selectedItems={webhookEvents.filter(event =>
                      events.includes(event.id),
                    )}
                    itemToString={item => item?.name || ''}
                    onChange={({ selectedItems }) =>
                      setEvents((selectedItems || []).map(event => event.id))
                    }
                    helperText="Select which assistant lifecycle events should send a webhook."
                  />
                </InputGroup>

                <div className="grid lg:grid-cols-2">
                  <InputGroup
                    title={renderLabelWithTooltip(
                      'Retry',
                      'Control how the webhook retries after failed responses.',
                    )}
                    childClass="space-y-4"
                  >
                    <Stack gap={5}>
                      <div className="max-w-xs">
                        <CarbonSelect
                          id="webhook-max-retries"
                          labelText={renderLabelWithTooltip(
                            'Max retry count',
                            'Maximum number of retry attempts after a matching failure response.',
                          )}
                          value={maxRetries.toString()}
                          onChange={e =>
                            setMaxRetries(parseInt(e.target.value))
                          }
                        >
                          <SelectItem value="1" text="1" />
                          <SelectItem value="2" text="2" />
                          <SelectItem value="3" text="3" />
                        </CarbonSelect>
                      </div>
                      <div className="flex flex-wrap gap-4">
                        {['40X', '50X'].map(status => (
                          <Checkbox
                            key={status}
                            id={`retry-status-${status}`}
                            labelText={status}
                            checked={retryOnStatus.includes(status)}
                            onChange={(_, { checked }) => {
                              if (checked) {
                                setRetryOnStatus([...retryOnStatus, status]);
                              } else {
                                setRetryOnStatus(
                                  retryOnStatus.filter(s => s !== status),
                                );
                              }
                            }}
                          />
                        ))}
                      </div>
                    </Stack>
                  </InputGroup>

                  <div className="grid gap-6">
                    <InputGroup
                      title={renderLabelWithTooltip(
                        'Timeout',
                        'Set how long the webhook waits before the request times out.',
                      )}
                      childClass="space-y-4"
                    >
                      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
                        <Slider
                          min={180}
                          max={300}
                          step={1}
                          value={requestTimeout}
                          onSlide={value => setRequestTimeout(value)}
                          className="w-full sm:flex-1"
                        />
                        <NumberInput
                          id="webhook-timeout"
                          hideLabel
                          label={renderLabelWithTooltip(
                            'Timeout (seconds)',
                            'Webhook request timeout in seconds.',
                          )}
                          min={180}
                          max={300}
                          step={1}
                          value={requestTimeout}
                          onChange={(e: any, { value }: any) =>
                            setRequestTimeout(Number(value))
                          }
                          className="!w-full sm:!w-24"
                        />
                      </div>

                      <div className="max-w-[12rem]">
                        <NumberInput
                          id="webhook-priority"
                          label={renderLabelWithTooltip(
                            'Priority',
                            'Execution order when multiple webhooks trigger at the same time.',
                          )}
                          min={0}
                          value={priority}
                          onChange={(e: any, { value }: any) =>
                            setPriority(Number(value))
                          }
                          helperText="Lower numbers execute first when multiple webhooks are triggered."
                        />
                      </div>
                    </InputGroup>
                  </div>
                </div>
              </div>
            ),
          },
        ]}
      />
    </>
  );
};

export const TypeKeySelector: FC<{
  type:
    | 'event'
    | 'assistant'
    | 'client'
    | 'conversation'
    | 'argument'
    | 'metadata'
    | 'option'
    | 'analysis';
  value: string;
  onChange: (newValue: string) => void;
}> = ({ type, value, onChange }) => {
  switch (type) {
    case 'event':
      return (
        <CarbonSelect
          id="type-key-event"
          labelText=""
          hideLabel
          value={value}
          onChange={e => onChange(e.target.value)}
          size="md"
        >
          <SelectItem value="" text="Select key" />
          <SelectItem value="type" text="Type" />
          <SelectItem value="data" text="Data" />
        </CarbonSelect>
      );
    case 'assistant':
      return (
        <CarbonSelect
          id="type-key-assistant"
          labelText=""
          hideLabel
          value={value}
          onChange={e => onChange(e.target.value)}
          size="md"
        >
          <SelectItem value="" text="Select key" />
          <SelectItem value="id" text="ID" />
          <SelectItem value="name" text="Name" />
          <SelectItem value="version" text="Version" />
        </CarbonSelect>
      );
    case 'client':
      return (
        <CarbonSelect
          id="type-key-client"
          labelText=""
          hideLabel
          value={value}
          onChange={e => onChange(e.target.value)}
          size="md"
        >
          <SelectItem value="" text="Select key" />
          <SelectItem value="phone" text="Phone" />
          <SelectItem value="assistantPhone" text="Assistant Phone" />
          <SelectItem value="direction" text="Direction" />
          <SelectItem value="provider" text="Provider" />
          <SelectItem value="providerCallId" text="Provider Call ID" />
        </CarbonSelect>
      );
    case 'conversation':
      return (
        <CarbonSelect
          id="type-key-conversation"
          labelText=""
          hideLabel
          value={value}
          onChange={e => onChange(e.target.value)}
          size="md"
        >
          <SelectItem value="" text="Select key" />
          <SelectItem value="messages" text="Messages" />
          <SelectItem value="id" text="ID" />
        </CarbonSelect>
      );
    default:
      return (
        <TextInput
          id="type-key-custom"
          labelText=""
          hideLabel
          value={value}
          onChange={e => onChange(e.target.value)}
          placeholder="Key"
          size="md"
        />
      );
  }
};
