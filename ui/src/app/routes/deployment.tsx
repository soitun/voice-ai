import { MissionBox } from '@/app/components/container/mission-box';
import { ProtectedBox } from '@/app/components/container/protected-box';
import {
  DeploymentEndpointPage,
  DeploymentCreateEndpointPage,
  DeploymentViewEndpointPage,
  DeploymentCreateVersionEndpointPage,
  DeploymentConfigureEndpointPage,
} from '@/app/pages/endpoint';

import {
  DeploymentCreateAssistantPage,
  DeploymentAssistantPage,
  DeploymentViewAssistantPage,
  DeploymentCreateVersionAssistantPage,
  DeploymentConfigureAssistantWebDeploymentPage,
  DeploymentConfigureAssistantCallDeploymentPage,
  DeploymentConfigureAssistantToolPage,
  DeploymentConfigureAssistantAnalysisPage,
  DeploymentConfigureAssistantApiDeploymentPage,
  DeploymentConfigureAssistantDebuggerDeploymentPage,
  DeploymentEditAssistantPage,
  DeploymentConfigureAssistantWebhookPage,
  DeploymentCreateAssistantWebhookPage,
  DeploymentUpdateAssistantWebhookPage,
  DeploymentConfigureAssistantTelemetryPage,
  DeploymentCreateAssistantTelemetryPage,
  DeploymentUpdateAssistantTelemetryPage,
  DeploymentConversationDetailPage,
  DeploymentCreateAssistantAnalysisPage,
  DeploymentUpdateAssistantAnalysisPage,
  DeploymentUpdateAssistantToolPage,
  DeploymentCreateAssistantToolPage,
  DeploymentConfigureAssistantDeploymentPage,
  DeploymentCreateWebsocketVersionAssistantPage,
  DeploymentCreateAgentKitVersionAssistantPage,
  DeploymentCreateAgentKitPage,
  DeploymentCreateWebsocketPage,
} from '@/app/pages/assistant';
import { AssistantManageLayout } from '@/app/pages/assistant/actions/assistant-manage.layout';
import { AssistantViewLayout } from '@/app/pages/assistant/view/assistant-view.layout';
import { StaticPageNotFoundPage } from '@/app/pages/static-pages';
import { Navigate, Outlet, Route, Routes } from 'react-router-dom';

export function DeploymentRoute() {
  return (
    <Routes>
      <Route path="*" element={<StaticPageNotFoundPage />} />
      <Route
        key="/deployment/"
        path="/"
        element={
          <ProtectedBox>
            <MissionBox>
              <Outlet />
            </MissionBox>
          </ProtectedBox>
        }
      >
        {/* disvoer  */}
        {/* endpoint  */}
        <Route
          key={'/endpoint/'}
          path={''}
          element={<DeploymentEndpointPage />}
        />
        <Route
          key={'/deployment/endpoint'}
          path={'endpoint'}
          element={<DeploymentEndpointPage />}
        />

        <Route index element={<Navigate to="assistant/" replace />} />
        <Route
          path={'endpoint/create-endpoint'}
          element={<DeploymentCreateEndpointPage />}
        />
        <Route
          path={'endpoint/:endpointId/create-endpoint-version'}
          element={<DeploymentCreateVersionEndpointPage />}
        />
        {/*  */}
        <Route
          path={'endpoint/configure-endpoint'}
          element={<DeploymentCreateEndpointPage />}
        />
        <Route
          path={'endpoint/configure-endpoint/:endpointId'}
          element={<DeploymentConfigureEndpointPage />}
        />
        <Route
          path={'endpoint/:endpointId'}
          element={<DeploymentViewEndpointPage />}
        />
        <Route
          path={'endpoint/:endpointId/:endpointProviderId'}
          element={<DeploymentViewEndpointPage />}
        />
        {/* assistant routes */}
        <Route
          key={'/deployment/assistant'}
          path={'assistant'}
          element={<DeploymentAssistantPage />}
        />

        <Route
          path={'assistant/:assistantId'}
          element={
            <AssistantViewLayout>
              <Outlet />
            </AssistantViewLayout>
          }
        >
          <Route index element={<Navigate to="overview" replace />} />
          <Route path={':tab'} element={<DeploymentViewAssistantPage />} />

          {/* Session detail */}
          <Route
            path={'sessions/:sessionId'}
            element={<DeploymentConversationDetailPage />}
          />

          {/* Create version */}
          <Route
            path={'create-new-version'}
            element={<DeploymentCreateVersionAssistantPage />}
          />
          <Route
            path={'create-websocket-version'}
            element={<DeploymentCreateWebsocketVersionAssistantPage />}
          />
          <Route
            path={'create-agentkit-version'}
            element={<DeploymentCreateAgentKitVersionAssistantPage />}
          />

          {/* Configure pages */}
          <Route
            path={'edit-assistant/'}
            element={<DeploymentEditAssistantPage />}
          />
          <Route
            path={'configure-analysis/'}
            element={<DeploymentConfigureAssistantAnalysisPage />}
          />
          <Route
            path={'configure-analysis/create'}
            element={<DeploymentCreateAssistantAnalysisPage />}
          />
          <Route
            path={'configure-analysis/:analysisId'}
            element={<DeploymentUpdateAssistantAnalysisPage />}
          />
          <Route
            path={'configure-tool'}
            element={<DeploymentConfigureAssistantToolPage />}
          />
          <Route
            path={'configure-tool/create'}
            element={<DeploymentCreateAssistantToolPage />}
          />
          <Route
            path={'configure-tool/:assistantToolId'}
            element={<DeploymentUpdateAssistantToolPage />}
          />
          <Route
            path={'configure-webhook/'}
            element={<DeploymentConfigureAssistantWebhookPage />}
          />
          <Route
            path={'configure-webhook/create'}
            element={<DeploymentCreateAssistantWebhookPage />}
          />
          <Route
            path={'configure-webhook/:webhookId'}
            element={<DeploymentUpdateAssistantWebhookPage />}
          />
          <Route
            path={'configure-telemetry/'}
            element={<DeploymentConfigureAssistantTelemetryPage />}
          />
          <Route
            path={'configure-telemetry/create'}
            element={<DeploymentCreateAssistantTelemetryPage />}
          />
          <Route
            path={'configure-telemetry/:telemetryId'}
            element={<DeploymentUpdateAssistantTelemetryPage />}
          />
          <Route
            path={'deployment/'}
            element={<DeploymentConfigureAssistantDeploymentPage />}
          />
          <Route
            path={'deployment/web/'}
            element={<DeploymentConfigureAssistantWebDeploymentPage />}
          />
          <Route
            path={'deployment/call/'}
            element={<DeploymentConfigureAssistantCallDeploymentPage />}
          />
          <Route
            path={'deployment/api/'}
            element={<DeploymentConfigureAssistantApiDeploymentPage />}
          />
          <Route
            path={'deployment/debugger/'}
            element={<DeploymentConfigureAssistantDebuggerDeploymentPage />}
          />
        </Route>
        <Route
          key={'/deployment/assistant/create-assistant'}
          path={'assistant/create-assistant'}
          element={<DeploymentCreateAssistantPage />}
        />
        <Route
          key={'/deployment/assistant/connect-websocket'}
          path={'assistant/connect-websocket'}
          element={<DeploymentCreateWebsocketPage />}
        />
        <Route
          key={'/deployment/assistant/create-agentkit'}
          path={'assistant/connect-agentkit'}
          element={<DeploymentCreateAgentKitPage />}
        />
      </Route>
    </Routes>
  );
}
