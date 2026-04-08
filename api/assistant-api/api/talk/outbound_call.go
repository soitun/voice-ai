// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"fmt"

	channel_pipeline "github.com/rapidaai/api/assistant-api/internal/channel/pipeline"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// CreatePhoneCall initiates an outbound phone call.
// Thin controller — pipeline handles: validate, load assistant, create conversation,
// save context, create observer, dispatch. Controller just validates input and waits.
func (cApi *ConversationGrpcApi) CreatePhoneCall(ctx context.Context, ir *protos.CreatePhoneCallRequest) (*protos.CreatePhoneCallResponse, error) {
	auth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return utils.AuthenticateError[protos.CreatePhoneCallResponse]()
	}

	toNumber := ir.GetToNumber()
	if utils.IsEmpty(toNumber) {
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, fmt.Errorf("missing to_phone parameter"), "Please provide the required to_phone parameter.")
	}

	mtd, err := utils.AnyMapToInterfaceMap(ir.GetMetadata())
	if err != nil {
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Illegal metadata.")
	}
	args, err := utils.AnyMapToInterfaceMap(ir.GetArgs())
	if err != nil {
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Illegal arguments.")
	}
	opts, err := utils.AnyMapToInterfaceMap(ir.GetOptions())
	if err != nil {
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Illegal options.")
	}

	// Pipeline handles the full outbound flow
	result := cApi.channelPipeline.Run(ctx, channel_pipeline.OutboundRequestedPipeline{
		ID:          fmt.Sprintf("%d", ir.GetAssistant().GetAssistantId()),
		Auth:        auth,
		AssistantID: ir.GetAssistant().GetAssistantId(),
		Version:     ir.GetAssistant().GetVersion(),
		ToPhone:     toNumber,
		FromPhone:   ir.GetFromNumber(),
		Metadata:    mtd,
		Args:        args,
		Options:     opts,
	})

	if result.Error != nil {
		cApi.logger.Errorf("outbound call failed: %v", result.Error)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](500, result.Error, "Failed to initiate outbound call")
	}

	cApi.logger.Infof("outbound call dispatched: contextId=%s, conversationId=%d",
		result.ContextID, result.ConversationID)

	return utils.Success[protos.CreatePhoneCallResponse, *protos.AssistantConversation](&protos.AssistantConversation{
		Id: result.ConversationID,
	})
}

// InitiateBulkAssistantTalk implements protos.TalkServiceServer.
func (cApi *ConversationGrpcApi) CreateBulkPhoneCall(ctx context.Context, ir *protos.CreateBulkPhoneCallRequest) (*protos.CreateBulkPhoneCallResponse, error) {
	_, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return utils.AuthenticateError[protos.CreateBulkPhoneCallResponse]()
	}
	return utils.ErrorWithCode[protos.CreateBulkPhoneCallResponse](501, fmt.Errorf("not implemented"), "Bulk phone call not yet implemented")
}
