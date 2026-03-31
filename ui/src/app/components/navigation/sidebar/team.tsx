import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { UserMultiple } from '@carbon/icons-react';
import { useLocation } from 'react-router-dom';

export function Team() {
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/organization/users';
  return (
    <li>
      <SidebarSimpleListItem
        navigate={currentPath}
        active={pathname.includes(currentPath)}
      >
        <SidebarIconWrapper>
          <UserMultiple size={20} />
        </SidebarIconWrapper>
        <SidebarLabel>Users and Teams</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
