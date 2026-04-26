import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { ConfigureAssistantToolDialog } from '../index';

const mockValidateToolDefaultOptions = jest.fn();

jest.mock('@/configs', () => ({
  CONFIG: {
    workspace: {
      features: {
        knowledge: true,
      },
    },
  },
}));

jest.mock('@/app/components/carbon/modal', () => ({
  Modal: ({ open, children }: any) => (open ? <div>{children}</div> : null),
  ModalHeader: ({ title }: any) => <div>{title}</div>,
  ModalBody: ({ children }: any) => <div>{children}</div>,
  ModalFooter: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/carbon/button', () => ({
  PrimaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@/app/components/carbon/notification', () => ({
  Notification: ({ subtitle }: any) => <div>{subtitle}</div>,
}));

jest.mock('@/app/components/tools', () => ({
  BuildinTool: ({
    onChangeToolDefinition,
    onChangeBuildinTool,
    onChangeConfig,
    config,
  }: any) => (
    <div>
      <button
        onClick={() =>
          onChangeToolDefinition({
            name: 'tool_name',
            description: 'tool_description',
            parameters: '{"k":"v"}',
          })
        }
      >
        Set Valid Definition
      </button>
      <button
        onClick={() =>
          onChangeToolDefinition({
            name: 'tool_name',
            description: 'tool_description',
            parameters: '{bad-json',
          })
        }
      >
        Set Invalid JSON Definition
      </button>
      <button onClick={() => onChangeBuildinTool('mcp')}>Use MCP</button>
      <button
        onClick={() =>
          onChangeConfig({
            ...config,
            code: 'mcp',
            parameters: [{ getKey: () => 'mcp.server_url', getValue: () => 'https://mcp.example.com' }],
          })
        }
      >
        Set MCP Config
      </button>
    </div>
  ),
  BuildinToolConfig: {},
  GetDefaultToolDefintion: (code: string, existing?: any) => {
    if (existing?.name && existing?.description && existing?.parameters) {
      return existing;
    }
    if (code === 'mcp') {
      return {
        name: 'mcp_tool',
        description: 'MCP tool',
        parameters: '{"type":"object"}',
      };
    }
    return {
      name: 'knowledge_tool',
      description: 'Knowledge tool',
      parameters: '{"type":"object"}',
    };
  },
  GetDefaultToolConfigIfInvalid: (code: string, params: any[]) => params || [],
  ValidateToolDefaultOptions: (...args: unknown[]) =>
    mockValidateToolDefaultOptions(...args),
}));

describe('ConfigureAssistantToolDialog', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockValidateToolDefaultOptions.mockReturnValue(undefined);
  });

  it('submits valid create payload via onChange', () => {
    const onChange = jest.fn();

    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
        onChange={onChange}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Set Valid Definition' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save tool' }));

    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange.mock.calls[0][0]).toMatchObject({
      name: 'tool_name',
      description: 'tool_description',
      fields: '{"k":"v"}',
    });
  });

  it('shows validation error for invalid JSON parameters', () => {
    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );

    fireEvent.click(
      screen.getByRole('button', { name: 'Set Invalid JSON Definition' }),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Save tool' }));

    expect(
      screen.getByText(
        'Please provide a valid parameter, parameter must be a valid JSON.',
      ),
    ).toBeInTheDocument();
  });

  it('surfaces parent onValidateConfig error and blocks submit', () => {
    const onChange = jest.fn();

    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
        onChange={onChange}
        onValidateConfig={() => 'Please provide a unique tool name for tools.'}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Set Valid Definition' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save tool' }));

    expect(
      screen.getByText('Please provide a unique tool name for tools.'),
    ).toBeInTheDocument();
    expect(onChange).not.toHaveBeenCalled();
  });

  it('loads initial data on edit and validates using its tool code', () => {
    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={{
          name: 'old_tool',
          description: 'old desc',
          fields: '{"x":"y"}',
          buildinToolConfig: {
            code: 'knowledge_retrieval',
            parameters: [],
          },
        }}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Save tool' }));
    expect(mockValidateToolDefaultOptions).toHaveBeenCalledWith(
      'knowledge_retrieval',
      [],
    );
  });
});
