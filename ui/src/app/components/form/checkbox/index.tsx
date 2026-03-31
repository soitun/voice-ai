import { cn } from '@/utils';
import { Check } from 'lucide-react';
import { forwardRef, InputHTMLAttributes } from 'react';

interface InputCheckboxProps extends InputHTMLAttributes<HTMLInputElement> {
  children?: React.ReactNode;
}

/**
 * Carbon checkbox — 16×16px square, zero border-radius.
 * States: enabled → hover → checked (primary fill) → focus (inset outline) → disabled.
 *
 * Accessibility: native input kept in DOM via sr-only (not display:none)
 * so screen readers and keyboard nav work correctly.
 */
export const InputCheckbox = forwardRef<HTMLInputElement, InputCheckboxProps>(
  (props, ref) => {
    const { className, children, ...inputProps } = props;

    return (
      <label className="cursor-pointer inline-flex items-center gap-2">
        {/* Native input — visually hidden but accessible */}
        <input
          ref={ref}
          {...inputProps}
          type="checkbox"
          className="peer sr-only"
        />

        {/* Visual checkbox — 16×16, square, fills primary on checked */}
        <span
          className={cn(
            // Size — Carbon: 16×16px
            'relative flex-shrink-0 w-4 h-4',
            // Shape — Carbon: zero border-radius
            'rounded-none',
            // Border — 1px, gray when unchecked
            'border border-gray-500 dark:border-gray-400',
            // Background — white by default, primary when checked
            'bg-white dark:bg-gray-950',
            'peer-checked:bg-primary peer-checked:border-primary',
            // Text color propagates to Check icon
            'text-transparent peer-checked:text-white',
            // Hover
            'peer-hover:border-primary',
            // Disabled
            'peer-disabled:opacity-50 peer-disabled:cursor-not-allowed',
            // Focus — inset outline on the visual element
            'peer-focus-visible:outline peer-focus-visible:outline-2',
            'peer-focus-visible:outline-primary peer-focus-visible:outline-offset-[-2px]',
            // Layout
            'flex items-center justify-center',
            'transition-colors duration-100',
            className,
          )}
        >
          {/* Carbon uses a ~9px check inside a 16px box */}
          <Check className="w-3 h-3" strokeWidth={2.5} />
        </span>

        {children && (
          <span className="text-sm text-gray-900 dark:text-gray-100 select-none">
            {children}
          </span>
        )}
      </label>
    );
  },
);
