import type { FC, ReactNode, ChangeEvent, MouseEvent } from 'react';
import { forwardRef } from 'react';
import {
  Form as CarbonForm,
  Stack as CarbonStack,
  Checkbox as CarbonCheckbox,
  CheckboxSkeleton as CarbonCheckboxSkeleton,
  FormGroup as CarbonFormGroup,
  TextInput as CarbonTextInput,
  TextInputSkeleton as CarbonTextInputSkeleton,
  TextArea as CarbonTextArea,
  TextAreaSkeleton as CarbonTextAreaSkeleton,
} from '@carbon/react';
import { cn } from '@/utils';

// ─── Types ───────────────────────────────────────────────────────────────────

type InputSize = 'sm' | 'md' | 'lg' | 'xl';
type StackOrientation = 'horizontal' | 'vertical';

// ─── Form ────────────────────────────────────────────────────────────────────

export interface CarbonFormProps {
  children: ReactNode;
  className?: string;
  onSubmit?: (e: React.FormEvent<HTMLFormElement>) => void;
}

/** Carbon Form — standard form wrapper with Carbon spacing. */
export const Form: FC<CarbonFormProps> = ({
  children,
  className,
  onSubmit,
}) => {
  return (
    <CarbonForm className={cn(className)} onSubmit={onSubmit}>
      {children}
    </CarbonForm>
  );
};

// ─── Stack ───────────────────────────────────────────────────────────────────

export interface CarbonStackProps {
  children: ReactNode;
  className?: string;
  gap: number;
  orientation?: StackOrientation;
}

/** Carbon Stack — layout utility for consistent spacing between elements. */
export const Stack: FC<CarbonStackProps> = ({
  children,
  className,
  gap,
  orientation = 'vertical',
}) => {
  return (
    <CarbonStack
      className={cn(className)}
      gap={gap}
      orientation={orientation}
    >
      {children}
    </CarbonStack>
  );
};

// ─── FormGroup ───────────────────────────────────────────────────────────────

export interface CarbonFormGroupProps {
  children: ReactNode;
  legendText: ReactNode;
  className?: string;
  disabled?: boolean;
  invalid?: boolean;
  message?: boolean;
  messageText?: string;
}

/** Carbon FormGroup — fieldset wrapper with a legend label. */
export const FormGroup: FC<CarbonFormGroupProps> = ({
  children,
  legendText,
  className,
  disabled = false,
  invalid = false,
  message = false,
  messageText,
}) => {
  return (
    <CarbonFormGroup
      className={cn(className)}
      legendText={legendText}
      disabled={disabled}
      invalid={invalid}
      message={message}
      messageText={messageText}
    >
      {children}
    </CarbonFormGroup>
  );
};

// ─── TextInput ───────────────────────────────────────────────────────────────

export interface CarbonTextInputProps {
  id: string;
  labelText: ReactNode;
  className?: string;
  defaultValue?: string | number;
  disabled?: boolean;
  helperText?: ReactNode;
  hideLabel?: boolean;
  inline?: boolean;
  invalid?: boolean;
  invalidText?: ReactNode;
  name?: string;
  onChange?: (e: ChangeEvent<HTMLInputElement>) => void;
  onClick?: (e: MouseEvent<HTMLElement>) => void;
  placeholder?: string;
  readOnly?: boolean;
  required?: boolean;
  size?: InputSize;
  type?: string;
  value?: string | number;
  warn?: boolean;
  warnText?: ReactNode;
  enableCounter?: boolean;
  maxCount?: number;
  autoComplete?: string;
}

/** Carbon TextInput — single-line text field with label, helper, and validation. */
export const TextInput = forwardRef<HTMLInputElement, CarbonTextInputProps>(({
  id,
  labelText,
  className,
  size = 'md',
  ...rest
}, ref) => {
  return (
    <CarbonTextInput
      ref={ref}
      id={id}
      labelText={labelText}
      className={cn(className)}
      size={size}
      {...rest}
    />
  );
});

/** Carbon TextInputSkeleton — loading placeholder for TextInput. */
export const TextInputSkeleton: FC<{
  className?: string;
  hideLabel?: boolean;
}> = ({ className, hideLabel = false }) => {
  return (
    <CarbonTextInputSkeleton className={cn(className)} hideLabel={hideLabel} />
  );
};

// ─── TextArea ────────────────────────────────────────────────────────────────

export interface CarbonTextAreaProps {
  labelText: ReactNode;
  className?: string;
  cols?: number;
  defaultValue?: string | number;
  disabled?: boolean;
  helperText?: ReactNode;
  hideLabel?: boolean;
  id?: string;
  invalid?: boolean;
  invalidText?: ReactNode;
  name?: string;
  onChange?: (e: ChangeEvent<HTMLTextAreaElement>) => void;
  placeholder?: string;
  readOnly?: boolean;
  required?: boolean;
  rows?: number;
  value?: string | number;
  warn?: boolean;
  warnText?: ReactNode;
  enableCounter?: boolean;
  maxCount?: number;
}

/** Carbon TextArea — multi-line text field with label, helper, and validation. */
export const TextArea: FC<CarbonTextAreaProps> = ({
  labelText,
  className,
  rows = 4,
  ...rest
}) => {
  return (
    <CarbonTextArea
      labelText={labelText}
      className={cn(className)}
      rows={rows}
      {...rest}
    />
  );
};

/** Carbon TextAreaSkeleton — loading placeholder for TextArea. */
export const TextAreaSkeleton: FC<{
  className?: string;
  hideLabel?: boolean;
}> = ({ className, hideLabel = false }) => {
  return (
    <CarbonTextAreaSkeleton className={cn(className)} hideLabel={hideLabel} />
  );
};

// ─── Checkbox ────────────────────────────────────────────────────────────────

export interface CarbonCheckboxProps {
  id: string;
  labelText: ReactNode;
  className?: string;
  checked?: boolean;
  defaultChecked?: boolean;
  disabled?: boolean;
  helperText?: ReactNode;
  hideLabel?: boolean;
  indeterminate?: boolean;
  invalid?: boolean;
  invalidText?: ReactNode;
  name?: string;
  onChange?: (
    e: ChangeEvent<HTMLInputElement>,
    data: { checked: boolean; id: string },
  ) => void;
  onClick?: (e: MouseEvent<HTMLInputElement>) => void;
  warn?: boolean;
  warnText?: ReactNode;
}

/** Carbon Checkbox — single checkbox with label and validation support. */
export const Checkbox: FC<CarbonCheckboxProps> = ({
  id,
  labelText,
  className,
  ...rest
}) => {
  return (
    <CarbonCheckbox
      id={id}
      labelText={labelText}
      className={cn(className)}
      {...rest}
    />
  );
};

/** Carbon CheckboxSkeleton — loading placeholder for Checkbox. */
export const CheckboxSkeleton: FC<{ className?: string }> = ({
  className,
}) => {
  return <CarbonCheckboxSkeleton className={cn(className)} />;
};
