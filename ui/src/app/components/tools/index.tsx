import { Metadata } from '@rapidaai/react';
import { FC, useCallback, useMemo } from 'react';
import { Dropdown, Select, SelectItem, Tooltip } from '@carbon/react';
import { CONFIG } from '@/configs';
import { Information } from '@carbon/icons-react';
import { ConfigureAPIRequest } from '@/app/components/tools/api-request';
import {
  GetAPIRequestDefaultOptions,
  ValidateAPIRequestDefaultOptions,
} from '@/app/components/tools/api-request/constant';
import { ConfigureEndOfConversation } from '@/app/components/tools/end-of-conversation';
import {
  GetEndOfConversationDefaultOptions,
  ValidateEndOfConversationDefaultOptions,
} from '@/app/components/tools/end-of-conversation/constant';
import { ConfigureEndpoint } from '@/app/components/tools/endpoint';
import {
  GetEndpointDefaultOptions,
  ValidateEndpointDefaultOptions,
} from '@/app/components/tools/endpoint/constant';
import { ConfigureKnowledgeRetrieval } from '@/app/components/tools/knowledge-retrieval';
import {
  GetKnowledgeRetrievalDefaultOptions,
  ValidateKnowledgeRetrievalDefaultOptions,
} from '@/app/components/tools/knowledge-retrieval/constant';
import { ConfigureMCP } from '@/app/components/tools/mcp';
import {
  GetMCPDefaultOptions,
  ValidateMCPDefaultOptions,
} from '@/app/components/tools/mcp/constant';
import { ConfigureTransferCall } from '@/app/components/tools/transfer-call';
import { InputGroup } from '../input-group/index';
import {
  GetTransferCallDefaultOptions,
  ValidateTransferCallDefaultOptions,
} from '@/app/components/tools/transfer-call/constant';
import {
  APIRequestToolDefintion,
  BUILDIN_TOOLS,
  EndOfConverstaionToolDefintion,
  EndpointToolDefintion,
  KnowledgeRetrievalToolDefintion,
  TransferCallToolDefintion,
} from '@/llm-tools';
import {
  ConfigureToolProps,
  getToolConditionEntries,
  TOOL_CONDITION_SOURCE_OPTIONS,
  validateToolConditionMetadata,
  withToolConditionEntries,
  withNormalizedToolCondition,
} from './common';

const CONDITION_OPTIONS = [{ label: 'equals', value: '=' }];

// ============================================================================
// Types
// ============================================================================

export type ToolCode =
  | 'knowledge_retrieval'
  | 'api_request'
  | 'endpoint'
  | 'end_of_conversation'
  | 'transfer_call'
  | 'mcp';

export interface ToolDefinition {
  name: string;
  description: string;
  parameters: string;
}

export interface BuildinToolConfig {
  code: string;
  parameters: Metadata[];
}

// ============================================================================
// Tool Registry - Single source of truth for tool configurations
// ============================================================================

/**
 * Configuration interface for each tool in the registry.
 * @property definition - Static tool definition (optional for runtime-resolved tools like MCP)
 * @property getDefaultOptions - Returns default metadata parameters for the tool
 * @property validateOptions - Validates tool configuration and returns error message if invalid
 * @property Component - React component for tool configuration UI
 */
interface ToolConfig {
  definition?: ToolDefinition;
  getDefaultOptions: (params: Metadata[]) => Metadata[];
  validateOptions: (params: Metadata[]) => string | undefined;
  Component: FC<ConfigureToolProps>;
}

const TOOL_REGISTRY: Record<ToolCode, ToolConfig> = {
  knowledge_retrieval: {
    definition: KnowledgeRetrievalToolDefintion,
    getDefaultOptions: GetKnowledgeRetrievalDefaultOptions,
    validateOptions: ValidateKnowledgeRetrievalDefaultOptions,
    Component: ConfigureKnowledgeRetrieval,
  },
  api_request: {
    definition: APIRequestToolDefintion,
    getDefaultOptions: GetAPIRequestDefaultOptions,
    validateOptions: ValidateAPIRequestDefaultOptions,
    Component: ConfigureAPIRequest,
  },
  endpoint: {
    definition: EndpointToolDefintion,
    getDefaultOptions: GetEndpointDefaultOptions,
    validateOptions: ValidateEndpointDefaultOptions,
    Component: ConfigureEndpoint,
  },
  end_of_conversation: {
    definition: EndOfConverstaionToolDefintion,
    getDefaultOptions: GetEndOfConversationDefaultOptions,
    validateOptions: ValidateEndOfConversationDefaultOptions,
    Component: ConfigureEndOfConversation,
  },
  transfer_call: {
    definition: TransferCallToolDefintion,
    getDefaultOptions: GetTransferCallDefaultOptions,
    validateOptions: ValidateTransferCallDefaultOptions,
    Component: ConfigureTransferCall,
  },
  mcp: {
    // MCP tools don't have a static definition - resolved dynamically at runtime
    definition: undefined,
    getDefaultOptions: GetMCPDefaultOptions,
    validateOptions: ValidateMCPDefaultOptions,
    Component: ConfigureMCP,
  },
};

