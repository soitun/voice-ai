import { Endpoint } from '@rapidaai/react';
import { Helmet } from '@/app/components/helmet';
import { useEndpointPageStore, useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Version } from '@/app/pages/endpoint/view/version-list';
import { toHumanReadableRelativeTime } from '@/utils/date';
import { Playground } from '@/app/pages/endpoint/view/try-playground';
import { EndpointInstructionDialog } from '@/app/components/base/modal/endpoint-instruction-modal';
import { CreateTagDialog } from '@/app/components/base/modal/create-tag-modal';
import { Tag } from '@rapidaai/react';
import { useNavigate, useParams } from 'react-router-dom';
import { UpdateDescriptionDialog } from '@/app/components/base/modal/update-description-modal';
import { EndpointTag } from '@/app/components/form/tag-input/endpoint-tags';
import { EndpointTraces } from '@/app/pages/endpoint/view/traces';
import { EndpointSideNav } from '@/app/pages/endpoint/view/endpoint-side-nav';
import { Breadcrumb, BreadcrumbItem, ComboButton, MenuItem } from '@carbon/react';

export function ViewEndpointPage() {
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const [activeTab, setActiveTab] = useState('overview');
  const [navExpanded, setNavExpanded] = useState(true);
  const navigate = useNavigate();

  const handleTabChange = (tab: string) => {
    if (tab === 'create-version') {
      navigate(`/deployment/endpoint/${endpointId}/create-endpoint-version`);
      return;
    }
    setActiveTab(tab);
  };

  const {
    currentEndpoint,
    onChangeCurrentEndpoint,
    onChangeCurrentEndpointProviderModel,
    instructionVisible,
    onHideInstruction,
    currentEndpointProviderModel,
    editTagVisible,
    onHideEditTagVisible,
    onCreateEndpointTag,
    onGetEndpoint,
    updateDetailVisible,
    onHideUpdateDetailVisible,
    onUpdateEndpointDetail,
  } = useEndpointPageStore();

  const { endpointId, endpointProviderId } = useParams();

  const onError = useCallback(
    (err: string) => {
      hideLoader();
      toast.error(err);
    },
    [endpointId, endpointProviderId],
  );

  const onSuccess = useCallback(
    (data: Endpoint) => {
      onChangeCurrentEndpoint(data);
      const endpointProviderModel = data.getEndpointprovidermodel();
      if (endpointProviderModel)
        onChangeCurrentEndpointProviderModel(endpointProviderModel);
      hideLoader();
    },
    [endpointId, endpointProviderId],
  );

  const onReload = useCallback(() => {
    if (endpointId) {
      showLoader('overlay');
      onGetEndpoint(
        endpointId,
        endpointProviderId ? endpointProviderId : null,
        projectId,
        token,
        userId,
        onError,
        onSuccess,
      );
    }
  }, []);

  useEffect(() => {
    onReload();
  }, [endpointId, endpointProviderId]);

  const renderContent = () => {
    if (!currentEndpointProviderModel || !currentEndpoint) return null;
    switch (activeTab) {
      case 'overview':
        return (
          <Playground
            currentEndpoint={currentEndpoint}
            currentEndpointProviderModel={currentEndpointProviderModel}
          />
        );
      case 'Traces':
        return <EndpointTraces currentEndpoint={currentEndpoint} />;
      case 'versions':
        return <Version currentEndpoint={currentEndpoint} onReload={onReload} />;
      default:
        return null;
    }
  };

  return (
    <div className="h-full flex flex-1 overflow-hidden">
      {/* Modals */}
      <EndpointInstructionDialog
        className="w-1/2"
        modalOpen={instructionVisible}
        setModalOpen={onHideInstruction}
        currentEndpoint={currentEndpoint}
        currentEndpointProviderModel={currentEndpointProviderModel}
      />
      <UpdateDescriptionDialog
        title="Edit details"
        name={currentEndpoint?.getName()}
        modalOpen={updateDetailVisible}
        setModalOpen={onHideUpdateDetailVisible}
        description={currentEndpoint?.getDescription()}
        onUpdateDescription={(
          name: string,
          description: string,
          onError: (err: string) => void,
          onSuccess: () => void,
        ) => {
          let wId = currentEndpoint?.getId();
          if (!wId) {
            onError('Endpoint is undefined, please try again later.');
            return;
          }
          onUpdateEndpointDetail(
            wId, name, description, projectId, token, userId, onError,
            w => onSuccess(),
          );
        }}
      />
      <CreateTagDialog
        title="Edit tags"
        tags={currentEndpoint?.getEndpointtag()?.getTagList()}
        modalOpen={editTagVisible}
        allTags={EndpointTag}
        setModalOpen={onHideEditTagVisible}
        onCreateTag={(
          tags: string[],
          onError: (err: string) => void,
          onSuccess: (e: Tag) => void,
        ) => {
          let wId = currentEndpoint?.getId();
          if (!wId) {
            onError('Endpoint is undefined.');
            return;
          }
          onCreateEndpointTag(
            wId, tags, projectId, token, userId, onError,
            endpoint => {
              let tags = endpoint.getEndpointtag();
              if (tags) onSuccess(tags);
            },
          );
        }}
      />

      <Helmet title="Hosted endpoints" />

      {/* Side nav */}
      <EndpointSideNav
        activeTab={activeTab}
        onChangeTab={handleTabChange}
        expanded={navExpanded}
        onToggle={() => setNavExpanded(!navExpanded)}
      />

      {/* Main content */}
      <div className="flex flex-col flex-1 overflow-auto">
        {/* Page header — only on overview */}
        {activeTab === 'overview' && currentEndpoint && (
          <div className="px-4 pt-4 pb-4 border-b border-gray-200 dark:border-gray-800">
            <div className="flex items-start justify-between">
              <div>
                <Breadcrumb noTrailingSlash className="mb-2">
                  <BreadcrumbItem href="/deployment/endpoint">
                    Endpoints
                  </BreadcrumbItem>
                </Breadcrumb>
                <h1 className="text-2xl font-light tracking-tight">
                  {currentEndpoint.getName()}
                </h1>
                {currentEndpoint.getEndpointprovidermodel()?.getCreateddate() && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 tabular-nums">
                    Last updated{' '}
                    {toHumanReadableRelativeTime(
                      currentEndpoint.getEndpointprovidermodel()?.getCreateddate()!,
                    )}
                  </p>
                )}
              </div>
              <ComboButton
                label="Create new version"
                size="md"
                onClick={() =>
                  navigate(`/deployment/endpoint/${endpointId}/create-endpoint-version`)
                }
              >
                <MenuItem
                  label="View instructions"
                  onClick={() => {
                    const store = useEndpointPageStore.getState();
                    store.onShowInstruction();
                  }}
                />
                <MenuItem
                  label="Edit details"
                  onClick={() => {
                    if (currentEndpoint) {
                      const store = useEndpointPageStore.getState();
                      store.onShowUpdateDetailVisible(currentEndpoint);
                    }
                  }}
                />
                <MenuItem
                  label="Edit tags"
                  onClick={() => {
                    if (currentEndpoint) {
                      const store = useEndpointPageStore.getState();
                      store.onShowEditTagVisible(currentEndpoint);
                    }
                  }}
                />
              </ComboButton>
            </div>
          </div>
        )}
        {renderContent()}
      </div>
    </div>
  );
}
