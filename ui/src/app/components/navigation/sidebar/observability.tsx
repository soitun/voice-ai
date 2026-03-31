import { memo } from 'react';
import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { useLocation } from 'react-router-dom';
import { Activity, DataBase, Chat, Webhook, ToolKit } from '@carbon/icons-react';
import { useWorkspace } from '@/workspace';

export const Observability = memo(() => {
  const location = useLocation();
  const { pathname } = location;
  const workspace = useWorkspace();

  return (
    <li>
      <SidebarSimpleListItem
        active={pathname.endsWith('/logs')}
        navigate="/logs"
      >
        <SidebarIconWrapper>
          <Activity size={20} />
        </SidebarIconWrapper>
        <SidebarLabel>LLM logs</SidebarLabel>
      </SidebarSimpleListItem>

      <SidebarSimpleListItem
        active={pathname.includes('/logs/tool')}
        navigate="/logs/tool"
      >
        <SidebarIconWrapper>
          <ToolKit size={20} />
        </SidebarIconWrapper>
        <SidebarLabel>Tool logs</SidebarLabel>
      </SidebarSimpleListItem>
      <SidebarSimpleListItem
        active={pathname.includes('/logs/webhook')}
        navigate="/logs/webhook"
      >
        <SidebarIconWrapper>
          <Webhook size={20} />
        </SidebarIconWrapper>
        <SidebarLabel>Webhook logs</SidebarLabel>
      </SidebarSimpleListItem>
      {workspace.features?.knowledge !== false && (
        <SidebarSimpleListItem
          active={pathname.includes('/logs/knowledge')}
          navigate="/logs/knowledge"
        >
          <SidebarIconWrapper>
            <DataBase size={20} />
          </SidebarIconWrapper>
          <SidebarLabel>Knowledge logs</SidebarLabel>
        </SidebarSimpleListItem>
      )}
      <SidebarSimpleListItem
        active={pathname.includes('/logs/conversation')}
        navigate="/logs/conversation"
      >
        <SidebarIconWrapper>
          <Chat size={20} />
        </SidebarIconWrapper>
        <SidebarLabel>Conversation logs</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
});
