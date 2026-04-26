import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { AssistantConversationTelephonyEventDialog } from '../index';

jest.mock('@/app/components/base/modal/right-side-modal', () => ({
  RightSideModal: ({ modalOpen, children }: any) => (modalOpen ? <div>{children}</div> : null),
}));

jest.mock('@/app/components/code-highlighting', () => ({
  CodeHighlighting: ({ language, code }: any) => (
    <div data-testid="payload" data-language={language}>
      {code}
    </div>
  ),
}));

jest.mock('@/app/components/carbon/table-toolbar-filter', () => ({
  TableToolbarFilter: ({ onApplyFilter, onResetFilter }: any) => (
    <div>
      <button onClick={() => onApplyFilter(new Set(['session']))}>Apply Session Filter</button>
      <button onClick={() => onResetFilter()}>Reset Filter</button>
    </div>
  ),
}));

jest.mock('@carbon/react', () => ({
  DismissibleTag: ({ text, onClose }: any) => (
    <button onClick={onClose}>tag-{text}</button>
  ),
}));

jest.mock('@/utils/date', () => ({
  toHumanReadableDateTime: jest.fn(() => 'formatted-date'),
}));

type MockEvent = {
  getId: () => string;
  getProvider: () => string;
  getEventtype: () => string;
  getCreateddate: () => any;
  getPayload: () => { toJavaScript: () => unknown };
};

const makeEvent = (overrides: Partial<MockEvent> = {}): MockEvent => ({
  getId: () => 'evt_1',
  getProvider: () => 'twilio',
  getEventtype: () => 'ringing',
  getCreateddate: () => ({}),
  getPayload: () => ({ toJavaScript: () => ({ ok: true }) }),
  ...overrides,
});

describe('AssistantConversationTelephonyEventDialog', () => {
  it('renders JSON payload using json language when row is expanded', () => {
    render(
      <AssistantConversationTelephonyEventDialog
        modalOpen
        setModalOpen={jest.fn()}
        events={[makeEvent() as any]}
      />,
    );

    fireEvent.click(screen.getByText('evt_1').closest('button')!);

    expect(screen.getByTestId('payload')).toHaveAttribute('data-language', 'json');
  });

  it('shows fallback created date when event created date is missing', () => {
    render(
      <AssistantConversationTelephonyEventDialog
        modalOpen
        setModalOpen={jest.fn()}
        events={[makeEvent({ getCreateddate: () => undefined }) as any]}
      />,
    );

    expect(screen.getByText('N/A')).toBeInTheDocument();
  });

  it('filters rows by selected component type', () => {
    render(
      <AssistantConversationTelephonyEventDialog
        modalOpen
        setModalOpen={jest.fn()}
        events={[
          makeEvent({ getId: () => 'evt_session', getEventtype: () => 'session' }) as any,
          makeEvent({ getId: () => 'evt_llm', getEventtype: () => 'llm' }) as any,
        ]}
      />,
    );

    expect(screen.getByText('evt_session')).toBeInTheDocument();
    expect(screen.getByText('evt_llm')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Apply Session Filter' }));

    expect(screen.getByText('evt_session')).toBeInTheDocument();
    expect(screen.queryByText('evt_llm')).not.toBeInTheDocument();
  });
});
