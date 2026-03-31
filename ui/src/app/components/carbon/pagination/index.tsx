import type { FC } from 'react';
import { Pagination as CarbonPagination } from '@carbon/react';
import { cn } from '@/utils';

// ─── Types ───────────────────────────────────────────────────────────────────

type PaginationSize = 'sm' | 'md' | 'lg';

export interface CarbonPaginationProps {
  totalItems: number;
  page: number;
  pageSize: number;
  pageSizes: number[];
  onChange: (data: { page: number; pageSize: number }) => void;
  className?: string;
  size?: PaginationSize;
  disabled?: boolean;
  backwardText?: string;
  forwardText?: string;
  itemsPerPageText?: string;
  pageInputDisabled?: boolean;
  pageSizeInputDisabled?: boolean;
  pagesUnknown?: boolean;
  isLastPage?: boolean;
  id?: string | number;
}

/** Carbon Pagination — page navigation with items-per-page selector. */
export const Pagination: FC<CarbonPaginationProps> = ({
  totalItems,
  page,
  pageSize,
  pageSizes,
  onChange,
  className,
  size = 'md',
  disabled = false,
  backwardText = 'Previous page',
  forwardText = 'Next page',
  itemsPerPageText = 'Items per page:',
  pageInputDisabled = false,
  pageSizeInputDisabled = false,
  pagesUnknown = false,
  isLastPage = false,
  id,
}) => {
  return (
    <CarbonPagination
      totalItems={totalItems}
      page={page}
      pageSize={pageSize}
      pageSizes={pageSizes}
      onChange={onChange}
      className={cn(className)}
      size={size}
      disabled={disabled}
      backwardText={backwardText}
      forwardText={forwardText}
      itemsPerPageText={itemsPerPageText}
      pageInputDisabled={pageInputDisabled}
      pageSizeInputDisabled={pageSizeInputDisabled}
      pagesUnknown={pagesUnknown}
      isLastPage={isLastPage}
      id={id}
    />
  );
};
