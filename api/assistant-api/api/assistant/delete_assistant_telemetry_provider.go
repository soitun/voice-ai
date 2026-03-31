// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) DeleteAssistantTelemetryProvider(
	ctx context.Context,
	req *protos.DeleteAssistantTelemetryProviderRequest,
) (*protos.GetAssistantTelemetryProviderResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAssistantTelemetryProviderResponse]()
	}

	provider, err := assistantApi.assistantTelemetryService.Delete(
		ctx,
		iAuth,
		req.GetId(),
		req.GetAssistantId(),
	)
	if err != nil {
		return utils.Error[protos.GetAssistantTelemetryProviderResponse](
			err,
			"Unable to delete assistant telemetry provider, please try again in sometime",
		)
	}

	out := &protos.AssistantTelemetryProvider{}
	if err = utils.Cast(provider, out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant telemetry provider to response object")
	}

	return utils.Success[protos.GetAssistantTelemetryProviderResponse, *protos.AssistantTelemetryProvider](out)
}
