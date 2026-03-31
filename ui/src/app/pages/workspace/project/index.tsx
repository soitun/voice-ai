import { useCallback, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/helmet';
import {
  ArchiveProjectResponse,
  GetAllProjectResponse,
  Project,
} from '@rapidaai/react';
import { CreateProjectDialog } from '@/app/components/base/modal/create-project-modal';
import { GetAllProject, DeleteProject } from '@rapidaai/react';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { ServiceError } from '@rapidaai/react';
import { PrimaryButton } from '@/app/components/carbon/button';
import { Pagination } from '@/app/components/carbon/pagination';
import { Add, Renew } from '@carbon/icons-react';
import IconIndicator from '@carbon/react/es/components/IconIndicator';
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
  Button,
} from '@carbon/react';
import { ProjectUserGroupAvatar } from '@/app/components/avatar/project-user-group-avatar';
import { toHumanReadableDate } from '@/utils/date';
import { RoleIndicator } from '@/app/components/indicators/role';
import { ProjectOption } from '@/app/pages/workspace/project/project-options';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import { TableSection } from '@/app/components/sections/table-section';
import { connectionConfig } from '@/configs';

const statusMap: Record<string, { kind: string; label: string }> = {
  ACTIVE: { kind: 'succeeded', label: 'Active' },
  active: { kind: 'succeeded', label: 'Active' },
  DISABLED: { kind: 'failed', label: 'Disabled' },
  disabled: { kind: 'failed', label: 'Disabled' },
  INACTIVE: { kind: 'incomplete', label: 'Inactive' },
  inactive: { kind: 'incomplete', label: 'Inactive' },
  PENDING: { kind: 'pending', label: 'Pending' },
  pending: { kind: 'pending', label: 'Pending' },
  ARCHIVED: { kind: 'undefined', label: 'Archived' },
  archived: { kind: 'undefined', label: 'Archived' },
};

const defaultStatus = { kind: 'normal', label: 'Active' };

function getStatusKind(status?: string) {
  return (statusMap[status || ''] || defaultStatus).kind as any;
}

function getStatusLabel(status?: string) {
  return (statusMap[status || ''] || defaultStatus).label;
}

const headers = [
  { key: 'name', header: 'Name' },
  { key: 'createdDate', header: 'Date Created' },
  { key: 'role', header: 'Your Role' },
  { key: 'collaborators', header: 'Collaborators' },
  { key: 'status', header: 'Status' },
  { key: 'actions', header: '' },
];

export function ProjectPage() {
  const [createProjectModalOpen, setCreateProjectModalOpen] = useState(false);
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token] = useCredential();
  const [projects, setProjects] = useState<Project[]>([]);
  const [page, setPage] = useState<number>(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalCount, setTotalCount] = useState(0);
  const [criteria] = useState<{ key: string; value: string }[]>([]);

  const afterGettingProject = useCallback(
    (err: ServiceError | null, alpr: GetAllProjectResponse | null) => {
      hideLoader();
      if (err) {
        toast.error('Unable to process your request. please try again later.');
        return;
      }
      if (alpr?.getSuccess()) {
        setProjects(alpr.getDataList());
        let paginated = alpr.getPaginated();
        if (paginated) {
          setTotalCount(paginated.getTotalitem());
        }
      }
    },
    [],
  );

  const getAllProject = (
    page: number,
    pageSize: number,
    criteria: { key: string; value: string }[],
  ) => {
    showLoader();
    return GetAllProject(
      connectionConfig,
      page,
      pageSize,
      criteria,
      afterGettingProject,
      {
        authorization: token,
        'x-auth-id': userId,
      },
    );
  };

  useEffect(() => {
    getAllProject(page, pageSize, criteria);
  }, [page, pageSize, criteria]);

  const onDeleteProject = (projectId: string) => {
    DeleteProject(
      connectionConfig,
      projectId,
      (err: ServiceError | null, apr: ArchiveProjectResponse | null) => {
        if (err) {
          return;
        }
        if (apr?.getSuccess()) {
          const newList = projects?.filter(p => p.getId() !== apr.getId());
          setProjects(newList);
        }
      },
      {
        authorization: token,
        'x-auth-id': userId,
      },
    );
  };

  return (
    <>
      <Helmet title="Projects" />
      <PageHeaderBlock>
        <PageTitleWithCount count={projects.length} total={totalCount}>
          Projects
        </PageTitleWithCount>
      </PageHeaderBlock>
      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch placeholder="Search projects..." />
          <Button
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh"
            kind="ghost"
            onClick={() => getAllProject(page, pageSize, criteria)}
            tooltipPosition="bottom"
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            isLoading={loading}
            onClick={() => setCreateProjectModalOpen(true)}
          >
            Create new project
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
            {projects.map(project => (
              <TableRow key={project.getId()}>
                <TableCell>{project.getName()}</TableCell>
                <TableCell>
                  {project.getCreateddate() &&
                    toHumanReadableDate(project.getCreateddate()!)}
                </TableCell>
                <TableCell>
                  <RoleIndicator role={'SUPER_ADMIN'} />
                </TableCell>
                <TableCell>
                  <ProjectUserGroupAvatar
                    members={project
                      .getMembersList()
                      .map(m => ({ name: m.getName() }))}
                    size={7}
                    projectId={project.getId()}
                  />
                </TableCell>
                <TableCell>
                  <IconIndicator
                    kind={getStatusKind(project.getStatus?.())}
                    label={getStatusLabel(project.getStatus?.())}
                    size={16}
                  />
                </TableCell>
                <TableCell>
                  <ProjectOption
                    project={project.toObject()}
                    afterUpdateProject={() => {
                      getAllProject(page, pageSize, criteria);
                    }}
                    onDelete={() => onDeleteProject(project.getId())}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination
          totalItems={totalCount}
          page={page}
          pageSize={pageSize}
          pageSizes={[10, 20, 50]}
          onChange={({ page: newPage, pageSize: newSize }) => {
            setPage(newPage);
            setPageSize(newSize);
          }}
        />
      </TableSection>
      <CreateProjectDialog
        modalOpen={createProjectModalOpen}
        setModalOpen={setCreateProjectModalOpen}
        afterCreateProject={() => {
          getAllProject(page, pageSize, criteria);
        }}
      />
    </>
  );
}
