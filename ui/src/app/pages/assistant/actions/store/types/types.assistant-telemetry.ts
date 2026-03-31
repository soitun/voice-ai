import { AssistantTelemetryProvider } from '@rapidaai/react';
import { ColumnarType, PaginatedType } from '@/types';

export type AssistantTelemetryProperty = {
  telemetries: AssistantTelemetryProvider[];
};

export type AssistantTelemetryType = {
  onChangeAssistantTelemetries: (ep: AssistantTelemetryProvider[]) => void;
  getAssistantTelemetry: (
    assistantId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantTelemetryProvider[]) => void,
  ) => void;
  deleteAssistantTelemetry: (
    assistantId: string,
    telemetryId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantTelemetryProvider) => void,
  ) => void;
  clear: () => void;
} & AssistantTelemetryProperty &
  PaginatedType &
  ColumnarType;
