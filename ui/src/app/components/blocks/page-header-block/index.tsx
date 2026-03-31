import React, { FC, HTMLAttributes } from 'react';
import { cn } from '@/utils';

export const PageHeaderBlock: FC<HTMLAttributes<HTMLDivElement>> = props => {
  return (
    <div
      className={cn(
        'flex justify-between pl-4 pr-0 bg-white dark:bg-gray-900 items-center h-12 border-b border-gray-200 dark:border-gray-800 shrink-0',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
};
