import { FC, useState } from 'react';
import { Filter } from '@carbon/icons-react';
import { Button, Checkbox } from '@carbon/react';
import { cn } from '@/utils';

export interface FilterOption {
  id: string;
  label: string;
}

export interface TableToolbarFilterProps {
  filters: FilterOption[];
  activeFilters: Set<string>;
  onApplyFilter: (activeFilters: Set<string>) => void;
  onResetFilter: () => void;
  className?: string;
}

export const TableToolbarFilter: FC<TableToolbarFilterProps> = ({
  filters,
  activeFilters,
  onApplyFilter,
  onResetFilter,
  className,
}) => {
  const [open, setOpen] = useState(false);
  const [localFilters, setLocalFilters] = useState<Set<string>>(
    new Set(activeFilters),
  );

  const handleToggle = (id: string) => {
    setLocalFilters(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleApply = () => {
    onApplyFilter(localFilters);
    setOpen(false);
  };

  const handleReset = () => {
    setLocalFilters(new Set());
    onResetFilter();
    setOpen(false);
  };

  return (
    <div className={cn('relative flex items-center', className)}>
      <Button
        hasIconOnly
        renderIcon={Filter}
        iconDescription="Filter"
        kind="ghost"
        onClick={() => {
          setLocalFilters(new Set(activeFilters));
          setOpen(!open);
        }}
        className={cn(activeFilters.size > 0 && '!text-blue-600')}
        tooltipPosition="bottom"
      />
      {open && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => setOpen(false)}
          />
          <div className="absolute right-0 top-full z-50 w-56 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 shadow-lg">
            <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-700">
              <p className="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400">
                Filter by type
              </p>
            </div>
            <div className="max-h-64 overflow-y-auto py-2">
              {filters.map(f => (
                <div key={f.id} className="px-4 py-0.5">
                  <Checkbox
                    id={`filter-${f.id}`}
                    labelText={f.label}
                    checked={localFilters.has(f.id)}
                    onChange={() => handleToggle(f.id)}
                  />
                </div>
              ))}
            </div>
            <div className="flex border-t border-gray-200 dark:border-gray-700">
              <Button
                size="md"
                kind="secondary"
                className="!flex-1 !max-w-none !rounded-none"
                onClick={handleReset}
              >
                Reset
              </Button>
              <Button
                size="md"
                kind="primary"
                className="!flex-1 !max-w-none !rounded-none"
                onClick={handleApply}
              >
                Apply
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  );
};
