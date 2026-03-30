import { FC } from 'react';
import { cn } from '@/utils';
import { SidePanelOpen, SidePanelClose } from '@carbon/icons-react';
import {
  SideNav,
  SideNavItems,
  SideNavLink,
  SideNavMenu,
  SideNavMenuItem,
} from '@carbon/react';
import { endpointNavSections } from './endpoint-nav-config';

interface EndpointSideNavProps {
  activeTab: string;
  onChangeTab: (tab: string) => void;
  expanded: boolean;
  onToggle: () => void;
}

export const EndpointSideNav: FC<EndpointSideNavProps> = ({
  activeTab,
  onChangeTab,
  expanded,
  onToggle,
}) => {
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
        aria-label="Endpoint actions"
        expanded={expanded}
        isRail={!expanded}
        className="!relative !inset-auto !h-auto flex-1 !w-full !border-none !z-0"
      >
        <SideNavItems>
          {endpointNavSections.map((section, idx) => (
            <div key={idx}>
              {section.label && (
                <li className="cds--switcher__item--divider">
                  <span>{section.label}</span>
                </li>
              )}
              {section.items.map(item =>
                item.children ? (
                  <SideNavMenu
                    key={item.key}
                    title={item.label}
                    renderIcon={item.icon}
                    isActive={item.children.some(c => activeTab === c.tabKey)}
                    defaultExpanded={item.children.some(c => activeTab === c.tabKey)}
                  >
                    {item.children.map(child => (
                      <SideNavMenuItem
                        key={child.key}
                        isActive={activeTab === child.tabKey}
                        onClick={() => onChangeTab(child.tabKey)}
                      >
                        {child.label}
                      </SideNavMenuItem>
                    ))}
                  </SideNavMenu>
                ) : (
                  <SideNavLink
                    key={item.key}
                    renderIcon={item.icon}
                    isActive={activeTab === item.tabKey}
                    onClick={() => onChangeTab(item.tabKey)}
                  >
                    {item.label}
                  </SideNavLink>
                ),
              )}
            </div>
          ))}
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
