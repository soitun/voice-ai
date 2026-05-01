import React, { useDeferredValue, useEffect, useState } from 'react';
import {
  AssistantDefinition,
  ConnectionConfig,
  Criteria,
  GetAllAssistantTelemetry,
  GetAllAssistantTelemetryRequest,
  Paginate,
  TelemetryEvent,
  TelemetryMetric,
} from '@rapidaai/react';
import { ModalProps } from '@/app/components/base/modal';
import { connectionConfig } from '@/configs';
import { useCurrentCredential } from '@/hooks/use-credential';
import { Modal, ModalHeader, ModalBody } from '@/app/components/carbon/modal';
import { Pagination } from '@/app/components/carbon/pagination';
import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableExpandedRow,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Tag,
  DismissibleTag,
  Loading,
  CodeSnippet,
  Stack,
} from '@carbon/react';
import { TableToolbarFilter } from '@/app/components/carbon/table-toolbar-filter';
import { ChevronRight } from '@carbon/icons-react';
import { TextInput } from '@/app/components/carbon/form';

// ─── Types ───────────────────────────────────────────────────────────────────

interface ConversationTelemetryDialogProps extends ModalProps {
  assistantId: string;
  criterias?: Criteria[];
}

interface Chip {
  field: string;
  value: string | number;
  id: string;
}

type CriteriaInput = {
  key: string;
  value: string;
};

type TelemetrySearchQuery = {
  freeTextTerms: string[];
  filters: {
    type: string[];
    component: string[];
    scope: string[];
    name: string[];
    conversationId: string[];
    messageId: string[];
    contextId: string[];
  };
};

type TelemetrySearchDocument = {
  kind: 'event' | 'metric';
  componentType: string;
  typeLabel: string;
  name: string;
  scope: string;
  conversationId: string;
  messageId: string;
  contextId: string;
  rawText: string;
};

type TelemetryRow =
  | { kind: 'event'; ts: Date; key: string; record: TelemetryEvent }
  | { kind: 'metric'; ts: Date; key: string; record: TelemetryMetric };

// ─── Color map ───────────────────────────────────────────────────────────────

const EVENT_TAG_TYPE: Record<string, string> = {
  session: 'gray',
  telephony: 'teal',
  webrtc: 'cool-gray',
  stt: 'green',
  llm: 'blue',
  tts: 'purple',
  vad: 'warm-gray',
  eos: 'cyan',
  denoise: 'warm-gray',
  recording: 'purple',
  tool: 'magenta',
  knowledge: 'teal',
  metric: 'high-contrast',
};

const normalizeComponentType = (nameKey: string): string =>
  nameKey === 'sip' ? 'telephony' : nameKey;

export const splitStructuredTelemetryCriteria = (
  criteriaInputs: CriteriaInput[],
): {
  conversationId: string;
  messageId: string;
  remaining: CriteriaInput[];
} => {
  let conversationId = '';
  let messageId = '';
  const remaining: CriteriaInput[] = [];

  criteriaInputs.forEach(c => {
    if (c.key === 'conversationId') {
      conversationId = c.value;
      return;
    }
    if (c.key === 'messageId' || c.key === 'contextId') {
      messageId = c.value;
      return;
    }
    remaining.push(c);
  });

  return { conversationId, messageId, remaining };
};

export const buildTelemetryCriteriaInputs = (
  remaining: CriteriaInput[],
  conversationId: string,
  messageId: string,
): CriteriaInput[] => {
  const out = [...remaining];
  if (conversationId)
    out.push({ key: 'conversationId', value: conversationId });
  if (messageId) out.push({ key: 'messageId', value: messageId });
  return out;
};

const createEmptyTelemetrySearchQuery = (): TelemetrySearchQuery => ({
  freeTextTerms: [],
  filters: {
    type: [],
    component: [],
    scope: [],
    name: [],
    conversationId: [],
    messageId: [],
    contextId: [],
  },
});

