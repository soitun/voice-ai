import { FC } from 'react';
import { useLocation } from 'react-router-dom';
import { cn } from '@/utils';
import { SidePanelOpen, SidePanelClose } from '@carbon/icons-react';
import {
  SideNav,
  SideNavItems,
  SideNavLink,
  SideNavMenu,
  SideNavMenuItem,
} from '@carbon/react';
import {
  assistantNavSections,
  AssistantNavSection,
  AssistantNavItem,
} from './assistant-nav-config';

interface AssistantSideNavProps {
  assistantId: string;
  assistant: any;
  expanded: boolean;
  onToggle: () => void;
  actions?: Record<string, () => void>;
}

export const AssistantSideNav: FC<AssistantSideNavProps> = ({
  assistantId,
  assistant,
  expanded,
  onToggle,
  actions = {},
}) => {
  const { pathname } = useLocation();
  const basePath = `/deployment/assistant/${assistantId}`;

  const isPathActive = (path: string, exact?: boolean) => {
    const fullPath = `${basePath}/${path}`;
    return exact ? pathname === fullPath : pathname.startsWith(fullPath);
  };

  const renderItem = (item: AssistantNavItem) => {
    if (item.visible && !item.visible(assistant)) return null;

    // Items with children → SideNavMenu
    if (item.children && item.children.length > 0) {
      const isAnyChildActive = item.children.some(c => isPathActive(c.path));
      return (
        <SideNavMenu
          key={item.key}
          title={item.label}
          renderIcon={item.icon}
          isActive={isAnyChildActive}
          defaultExpanded={isAnyChildActive}
        >
          {item.children.map(child => {
            const childAction = child.action ? actions[child.action] : undefined;
            return (
              <SideNavMenuItem
                key={child.key}
                href={childAction ? undefined : `${basePath}/${child.path}`}
                onClick={childAction ? () => childAction() : undefined}
                isActive={isPathActive(child.path, true)}
              >
                {child.label}
              </SideNavMenuItem>
            );
          })}
        </SideNavMenu>
      );
    }

    // Simple link
    const active = isPathActive(item.path, item.exact);
    const actionFn = item.action ? actions[item.action] : undefined;

    return (
      <SideNavLink
        key={item.key}
        renderIcon={item.icon}
        href={actionFn ? undefined : `${basePath}/${item.path}`}
        onClick={actionFn ? () => actionFn() : undefined}
        isActive={active}
      >
        {item.label}
      </SideNavLink>
    );
  };

  const renderSection = (section: AssistantNavSection, idx: number) => {
    const visibleItems = section.items.filter(
      item => !item.visible || item.visible(assistant),
    );
    if (visibleItems.length === 0) return null;

    return (
      <div key={idx}>
        {section.label && (
          <li
            className={cn(
              'cds--switcher__item--divider transition-all duration-200',
              !expanded && 'opacity-0 h-0 overflow-hidden !py-0 !my-0 !border-none',
            )}
          >
            <span>{section.label}</span>
          </li>
        )}
        {section.items.map(renderItem)}
      </div>
    );
  };

  return (
    <div
      className={cn(
        'relative shrink-0 flex flex-col h-full',
        'border-r border-gray-200 dark:border-gray-800',
        'transition-all duration-200',
        expanded ? 'w-56' : 'w-12',
      )}
    >
      <SideNav
        aria-label="Assistant actions"
        expanded={expanded}
        isRail={!expanded}
        className="!relative !inset-auto !h-auto flex-1 !w-full !border-none !z-0"
      >
        <SideNavItems>
          {assistantNavSections.map(renderSection)}
        </SideNavItems>
      </SideNav>

      <div className="shrink-0 border-t border-gray-200 dark:border-gray-800">
        <button
          type="button"
          onClick={onToggle}
          className={cn(
            'flex items-center h-10 w-full cursor-pointer',
            'text-gray-400 dark:text-gray-500',
            'hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-600 dark:hover:text-gray-400',
            'transition-colors duration-100',
          )}
          aria-label={expanded ? 'Collapse nav' : 'Expand nav'}
        >
          <span className="flex items-center justify-center w-12 h-10 shrink-0 text-gray-400 dark:text-gray-500">
            {expanded ? <SidePanelClose size={16} /> : <SidePanelOpen size={16} />}
          </span>
          {expanded && <span className="text-xs truncate">Collapse</span>}
        </button>
      </div>
    </div>
  );
};
