import { Observability } from '@/app/components/navigation/sidebar/observability';
import { Deployment } from '@/app/components/navigation/sidebar/deployment';
import { Dashboard } from '@/app/components/navigation/sidebar/dashboard';
import { Team } from '@/app/components/navigation/sidebar/team';
import { Project } from '@/app/components/navigation/sidebar/project';
import { Vault } from '@/app/components/navigation/sidebar/vault';
import { Knowledge } from '@/app/components/navigation/sidebar/knowledge';
import { Aside } from '@/app/components/aside';
import { ExternalTool } from '@/app/components/navigation/sidebar/external-tools';
import { useWorkspace } from '@/workspace';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { RapidaTextIcon } from '@/app/components/Icon/RapidaText';
import { SidePanelClose, SidePanelOpen } from '@carbon/icons-react';
import { useSidebar } from '@/context/sidebar-context';
import { cn } from '../../../../utils/index';

/**
 * Carbon UI Shell — Side Navigation
 * Spec: h-8 nav items, 4px left accent on active, 48px logo header,
 *       label-01 group headers, lock/collapse button in footer.
 */
export function SidebarNavigation(props: {}) {
  const workspace = useWorkspace();
  const { locked, setLocked, open } = useSidebar();

  return (
    <Aside className="relative shrink-0 flex flex-col">
      {/* ── Logo row — Carbon UI Shell header: h-12, border-b ── */}
      <div className="h-12 flex items-center border-b border-gray-200 dark:border-gray-800 px-3 shrink-0">
        {workspace.logo ? (
          <>
            <img
              src={workspace.logo.light}
              alt={workspace.title}
              className="h-6 block dark:hidden"
            />
            <img
              src={workspace.logo.dark}
              alt={workspace.title}
              className="h-6 hidden dark:block"
            />
          </>
        ) : (
          <div className="flex items-center gap-2 text-primary">
            <RapidaIcon className="h-7 w-7 shrink-0" />
            <RapidaTextIcon
              className={cn(
                'h-5 transition-all duration-200',
                open ? 'opacity-100' : 'opacity-0 w-0',
              )}
            />
          </div>
        )}
      </div>

      {/* ── Nav groups — scrollable ── */}
      <nav className="flex-1 overflow-y-auto no-scrollbar py-2">
          {/* Group 1 — primary nav */}
          <ul>
            <Dashboard />
            <Deployment />
            {workspace.features?.knowledge !== false && <Knowledge />}
          </ul>

          {/* Group 2 — Observability */}
          <div className="mt-2">
            <div
              className={cn(
                'flex items-center px-4 py-2 border-b border-gray-200 dark:border-gray-800',
                'text-[10px] font-medium capitalize tracking-[0.1em]',
                'text-gray-500 dark:text-gray-400',
                'transition-all duration-200',
                open ? 'opacity-100' : 'opacity-0 h-0 py-0 overflow-hidden border-none',
              )}
            >
              Observability
            </div>
            <ul>
              <Observability />
            </ul>
          </div>

          {/* Group 3 — Integrations */}
          <div className="mt-2">
            <div
              className={cn(
                'flex items-center px-4 py-2 border-b border-gray-200 dark:border-gray-800',
                'text-[10px] font-medium capitalize tracking-[0.1em]',
                'text-gray-500 dark:text-gray-400',
                'transition-all duration-200',
                open ? 'opacity-100' : 'opacity-0 h-0 py-0 overflow-hidden border-none',
              )}
            >
              Integrations
            </div>
            <ul>
              <ExternalTool />
              <Vault />
            </ul>
          </div>

          {/* Group 4 — Organizations */}
          <div className="mt-2">
            <div
              className={cn(
                'flex items-center px-4 py-2 border-b border-gray-200 dark:border-gray-800',
                'text-[10px] font-medium capitalize tracking-[0.1em]',
                'text-gray-500 dark:text-gray-400',
                'transition-all duration-200',
                open ? 'opacity-100' : 'opacity-0 h-0 py-0 overflow-hidden border-none',
              )}
            >
              Organizations
            </div>
            <ul>
              <Team />
              <Project />
            </ul>
          </div>
        </nav>

      {/* ── Footer — collapse/expand button ── */}
      <div className="shrink-0 border-t border-gray-200 dark:border-gray-800">
        <button
          type="button"
          onClick={() => setLocked(!locked)}
          aria-label={locked ? 'Collapse sidebar' : 'Expand sidebar'}
          className={cn(
            'flex items-center h-10 w-full cursor-pointer px-4',
            'text-gray-400 dark:text-gray-500',
            'hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-600 dark:hover:text-gray-400',
            'transition-colors duration-100',
          )}
        >
          <span className="shrink-0">
            {locked ? <SidePanelClose size={16} /> : <SidePanelOpen size={16} />}
          </span>
          <span
            className={cn(
              'text-xs truncate transition-all duration-200 ml-3',
              open ? 'opacity-100' : 'opacity-0 w-0 ml-0 overflow-hidden',
            )}
          >
            {locked ? 'Collapse' : 'Expand'}
          </span>
        </button>
      </div>
    </Aside>
  );
}
