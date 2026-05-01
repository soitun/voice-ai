import {
  buildTelemetryCriteriaInputs,
  matchesTelemetrySearchDocument,
  parseTelemetrySearchQuery,
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

  it('parses sentry-like field filters and free text terms from search input', () => {
    const query = parseTelemetrySearchQuery(
      'type:metric scope:llm conversation:123 "timeout error"',
    );

    expect(query).toEqual({
      freeTextTerms: ['timeout error'],
      filters: {
        type: ['metric'],
        component: [],
        scope: ['llm'],
        name: [],
        conversationId: ['123'],
        messageId: [],
        contextId: [],
      },
    });
  });

  it('matches telemetry documents against structured search filters', () => {
    const query = parseTelemetrySearchQuery(
      'type:event component:telephony message:call-9 connected',
    );

    expect(
      matchesTelemetrySearchDocument(
        {
          kind: 'event',
          componentType: 'telephony',
          typeLabel: 'sip.call.connected',
          name: 'sip.call.connected',
          scope: '',
          conversationId: '100',
          messageId: 'call-9',
          contextId: '',
          rawText: '{"status":"connected"}',
        },
        query,
      ),
    ).toBe(true);

    expect(
      matchesTelemetrySearchDocument(
        {
          kind: 'metric',
          componentType: 'metric',
          typeLabel: 'metric.llm',
          name: '',
          scope: 'llm',
          conversationId: '100',
          messageId: '',
          contextId: 'ctx-1',
          rawText: '{"status":"connected"}',
        },
        query,
      ),
    ).toBe(false);
  });
});