const DEFAULT_TOOL_CODE: ToolCode = 'endpoint';

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Type guard to check if a string is a valid tool code
 */
const isValidToolCode = (code: string): code is ToolCode => {
  return code in TOOL_REGISTRY;
};

/**
 * Safely retrieves tool configuration with fallback to default
 */
const getToolConfig = (code: string): ToolConfig => {
  return isValidToolCode(code)
    ? TOOL_REGISTRY[code]
    : TOOL_REGISTRY[DEFAULT_TOOL_CODE];
};

/**
 * Returns the default tool definition for a given tool code.
 * If an existing definition has all required fields, it returns the existing one.
 * MCP tools return a placeholder definition as they are resolved at runtime.
 * This should only be called during initialization, not on every render.
 */
export const GetDefaultToolDefintion = (
  code: string,
  existing?: Partial<ToolDefinition>,
): ToolDefinition => {
  // For MCP, use existing or return placeholder
  if (code === 'mcp') {
    if (existing?.name && existing?.description && existing?.parameters) {
      return existing as ToolDefinition;
    }
    // Return placeholder for MCP - actual definition resolved at runtime
    return {
      name: 'mcp_tool',
      description: 'MCP server tool - resolved at runtime',
      parameters: JSON.stringify({ type: 'object', properties: {} }),
    };
  }

  const hasValidExisting =
    existing?.name && existing?.description && existing?.parameters;

  if (hasValidExisting) {
    return existing as ToolDefinition;
  }

  const config = getToolConfig(code);
  if (!config.definition) {
    throw new Error(`Tool definition not found for code: ${code}`);
  }

  return config.definition;
};

/**
 * Returns default tool config parameters, merging with existing if valid.
 */
export const GetDefaultToolConfigIfInvalid = (
  code: string,
  parameters: Metadata[],
): Metadata[] => {
  const config = getToolConfig(code);
  return withNormalizedToolCondition(
    config.getDefaultOptions(parameters),
    parameters,
  );
};

/**
 * Validates tool parameters and returns an error message if invalid.
 * Returns undefined if validation passes.
 */
export const ValidateToolDefaultOptions = (
  code: string,
  parameters: Metadata[],
): string | undefined => {
  if (!isValidToolCode(code)) {
    return `Invalid tool code: ${code}`;
  }
  return (
    TOOL_REGISTRY[code].validateOptions(parameters) ||
    validateToolConditionMetadata(parameters)
  );
};

// ============================================================================
// Components
// ============================================================================

const ConfigureBuildinTool: FC<{
  toolDefinition: ToolDefinition;
  onChangeToolDefinition?: (value: ToolDefinition) => void;
  config: BuildinToolConfig;
  onParameterChange: (params: Metadata[]) => void;
  inputClass?: string;
}> = ({
  config,
  inputClass,
  toolDefinition,
  onChangeToolDefinition,
  onParameterChange,
}) => {
  if (!isValidToolCode(config.code)) {
    return null;
  }

  const { Component } = TOOL_REGISTRY[config.code];

  return (
    <Component
      toolDefinition={toolDefinition}
      onChangeToolDefinition={onChangeToolDefinition}
      parameters={config.parameters}
      inputClass={inputClass}
      onParameterChange={onParameterChange}
    />
  );
};

