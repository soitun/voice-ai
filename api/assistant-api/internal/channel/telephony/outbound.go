// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
	"context"
	"fmt"

	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
)

type OutboundDispatcher struct {
	cfg              *config.AssistantConfig
	store            callcontext.Store
	logger           commons.Logger
	vaultClient      web_client.VaultClient
	assistantService internal_services.AssistantService
	telephonyOpt     TelephonyOption
}

func NewOutboundDispatcher(deps TelephonyDispatcherDeps) *OutboundDispatcher {
	return &OutboundDispatcher{
		cfg:              deps.Cfg,
		store:            deps.Store,
		logger:           deps.Logger,
		vaultClient:      deps.VaultClient,
		assistantService: deps.AssistantService,
		telephonyOpt:     deps.TelephonyOpt,
	}
}

func (d *OutboundDispatcher) Dispatch(ctx context.Context, contextID string) error {
	cc, err := d.store.Claim(ctx, contextID)
	if err != nil {
		d.logger.Errorf("outbound dispatcher: failed to claim call context %s: %v", contextID, err)
		return err
	}

	d.logger.Infof("outbound dispatcher[%s]: processing contextId=%s, assistant=%d, conversation=%d",
		cc.Provider, cc.ContextID, cc.AssistantID, cc.ConversationID)

	if err := d.performOutbound(ctx, cc); err != nil {
		d.logger.Errorf("outbound dispatcher[%s]: call failed for contextId=%s: %v", cc.Provider, contextID, err)
		if updateErr := d.store.UpdateField(ctx, contextID, "status", callcontext.StatusFailed); updateErr != nil {
			d.logger.Errorf("outbound dispatcher[%s]: failed to update status for %s: %v", cc.Provider, contextID, updateErr)
		}
		return err
	}

	d.logger.Infof("outbound dispatcher[%s]: call initiated for contextId=%s", cc.Provider, contextID)
	return nil
}

func (d *OutboundDispatcher) performOutbound(ctx context.Context, cc *callcontext.CallContext) error {
	telephony, err := GetTelephony(Telephony(cc.Provider), d.cfg, d.logger, d.telephonyOpt)
	if err != nil {
		return fmt.Errorf("telephony provider %s not available: %w", cc.Provider, err)
	}

	auth := cc.ToAuth()

	assistant, err := d.assistantService.Get(ctx, auth, cc.AssistantID, nil, &internal_services.GetAssistantOption{InjectPhoneDeployment: true})
	if err != nil {
		return fmt.Errorf("failed to load assistant %d: %w", cc.AssistantID, err)
	}
	if !assistant.IsPhoneDeploymentEnable() {
		return fmt.Errorf("phone deployment not enabled for assistant %d", cc.AssistantID)
	}

	credentialID, err := assistant.AssistantPhoneDeployment.GetOptions().GetUint64("rapida.credential_id")
	if err != nil {
		return fmt.Errorf("failed to get credential ID: %w", err)
	}

	vltC, err := d.vaultClient.GetCredential(ctx, auth, credentialID)
	if err != nil {
		return fmt.Errorf("failed to get vault credential: %w", err)
	}

	opts := assistant.AssistantPhoneDeployment.GetOptions()
	opts["rapida.context_id"] = cc.ContextID

	callInfo, callErr := telephony.OutboundCall(auth, cc.CallerNumber, cc.FromNumber, cc.AssistantID, cc.ConversationID, vltC, opts)
	if callErr != nil {
		d.logger.Errorf("outbound dispatcher[%s]: telephony call failed for contextId=%s: %v", cc.Provider, cc.ContextID, callErr)
	}
	if callInfo == nil {
		return callErr
	}

	if callInfo.ChannelUUID != "" {
		if updateErr := d.store.UpdateField(ctx, cc.ContextID, "channel_uuid", callInfo.ChannelUUID); updateErr != nil {
			d.logger.Warnf("outbound dispatcher[%s]: failed to store channel UUID: %v", cc.Provider, updateErr)
		}
	}

	return callErr
}
