import { useAssistantProviderPageStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Assistant, GetAllAssistantProviderResponse } from '@rapidaai/react';
import { SectionLoader } from '@/app/components/loader/section-loader';
import { TableSection } from '@/app/components/sections/table-section';
import { Pagination } from '@/app/components/carbon/pagination';
import { toHumanReadableDateTime } from '@/utils/date';
import {
  Tag,
  Button,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  TableSelectRow,
  TableSelectAll,
  TableBatchActions,
  TableBatchAction,
  RadioButton,
} from '@carbon/react';
import { Copy, Checkmark, Rocket, Renew } from '@carbon/icons-react';
import IconIndicator from '@carbon/react/es/components/IconIndicator';

interface VersionProps {
  assistant: Assistant;
  onReload: () => void;
}

const headers = [
  { key: 'version', header: 'Version' },
  { key: 'type', header: 'Type' },
  { key: 'description', header: 'Description' },
  { key: 'status', header: 'Status' },
  { key: 'createdBy', header: 'Created By' },
  { key: 'createdDate', header: 'Created' },
];

function VersionId({ id }: { id: string }) {
  const [copied, setCopied] = useState(false);
  const version = `vrsn_${id}`;
  const copy = () => {
    navigator.clipboard.writeText(version);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <span className="inline-flex items-center gap-1 font-mono text-xs text-gray-600 dark:text-gray-400">
      {version}
      <Button
        hasIconOnly
        renderIcon={copied ? Checkmark : Copy}
        iconDescription="Copy"
        kind="ghost"
        size="sm"
        onClick={copy}
        className="!min-h-0 !p-1"
      />
    </span>
  );
}

