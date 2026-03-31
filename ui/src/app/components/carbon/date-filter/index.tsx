import { FC, useState } from 'react';
import { Filter } from '@carbon/icons-react';
import {
  Button,
  DatePicker as CarbonDatePicker,
  DatePickerInput,
} from '@carbon/react';
import { cn } from '@/utils';

export interface DateFilterProps {
  onApply: (from: Date, to: Date) => void;
  onReset?: () => void;
  className?: string;
}

export const DateFilter: FC<DateFilterProps> = ({
  onApply,
  onReset,
  className,
}) => {
  const [open, setOpen] = useState(false);
  const [dates, setDates] = useState<Date[]>([]);

  const handleApply = () => {
    if (dates.length === 2) {
      onApply(dates[0], dates[1]);
    }
    setOpen(false);
  };

  const handleReset = () => {
    setDates([]);
    onReset?.();
    setOpen(false);
  };

  return (
    <div className={cn('relative flex items-center', className)}>
      <Button
        hasIconOnly
        renderIcon={Filter}
        iconDescription="Filter by date"
        kind="ghost"
        onClick={() => setOpen(!open)}
        className={cn(dates.length === 2 && '!text-blue-600')}
        tooltipPosition="bottom"
      />
      {open && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => setOpen(false)}
          />
          <div className="absolute right-0 top-full z-50 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 shadow-lg">
            <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-700">
              <p className="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400">
                Filter by date
              </p>
            </div>
            <div className="p-4">
              <CarbonDatePicker
                datePickerType="range"
                dateFormat="m/d/Y"
                value={dates.length > 0 ? dates : undefined}
                onChange={(selectedDates: Date[]) => setDates(selectedDates)}
              >
                <DatePickerInput
                  id="date-filter-from"
                  placeholder="Start date"
                  labelText="From"
                  size="md"
                />
                <DatePickerInput
                  id="date-filter-to"
                  placeholder="End date"
                  labelText="To"
                  size="md"
                />
              </CarbonDatePicker>
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
