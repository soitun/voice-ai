import { CustomLink } from '@/app/components/custom-link';
import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';
import { SkeletonIcon, SkeletonText } from '@carbon/react';
import { useSidebar } from '@/context/sidebar-context';

interface SidebarLinkItemProps extends HTMLAttributes<HTMLDivElement> {
  active?: boolean;
  redirect?: boolean;
  navigate: string;
  loading?: boolean;
}

export function SidebarSimpleListItem(props: SidebarLinkItemProps) {
  const { active, redirect, navigate, loading, ...dProps } = props;
  const { open } = useSidebar();

  if (loading) {
    return (
      <div className="flex items-center h-10 w-full px-1">
        <div className="flex-shrink-0 flex items-center justify-center w-12 h-8">
          <SkeletonIcon className="!w-5 !h-5" />
        </div>
        {open && <SkeletonText className="!mb-0 flex-1" width="70%" />}
      </div>
    );
  }

  return (
    <CustomLink to={navigate} isExternal={redirect}>
      <div
        {...dProps}
        className={cn(
          'relative flex items-center h-10 w-full cursor-pointer',
          'text-gray-700 dark:text-gray-300',
          'hover:bg-gray-100 dark:hover:bg-gray-800',
          active && [
            'bg-gray-100 dark:bg-gray-800 font-semibold',
            'before:absolute before:inset-y-0 before:left-0 before:w-1 before:bg-primary before:content-[""]',
          ],
          props.className,
        )}
      >
        {props.children}
      </div>
    </CustomLink>
  );
}