export function Version(props: VersionProps) {
  const [userId, token, projectId] = useCredential();
  const assistantProviderAction = useAssistantProviderPageStore();
  const [isFetching, setIsFetching] = useState(true);
  const [deployingProviderId, setDeployingProviderId] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(null);

  useEffect(() => {
    setIsFetching(true);
    assistantProviderAction.onChangeAssistant(props.assistant);
    assistantProviderAction.getAssistantProviders(
      props.assistant.getId(),
      projectId,
      token,
      userId,
      (err: string) => {
        setIsFetching(false);
        toast.error(err);
      },
      data => {
        setIsFetching(false);
      },
    );
  }, [
    props.assistant.getId(),
    projectId,
    assistantProviderAction.page,
    assistantProviderAction.pageSize,
    assistantProviderAction.criteria,
  ]);

  const deployRevision = (
    assistantProvider: string,
    assistantProviderId: string,
  ) => {
    setDeployingProviderId(assistantProviderId);
    assistantProviderAction.onReleaseVersion(
      assistantProvider,
      assistantProviderId,
      projectId,
      token,
      userId,
      error => {
        setDeployingProviderId(null);
        toast.error(error);
      },
      e => {
        toast.success('New version of assistant has been deployed successfully.');
        assistantProviderAction.onChangeAssistant(e);
        props.onReload();
        setDeployingProviderId(null);
      },
    );
  };

  if (isFetching) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  const getProviderData = (apm: any) => {
    const caseType = apm.getAssistantproviderCase();
    const Cases = GetAllAssistantProviderResponse.AssistantProvider.AssistantproviderCase;

    switch (caseType) {
      case Cases.ASSISTANTPROVIDERMODEL: {
        const m = apm.getAssistantprovidermodel();
        return {
          id: m?.getId()!,
          type: 'LLM',
          typeColor: 'blue' as const,
          description: m?.getDescription() || 'Initial assistant version',
          createdBy: m?.getCreateduser()?.getName() || '',
          createdDate: m?.getCreateddate(),
          deployType: 'MODEL',
        };
      }
      case Cases.ASSISTANTPROVIDERAGENTKIT: {
        const a = apm.getAssistantprovideragentkit();
        return {
          id: a?.getId()!,
          type: 'AgentKit',
          typeColor: 'purple' as const,
          description: a?.getDescription() || 'Initial assistant version',
          createdBy: a?.getCreateduser()?.getName() || '',
          createdDate: a?.getCreateddate(),
          deployType: 'AGENTKIT',
        };
      }
      case Cases.ASSISTANTPROVIDERWEBSOCKET: {
        const w = apm.getAssistantproviderwebsocket();
        return {
          id: w?.getId()!,
          type: 'WebSocket',
          typeColor: 'teal' as const,
          description: w?.getDescription() || 'Initial assistant version',
          createdBy: w?.getCreateduser()?.getName() || '',
          createdDate: w?.getCreateddate(),
          deployType: 'WEBSOCKET',
        };
      }
      default:
        return null;
    }
  };

  const refresh = () => {
    setIsFetching(true);
    assistantProviderAction.getAssistantProviders(
      props.assistant.getId(),
      projectId,
      token,
      userId,
      (err: string) => { setIsFetching(false); toast.error(err); },
      () => { setIsFetching(false); },
    );
  };

  const allRows = assistantProviderAction.assistantProviders
    .map(apm => ({ apm, data: getProviderData(apm) }))
    .filter(({ data }) => data !== null) as { apm: any; data: NonNullable<ReturnType<typeof getProviderData>> }[];

  const filteredRows = searchTerm
    ? allRows.filter(({ data }) =>
        `vrsn_${data.id}`.toLowerCase().includes(searchTerm.toLowerCase()) ||
        data.type.toLowerCase().includes(searchTerm.toLowerCase()) ||
        data.description.toLowerCase().includes(searchTerm.toLowerCase()),
      )
    : allRows;

  return (
    <TableSection>
      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={!!selectedVersionId}
          totalSelected={selectedVersionId ? 1 : 0}
          onCancel={() => setSelectedVersionId(null)}
          totalCount={filteredRows.length}
        >
          <TableBatchAction
            renderIcon={Rocket}
            kind="ghost"
            onClick={() => {
              const row = filteredRows.find(r => r.data.id === selectedVersionId);
              if (row) {
                deployRevision(row.data.deployType, row.data.id);
                setSelectedVersionId(null);
              }
            }}
          >
            Deploy version
          </TableBatchAction>
        </TableBatchActions>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search versions..."
            onChange={(e: any) => setSearchTerm(e.target?.value || '')}
          />
          <Button
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh"
            kind="ghost"
            onClick={refresh}
            tooltipPosition="bottom"
          />
        </TableToolbarContent>
      </TableToolbar>
      <Table>
        <TableHead>
          <TableRow>
            <TableHeader className="!w-12" />
            {headers.map(h => (
              <TableHeader key={h.key}>{h.header}</TableHeader>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {filteredRows.map(({ data }, idx) => {
            const isCurrent =
              assistantProviderAction.assistant?.getAssistantproviderid() === data.id;
            const isDeploying = deployingProviderId === data.id;

            return (
              <TableRow
                key={idx}
                isSelected={selectedVersionId === data.id}
                onClick={() => !isCurrent && setSelectedVersionId(selectedVersionId === data.id ? null : data.id)}
                className={!isCurrent ? 'cursor-pointer' : ''}
              >
                <TableCell className="!w-12 !pr-0">
                  <RadioButton
                    id={`version-select-${data.id}`}
                    name="version-select"
                    labelText=""
                    hideLabel
                    checked={selectedVersionId === data.id}
                    onClick={() =>
                      setSelectedVersionId(
                        selectedVersionId === data.id ? null : data.id,
                      )
                    }
                    disabled={isCurrent}
                  />
                </TableCell>
                <TableCell>
                  <VersionId id={data.id} />
                </TableCell>
                <TableCell>
                  <Tag type={data.typeColor} size="sm">{data.type}</Tag>
                </TableCell>
                <TableCell>{data.description}</TableCell>
                <TableCell>
                  {isCurrent ? (
                    <IconIndicator kind="succeeded" label="In use" size={16} />
                  ) : isDeploying ? (
                    <IconIndicator kind="in-progress" label="Deploying" size={16} />
                  ) : (
                    <IconIndicator kind="incomplete" label="Available" size={16} />
                  )}
                </TableCell>
                <TableCell>{data.createdBy}</TableCell>
                <TableCell>
                  {data.createdDate && toHumanReadableDateTime(data.createdDate)}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      <Pagination
        totalItems={assistantProviderAction.totalCount}
        page={assistantProviderAction.page}
        pageSize={assistantProviderAction.pageSize}
        pageSizes={[10, 20, 50]}
        onChange={({ page, pageSize }) => {
          assistantProviderAction.setPage(page);
          if (pageSize !== assistantProviderAction.pageSize)
            assistantProviderAction.setPageSize(pageSize);
        }}
      />
    </TableSection>
  );
}
