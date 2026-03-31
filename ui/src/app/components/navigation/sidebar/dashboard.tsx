import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { Dashboard as DashboardIcon } from '@carbon/icons-react';
import { useLocation } from 'react-router-dom';

export function Dashboard() {
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/dashboard';
  return (
    <SidebarSimpleListItem
      navigate={currentPath}
      active={pathname.includes(currentPath)}
    >
      <SidebarIconWrapper>
        <DashboardIcon size={20} />
      </SidebarIconWrapper>
      <SidebarLabel>Dashboard</SidebarLabel>
    </SidebarSimpleListItem>
  );
}
