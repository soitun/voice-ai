import { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/helmet';
import { DateFilter } from '@/app/components/carbon/date-filter';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { formatNanoToReadableMilli, toDateString, toDate } from '@/utils/date';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { useToolActivityLogPage } from '@/hooks/use-tool-activity-log-page-store';
import { ToolLogDialog } from '@/app/components/base/modal/tool-log-modal';
import { CarbonStatusIndicator } from '@/app/components/carbon/status-indicator';
import { Pagination } from '@/app/components/carbon/pagination';
import { IconOnlyButton } from '@/app/components/carbon/button';

import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Loading,
  DefinitionTooltip,
} from '@carbon/react';
import { TableLink } from '@/app/components/carbon/table-link';
import { Renew, View, Launch, ToolKit } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/carbon/empty-state';

export function ListingPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token, projectId] = useCredential();
  const [currentActivityId, setCurrentActivityId] = useState('');
  const [showLogModal, setShowLogModal] = useState(false);

  const {
    getActivities,
    addCriterias,
    activities,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    visibleColumn,
    setPageSize,
    setColumns,
  } = useToolActivityLogPage();

  const onDateSelect = (to: Date, from: Date) => {
    addCriterias([
      { k: 'created_date', v: toDateString(to), logic: '<=' },
      { k: 'created_date', v: toDateString(from), logic: '>=' },
    ]);
  };

  useEffect(() => {
    showLoader();
    onGetActivities();
  }, [projectId, page, pageSize, JSON.stringify(criteria)]);

  const onGetActivities = () => {
    getActivities(
      projectId,
      token,
      userId,
      err => {
        hideLoader();
        toast.error(err);
      },
      logs => {
        hideLoader();
      },
    );
  };

  const visibleColumns = columns.filter(c => c.visible);

  return (
    <>
      {currentActivityId && (
        <ToolLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentActivityId={currentActivityId}
        />
      )}

      <Helmet title="Tool Logs" />
      <PageHeaderBlock>
        <PageTitleWithCount count={activities.length} total={totalCount}>
          Tool Logs
        </PageTitleWithCount>
      </PageHeaderBlock>

      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch placeholder="Search tool logs" />
          <DateFilter
            onApply={(from, to) => onDateSelect(to, from)}
            onReset={() => addCriterias([])}
          />
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={() => onGetActivities()}
          />
        </TableToolbarContent>
      </TableToolbar>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <Loading withOverlay={false} small />
        </div>
      ) : activities.length > 0 ? (
        <div className="overflow-auto flex-1">
          <Table>
            <TableHead>
              <TableRow>
                {visibleColumns.map(col => (
                  <TableHeader key={col.key}>{col.name}</TableHeader>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {activities.map((at, idx) => (
                <TableRow key={idx}>
                  {visibleColumn('assistant_id') && (
                    <TableCell>
                      <TableLink href={`/deployment/assistant/${at.getAssistantid()}`}>
                        {at.getAssistantid()}
                      </TableLink>
                    </TableCell>
                  )}
                  {visibleColumn('assistant_conversation_id') && (
                    <TableCell>
                      <TableLink href={`/deployment/assistant/${at.getAssistantid()}/sessions/${at.getAssistantconversationid()}`}>
                        {at.getAssistantconversationid()}
                      </TableLink>
                    </TableCell>
                  )}
                  {visibleColumn('assistant_tool_name') && (
                    <TableCell>{at.getAssistanttoolname()}</TableCell>
                  )}
                  {visibleColumn('tool_call_id') && (
                    <TableCell>
                      <span className="font-mono">{at.getToolcallid()}</span>
                    </TableCell>
                  )}
                  {visibleColumn('action') && (
                    <TableCell>
                      <div className="flex items-center gap-0">
                        <IconOnlyButton
                          kind="ghost"
                          size="md"
                          renderIcon={View}
                          iconDescription="View detail"
                          onClick={() => {
                            setCurrentActivityId(at.getId());
                            setShowLogModal(true);
                          }}
                        />
                        <IconOnlyButton
                          kind="ghost"
                          size="md"
                          renderIcon={Launch}
                          iconDescription="View conversation"
                          onClick={() => {
                            window.location.href = `/deployment/assistant/${at.getAssistantid()}/sessions/${at.getAssistantconversationid()}`;
                          }}
                        />
                      </div>
                    </TableCell>
                  )}
                  {visibleColumn('status') && (
                    <TableCell>
                      <CarbonStatusIndicator state={at.getStatus()} />
                    </TableCell>
                  )}
                  {visibleColumn('time_taken') && (
                    <TableCell className="!font-mono !text-xs">
                      {formatNanoToReadableMilli(at.getTimetaken())}
                    </TableCell>
                  )}
                  {visibleColumn('created_date') && (
                    <TableCell className="!text-xs whitespace-nowrap">
                      {at.getCreateddate() && (
                        <DefinitionTooltip
                          definition={toDate(at.getCreateddate()!).toUTCString()}
                          openOnHover
                        >
                          {toDate(at.getCreateddate()!).toLocaleString()}
                        </DefinitionTooltip>
                      )}
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <EmptyState
          icon={ToolKit}
          title="No tool activity logs found"
          subtitle="Tool calls made by your assistants during conversations will appear here once tools are configured and invoked."
        />
      )}

      {activities.length > 0 && (
        <Pagination
          totalItems={totalCount}
          page={page}
          pageSize={pageSize}
          pageSizes={[10, 20, 25, 50, 100]}
          onChange={({ page: p, pageSize: ps }) => {
            if (ps !== pageSize) setPageSize(ps);
            else setPage(p);
          }}
        />
      )}
    </>
  );
}
