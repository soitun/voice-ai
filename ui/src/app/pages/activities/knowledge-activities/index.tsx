import { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/helmet';
import { DateFilter } from '@/app/components/carbon/date-filter';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { formatNanoToReadableMilli, toDateString, toHumanReadableDateTime } from '@/utils/date';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { useKnowledgeActivityLogPage } from '@/hooks/use-knowledge-activity-log-page-store';
import { KnowledgeLogDialog } from '@/app/components/base/modal/knowledge-log-modal';
import { CarbonStatusIndicator } from '@/app/components/carbon/status-indicator';
import { Pagination } from '@/app/components/carbon/pagination';
import { IconOnlyButton } from '@/app/components/carbon/button';
import { Renew, View, DataBase } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/carbon/empty-state';

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
} from '@carbon/react';
import { TableLink } from '@/app/components/carbon/table-link';

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
  } = useKnowledgeActivityLogPage();

  const onDateSelect = (to: Date, from: Date) => {
    addCriterias([
      { k: 'created_date', v: toDateString(to), logic: '<=' },
      { k: 'created_date', v: toDateString(from), logic: '>=' },
    ]);
  };

  useEffect(() => {
    showLoader();
    onGetAcitvities();
  }, [projectId, page, pageSize, JSON.stringify(criteria)]);

  const onGetAcitvities = () => {
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
        <KnowledgeLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentActivityId={currentActivityId}
        />
      )}

      <Helmet title="Knowledge Logs" />
      <PageHeaderBlock>
        <PageTitleWithCount count={activities.length} total={totalCount}>
          Knowledge Logs
        </PageTitleWithCount>
      </PageHeaderBlock>

      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch placeholder="Search knowledge logs" />
          <DateFilter
            onApply={(from, to) => onDateSelect(to, from)}
            onReset={() => addCriterias([])}
          />
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={() => onGetAcitvities()}
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
                <TableHeader>Actions</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {activities.map((at, idx) => (
                <TableRow key={idx}>
                  {visibleColumn('knowledge_id') && (
                    <TableCell>
                      <TableLink href={`/knowledge/${at.getKnowledgeid()}`}>
                        {at.getKnowledgeid()}
                      </TableLink>
                    </TableCell>
                  )}
                  {visibleColumn('retrieval_method') && (
                    <TableCell>{at.getRetrievalmethod()}</TableCell>
                  )}
                  {visibleColumn('top_k') && (
                    <TableCell>{at.getTopk()}</TableCell>
                  )}
                  {visibleColumn('score_threshold') && (
                    <TableCell>{at.getScorethreshold()}</TableCell>
                  )}
                  {visibleColumn('document_count') && (
                    <TableCell>{at.getDocumentcount()}</TableCell>
                  )}
                  {visibleColumn('time_taken') && (
                    <TableCell className="!font-mono !text-xs">
                      {formatNanoToReadableMilli(at.getTimetaken())}
                    </TableCell>
                  )}
                  {visibleColumn('status') && (
                    <TableCell>
                      <CarbonStatusIndicator state={at.getStatus()} />
                    </TableCell>
                  )}
                  {visibleColumn('created_date') && (
                    <TableCell className="!font-mono !text-xs whitespace-nowrap">
                      {at.getCreateddate() && toHumanReadableDateTime(at.getCreateddate()!)}
                    </TableCell>
                  )}
                  <TableCell>
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
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <EmptyState
          icon={DataBase}
          title="No knowledge activities found"
          subtitle="Knowledge base retrieval activities will appear here once your assistants query connected knowledge sources."
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
