import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { TabLink } from '@/app/components/tab-link';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ChevronLeft } from 'lucide-react';
import { FC, HTMLAttributes } from 'react';
import { Outlet, useParams } from 'react-router-dom';

export const AssistantManageLayout: FC<HTMLAttributes<HTMLDivElement>> = () => {
  const { assistantId } = useParams();
  const { goToAssistant } = useGlobalNavigation();

  return (
    <div className="flex flex-col h-full flex-1 overflow-auto bg-white dark:bg-gray-900">
      <PageHeaderBlock>
        <div className="flex items-center gap-1.5 min-w-0">
          <div
            onClick={() => goToAssistant(assistantId!)}
            className="flex items-center gap-1.5 text-gray-500 dark:text-gray-400 hover:text-primary transition-colors cursor-pointer shrink-0"
          >
            <ChevronLeft className="w-4 h-4" strokeWidth={1.5} />
            <span className="text-sm font-medium">Assistants</span>
          </div>
          <span className="px-1 text-gray-300 dark:text-gray-600 shrink-0">/</span>
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100 font-mono truncate">
            {assistantId}
          </span>
        </div>
      </PageHeaderBlock>
      <div className="sticky top-0 z-3 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 shrink-0">
        <div className="flex items-stretch h-10">
          <TabLink to={`/deployment/assistant/${assistantId}/deployment`}>
            Deployment
          </TabLink>
          <TabLink to={`/deployment/assistant/${assistantId}/configure-tool`}>
            Tools & MCP
          </TabLink>
          <TabLink to={`/deployment/assistant/${assistantId}/configure-analysis`}>
            Analysis
          </TabLink>
          <TabLink to={`/deployment/assistant/${assistantId}/configure-webhook`}>
            Webhooks
          </TabLink>
          <TabLink to={`/deployment/assistant/${assistantId}/edit-assistant`}>
            Settings
          </TabLink>
        </div>
      </div>
      <Outlet />
    </div>
  );
};
