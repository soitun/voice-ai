import { Button as CarbonButton } from '@carbon/react';
import { Spinner } from '@/app/components/loader/spinner';
import { cn } from '@/utils';
import type { ButtonProps } from './v1.button';

export * from './v1.button';

const sizeMap: Record<NonNullable<ButtonProps['size']>, 'sm' | 'md' | 'lg'> = {
  sm: 'sm',
  md: 'md',
  lg: 'lg',
};

export function Button({
  isLoading,
  size = 'md',
  type = 'button',
  disabled,
  children,
  className,
  ...props
}: ButtonProps) {
  return (
    <CarbonButton
      {...props}
      type={type}
      kind="primary"
      size={sizeMap[size]}
      disabled={Boolean(disabled || isLoading)}
      className={cn('button', className)}
    >
      {isLoading ? (
        <Spinner className="w-4 h-4 border-white" size="xs" />
      ) : (
        children
      )}
    </CarbonButton>
  );
}
