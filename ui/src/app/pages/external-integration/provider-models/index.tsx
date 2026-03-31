import { Helmet } from '@/app/components/helmet';
import { ProviderCard } from '@/app/components/base/cards/provider-card';
import { BluredWrapper } from '@/app/components/wrapper/blured-wrapper';
import { SearchIconInput } from '@/app/components/form/input/IconInput';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { cn } from '@/utils';
import { INTEGRATION_PROVIDER } from '@/providers';
import { useState } from 'react';
/**
 *
 * @returns
 */
export function ProviderModelPage() {
  const [searchTerm, setSearchTerm] = useState(''); // State to store search input
  const filteredProviders = INTEGRATION_PROVIDER.filter(provider =>
    provider.name.toLowerCase().includes(searchTerm.toLowerCase()),
  );

  return (
    <div className={cn('flex flex-col h-full flex-1 overflow-auto')}>
      <Helmet title="Providers and Models" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Providers and Models</PageTitleBlock>
          <span className="text-xs text-gray-500 dark:text-gray-400 tabular-nums">
            {`${INTEGRATION_PROVIDER.length}/${INTEGRATION_PROVIDER.length}`}
          </span>
        </div>
      </PageHeaderBlock>
      <BluredWrapper className="sticky top-0 z-11">
        <SearchIconInput
          className="bg-light-background flex-1"
          onChange={e => setSearchTerm(e.target.value)}
        />
      </BluredWrapper>
      <section className="grid content-start grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4 grow shrink-0 m-4">
        {filteredProviders.map((mp, idx) => {
          return (
            <ProviderCard
              key={`spc-${idx}`}
              provider={mp}
              className="col-span-1"
            />
          );
        })}
      </section>
    </div>
  );
}
