import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';
import { useSidebar } from '@/context/sidebar-context';

export function SidebarLabel(props: HTMLAttributes<HTMLSpanElement>) {
  const { open } = useSidebar();
  return (
    <span
      className={cn(
        'text-sm truncate flex-1 transition-all duration-200 font-semibold',
        open ? 'opacity-100' : 'opacity-0 w-0',
        props.className,
      )}
      {...props}
    >
      {props.children}
    </span>
  );
}
