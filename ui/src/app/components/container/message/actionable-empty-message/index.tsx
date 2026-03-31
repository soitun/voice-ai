import { IBlueBorderPlusButton } from '@/app/components/form/button';
import { FC } from 'react';

export const ActionableEmptyMessage: FC<{
  title: string;
  subtitle: string;
  action?: string;
  actionComponent?: any;
  onActionClick?: () => void;
  /** Wrap in a flex-1 full-width centering container */
  centered?: boolean;
}> = ({
  title,
  subtitle,
  action,
  actionComponent,
  onActionClick,
  centered,
}) => {
  const content = (
    <div className="px-4 py-6 flex flex-col justify-center items-center">
      <div className="font-semibold">{title}</div>
      <div>{subtitle}</div>
      {actionComponent && actionComponent}
      {action && (
        <IBlueBorderPlusButton
          onClick={onActionClick}
          className="mt-3 bg-white dark:bg-gray-950"
        >
          {action}
        </IBlueBorderPlusButton>
      )}
    </div>
  );
  if (centered) {
    return (
      <div className="flex flex-1 w-full justify-center items-center">
        {content}
      </div>
    );
  }
  return content;
};
