import { Spinner } from '@/app/components/loader/spinner';
import React, { FC } from 'react';
import { cn } from '@/utils';
import { ChevronRight, Plus } from 'lucide-react';

// ─── Shared prop interface ────────────────────────────────────────────────────

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  children?: React.ReactNode;
  /** Shows a spinner and disables the button while true */
  isLoading?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

export interface LinkProps
  extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  children?: React.ReactNode;
  isLoading?: boolean;
}

// ─── Base class fragments ─────────────────────────────────────────────────────

/** Shared structural classes applied to every button variant */
const base =
  'inline-flex items-center justify-center gap-2 w-fit ' +
  'text-sm font-medium ' +
  'rounded-none border-0 ' +
  'transition-colors duration-100 ' +
  'disabled:opacity-50 disabled:cursor-not-allowed ' +
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary ' +
  'button';

/** Carbon md height = 40px, lg = 48px, sm = 32px */
const sizes: Record<NonNullable<ButtonProps['size']>, string> = {
  sm: 'h-8 px-4 text-xs',
  md: 'h-10 px-4 text-sm',
  lg: 'h-12 px-4 text-base',
};

// ─── Primary ──────────────────────────────────────────────────────────────────

/**
 * Carbon Primary button — filled primary, white text.
 * Use for the single most important action on a page.
 */
export function Button({ isLoading, size = 'md', ...props }: ButtonProps) {
  return (
    <button
      {...props}
      className={cn(
        base,
        sizes[size],
        'bg-primary text-white hover:bg-primary/90 active:bg-primary/80',
        props.className,
      )}
    >
      {isLoading ? (
        <Spinner className="border-white" size="xs" />
      ) : (
        props.children
      )}
    </button>
  );
}

// ─── Primary aliases (used throughout existing pages) ─────────────────────────

/** @alias Button — primary filled */
export function IBlueBGButton({ isLoading, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-primary text-white hover:bg-primary/90 active:bg-primary/80',
        isLoading && 'cursor-wait',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

/**
 * Carbon CTA / wizard "Next" button.
 * Text is left-aligned; icon is pinned to the far right (justify-between).
 * min-w-[10rem] ensures visible breathing room between label and icon.
 */
export function IBlueBGArrowButton({ isLoading, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 pl-4 pr-4',
        // Carbon icon-right: text left, icon pinned to right edge
        'justify-between gap-8 min-w-[10rem]',
        'bg-primary text-white hover:bg-primary/90 active:bg-primary/80',
        isLoading && 'cursor-wait',
        props.className,
      )}
    >
      <span>{props.children}</span>
      {isLoading ? (
        <Spinner className="w-4 h-4 border-white shrink-0" size="xs" />
      ) : (
        <ChevronRight className="w-4 h-4 shrink-0" strokeWidth={1.5} />
      )}
    </button>
  );
}

// ─── Secondary ────────────────────────────────────────────────────────────────

/**
 * Carbon Secondary button — transparent bg, gray border, gray text.
 * Pairs with a primary button as a non-destructive alternative.
 */
export function ICancelButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent',
        'text-primary',
        'hover:bg-gray-100 dark:hover:bg-gray-800',
        'active:bg-gray-200 dark:active:bg-gray-700',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export function ISecondaryButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-gray-200 dark:bg-gray-800',
        'hover:bg-gray-300 dark:hover:bg-gray-950',
        'active:bg-gray-200 dark:active:bg-gray-700',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

/** @alias ICancelButton */
export function BorderButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent border border-gray-300 dark:border-gray-600',
        'text-gray-700 dark:text-gray-300',
        'hover:bg-gray-50 dark:hover:bg-gray-800',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

/** @alias ICancelButton */
export function IBorderButton({ isLoading, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent border border-gray-300 dark:border-gray-600',
        'text-gray-700 dark:text-gray-300',
        'hover:bg-gray-700 hover:text-white hover:border-gray-700',
        'dark:hover:bg-gray-600 dark:hover:border-gray-600',
        props.className,
      )}
    >
      {isLoading ? <Spinner className="w-4 h-4" size="xs" /> : props.children}
    </button>
  );
}

// ─── Tertiary ─────────────────────────────────────────────────────────────────

/**
 * Carbon Tertiary button — primary border + text, transparent bg.
 * Hover fills with primary color.
 */
export function BlueBorderButton({ isLoading, ...props }: ButtonProps) {
  return (
    <button
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent border border-primary',
        'text-primary',
        'hover:bg-primary hover:text-white',
        'dark:hover:bg-primary',
        props.className,
      )}
    >
      {isLoading ? <Spinner size="xs" /> : props.children}
    </button>
  );
}

