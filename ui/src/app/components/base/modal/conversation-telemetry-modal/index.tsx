import React, { useEffect, useState } from 'react';
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
import { Tabs } from '@/app/components/carbon/tabs';
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
  Dropdown,
  MultiSelect,
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

type TelemetryFilterState = {
  searchText: string;
  names: string[];
  messageOrContextId: string;
  eventDataType: string;
  metricScope: string;
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
  eventDataType: string;
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

const EVENT_NAME_OPTIONS = [
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
].map(id => ({
  id,
  label: id,
}));

const METRIC_SCOPE_OPTIONS = ['message', 'conversation'].map(id => ({
  id,
  label: id.charAt(0) + id.slice(1),
}));

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

export const matchesTelemetryFilters = (
  document: TelemetrySearchDocument,
  filters: TelemetryFilterState,
): boolean => {
  const normalizeSearchValue = (value?: string) =>
    String(value || '')
      .toLowerCase()
      .replace(/\s+/g, ' ')
      .trim();
  const compactSearchValue = (value?: string) =>
    String(value || '')
      .toLowerCase()
      .replace(/[\s"'`]+/g, '');
  const contains = (source: string, term: string) =>
    normalizeSearchValue(source).includes(normalizeSearchValue(term)) ||
    compactSearchValue(source).includes(compactSearchValue(term));
  const searchTerm = filters.searchText.trim();

  if (
    searchTerm &&
    !contains(document.typeLabel, searchTerm) &&
    !contains(document.rawText, searchTerm)
  ) {
    return false;
  }

  if (
    filters.names.length > 0 &&
    !filters.names.some(name => contains(document.name, name))
  ) {
    return false;
  }

  if (
    filters.messageOrContextId &&
    !contains(document.messageId, filters.messageOrContextId) &&
    !contains(document.contextId, filters.messageOrContextId)
  ) {
    return false;
  }

  if (
    filters.eventDataType &&
    !contains(document.eventDataType, filters.eventDataType)
  ) {
    return false;
  }

  if (filters.metricScope && !contains(document.scope, filters.metricScope))
    return false;

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
    const eventJson = json as { data?: Record<string, string> };
    return {
      kind: 'event',
      componentType: normalizeComponentType(row.record.getName().split('.')[0]),
      typeLabel,
      name: row.record.getName(),
      scope: '',
      conversationId: row.record.getAssistantconversationid(),
      messageId: row.record.getMessageid(),
      contextId: '',
      eventDataType: eventJson.data?.type || '',
      rawText: `${JSON.stringify(json)}\n${JSON.stringify(json, null, 2)}`,
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
    eventDataType: '',
    rawText: `${JSON.stringify(json)}\n${JSON.stringify(json, null, 2)}`,
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
  const [selectedTab, setSelectedTab] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalItem, setTotalItem] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [criteriaReady, setCriteriaReady] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [conversationIdInput, setConversationIdInput] = useState('');
  const [messageIdInput, setMessageIdInput] = useState('');
  const [eventNameInputs, setEventNameInputs] = useState<string[]>([]);
  const [eventDataTypeInput, setEventDataTypeInput] = useState('');
  const [metricScopeInput, setMetricScopeInput] = useState('');
  const [appliedConversationId, setAppliedConversationId] = useState('');
  const [appliedMessageId, setAppliedMessageId] = useState('');
  const [appliedEventNames, setAppliedEventNames] = useState<string[]>([]);
  const [appliedEventDataType, setAppliedEventDataType] = useState('');
  const [appliedMetricScope, setAppliedMetricScope] = useState('');
  const [structuredError, setStructuredError] = useState('');
  const activeTabKind: 'event' | 'metric' =
    selectedTab === 0 ? 'event' : 'metric';
  const hasSearchQuery = searchText.trim() !== '';
  const hasLocalFilters =
    activeTabKind === 'event'
      ? appliedEventNames.length > 0 || appliedEventDataType !== ''
      : appliedMetricScope !== '';
  const shouldFetchAllRows = hasSearchQuery || hasLocalFilters;
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
    setSelectedTab(0);
    setPage(1);
    setChips(initialChips);
    setSearchText('');
    setConversationIdInput(normalized.conversationId);
    setMessageIdInput(normalized.messageId);
    setEventNameInputs([]);
    setEventDataTypeInput('');
    setMetricScopeInput('');
    setAppliedConversationId(normalized.conversationId);
    setAppliedMessageId(normalized.messageId);
    setAppliedEventNames([]);
    setAppliedEventDataType('');
    setAppliedMetricScope('');
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

  const applyEventFilters = (): boolean => {
    if (!applyStructuredCriteria()) {
      return false;
    }
    setAppliedEventNames(eventNameInputs);
    setAppliedEventDataType(eventDataTypeInput.trim());
    setPage(1);
    return true;
  };

  const resetEventFilters = () => {
    setEventNameInputs([]);
    setEventDataTypeInput('');
    setAppliedEventNames([]);
    setAppliedEventDataType('');
    resetStructuredCriteria();
  };

  const applyMetricFilters = (): boolean => {
    setAppliedMetricScope(metricScopeInput);
    setPage(1);
    return true;
  };

  const resetMetricFilters = () => {
    setMetricScopeInput('');
    setAppliedMetricScope('');
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

  const getFilteredRows = (kind: 'event' | 'metric') =>
    rows.filter(row => {
      const { typeLabel, json } = getRowData(row);
      if (row.kind !== kind) {
        return false;
      }
      return matchesTelemetryFilters(
        getTelemetrySearchDocument(row, typeLabel, json),
        {
          searchText,
          names: kind === 'event' ? appliedEventNames : [],
          messageOrContextId: kind === 'event' ? appliedMessageId : '',
          eventDataType: kind === 'event' ? appliedEventDataType : '',
          metricScope: kind === 'metric' ? appliedMetricScope : '',
        },
      );
    });

  const filteredRows = getFilteredRows(activeTabKind);

  useEffect(() => {
    if (!shouldFetchAllRows) return;
    const maxPage = Math.max(1, Math.ceil(filteredRows.length / pageSize));
    if (page > maxPage) {
      setPage(maxPage);
    }
  }, [filteredRows.length, page, pageSize, shouldFetchAllRows]);

  useEffect(() => {
    setExpandedRows(new Set());
    setPage(1);
  }, [selectedTab]);

  const totalItems = shouldFetchAllRows ? filteredRows.length : totalItem;
  const renderTelemetryTable = (kind: 'event' | 'metric') => {
    const isActiveTab = activeTabKind === kind;
    const isEventTab = kind === 'event';
    const tabTitle = isEventTab ? 'Events' : 'Metrics';
    const tabRows = isActiveTab ? filteredRows : getFilteredRows(kind);
    const tabVisibleRows =
      isActiveTab && shouldFetchAllRows
        ? tabRows.slice((page - 1) * pageSize, page * pageSize)
        : tabRows;
    const tabTotalItems = isActiveTab
      ? totalItems
      : shouldFetchAllRows
        ? tabRows.length
        : tabRows.length;

    return (
      <>
        <TableToolbar>
          <TableToolbarContent>
            <TableToolbarSearch
              placeholder={`Search ${tabTitle.toLowerCase()} payload or text`}
              value={searchText}
              onChange={(e: any) => {
                setSearchText(e.target?.value || '');
                setPage(1);
              }}
            />
            <TableToolbarFilter
              panelClassName="!w-[48rem] max-w-[calc(100vw-4rem)]"
              filters={[]}
              activeFilters={new Set()}
              onApplyFilter={() => {}}
              onResetFilter={() => {}}
              onApply={isEventTab ? applyEventFilters : applyMetricFilters}
              onReset={isEventTab ? resetEventFilters : resetMetricFilters}
              extraContent={
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  {isEventTab ? (
                    <>
                      <MultiSelect
                        id="telemetry-filter-event-name"
                        titleText="Name"
                        label="Choose names"
                        items={EVENT_NAME_OPTIONS}
                        itemToString={(item: any) => item?.label || ''}
                        selectedItems={EVENT_NAME_OPTIONS.filter(item =>
                          eventNameInputs.includes(item.id),
                        )}
                        onChange={({ selectedItems }: any) =>
                          setEventNameInputs(
                            selectedItems.map((item: any) => item.id),
                          )
                        }
                      />
                      <TextInput
                        id="telemetry-filter-event-message-id"
                        labelText="MessageID / ContextID"
                        placeholder="MessageID or ContextID"
                        value={messageIdInput}
                        onChange={(e: any) =>
                          setMessageIdInput(e.target?.value || '')
                        }
                      />
                      <TextInput
                        id="telemetry-filter-event-data-type"
                        labelText="Type"
                        placeholder="Type"
                        value={eventDataTypeInput}
                        onChange={(e: any) =>
                          setEventDataTypeInput(e.target?.value || '')
                        }
                      />
                    </>
                  ) : (
                    <Dropdown
                      id="telemetry-filter-metric-scope"
                      titleText="Scope"
                      label="Choose scope"
                      items={METRIC_SCOPE_OPTIONS}
                      itemToString={(item: any) => item?.label || ''}
                      selectedItem={
                        METRIC_SCOPE_OPTIONS.find(
                          item => item.id === metricScopeInput,
                        ) || null
                      }
                      onChange={({ selectedItem }: any) =>
                        setMetricScopeInput(selectedItem?.id || '')
                      }
                    />
                  )}
                </div>
              }
            />
          </TableToolbarContent>
        </TableToolbar>

        {(chips.length > 0 ||
          (isEventTab
            ? appliedEventNames.length > 0 ||
              appliedMessageId !== '' ||
              appliedEventDataType !== ''
            : appliedMetricScope !== '') ||
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
            {isEventTab &&
              appliedEventNames.map(appliedEventName => (
                <DismissibleTag
                  key={appliedEventName}
                  type="cyan"
                  text={`name: ${EVENT_NAME_OPTIONS.find(option => option.id === appliedEventName)?.label || appliedEventName}`}
                  onClose={() => {
                    setEventNameInputs(prev =>
                      prev.filter(value => value !== appliedEventName),
                    );
                    setAppliedEventNames(prev =>
                      prev.filter(value => value !== appliedEventName),
                    );
                    setPage(1);
                  }}
                />
              ))}
            {!isEventTab && appliedMetricScope !== '' && (
              <DismissibleTag
                type="cyan"
                text={`scope: ${appliedMetricScope}`}
                onClose={() => {
                  setMetricScopeInput('');
                  setAppliedMetricScope('');
                  setPage(1);
                }}
              />
            )}
            {isEventTab && appliedEventDataType !== '' && (
              <DismissibleTag
                type="cyan"
                text={`data.type: ${appliedEventDataType}`}
                onClose={() => {
                  setEventDataTypeInput('');
                  setAppliedEventDataType('');
                  setPage(1);
                }}
              />
            )}
          </div>
        )}
        {structuredError !== '' && (
          <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-800 text-xs text-red-600 dark:text-red-400">
            {structuredError}
          </div>
        )}

        <div className="flex-1 overflow-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loading withOverlay={false} small />
            </div>
          ) : tabVisibleRows.length === 0 ? (
            <div className="flex items-center justify-center py-16 text-gray-400 dark:text-gray-500 text-sm">
              No {tabTitle.toLowerCase()} found
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
                {tabVisibleRows.map(row => {
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

        {isActiveTab && tabTotalItems > 0 && (
          <Pagination
            totalItems={tabTotalItems}
            page={page}
            pageSize={pageSize}
            pageSizes={[25, 50, 100]}
            onChange={({ page: p, pageSize: ps }) => {
              setPageSize(ps);
              setPage(p);
            }}
          />
        )}
      </>
    );
  };

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
        <Tabs
          tabs={['Events', 'Metrics']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          fill
          aria-label="Telemetry tabs"
          panelClassName="!p-0"
        >
          <div className="flex flex-1 min-h-0 flex-col">
            {renderTelemetryTable('event')}
          </div>
          <div className="flex flex-1 min-h-0 flex-col">
            {renderTelemetryTable('metric')}
          </div>
        </Tabs>
      </ModalBody>
    </Modal>
  );
}
