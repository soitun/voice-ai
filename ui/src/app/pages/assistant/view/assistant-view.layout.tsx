import { Helmet } from '@/app/components/helmet';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { FC, HTMLAttributes, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Outlet, useParams } from 'react-router-dom';
import { toHumanReadableRelativeTime } from '@/utils/date';
import { cn } from '@/utils';
import {
  AssistantDefinition,
  ConnectionConfig,
  GetAssistant,
  GetAssistantRequest,
} from '@rapidaai/react';
import { useAssistantPageStore } from '@/hooks/use-assistant-page-store';
import { TabLink } from '@/app/components/tab-link';
import { IButton } from '@/app/components/form/button';
import { Bolt, ChevronDown, GitPullRequestCreate } from 'lucide-react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { ErrorContainer } from '@/app/components/error-container';
import { connectionConfig } from '@/configs';
import { Popover } from '@/app/components/popover';
/**
 *
 * @returns
 */
export const AssistantViewLayout: FC<HTMLAttributes<HTMLDivElement>> = () => {
  /**
   * authentication information
   */
  const [userId, token, projectId] = useCredential();

  /**
   * global loader
   */
  const { showLoader, hideLoader } = useRapidaStore();

  /**
   * zustand state for the page / this also contains of listing
   */
  const assistantAction = useAssistantPageStore();

  /**
   * get all the models when type change
   */
  const { assistantId } = useParams();

  //
  const [createVersionPopover, setCreateVersionPopover] = useState(false);

  //
  const [previewPopover, setPreviewPopover] = useState(false);

  /**
   * navigation
   */
  const {
    goToAssistantPreview,
    goToAssistantPreviewCall,
    goToCreateAssistantVersion,
    goToCreateAssistantAgentKitVersion,
    goToAssistantListing,
    goToManageAssistant,
  } = useGlobalNavigation();

  /**
   *
   */

  const [unknownState, setUnknownState] = useState(false);
  /**
   *
   */
  useEffect(() => {
    assistantAction.clear();
    if (assistantId) {
      showLoader();
      const request = new GetAssistantRequest();
      const assistantDef = new AssistantDefinition();
      assistantDef.setAssistantid(assistantId);
      request.setAssistantdefinition(assistantDef);
      GetAssistant(
        connectionConfig,
        request,
        ConnectionConfig.WithDebugger({
          authorization: token,
          userId: userId,
          projectId: projectId,
        }),
      )
        .then(epmr => {
          hideLoader();
          if (epmr?.getSuccess()) {
            let assistant = epmr.getData();
            if (assistant) assistantAction.onChangeCurrentAssistant(assistant);
          } else {
            setUnknownState(true);
            const errorMessage =
              'Unable to get your assistant. please try again later.';
            const error = epmr?.getError();
            if (error) {
              toast.error(error.getHumanmessage());
              return;
            }
            toast.error(errorMessage);
            return;
          }
        })
        .catch(err => {
          hideLoader();
        });
    }
  }, [assistantId]);

  if (unknownState)
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={goToAssistantListing}
          code="403"
          actionLabel="Go to listing"
          title="Assistant not available"
          description="This assistant may be archived or you don't have access to it. Please check with your administrator or try another assistant."
        />
      </div>
    );

  //
  return (
    <div className={cn('flex flex-col h-full flex-1 overflow-auto bg-white dark:bg-gray-900')}>
      <Helmet title="Hosted Assistant" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>
            <span className="text-gray-400 dark:text-gray-500 font-normal">
              Assistant
            </span>
            <span className="px-2 text-gray-300 dark:text-gray-600">/</span>
            {assistantAction.currentAssistant?.getName()}
          </PageTitleBlock>
          {assistantAction.currentAssistant
            ?.getAssistantprovidermodel()
            ?.getCreateddate() && (
            <span className="text-xs text-gray-500 dark:text-gray-400 tabular-nums">
              {toHumanReadableRelativeTime(
                assistantAction.currentAssistant
                  ?.getAssistantprovidermodel()
                  ?.getCreateddate()!,
              )}
            </span>
          )}
        </div>
        {assistantAction.currentAssistant && (
          <div className="flex items-stretch h-12 border-l border-gray-200 dark:border-gray-800">
            {/* Create New Version */}
            <div className="border-r border-gray-200 dark:border-gray-800 flex items-stretch">
              <button
                type="button"
                onClick={() => setCreateVersionPopover(true)}
                className={cn(
                  'flex items-center gap-2 px-4 text-sm text-white bg-primary hover:bg-primary/90 transition-colors whitespace-nowrap',
                  createVersionPopover && 'bg-primary/80',
                )}
              >
                Create New Version
                <GitPullRequestCreate className="w-4 h-4" strokeWidth={1.5} />
              </button>
              <Popover
                align={'bottom-end'}
                className="w-60 pb-2"
                open={createVersionPopover}
                setOpen={setCreateVersionPopover}
              >
                <div className="space-y-0.5 text-sm/6">
                  <p className="px-4 py-2 text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                    New Version
                  </p>
                  <div className="border-t border-gray-200 dark:border-gray-800" />
                  <IButton
                    className="w-full justify-start"
                    onClick={() => goToCreateAssistantVersion(assistantId!)}
                  >
                    Create New version
                  </IButton>
                  <IButton
                    className="w-full justify-start"
                    onClick={() =>
                      goToCreateAssistantAgentKitVersion(assistantId!)
                    }
                  >
                    Connect new AgentKit
                  </IButton>
                </div>
              </Popover>
            </div>

            {/* Configure assistant */}
            <div className="border-r border-gray-200 dark:border-gray-800 flex items-stretch">
              <IButton
                className="h-full px-4"
                onClick={() => goToManageAssistant(assistantId!)}
              >
                Configure assistant
                <Bolt className="w-4 h-4" strokeWidth={1.5} />
              </IButton>
            </div>

            {/* Preview */}
            <div className="flex items-stretch">
              <IButton
                className="h-full px-4"
                onClick={() => setPreviewPopover(!previewPopover)}
              >
                Preview
                <ChevronDown
                  className={cn(
                    'w-4 h-4 transition-all delay-200',
                    previewPopover && 'rotate-180',
                  )}
                  strokeWidth={1.5}
                />
              </IButton>
              <Popover
                align={'bottom-end'}
                className="w-60 pb-2"
                open={previewPopover}
                setOpen={setPreviewPopover}
              >
                <div className="space-y-0.5 text-sm/6">
                  <p className="px-4 py-2 text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                    Agent Preview
                  </p>
                  <div className="border-t border-gray-200 dark:border-gray-800" />
                  {assistantAction.currentAssistant.getPhonedeployment() && (
                    <IButton
                      className="w-full justify-start"
                      onClick={() => goToAssistantPreviewCall(assistantId!)}
                    >
                      Preview phone call
                    </IButton>
                  )}
                  <IButton
                    className="w-full justify-start"
                    onClick={() => goToAssistantPreview(assistantId!)}
                  >
                    Debugging
                  </IButton>
                </div>
              </Popover>
            </div>
          </div>
        )}
      </PageHeaderBlock>
      <div className="sticky top-0 z-3 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
        <div className="flex items-stretch h-10">
          <TabLink to={`/deployment/assistant/${assistantId}/overview`}>
            Overview
          </TabLink>
          <TabLink to={`/deployment/assistant/${assistantId}/sessions`}>
            Sessions
          </TabLink>
          <TabLink to={`/deployment/assistant/${assistantId}/version-history`}>
            Versions
          </TabLink>
        </div>
      </div>
      <Outlet />
    </div>
  );
};
