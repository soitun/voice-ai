import { create } from 'zustand';
import { initialPaginated } from '@/types/types.paginated';
import {
  AssistantTelemetryProvider,
  DeleteAssistantTelemetryProvider,
  GetAllAssistantTelemetryProvider,
  GetAllAssistantTelemetryProviderResponse,
  GetAssistantTelemetryProviderResponse,
  ServiceError,
} from '@rapidaai/react';
import {
  AssistantTelemetryProperty,
  AssistantTelemetryType,
} from './types/types.assistant-telemetry';
import { connectionConfig } from '@/configs';

const initialAssistantTelemetry: AssistantTelemetryProperty = {
  telemetries: [],
};

export const useAssistantTelemetryPageStore = create<AssistantTelemetryType>(
  (set, get) => ({
    ...initialAssistantTelemetry,
    ...initialPaginated,

    setPageSize: (pageSize: number) => {
      set({ page: 1, pageSize });
    },

    setPage: (pg: number) => {
      set({ page: pg });
    },

    setTotalCount: (tc: number) => {
      set({ totalCount: tc });
    },

    onChangeAssistantTelemetries: (ep: AssistantTelemetryProvider[]) => {
      set({ telemetries: ep });
    },

    addCriteria: (k: string, v: string, logic: string) => {
      let current = get().criteria.filter(
        x => x.key !== k && x.logic !== logic,
      );
      if (v) current.push({ key: k, value: v, logic: logic });
      set({ criteria: current });
    },

    addCriterias: (v: { k: string; v: string; logic: string }[]) => {
      let current = get().criteria.filter(
        x => !v.find(y => y.k === x.key && x.logic === y.logic),
      );
      v.forEach(c => {
        current.push({ key: c.k, value: c.v, logic: c.logic });
      });
      set({ criteria: current });
    },

    getAssistantTelemetry: (
      assistantId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: AssistantTelemetryProvider[]) => void,
    ) => {
      const afterGetAllAssistantTelemetry = (
        err: ServiceError | null,
        gur: GetAllAssistantTelemetryProviderResponse | null,
      ) => {
        if (err) {
          onError(err.message || 'Unable to fetch assistant telemetry providers.');
          return;
        }

        if (gur?.getSuccess()) {
          get().onChangeAssistantTelemetries(gur.getDataList());
          const paginated = gur.getPaginated();
          if (paginated) {
            get().setTotalCount(paginated.getTotalitem());
          }
          onSuccess(gur.getDataList());
          return;
        }

        const errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }

        onError('Unable to get assistant telemetry providers, please try again later.');
      };

      GetAllAssistantTelemetryProvider(
        connectionConfig,
        assistantId,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllAssistantTelemetry,
        {
          authorization: token,
          'x-project-id': projectId,
          'x-auth-id': userId,
        },
      );
    },

    deleteAssistantTelemetry: (
      assistantId: string,
      telemetryId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: AssistantTelemetryProvider) => void,
    ) => {
      const afterDeleteAssistantTelemetry = (
        err: ServiceError | null,
        gur: GetAssistantTelemetryProviderResponse | null,
      ) => {
        if (err) {
          onError(err.message || 'Unable to delete assistant telemetry provider.');
          return;
        }

        if (gur?.getSuccess() && gur.getData()) {
          onSuccess(gur.getData()!);
          return;
        }

        const errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }

        onError('Unable to delete assistant telemetry provider, please try again later.');
      };

      DeleteAssistantTelemetryProvider(
        connectionConfig,
        assistantId,
        telemetryId,
        afterDeleteAssistantTelemetry,
        {
          authorization: token,
          'x-project-id': projectId,
          'x-auth-id': userId,
        },
      );
    },

    columns: [
      { name: 'ID', key: 'id', visible: false },
      { name: 'Provider', key: 'providerType', visible: true },
      { name: 'Enabled', key: 'enabled', visible: true },
      { name: 'Options', key: 'options', visible: true },
      { name: 'Created Date', key: 'createdDate', visible: true },
    ],

    setColumns(cl: { name: string; key: string; visible: boolean }[]) {
      set({ columns: cl });
    },

    visibleColumn: (k: string): boolean => {
      const column = get().columns.find(c => c.key === k);
      return column ? column.visible : false;
    },

    clear: () => set({ ...initialAssistantTelemetry, ...initialPaginated }),
  }),
);
