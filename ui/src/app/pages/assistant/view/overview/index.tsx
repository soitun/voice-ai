import { Assistant } from '@rapidaai/react';
import { SectionLoader } from '@/app/components/loader/section-loader';
import { AssistantAnalytics } from '@/app/pages/assistant/view/overview/assistant-analytics';
import { useRapidaStore } from '@/hooks';
import { FC } from 'react';
import { LinkNotification } from '@/app/components/carbon/notification';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableRelativeTime } from '@/utils/date';
import { Breadcrumb, BreadcrumbItem, ComboButton, MenuItem } from '@carbon/react';

export const Overview: FC<{ currentAssistant: Assistant }> = ({
  currentAssistant,
}) => {
  const rapidaContext = useRapidaStore();
  const navigation = useGlobalNavigation();

  if (rapidaContext.loading) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="flex flex-col flex-1 grow">
      {/* ── IBM Page header — breadcrumb + title + actions ── */}
      <div className="px-4 pt-4 pb-6 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
        <div className="flex items-start justify-between">
          <div>
            <Breadcrumb noTrailingSlash className="mb-2">
              <BreadcrumbItem href="/deployment/assistant">
                Assistants
              </BreadcrumbItem>
            </Breadcrumb>
            <h1 className="text-2xl font-light tracking-tight">
              {currentAssistant.getName()}
            </h1>
            {currentAssistant.getAssistantprovidermodel()?.getCreateddate() && (
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 tabular-nums">
                Last updated{' '}
                {toHumanReadableRelativeTime(
                  currentAssistant.getAssistantprovidermodel()?.getCreateddate()!,
                )}
              </p>
            )}
          </div>
          <ComboButton
            label="Create new version"
            size="md"
            onClick={() =>
              navigation.goToCreateAssistantVersion(currentAssistant.getId())
            }
          >
            <MenuItem
              label="Create AgentKit"
              onClick={() =>
                navigation.goToCreateAssistantAgentKitVersion(
                  currentAssistant.getId(),
                )
              }
            />
          </ComboButton>
        </div>
      </div>

      {/* ── Notifications ── */}
      {!currentAssistant.getApideployment() &&
        !currentAssistant.getDebuggerdeployment() &&
        !currentAssistant.getWebplugindeployment() &&
        !currentAssistant.getPhonedeployment() && (
          <LinkNotification
            kind="warning"
            title="Your assistant is ready, but not live yet."
            subtitle="It looks like your assistant isn't deployed to any channel."
            linkText="Enable deployment"
            onLinkClick={() =>
              navigation.goToDeploymentAssistant(currentAssistant.getId())
            }
          />
        )}

      {/* ── Dashboard content ── */}
      <AssistantAnalytics assistant={currentAssistant} />
    </div>
  );
};
