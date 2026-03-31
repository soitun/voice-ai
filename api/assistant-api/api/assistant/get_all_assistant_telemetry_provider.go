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

func (assistantApi *assistantGrpcApi) GetAllAssistantTelemetryProvider(
	ctx context.Context,
	req *protos.GetAllAssistantTelemetryProviderRequest,
) (*protos.GetAllAssistantTelemetryProviderResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAllAssistantTelemetryProviderResponse]()
	}

	paginate := req.GetPaginate()
	if paginate == nil {
		paginate = &protos.Paginate{
			Page:     1,
			PageSize: 50,
		}
	}

	cnt, providers, err := assistantApi.assistantTelemetryService.GetAll(
		ctx,
		iAuth,
		req.GetAssistantId(),
		req.GetCriterias(),
		paginate,
	)
	if err != nil {
		return exceptions.BadRequestError[protos.GetAllAssistantTelemetryProviderResponse](
			"Unable to get assistant telemetry providers.",
		)
	}

	out := []*protos.AssistantTelemetryProvider{}
	if err = utils.Cast(providers, &out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant telemetry providers %v", err)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantTelemetryProviderResponse, []*protos.AssistantTelemetryProvider](
		uint32(cnt),
		paginate.GetPage(),
		out,
	)
}
