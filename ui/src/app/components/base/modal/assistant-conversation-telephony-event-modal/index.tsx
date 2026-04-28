import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { ModalProps } from '@/app/components/base/modal';
import { FC, HTMLAttributes, memo, useState } from 'react';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { AssistantConversationTelephonyEvent } from '@rapidaai/react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { toHumanReadableDateTime } from '@/utils/date';
import { CodeHighlighting } from '@/app/components/code-highlighting';

interface AssistantConversationTelephonyEventDialogProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  events: AssistantConversationTelephonyEvent[];
}
export const AssistantConversationTelephonyEventDialog: FC<AssistantConversationTelephonyEventDialogProps> =
  memo(({ events, ...mldAttr }) => {
    const [expandedRow, setExpandedRow] = useState<string | null>(null);
    return (
      <RightSideModal {...mldAttr} className={'min-w-[30vw]! overflow-visible'}>
        <div className="flex items-center p-4 border-b text-base/6">
          <div className="font-medium">Assistant</div>
          <ChevronRight size={18} className="mx-2" />
          <div className="font-medium">Session</div>
          <ChevronRight size={18} className="mx-2" />
          <div className="font-medium text-base">Telephony</div>
        </div>
        <div className="relative overflow-auto flex flex-col flex-1 justify-between">
          <ModalBody>
            {/* Summary Stats */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
              <div className="border dark:bg-gray-950 bg-light-background rounded-[2px] p-3">
                <p className="text-xs/6 font-medium uppercase text-muted mb-1">
                  Total Events
                </p>
                <p className="text-lg font-semibold">{events.length}</p>
              </div>
              <div className="border dark:bg-gray-950 bg-light-background rounded-[2px] p-3">
                <p className="text-xs/6 font-medium uppercase text-muted mb-1">
                  Provider
                </p>
                <p className="text-lg font-medium capitalize">
                  {events.at(events.length - 1)?.getProvider()}
                </p>
              </div>

              <div className="border dark:bg-gray-950 bg-light-background rounded-[2px] p-3">
                <p className="text-xs/6 font-medium uppercase text-muted mb-1">
                  Current Status
                </p>
                <p className="text-lg font-medium capitalize">
                  {events.at(events.length - 1)?.getEventtype()}
                </p>
              </div>
            </div>

            {/* Events Table */}
            <div className="border rounded-[2px] overflow-hidden">
              {/* Header */}
              <div className="grid grid-cols-7 gap-2 px-4 py-3 border-b text-xs font-semibold uppercase tracking-wide text-muted">
                <div className="col-span-2">Event ID</div>
                <div className="col-span-2">Event Type</div>
                <div className="col-span-2">Created</div>
                <div className="col-span-1 text-right">Action</div>
              </div>

              {/* Rows */}
              {events.map(event => {
                const createdDate = event.getCreateddate();
                return (
                  <div key={event.getId()} className="flex flex-col">
                    <button
                      onClick={() =>
                        setExpandedRow(
                          expandedRow === event.getId() ? null : event.getId(),
                        )
                      }
                      className="w-full grid grid-cols-7 gap-2 px-4 py-3 text-left border-b hover:bg-gray-50 dark:hover:bg-gray-950 transition-colors text-sm"
                    >
                      <div className="col-span-2 font-mono text-xs truncate">
                        {event.getId()}
                      </div>
                      <div className="col-span-2 font-medium capitalize">
                        {event.getEventtype()}
                      </div>
                      <div className="col-span-2  text-xs">
                        {createdDate ? toHumanReadableDateTime(createdDate) : 'N/A'}
                      </div>
                      <div className="col-span-1 flex justify-end">
                        <ChevronDown
                          className={`w-4 h-4 text-gray-400 transition-transform`}
                        />
                      </div>
                    </button>

                    {expandedRow === event.getId() && (
                      <>
                        <p className="text-xs font-semibold text-gray-700 uppercase tracking-wide px-3 py-2">
                          Payload (Raw)
                        </p>
                        <CodeHighlighting
                          language="json"
                          className="h-[200px] !text-xs"
                          lineNumbers={false}
                          foldGutter={false}
                          code={JSON.stringify(
                            event.getPayload()?.toJavaScript(),
                            null,
                            2,
                          )}
                        />
                      </>
                    )}
                  </div>
                );
              })}
            </div>
          </ModalBody>
        </div>
      </RightSideModal>
    );
  });
