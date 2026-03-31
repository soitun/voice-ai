import { useEffect, useState, useCallback } from 'react';
import { Helmet } from '@/app/components/helmet';
import { InviteUserDialog } from '@/app/components/base/modal/invite-user-modal';
import { User } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { useUserPageStore } from '@/hooks';
import { SingleUser } from '@/app/pages/workspace/user/single-user';
import { PrimaryButton } from '@/app/components/carbon/button';
import { Pagination } from '@/app/components/carbon/pagination';
import { Add, Renew } from '@carbon/icons-react';
import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Button,
} from '@carbon/react';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import { TableSection } from '@/app/components/sections/table-section';

const headers = [
  { key: 'id', header: 'ID' },
  { key: 'name', header: 'Name' },
  { key: 'email', header: 'Email' },
  { key: 'role', header: 'Role' },
  { key: 'createdDate', header: 'Date Created' },
  { key: 'status', header: 'Status' },
  { key: 'actions', header: '' },
];

export function UserPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [createUserModalOpen, setCreateUserModalOpen] = useState(false);
  const { projectId, authId, token } = useCurrentCredential();
  const userActions = useUserPageStore();

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((data: User[]) => {
    hideLoader();
  }, []);

  const getUsers = useCallback((token, userId, projectId) => {
    showLoader();
    userActions.getAllUser(token, userId, projectId, onError, onSuccess);
  }, []);

  useEffect(() => {
    getUsers(token, authId, projectId);
  }, [userActions.page, userActions.pageSize, userActions.criteria]);

  return (
    <>
      <Helmet title="User and Teams" />
      <PageHeaderBlock>
        <PageTitleWithCount
          count={userActions.users.length}
          total={userActions.totalCount}
        >
          Users
        </PageTitleWithCount>
      </PageHeaderBlock>
      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch placeholder="Search users..." />
          <Button
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh"
            kind="ghost"
            onClick={() => getUsers(token, authId, projectId)}
            tooltipPosition="bottom"
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            isLoading={loading}
            onClick={() => setCreateUserModalOpen(true)}
          >
            Invite user
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>
      <TableSection>
        <Table>
          <TableHead>
            <TableRow>
              {headers.map(h => (
                <TableHeader key={h.key}>{h.header}</TableHeader>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {userActions.users.map((usr, idx) => (
              <SingleUser key={idx} user={usr} />
            ))}
          </TableBody>
        </Table>
        <Pagination
          totalItems={userActions.totalCount}
          page={userActions.page}
          pageSize={userActions.pageSize}
          pageSizes={[10, 20, 50]}
          onChange={({ page, pageSize }) => {
            if (pageSize !== userActions.pageSize) {
              userActions.setPageSize(pageSize);
            } else {
              userActions.setPage(page);
            }
          }}
        />
      </TableSection>
      <InviteUserDialog
        modalOpen={createUserModalOpen}
        setModalOpen={setCreateUserModalOpen}
        onSuccess={() => {
          getUsers(token, authId, projectId);
        }}
      />
    </>
  );
}
