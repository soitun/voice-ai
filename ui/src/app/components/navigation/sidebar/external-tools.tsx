import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { Plug } from '@carbon/icons-react';
import { useLocation } from 'react-router-dom';

export function ExternalTool() {
  const location = useLocation();
  const { pathname } = location;
  return (
    <li>
      <SidebarSimpleListItem
        navigate="/integration/models"
        active={pathname.includes('/integration/models')}
      >
        <SidebarIconWrapper>
          <Plug size={20} />
        </SidebarIconWrapper>
        <SidebarLabel>External integrations</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
