import { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/helmet';
import { DateFilter } from '@/app/components/carbon/date-filter';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { formatNanoToReadableMilli, toDateString, toHumanReadableDateTime } from '@/utils/date';
import { HttpStatusSpanIndicator } from '@/app/components/indicators/http-status';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import { useWebhookLogPage } from '@/hooks/use-webhook-log-page-store';
import { WebhookLogDialog } from '@/app/components/base/modal/webhook-log-modal';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';

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
  Tag,
} from '@carbon/react';
import { TableLink } from '@/app/components/carbon/table-link';
import { Pagination } from '@/app/components/carbon/pagination';
import { IconOnlyButton } from '@/app/components/carbon/button';
import { Renew, View, EventSchedule } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/carbon/empty-state';

export function ListingPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token, projectId] = useCredential();
  const [currentActivityId, setCurrentActivityId] = useState('');
  const [showLogModal, setShowLogModal] = useState(false);

  const {
    getActivities,
    addCriterias,
    webhookLogs,
    onChangeActivities,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    visibleColumn,
    setPageSize,
    setColumns,
  } = useWebhookLogPage();

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
        onChangeActivities(logs);
      },
    );
  };

  const visibleColumns = columns.filter(c => c.visible);

  return (
    <>
      {currentActivityId && (
        <WebhookLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentWebhookId={currentActivityId}
        />
      )}

      <Helmet title="Webhook Logs" />
      <PageHeaderBlock>
        <PageTitleWithCount count={webhookLogs.length} total={totalCount}>
          Webhook Logs
        </PageTitleWithCount>
      </PageHeaderBlock>

      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch placeholder="Search webhook logs" />
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
      ) : webhookLogs.length > 0 ? (
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
              {webhookLogs.map((at, idx) => (
                <TableRow key={idx}>
                  {visibleColumn('webhookid') && (
                    <TableCell>
                      <TableLink href={`/deployment/assistant/${at.getAssistantid()}/manage/configure-webhook`}>
                        {at.getWebhookid()}
                      </TableLink>
                    </TableCell>
                  )}
                  {visibleColumn('sessionid') && (
                    <TableCell>
                      <TableLink href={`/deployment/assistant/${at.getAssistantid()}/sessions/${at.getAssistantconversationid()}`}>
                        {at.getAssistantconversationid()}
                      </TableLink>
                    </TableCell>
                  )}
                  {visibleColumn('event') && (
                    <TableCell>
                      <Tag size="sm" type="blue">
                        {at.getEvent()}
                      </Tag>
                    </TableCell>
                  )}
                  {visibleColumn('endpoint') && (
                    <TableCell className="!text-xs">
                      {at.getHttpmethod()}:{at.getHttpurl()}
                    </TableCell>
                  )}
                  {visibleColumn('responsestatus') && (
                    <TableCell>
                      <HttpStatusSpanIndicator status={Number(at.getResponsestatus())} />
                    </TableCell>
                  )}
                  {visibleColumn('timetaken') && (
                    <TableCell className="!font-mono !text-xs">
                      {formatNanoToReadableMilli(at.getTimetaken())}
                    </TableCell>
                  )}
                  {visibleColumn('retrycount') && (
                    <TableCell className="!text-xs">{at.getRetrycount()}</TableCell>
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
          icon={EventSchedule}
          title="No webhook logs found"
          subtitle="Webhook activities triggered by your assistant conversations will appear here once webhooks are configured and events are fired."
        />
      )}

      {webhookLogs.length > 0 && (
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
