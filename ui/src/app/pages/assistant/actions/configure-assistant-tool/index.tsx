import { useRapidaStore } from '@/hooks';
import { FC, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { Add, Renew, Edit, TrashCan, ToolKit } from '@carbon/icons-react';
import { CreateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/create-assistant-tool';
import { SectionLoader } from '@/app/components/loader/section-loader';
import { useAssistantToolPageStore } from '@/app/pages/assistant/actions/store/use-tool-page-store';
import { useCurrentCredential } from '@/hooks/use-credential';
import { UpdateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/update-assistant-tool';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { IconOnlyButton, PrimaryButton } from '@/app/components/carbon/button';
import { BUILDIN_TOOLS } from '@/llm-tools';
import {
  getToolConditionSource,
  getToolConditionSourceLabel,
} from '@/app/components/tools/common';
import {
  Breadcrumb,
  BreadcrumbItem,
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
  TableBatchActions,
  TableBatchAction,
  RadioButton,
  Tag,
} from '@carbon/react';

export function ConfigureAssistantToolPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <ConfigureAssistantTool assistantId={assistantId} />}</>
  );
}

export function CreateAssistantToolPage() {
  const { assistantId } = useParams();
  return <>{assistantId && <CreateTool assistantId={assistantId} />}</>;
}

export function UpdateAssistantToolPage() {
  const { assistantId } = useParams();
  return <>{assistantId && <UpdateTool assistantId={assistantId} />}</>;
}

const ConfigureAssistantTool: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const navigation = useGlobalNavigation();
  const axtion = useAssistantToolPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedToolId, setSelectedToolId] = useState<string | null>(null);

  useEffect(() => {
    showLoader('block');
    get();
  }, []);

  const get = () => {
    axtion.getAssistantTool(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        hideLoader();
      },
    );
  };

  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({
    title: 'Are you sure?',
    content: 'You want to delete? The tool will be removed from assistant.',
  });

  const deleteAssistantTool = (
    assistantId: string,
    assistantToolId: string,
  ) => {
    showLoader('block');
    axtion.deleteAssistantTool(
      assistantId,
      assistantToolId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        toast.success('Assistant tool deleted successfully');
        get();
      },
    );
  };

  const normalizedSearch = searchTerm.trim().toLowerCase();
  const filteredTools = normalizedSearch
    ? axtion.tools.filter(itm =>
        [itm.getName(), itm.getDescription(), itm.getExecutionmethod()]
          .join(' ')
          .toLowerCase()
          .includes(normalizedSearch),
      )
    : axtion.tools;

  const refreshTools = () => {
    setSelectedToolId(null);
    showLoader('block');
    get();
  };

  if (loading) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="relative flex flex-col flex-1">
      <ConfirmDialogComponent />

      {/* Page header */}
      <div className="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
        <div>
          <Breadcrumb noTrailingSlash className="mb-2">
            <BreadcrumbItem
              href={`/deployment/assistant/${assistantId}/overview`}
            >
              Assistant
            </BreadcrumbItem>
          </Breadcrumb>
          <h1 className="text-2xl font-light tracking-tight">Tools & MCP</h1>
        </div>
      </div>

      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={!!selectedToolId}
          totalSelected={selectedToolId ? 1 : 0}
          totalCount={axtion.tools.length}
          onCancel={() => setSelectedToolId(null)}
          className="[&_[class*=divider]]:hidden [&_.cds--btn]:transition-colors [&_.cds--btn:hover]:!bg-primary [&_.cds--btn:hover]:!text-white"
        >
          {selectedToolId && (
            <>
              <TableBatchAction
                renderIcon={Edit}
                onClick={() => {
                  navigation.goToEditAssistantTool(assistantId, selectedToolId);
                  setSelectedToolId(null);
                }}
              >
                Edit tool
              </TableBatchAction>
              <TableBatchAction
                renderIcon={TrashCan}
                onClick={() => {
                  showDialog(() => {
                    deleteAssistantTool(assistantId, selectedToolId);
                  });
                  setSelectedToolId(null);
                }}
              >
                Delete tool
              </TableBatchAction>
            </>
          )}
        </TableBatchActions>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search tools..."
            onChange={(e: any) => setSearchTerm(e.target?.value || '')}
          />
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={refreshTools}
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            onClick={() => navigator.goToCreateAssistantTool(assistantId)}
          >
            Add tool
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>

      <div className="overflow-auto flex flex-col flex-1">
        {axtion.tools.length > 0 && filteredTools.length > 0 ? (
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader className="!w-12" />
                <TableHeader>Name</TableHeader>
                <TableHeader>Type</TableHeader>
                <TableHeader>Description</TableHeader>
                <TableHeader>Actions</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredTools.map((itm, idx) => {
                const method = itm.getExecutionmethod();
                const methodMeta = BUILDIN_TOOLS.find(x => x.code === method);
                const isMcp = method === 'mcp';
                const selected = selectedToolId === itm.getId();
                const conditionSource = getToolConditionSource(
                  itm.getExecutionoptionsList(),
                );

                return (
                  <TableRow
                    key={`tool-row-${idx}`}
                    isSelected={selected}
                    onClick={() =>
                      setSelectedToolId(selected ? null : itm.getId())
                    }
                    className="cursor-pointer"
                  >
                    <TableCell
                      className="!w-12 !pr-0"
                      onClick={e => e.stopPropagation()}
                    >
                      <RadioButton
                        id={`tool-select-${itm.getId()}`}
                        name="tool-select"
                        labelText=""
                        hideLabel
                        checked={selected}
                        onChange={() =>
                          setSelectedToolId(selected ? null : itm.getId())
                        }
                      />
                    </TableCell>
                    <TableCell>{itm.getName()}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        {methodMeta && (
                          <Tag size="sm" type="gray">
                            {methodMeta.name}
                          </Tag>
                        )}
                        {isMcp && (
                          <Tag size="sm" type="purple">
                            MCP
                          </Tag>
                        )}
                        {!methodMeta && !isMcp && (
                          <Tag size="sm" type="gray">
                            {(method || 'Unknown').replace(/_/g, ' ')}
                          </Tag>
                        )}
                        {conditionSource !== 'all' && (
                          <Tag size="sm" type="blue">
                            Source:{' '}
                            {getToolConditionSourceLabel(conditionSource)}
                          </Tag>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="max-w-[360px] truncate">
                      {itm.getDescription()}
                    </TableCell>
                    <TableCell onClick={e => e.stopPropagation()}>
                      <div className="flex items-center gap-0">
                        <Button
                          hasIconOnly
                          renderIcon={Edit}
                          iconDescription="Edit tool"
                          kind="ghost"
                          size="sm"
                          onClick={() =>
                            navigation.goToEditAssistantTool(
                              assistantId,
                              itm.getId(),
                            )
                          }
                        />
                        <Button
                          hasIconOnly
                          renderIcon={TrashCan}
                          iconDescription="Delete tool"
                          kind="ghost"
                          size="sm"
                          onClick={() =>
                            showDialog(() => {
                              deleteAssistantTool(assistantId, itm.getId());
                            })
                          }
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        ) : axtion.tools.length > 0 ? (
          <EmptyState
            icon={ToolKit}
            title="No tools found"
            subtitle="No tool matched your search."
          />
        ) : (
          <EmptyState
            icon={ToolKit}
            title="No tools found"
            subtitle="Any tools or MCPs you add will be listed here."
          />
        )}
      </div>
    </div>
  );
};
