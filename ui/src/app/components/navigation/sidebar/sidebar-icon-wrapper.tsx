import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';

export function SidebarIconWrapper(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'flex-shrink-0 flex items-center justify-center w-12 h-8',
        '[&_svg]:w-5 [&_svg]:h-5',
        props.className,
      )}
      {...props}
    >
      {props.children}
    </div>
  );
}
