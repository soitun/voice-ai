import React, { FC } from 'react';
import { Assistant } from '@rapidaai/react';
import {
  toDate,
  toHumanReadableDate,
  toHumanReadableDateFromDate,
  toHumanReadableRelativeDay,
} from '@/utils/date';
import TooltipPlus from '@/app/components/base/tooltip-plus';
import SourceIndicator from '@/app/components/indicators/source';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { AssistantConversation } from '@rapidaai/react';
import { ArrowRight } from '@carbon/icons-react';
import { ActionCard } from '@/app/components/base/cards';

const SingleAssistant: FC<{ assistant: Assistant }> = ({ assistant }) => {
  const gn = useGlobalNavigation();

  const hasDeployment =
    assistant.getApideployment() ||
    assistant.getDebuggerdeployment() ||
    assistant.getWebplugindeployment() ||
    assistant.getPhonedeployment();

  return (
    <ActionCard onClick={() => gn.goToAssistant(assistant.getId())}>
      {/* ── Card header ── */}
      <div className="px-4 pt-4 pb-3">
        <div className="text-base text-gray-900 dark:text-gray-300 leading-snug mb-1.5 truncate">
          {assistant.getName()}
        </div>
        {/* Stats row */}
        <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span>
            Sessions: {assistant.getAssistantconversationsList().length}
          </span>
          <span className="w-px h-3 bg-gray-300 dark:bg-gray-600" />
          <span>
            Users:{' '}
            {
              assistant
                .getAssistantconversationsList()
                .map(x => x.getIdentifier())
                .filter((v, i, self) => self.indexOf(v) === i).length
            }
          </span>
        </div>
      </div>

      {/* ── Sparkline chart ── */}
      <ConversationChart
        conversations={assistant.getAssistantconversationsList()}
      />

      {/* ── Footer: deployments + open arrow ── */}
      <div className="px-4 py-3 border-t border-gray-100 dark:border-gray-800 flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400 mb-2">
            Deployments
          </p>
          {hasDeployment ? (
            <div className="flex flex-wrap gap-1.5 items-center">
              {assistant.getApideployment() && (
                <DeploymentBadge
                  source="react-sdk"
                  tooltip={`API Deployment · ${toHumanReadableRelativeDay(assistant.getApideployment()?.getCreateddate()!)}`}
                />
              )}
              {assistant.getDebuggerdeployment() && (
                <DeploymentBadge
                  source="debugger"
                  tooltip={`Debugger · ${toHumanReadableRelativeDay(assistant.getDebuggerdeployment()?.getCreateddate()!)}`}
                />
              )}
              {assistant.getWebplugindeployment() && (
                <DeploymentBadge
                  source="web-plugin"
                  tooltip={`Web Plugin · ${toHumanReadableRelativeDay(assistant.getWebplugindeployment()?.getCreateddate()!)}`}
                />
              )}
              {assistant.getPhonedeployment() && (
                <DeploymentBadge
                  source="twilio-call"
                  tooltip={`Phone · ${toHumanReadableRelativeDay(assistant.getPhonedeployment()?.getCreateddate()!)}`}
                />
              )}
            </div>
          ) : (
            // Carbon empty-state inline CTA
            <button
              className="flex items-center gap-1 text-xs text-primary hover:underline underline-offset-2"
              onClick={e => {
                e.stopPropagation();
                gn.goToManageAssistant(assistant.getId());
              }}
            >
              Set up deployment
              <ArrowRight size={12} />
            </button>
          )}
        </div>

        {/* Carbon clickable-tile arrow — communicates the card is navigable */}
        <ArrowRight
          size={16}
          className="shrink-0 text-gray-300 dark:text-gray-600 group-hover:text-primary transition-colors mt-0.5"
        />
      </div>
    </ActionCard>
  );
};

/** Deployment badge with tooltip */
const DeploymentBadge: FC<{ source: string; tooltip: string }> = ({
  source,
}) => (
  <SourceIndicator source={source} withLabel={false} />
);

const ConversationChart: FC<{
  conversations: Array<AssistantConversation>;
}> = ({ conversations }) => {
  const groupedData = conversations.reduce(
    (acc, conversation) => {
      const date = toHumanReadableDate(conversation.getCreateddate()!);
      if (!acc[date]) {
        acc[date] = { activeUsers: new Set(), totalSessions: 0 };
      }
      acc[date].activeUsers.add(conversation.getIdentifier());
      acc[date].totalSessions += 1;
      return acc;
    },
    {} as Record<string, { activeUsers: Set<string>; totalSessions: number }>,
  );

  // Compute range: from earliest conversation (or 30 days ago, whichever is earlier) to today
  const today = new Date();
  const thirtyDaysAgo = new Date();
  thirtyDaysAgo.setDate(today.getDate() - 30);

  let earliest = thirtyDaysAgo;
  conversations.forEach(c => {
    const d = toDate(c.getCreateddate()!);
    if (d < earliest) earliest = d;
  });

  const dayCount = Math.max(
    Math.ceil((today.getTime() - earliest.getTime()) / (1000 * 60 * 60 * 24)) + 1,
    30,
  );

  const last30Days = Array.from({ length: dayCount }, (_, i) => {
    const date = new Date(today);
    date.setDate(today.getDate() - (dayCount - 1 - i));
    return toHumanReadableDateFromDate(date);
  });

  const dataArray = last30Days.map(date => ({
    date,
    activeUsers: 0,
    totalSessions: 0,
  }));

  Object.entries(groupedData).forEach(([date, data]) => {
    const index = dataArray.findIndex(d => d.date === date);
    if (index !== -1) {
      dataArray[index].activeUsers = data.activeUsers.size;
      dataArray[index].totalSessions = data.totalSessions;
    }
  });

  const maxSessions = Math.max(...dataArray.map(d => d.totalSessions));
  const maxHeight = 48;

  return (
    <div className="relative w-full h-16 bg-gray-50 dark:bg-gray-950/70">
      {dataArray.length > 0 && (
        <div className="absolute inset-0 flex items-end px-px">
          {dataArray.map((data, i) => {
            const barHeight =
              (data.totalSessions / maxSessions) * maxHeight || 3;

            return (
              <div key={i} className="flex-1 px-px">
                <TooltipPlus
                  className="bg-white dark:bg-gray-950 border rounded-none px-0 py-0 w-56"
                  popupContent={
                    <div className="divide-y text-xs dark:text-gray-400 text-gray-700">
                      <div className="px-3 py-2 space-y-1">
                        <div className="flex justify-between">
                          <span>Active Users</span>
                          <span className="font-semibold">
                            {data.activeUsers}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Sessions</span>
                          <span className="font-semibold">
                            {data.totalSessions}
                          </span>
                        </div>
                      </div>
                      <div className="px-3 py-1.5 text-gray-400">
                        {data.date}
                      </div>
                    </div>
                  }
                >
                  <div className="h-full flex items-end">
                    <div
                      className="bg-primary w-full"
                      style={{
                        height: `${barHeight}px`,
                        opacity:
                          data.totalSessions === 0
                            ? 0.15
                            : 0.2 + (data.totalSessions / maxSessions) * 0.8,
                      }}
                    />
                  </div>
                </TooltipPlus>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default React.memo(SingleAssistant);