export const BuildinTool: FC<{
  toolDefinition: ToolDefinition;
  onChangeToolDefinition: (value: ToolDefinition) => void;
  onChangeBuildinTool: (code: string) => void;
  onChangeConfig: (config: BuildinToolConfig) => void;
  inputClass?: string;
  config: BuildinToolConfig;
  showDefinitionForm?: boolean;
}> = ({
  toolDefinition,
  onChangeToolDefinition,
  onChangeBuildinTool,
  onChangeConfig,
  config,
  inputClass,
  showDefinitionForm = true,
}) => {
  const conditionEntries = useMemo(
    () => getToolConditionEntries(config.parameters),
    [config.parameters],
  );
  const sourceCondition = useMemo(
    () =>
      conditionEntries.find(entry => entry.key === 'source') || {
        key: 'source',
        condition: '=',
        value: 'all',
      },
    [conditionEntries],
  );
  const selectedSourceOption = useMemo(
    () =>
      TOOL_CONDITION_SOURCE_OPTIONS.find(
        option => option.value === sourceCondition.value,
      ) || TOOL_CONDITION_SOURCE_OPTIONS[0],
    [sourceCondition.value],
  );

  const handleParameterChange = useCallback(
    (params: Metadata[]) => {
      onChangeConfig({
        ...config,
        parameters: withNormalizedToolCondition(params, config.parameters),
      });
    },
    [config, onChangeConfig],
  );

  const availableTools = useMemo(
    () =>
      CONFIG.workspace.features?.knowledge !== false
        ? BUILDIN_TOOLS
        : BUILDIN_TOOLS.filter(tool => tool.code !== 'knowledge_retrieval'),
    [],
  );

  const currentTool = useMemo(
    () => availableTools.find(tool => tool.code === config.code),
    [config.code, availableTools],
  );

  return (
    <>
      <InputGroup
        title="Condition"
        className="relative z-20"
        childClass="overflow-visible relative z-20"
      >
        <div className="mb-2 text-xs text-gray-500 flex items-center gap-1">
          <span>Rule</span>
          <Tooltip
            align="right"
            label="This rule is tested before the tool is added to the LLM tool list."
          >
            <Information size={14} />
          </Tooltip>
        </div>
        <table className="w-full border-collapse border border-gray-200 dark:border-gray-700 text-sm [&_input]:!border-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none [&_.cds--select-input]:!border-none [&_.cds--form-item]:!m-0">
          <thead>
            <tr className="bg-gray-50 dark:bg-gray-900">
              <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/4">
                <span className="inline-flex items-center gap-1">
                  Key
                  <Tooltip
                    align="right"
                    label="The variable to evaluate for this condition. 'source' refers to the channel the call is coming from."
                  >
                    <Information size={11} />
                  </Tooltip>
                </span>
              </th>
              <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/4">
                <span className="inline-flex items-center gap-1">
                  Condition
                  <Tooltip
                    align="right"
                    label="The condition to evaluate for this variable."
                  >
                    <Information size={11} />
                  </Tooltip>
                </span>
              </th>
              <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-gray-200 dark:border-gray-700">
                <span className="inline-flex items-center gap-1">
                  Value
                  <Tooltip
                    align="right"
                    label="The value to compare against the variable."
                  >
                    <Information size={11} />
                  </Tooltip>
                </span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b border-gray-200 dark:border-gray-700 last:border-b-0">
              <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                <Select
                  id="tool-condition-key"
                  labelText=""
                  hideLabel
                  value={sourceCondition.key}
                  onChange={e => {
                    const next = [
                      {
                        key: e.target.value,
                        condition: sourceCondition.condition,
                        value: sourceCondition.value,
                      },
                    ];
                    onChangeConfig({
                      ...config,
                      parameters: withToolConditionEntries(
                        config.parameters,
                        next,
                      ),
                    });
                  }}
                  size="md"
                >
                  <SelectItem value="source" text="Source" />
                </Select>
              </td>
              <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                <Select
                  id="tool-condition-op"
                  labelText=""
                  hideLabel
                  value={sourceCondition.condition}
                  onChange={e => {
                    const next = [
                      {
                        key: sourceCondition.key,
                        condition: e.target.value,
                        value: sourceCondition.value,
                      },
                    ];
                    onChangeConfig({
                      ...config,
                      parameters: withToolConditionEntries(
                        config.parameters,
                        next,
                      ),
                    });
                  }}
                  size="md"
                >
                  {CONDITION_OPTIONS.map(option => (
                    <SelectItem
                      key={option.value}
                      value={option.value}
                      text={option.label}
                    />
                  ))}
                </Select>
              </td>

              <td className="p-0 min-w-[240px]">
                <Select
                  id="tool-condition-source-value"
                  labelText=""
                  hideLabel
                  value={selectedSourceOption.value}
                  onChange={e => {
                    const next = [
                      {
                        key: sourceCondition.key,
                        condition: sourceCondition.condition,
                        value: e.target.value,
                      },
                    ];
                    onChangeConfig({
                      ...config,
                      parameters: withToolConditionEntries(
                        config.parameters,
                        next,
                      ),
                    });
                  }}
                  size="md"
                >
                  {TOOL_CONDITION_SOURCE_OPTIONS.map(option => (
                    <SelectItem
                      key={option.value}
                      value={option.value}
                      text={option.label}
                    />
                  ))}
                </Select>
              </td>
            </tr>
          </tbody>
        </table>
      </InputGroup>

      <InputGroup title="Action">
        <Dropdown
          id="tool-action-select"
          titleText="Action"
          label="Select provider"
          items={availableTools}
          selectedItem={currentTool}
          itemToString={(item: any) => item?.name || ''}
          onChange={({ selectedItem }: any) => {
            if (selectedItem) onChangeBuildinTool(selectedItem.code);
          }}
        />
      </InputGroup>

      <ConfigureBuildinTool
        toolDefinition={toolDefinition}
        onChangeToolDefinition={
          showDefinitionForm ? onChangeToolDefinition : undefined
        }
        config={config}
        onParameterChange={handleParameterChange}
        inputClass={inputClass}
      />
    </>
  );
};
