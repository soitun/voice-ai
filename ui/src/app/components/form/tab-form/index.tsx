import { cn } from '@/utils';
import { FC, HTMLAttributes, ReactElement } from 'react';
import { Notification } from '@/app/components/carbon/notification';

interface TabFormProps extends HTMLAttributes<HTMLDivElement> {
  activeTab?: string;
  onChangeActiveTab: (code: string) => void;
  errorMessage?: string;
  formHeading?: string;
  form: {
    code: string;
    name: string;
    description?: string;
    body: ReactElement;
    actions: ReactElement[];
  }[];
}

export const TabForm: FC<TabFormProps> = ({
  activeTab,
  onChangeActiveTab,
  errorMessage,
  formHeading,
  form,
}) => {
  const activeIndex = form.findIndex(f => f.code === activeTab);

  return (
    <section className="flex flex-1 min-h-0 overflow-hidden">
      {/* IBM Carbon Progress Indicator — sidebar */}
      <aside className="w-80 hidden md:flex flex-col shrink-0 border-r border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-950">
        <div className="px-6 pt-6 pb-5 border-b border-gray-200 dark:border-gray-800">
          <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400 mb-1.5">
            Setup Progress
          </p>
          {formHeading && (
            <p className="text-xs text-gray-500 dark:text-gray-500 leading-relaxed">
              {formHeading}
            </p>
          )}
        </div>

        <nav className="flex-1 px-6 py-6 overflow-y-auto">
          <ol>
            {form.map((item, index) => {
              const isActive = item.code === activeTab;
              const isCompleted = index < activeIndex;
              const isLast = index === form.length - 1;

              return (
                <li key={item.code} className="relative flex gap-0">
                  {/* Vertical connecting line — centered under the 32px circle */}
                  <div className="flex flex-col items-center mr-4 shrink-0">
                    {/* Step circle */}
                    <span
                      className={cn(
                        'flex-shrink-0 flex items-center justify-center w-8 h-8 rounded-full text-xs font-semibold transition-colors duration-150',
                        isCompleted && 'bg-primary text-white',
                        isActive &&
                          !isCompleted &&
                          'bg-primary text-white ring-4 ring-primary/10',
                        !isActive &&
                          !isCompleted &&
                          'bg-white dark:bg-gray-900 border-2 border-gray-300 dark:border-gray-700 text-gray-500 dark:text-gray-400',
                      )}
                    >
                      {isCompleted ? (
                        <svg
                          className="w-3.5 h-3.5"
                          viewBox="0 0 24 24"
                          fill="none"
                          aria-hidden="true"
                        >
                          <path
                            d="M5 13L9 17L19 7"
                            stroke="currentColor"
                            strokeWidth="2.5"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          />
                        </svg>
                      ) : (
                        <span>{String(index + 1).padStart(2, '0')}</span>
                      )}
                    </span>
                    {/* Connecting line */}
                    {!isLast && (
                      <span
                        className={cn(
                          'w-px flex-1 min-h-8',
                          isCompleted
                            ? 'bg-primary'
                            : 'bg-gray-200 dark:bg-gray-800',
                        )}
                      />
                    )}
                  </div>

                  {/* Step labels */}
                  <button
                    type="button"
                    className={cn(
                      'flex flex-col pb-8 pt-0.5 text-left min-w-0 flex-1',
                      isLast && 'pb-0',
                    )}
                    onClick={() => onChangeActiveTab(item.code)}
                  >
                    <span
                      className={cn(
                        'text-[10px] font-semibold tracking-[0.1em] uppercase mb-0.5',
                        isActive
                          ? 'text-primary'
                          : 'text-gray-500 dark:text-gray-400',
                      )}
                    >
                      Step {index + 1}
                    </span>
                    <span
                      className={cn(
                        'text-sm font-medium leading-snug',
                        isActive
                          ? 'text-gray-900 dark:text-gray-100'
                          : isCompleted
                            ? 'text-gray-500 dark:text-gray-400'
                            : 'text-gray-500 dark:text-gray-400',
                      )}
                    >
                      {item.name}
                    </span>
                    {item.description && (
                      <span className="text-xs text-gray-500 dark:text-gray-400 mt-1 leading-relaxed">
                        {item.description}
                      </span>
                    )}
                  </button>
                </li>
              );
            })}
          </ol>
        </nav>
      </aside>

      {/* Content area — flex column so footer sits below scroll */}
      <div className="flex-1 min-h-0 flex flex-col bg-white dark:bg-gray-900">
        {/* Scrollable region */}
        <div className="flex-1 min-h-0 overflow-y-auto">
          {form.map(
            item =>
              item.code === activeTab && (
                <div key={`form-body-${item.code}`}>
                  {/* IBM Carbon step content header */}
                  <header className="px-8 pt-8 pb-6 border-b border-gray-200 dark:border-gray-800">
                    <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400 mb-1.5">
                      Step {activeIndex + 1} of {form.length}
                    </p>
                    <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 leading-tight">
                      {item.name}
                    </h1>
                    {item.description && (
                      <p className="text-sm text-gray-500 dark:text-gray-500 mt-1.5 leading-relaxed">
                        {item.description}
                      </p>
                    )}
                  </header>

                  {item.body}
                </div>
              ),
          )}
        </div>

        {/* Footer — full width buttons like Carbon modal footer */}
        {form.map(
          item =>
            item.code === activeTab && (
              <div
                key={`footer-${item.code}`}
                className="shrink-0"
              >
                {errorMessage && (
                  <Notification kind="error" title="Error" subtitle={errorMessage} />
                )}
                {item.actions.map((action, idx) => (
                  <div key={`action-${idx}`}>{action}</div>
                ))}
              </div>
            ),
        )}
      </div>
    </section>
  );
};
