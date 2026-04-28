import {
  buildTelemetryCriteriaInputs,
  splitStructuredTelemetryCriteria,
} from '@/app/components/base/modal/conversation-telemetry-modal';

describe('conversation telemetry structured criteria helpers', () => {
  it('extracts conversation and message/context ids from criteria list', () => {
    const parsed = splitStructuredTelemetryCriteria([
      { key: 'conversationId', value: '123' },
      { key: 'contextId', value: 'ctx-1' },
      { key: 'scope', value: 'telephony' },
    ]);

    expect(parsed.conversationId).toBe('123');
    expect(parsed.messageId).toBe('ctx-1');
    expect(parsed.remaining).toEqual([{ key: 'scope', value: 'telephony' }]);
  });

  it('builds server criteria with backend keys conversationId and messageId', () => {
    const criteria = buildTelemetryCriteriaInputs(
      [{ key: 'scope', value: 'telephony' }],
      '321',
      'msg-9',
    );

    expect(criteria).toEqual([
      { key: 'scope', value: 'telephony' },
      { key: 'conversationId', value: '321' },
      { key: 'messageId', value: 'msg-9' },
    ]);
  });
});
