import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { Helmet } from '@/app/components/helmet';
import { Tab } from '@/app/components/tab';
import { AccountSetting } from '@/app/pages/user/account-setting/account-setting';
import { NotificationSetting } from '@/app/pages/user/account-setting/notification-setting';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ChevronLeft } from 'lucide-react';

export const AccountSettingPage = () => {
  const { goToDashboard } = useGlobalNavigation();
  return (
    <div className="flex flex-col flex-1 bg-white dark:bg-gray-900 h-full">
      <Helmet title="Account settings" />
      <PageHeaderBlock>
        <div className="flex items-center gap-1.5 min-w-0">
          <div
            onClick={() => goToDashboard()}
            className="flex items-center gap-1.5 text-gray-500 dark:text-gray-400 hover:text-primary transition-colors cursor-pointer shrink-0"
          >
            <ChevronLeft className="w-4 h-4" strokeWidth={1.5} />
            <span className="text-sm font-medium">Dashboard</span>
          </div>
        </div>
      </PageHeaderBlock>
      <div className="flex min-h-0 flex-1 flex-col">
        <Tab
          strict={false}
          active="Account"
          className="sticky top-0 z-3 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900"
          tabs={[
            {
              label: 'Account',
              element: <AccountSetting />,
            },
            {
              label: 'Notification',
              element: <NotificationSetting />,
            },
          ]}
        />
      </div>
    </div>
  );
};
