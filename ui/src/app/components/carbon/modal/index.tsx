import type { FC, MouseEvent, ReactNode } from 'react';
import {
  ComposedModal,
  ModalHeader as CarbonModalHeader,
  ModalBody as CarbonModalBody,
  ModalFooter as CarbonModalFooter,
} from '@carbon/react';
import { cn } from '@/utils';

// ─── Types ───────────────────────────────────────────────────────────────────

type ModalSize = 'xs' | 'sm' | 'md' | 'lg';

export interface CarbonModalProps {
  open: boolean;
  onClose: (e?: MouseEvent) => void;
  className?: string;
  containerClassName?: string;
  size?: ModalSize;
  danger?: boolean;
  preventCloseOnClickOutside?: boolean;
  selectorPrimaryFocus?: string;
  'aria-label'?: string;
  children: ReactNode;
}

export interface CarbonModalHeaderProps {
  className?: string;
  title?: string;
  label?: string;
  children?: ReactNode;
  onClose?: () => void;
}

export interface CarbonModalBodyProps {
  className?: string;
  children: ReactNode;
  hasForm?: boolean;
  hasScrollingContent?: boolean;
}

export interface CarbonModalFooterProps {
  className?: string;
  children: ReactNode;
  danger?: boolean;
}

// ─── Modal ───────────────────────────────────────────────────────────────────

/**
 * Carbon ComposedModal — flexible modal with header, body, and footer slots.
 */
export const Modal: FC<CarbonModalProps> = ({
  open,
  onClose,
  className,
  containerClassName,
  size = 'md',
  danger = false,
  preventCloseOnClickOutside = false,
  selectorPrimaryFocus = '[data-modal-primary-focus]',
  children,
  ...rest
}) => {
  return (
    <ComposedModal
      open={open}
      onClose={onClose}
      className={cn(className)}
      containerClassName={cn(containerClassName)}
      size={size}
      danger={danger}
      preventCloseOnClickOutside={preventCloseOnClickOutside}
      selectorPrimaryFocus={selectorPrimaryFocus}
      {...rest}
    >
      {children}
    </ComposedModal>
  );
};

// ─── Modal Header ────────────────────────────────────────────────────────────

/**
 * Carbon ModalHeader — title bar with optional label and close button.
 */
export const ModalHeader: FC<CarbonModalHeaderProps> = ({
  className,
  title,
  label,
  children,
  onClose,
}) => {
  return (
    <CarbonModalHeader
      className={cn(className)}
      title={title}
      label={label}
      buttonOnClick={onClose}
    >
      {children}
    </CarbonModalHeader>
  );
};

// ─── Modal Body ──────────────────────────────────────────────────────────────

/**
 * Carbon ModalBody — scrollable content area.
 */
export const ModalBody: FC<CarbonModalBodyProps> = ({
  className,
  children,
  hasForm = false,
  hasScrollingContent = false,
}) => {
  return (
    <CarbonModalBody
      className={cn(className)}
      hasForm={hasForm}
      hasScrollingContent={hasScrollingContent}
    >
      {children}
    </CarbonModalBody>
  );
};

// ─── Modal Footer ────────────────────────────────────────────────────────────

/**
 * Carbon ModalFooter — action buttons area at bottom of modal.
 */
export const ModalFooter: FC<CarbonModalFooterProps> = ({
  className,
  children,
  danger = false,
}) => {
  return (
    <CarbonModalFooter className={cn(className)} danger={danger}>
      {children}
    </CarbonModalFooter>
  );
};