/** @alias BlueBorderButton */
export function IBlueBorderButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent border border-primary',
        'text-primary',
        'hover:bg-primary hover:text-white',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

/** Tertiary with trailing Plus icon */
export function IBlueBorderPlusButton({ isLoading, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4 justify-between gap-8',
        'bg-transparent border border-primary',
        'text-primary',
        'hover:bg-primary hover:text-white',
        props.className,
      )}
    >
      {props.children}
      {isLoading ? (
        <Spinner className="border-current" size="xs" />
      ) : (
        <Plus className="w-4 h-4" strokeWidth={1.5} />
      )}
    </button>
  );
}

// ─── Ghost ────────────────────────────────────────────────────────────────────

/**
 * Carbon Ghost button — no background, no border. Text color only.
 * Used for low-emphasis actions, often inline with content.
 */
export function HoverButton({ isLoading, ...props }: ButtonProps) {
  return (
    <button
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent text-gray-700 dark:text-gray-300',
        'hover:bg-gray-100 dark:hover:bg-gray-800',
        props.className,
      )}
    >
      {isLoading ? <Spinner size="xs" /> : props.children}
    </button>
  );
}

/** @alias HoverButton */
export function IButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent text-gray-700 dark:text-gray-300',
        'hover:bg-gray-100 dark:hover:bg-gray-800',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

/** Ghost — primary text color */
export function IBlueButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent text-primary',
        'hover:bg-primary/10 dark:hover:bg-primary/10',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

/** Icon button — square, ghost */
export function IconButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'w-8 h-8 p-1.5',
        'bg-transparent text-gray-600 dark:text-gray-400',
        'hover:bg-gray-100 dark:hover:bg-gray-800',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

/** Simple hover — minimal ghost, used for toolbar/icon rows */
export function SimpleButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent text-gray-700 dark:text-gray-300',
        'hover:bg-gray-100 dark:hover:bg-gray-800',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

// ─── Danger ───────────────────────────────────────────────────────────────────

/**
 * Carbon Danger button — red fill, white text.
 * Use for destructive or irreversible actions.
 */
export function IRedBGButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-red-600 text-white',
        'hover:bg-red-700 active:bg-red-800',
        'disabled:opacity-50',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

/** Carbon Danger tertiary — red border + text, fills red on hover */
export function IRedBorderButton({ isLoading, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-transparent border border-red-600',
        'text-red-600',
        'hover:bg-red-600 hover:text-white',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

// ─── Success ──────────────────────────────────────────────────────────────────

/** Non-Carbon custom: success/green fill */
export function IGreenBGButton({ isLoading, size, ...props }: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        base,
        'h-10 px-4',
        'bg-green-600 text-white',
        'hover:bg-green-700 active:bg-green-800',
        'disabled:opacity-50',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

// ─── Outline (wraps primary) ─────────────────────────────────────────────────

/** Wraps primary Button with uppercase label */
export const OutlineButton = ({
  isLoading,
  className,
  ...props
}: ButtonProps) => (
  <Button
    className={cn('px-4 uppercase tracking-wide', className)}
    type="submit"
    {...props}
  >
    {props.children}
    {isLoading && <Spinner className="w-4 h-4 border-white" size="xs" />}
  </Button>
);

// ─── Link variants ────────────────────────────────────────────────────────────

/** Anchor styled as primary button */
export function ILinkButton({ isLoading, ...props }: LinkProps) {
  return (
    <a
      {...props}
      className={cn(
        'inline-flex items-center justify-center gap-2 w-fit',
        'h-10 px-4 text-sm font-medium',
        'bg-primary text-white hover:bg-primary/90',
        'rounded-none transition-colors duration-100',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary',
        props.className,
      )}
    >
      {props.children}
    </a>
  );
}

/** Anchor styled as secondary button */
export const ILinkBorderButton: FC<LinkProps> = ({ isLoading, ...props }) => (
  <a
    {...props}
    className={cn(
      'inline-flex items-center justify-center gap-2 w-fit',
      'h-10 px-4 text-sm font-medium',
      'bg-transparent border border-gray-300 dark:border-gray-600',
      'text-gray-700 dark:text-gray-300',
      'hover:bg-gray-50 dark:hover:bg-gray-800',
      'rounded-none transition-colors duration-100',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary',
      props.className,
    )}
  >
    {isLoading ? <Spinner size="xs" /> : props.children}
  </a>
);