const tokenizeTelemetrySearchQuery = (value: string): string[] =>
  (value.match(/"[^"]+"|\S+/g) || [])
    .map(token => token.trim())
    .filter(Boolean)
    .map(token =>
      token.startsWith('"') && token.endsWith('"')
        ? token.slice(1, -1).trim()
        : token,
    )
    .filter(Boolean);

const normalizeTelemetrySearchField = (field: string): string => {
  switch (field.toLowerCase()) {
    case 'event':
      return 'name';
    case 'conversation':
      return 'conversationId';
    case 'message':
      return 'messageId';
    case 'context':
      return 'contextId';
    default:
      return field.toLowerCase();
  }
};

export const parseTelemetrySearchQuery = (
  value: string,
): TelemetrySearchQuery => {
  const query = createEmptyTelemetrySearchQuery();

  tokenizeTelemetrySearchQuery(value).forEach(token => {
    const delimiterIndex = token.indexOf(':');
    if (delimiterIndex <= 0) {
      query.freeTextTerms.push(token.toLowerCase());
      return;
    }

    const field = normalizeTelemetrySearchField(
      token.slice(0, delimiterIndex).trim(),
    );
    const fieldValue = token.slice(delimiterIndex + 1).trim().toLowerCase();

    if (!fieldValue) {
      return;
    }

    switch (field) {
      case 'type':
      case 'component':
      case 'scope':
      case 'name':
      case 'conversationId':
      case 'messageId':
      case 'contextId':
        query.filters[field].push(fieldValue);
        break;
      default:
        query.freeTextTerms.push(token.toLowerCase());
    }
  });

  return query;
};

export const matchesTelemetrySearchDocument = (
  document: TelemetrySearchDocument,
  query: TelemetrySearchQuery,
): boolean => {
  const contains = (source: string, term: string) =>
    source.toLowerCase().includes(term);
  const matchesAny = (values: string[], candidate: string) =>
    values.length === 0 || values.some(value => contains(candidate, value));

  if (
    query.freeTextTerms.some(
      term =>
        !contains(document.typeLabel, term) && !contains(document.rawText, term),
    )
  ) {
    return false;
  }

  if (!matchesAny(query.filters.type, document.kind)) return false;
  if (!matchesAny(query.filters.component, document.componentType)) return false;
  if (!matchesAny(query.filters.scope, document.scope)) return false;
  if (!matchesAny(query.filters.name, document.name)) return false;
  if (!matchesAny(query.filters.conversationId, document.conversationId))
    return false;
  if (!matchesAny(query.filters.contextId, document.contextId)) return false;

  if (
    query.filters.messageId.length > 0 &&
    !query.filters.messageId.some(
      value =>
        contains(document.messageId, value) ||
        contains(document.contextId, value),
    )
  ) {
    return false;
  }

  return true;
};

// ─── Helpers ─────────────────────────────────────────────────────────────────

function formatDateTime(d: Date): string {
  const pad = (n: number, w = 2) => String(n).padStart(w, '0');
  return (
    `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ` +
    `${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}.${pad(d.getUTCMilliseconds(), 3)}`
  );
}

function eventToJson(event: TelemetryEvent): object {
  const data = Object.fromEntries(
    event.getDataMap().toArray() as [string, string][],
  );
  return {
    name: event.getName(),
    messageId: event.getMessageid(),
    conversationId: event.getAssistantconversationid(),
    data,
  };
}

function metricToJson(metric: TelemetryMetric): object {
  return {
    scope: metric.getScope(),
    contextId: metric.getContextid(),
    conversationId: metric.getAssistantconversationid(),
    metrics: metric
      .getMetricsList()
      .map(m => ({ name: m.getName(), value: m.getValue() })),
  };
}

function getTelemetrySearchDocument(
  row: TelemetryRow,
  typeLabel: string,
  json: object,
): TelemetrySearchDocument {
  if (row.kind === 'event') {
    return {
      kind: 'event',
      componentType: normalizeComponentType(row.record.getName().split('.')[0]),
      typeLabel,
      name: row.record.getName(),
      scope: '',
      conversationId: row.record.getAssistantconversationid(),
      messageId: row.record.getMessageid(),
      contextId: '',
      rawText: JSON.stringify(json),
    };
  }

  return {
    kind: 'metric',
    componentType: 'metric',
    typeLabel,
    name: '',
    scope: row.record.getScope(),
    conversationId: row.record.getAssistantconversationid(),
    messageId: '',
    contextId: row.record.getContextid(),
    rawText: JSON.stringify(json),
  };
}

// ─── Main dialog ─────────────────────────────────────────────────────────────

export function ConversationTelemetryDialog(
  props: ConversationTelemetryDialogProps,
) {
  const { token, authId, projectId } = useCurrentCredential();
  const [chips, setChips] = useState<Chip[]>([]);
  const [rows, setRows] = useState<TelemetryRow[]>([]);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalItem, setTotalItem] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [criteriaReady, setCriteriaReady] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [activeFilters, setActiveFilters] = useState<Set<string>>(new Set());
  const [conversationIdInput, setConversationIdInput] = useState('');
  const [messageIdInput, setMessageIdInput] = useState('');
  const [appliedConversationId, setAppliedConversationId] = useState('');
  const [appliedMessageId, setAppliedMessageId] = useState('');
  const [structuredError, setStructuredError] = useState('');
  const deferredSearchText = useDeferredValue(searchText);
  const parsedSearchQuery = parseTelemetrySearchQuery(deferredSearchText);
  const hasSearchQuery = searchText.trim() !== '';
  const hasTypeFilters = activeFilters.size > 0;
  const shouldFetchAllRows = hasSearchQuery || hasTypeFilters;
  const requestPage = shouldFetchAllRows ? 1 : page;
  const requestPageSize = shouldFetchAllRows ? 100 : pageSize;

  useEffect(() => {
    const normalized = splitStructuredTelemetryCriteria(
      (props.criterias || []).map(criteria => ({
        key: criteria.getKey(),
        value: criteria.getValue(),
      })),
    );
    const initialChips = normalized.remaining.map((criteria, index) => ({
      field: criteria.key,
      value: criteria.value,
      id: `${Date.now()}-${index}`,
    }));
    setRows([]);
    setExpandedRows(new Set());
    setTotalItem(0);
    setPage(1);
    setChips(initialChips);
    setConversationIdInput(normalized.conversationId);
    setMessageIdInput(normalized.messageId);
    setAppliedConversationId(normalized.conversationId);
    setAppliedMessageId(normalized.messageId);
    setStructuredError('');
    setCriteriaReady(true);
  }, [props.criterias]);

  useEffect(() => {
    if (!criteriaReady) return;
    let active = true;
    setIsLoading(true);
    setRows([]);
    setExpandedRows(new Set());

    const criteriaList = buildTelemetryCriteriaInputs(
      chips.map(chip => ({ key: chip.field, value: String(chip.value) })),
      appliedConversationId,
      appliedMessageId,
    ).map(c => {
      const criteria = new Criteria();
      criteria.setKey(c.key);
      criteria.setValue(c.value);
      criteria.setLogic('match');
      return criteria;
    });

    const buildRequest = (nextPage: number, nextPageSize: number) => {
      const request = new GetAllAssistantTelemetryRequest();
      const paginate = new Paginate();
      paginate.setPage(nextPage);
      paginate.setPagesize(nextPageSize);
      request.setPaginate(paginate);

      const assistantDef = new AssistantDefinition();
      assistantDef.setAssistantid(props.assistantId);
      request.setAssistant(assistantDef);
      request.setCriteriasList(criteriaList);
      return request;
    };

    const toTelemetryRows = (response: any, pageOffset: number) => {
      const merged: TelemetryRow[] = [];
      response.getDataList().forEach((record: any, index: number) => {
        const event = record.getEvent();
        const metric = record.getMetric();
        if (event) {
          merged.push({
            kind: 'event',
            ts: event.getTime()?.toDate() ?? new Date(0),
            key: `e-${pageOffset + index}`,
            record: event,
          });
        } else if (metric) {
          merged.push({
            kind: 'metric',
            ts: metric.getTime()?.toDate() ?? new Date(0),
            key: `m-${pageOffset + index}`,
            record: metric,
          });
        }
      });
      return merged;
    };

    const fetchTelemetry = async () => {
      try {
        const firstResponse = await GetAllAssistantTelemetry(
          connectionConfig,
          buildRequest(requestPage, requestPageSize),
          ConnectionConfig.WithDebugger({
            authorization: token,
            userId: authId,
            projectId: projectId,
          }),
        );
        if (!active) return;

        const total = firstResponse.getPaginated()?.getTotalitem() ?? 0;
        const mergedRows = toTelemetryRows(firstResponse, 0);

        if (shouldFetchAllRows && total > requestPageSize) {
          const totalPages = Math.ceil(total / requestPageSize);
          for (let nextPage = 2; nextPage <= totalPages; nextPage += 1) {
            const response = await GetAllAssistantTelemetry(
              connectionConfig,
              buildRequest(nextPage, requestPageSize),
              ConnectionConfig.WithDebugger({
                authorization: token,
                userId: authId,
                projectId: projectId,
              }),
            );
            if (!active) return;
            mergedRows.push(
              ...toTelemetryRows(response, (nextPage - 1) * requestPageSize),
            );
          }
        }

        mergedRows.sort((a, b) => a.ts.getTime() - b.ts.getTime());
        setRows(mergedRows);
        setTotalItem(total);
      } catch {
        if (!active) return;
        setRows([]);
        setTotalItem(0);
      } finally {
        if (!active) return;
        setIsLoading(false);
      }
    };

    fetchTelemetry();

    return () => {
      active = false;
    };
  }, [
    token,
    authId,
    projectId,
    props.assistantId,
    JSON.stringify(chips),
    appliedConversationId,
    appliedMessageId,
    requestPageSize,
    requestPage,
    criteriaReady,
    shouldFetchAllRows,
  ]);

  const toggleRow = (key: string) => {
    setExpandedRows(prev => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const removeChip = (chipId: string) => {
    setChips(prev => prev.filter(c => c.id !== chipId));
    setPage(1);
  };

  const applyStructuredCriteria = (): boolean => {
    const nextConversationId = conversationIdInput.trim();
    const nextMessageId = messageIdInput.trim();
    if (nextConversationId && !/^\d+$/.test(nextConversationId)) {
      setStructuredError('Conversation ID must be numeric.');
      return false;
    }
    setStructuredError('');
    setAppliedConversationId(nextConversationId);
    setAppliedMessageId(nextMessageId);
    setPage(1);
    return true;
  };

  const resetStructuredCriteria = () => {
    setStructuredError('');
    setConversationIdInput('');
    setMessageIdInput('');
    setAppliedConversationId('');
    setAppliedMessageId('');
    setPage(1);
  };

  const EVENT_TYPES = [
    'session',
    'telephony',
    'webrtc',
    'stt',
    'llm',
    'tts',
    'vad',
    'eos',
    'denoise',
    'recording',
    'tool',
    'knowledge',
  ];

  const toggleFilter = (type: string) => {
    setActiveFilters(prev => {
      const next = new Set(prev);
      if (next.has(type)) next.delete(type);
      else next.add(type);
      return next;
    });
    setPage(1);
  };

  const getRowData = (row: TelemetryRow) => {
    if (row.kind === 'event') {
      const nameKey = normalizeComponentType(
        row.record.getName().split('.')[0],
      );
      return {
        typeLabel: row.record.getName(),
        tagType: EVENT_TAG_TYPE[nameKey] ?? 'gray',
        json: eventToJson(row.record),
      };
    }
    return {
      typeLabel: `metric·${row.record.getScope()}`,
      tagType: 'high-contrast',
      json: metricToJson(row.record),
    };
  };

  const filteredRows = rows.filter(row => {
    const { typeLabel, json } = getRowData(row);
    const matchesSearch =
      deferredSearchText.trim() === ''
        ? true
        : matchesTelemetrySearchDocument(
            getTelemetrySearchDocument(row, typeLabel, json),
            parsedSearchQuery,
          );
    const matchesFilter =
      activeFilters.size === 0
        ? true
        : activeFilters.has(
            row.kind === 'event'
              ? normalizeComponentType(row.record.getName().split('.')[0])
              : 'metric',
          );
    return matchesSearch && matchesFilter;
  });

  useEffect(() => {
    if (!shouldFetchAllRows) return;
    const maxPage = Math.max(1, Math.ceil(filteredRows.length / pageSize));
    if (page > maxPage) {
      setPage(maxPage);
    }
  }, [filteredRows.length, page, pageSize, shouldFetchAllRows]);

  const visibleRows = shouldFetchAllRows
    ? filteredRows.slice((page - 1) * pageSize, page * pageSize)
    : filteredRows;
  const totalItems = shouldFetchAllRows ? filteredRows.length : totalItem;

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="lg"
      preventCloseOnClickOutside
      containerClassName="!h-[90vh] !w-[90vw] !max-h-[90vh] !max-w-[90vw]"
    >
      <ModalHeader
        label="Observability"
        title="Telemetry Events"
        onClose={() => props.setModalOpen(false)}
      />
      <ModalBody className="!p-0 !overflow-hidden !flex !flex-col">
        {/* Toolbar */}
        <TableToolbar>
          <TableToolbarContent>
            <TableToolbarSearch
              placeholder='Search telemetry, e.g. type:metric scope:llm "timeout"'
              value={searchText}
              onChange={(e: any) => {
                setSearchText(e.target?.value || '');
                setPage(1);
              }}
            />
            <TableToolbarFilter
              filters={EVENT_TYPES.map(t => ({
                id: t,
                label: t.charAt(0).toUpperCase() + t.slice(1),
              }))}
              activeFilters={activeFilters}
              onApplyFilter={filters => {
                setActiveFilters(filters);
                setPage(1);
              }}
              onResetFilter={() => {
                setActiveFilters(new Set());
                setPage(1);
              }}
              onApply={applyStructuredCriteria}
              onReset={resetStructuredCriteria}
              extraContent={
                <Stack orientation="vertical" gap={6} className="py-2">
                  <TextInput
                    id="telemetry-criteria-conversation-id"
                    labelText="Conversation ID"
                    placeholder="Conversation ID"
                    value={conversationIdInput}
                    onChange={(e: any) =>
                      setConversationIdInput(e.target?.value || '')
                    }
                  />
                  <TextInput
                    id="telemetry-criteria-message-id"
                    labelText="Message / Context ID"
                    placeholder="Message ID or Context ID"
                    value={messageIdInput}
                    onChange={(e: any) =>
                      setMessageIdInput(e.target?.value || '')
                    }
                  />
                </Stack>
              }
            />
          </TableToolbarContent>
        </TableToolbar>

        {/* Active filter + criteria chips */}
        {(chips.length > 0 ||
          activeFilters.size > 0 ||
          appliedConversationId !== '' ||
          appliedMessageId !== '') && (
          <div className="flex flex-wrap gap-1.5 px-4 py-2 border-b border-gray-200 dark:border-gray-800">
            {appliedConversationId !== '' && (
              <DismissibleTag
                type="teal"
                text={`assistantConversationId: ${appliedConversationId}`}
                onClose={() => {
                  setConversationIdInput('');
                  setAppliedConversationId('');
                  setPage(1);
                }}
              />
            )}
            {appliedMessageId !== '' && (
              <DismissibleTag
                type="teal"
                text={`messageId/contextId: ${appliedMessageId}`}
                onClose={() => {
                  setMessageIdInput('');
                  setAppliedMessageId('');
                  setPage(1);
                }}
              />
            )}
            {chips.map(chip => (
              <DismissibleTag
                key={chip.id}
                type="blue"
                text={`${chip.field}: ${chip.value}`}
                onClose={() => removeChip(chip.id)}
              />
            ))}
            {Array.from(activeFilters).map(f => (
              <DismissibleTag
                key={f}
                type="cyan"
                text={f}
                onClose={() => toggleFilter(f)}
              />
            ))}
          </div>
        )}
        {structuredError !== '' && (
          <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-800 text-xs text-red-600 dark:text-red-400">
            {structuredError}
          </div>
        )}

        {/* Table */}
        <div className="flex-1 overflow-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loading withOverlay={false} small />
            </div>
          ) : visibleRows.length === 0 ? (
            <div className="flex items-center justify-center py-16 text-gray-400 dark:text-gray-500 text-sm">
              No telemetry events found
            </div>
          ) : (
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader className="!w-8" />
                  <TableHeader className="!w-[180px]">Time</TableHeader>
                  <TableHeader className="!w-[120px]">Type</TableHeader>
                  <TableHeader>Preview</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {visibleRows.map(row => {
                  const { typeLabel, tagType, json } = getRowData(row);
                  const isExpanded = expandedRows.has(row.key);
                  return (
                    <React.Fragment key={row.key}>
                      <TableRow
                        className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50"
                        onClick={() => toggleRow(row.key)}
                      >
                        <TableCell className="!w-8 !px-2">
                          <ChevronRight
                            size={16}
                            className={`transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                          />
                        </TableCell>
                        <TableCell className="!font-mono !text-xs tabular-nums whitespace-nowrap">
                          {formatDateTime(row.ts)}
                        </TableCell>
                        <TableCell className="!w-[120px]">
                          <Tag size="sm" type={tagType as any}>
                            {typeLabel}
                          </Tag>
                        </TableCell>
                        <TableCell className="!text-xs !text-gray-500 dark:!text-gray-400 truncate max-w-[300px]">
                          {JSON.stringify(json)}
                        </TableCell>
                      </TableRow>
                      {isExpanded && (
                        <TableExpandedRow colSpan={5}>
                          <CodeSnippet
                            type="multi"
                            feedback="Copied!"
                            className="!max-w-full"
                          >
                            {JSON.stringify(json, null, 2)}
                          </CodeSnippet>
                        </TableExpandedRow>
                      )}
                    </React.Fragment>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </div>

        {/* Pagination */}
        {totalItems > 0 && (
          <Pagination
            totalItems={totalItems}
            page={page}
            pageSize={pageSize}
            pageSizes={[25, 50, 100]}
            onChange={({ page: p, pageSize: ps }) => {
              setPageSize(ps);
              setPage(p);
            }}
          />
        )}
      </ModalBody>
    </Modal>
  );
}
