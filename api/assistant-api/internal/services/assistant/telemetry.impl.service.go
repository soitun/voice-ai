// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"fmt"
	"time"

	internal_telemetry_entity "github.com/rapidaai/api/assistant-api/internal/entity/telemetry"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

type assistantTelemetryProviderService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewAssistantTelemetryProviderService(
	logger commons.Logger,
	postgres connectors.PostgresConnector,
) internal_services.AssistantTelemetryProviderService {
	return &assistantTelemetryProviderService{
		logger:   logger,
		postgres: postgres,
	}
}

func (eService *assistantTelemetryProviderService) Get(
	ctx context.Context,
	auth types.SimplePrinciple,
	telemetryProviderId uint64,
	assistantId uint64,
) (*internal_telemetry_entity.AssistantTelemetryProvider, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var provider *internal_telemetry_entity.AssistantTelemetryProvider
	tx := db.Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
		Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ?",
			telemetryProviderId,
			assistantId,
			*auth.GetCurrentOrganizationId(),
			*auth.GetCurrentProjectId(),
		).
		First(&provider)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Get", time.Since(start))
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantTelemetryProviderService.Get", time.Since(start))
	return provider, nil
}

func (eService *assistantTelemetryProviderService) GetAll(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
) (int64, []*internal_telemetry_entity.AssistantTelemetryProvider, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		providers []*internal_telemetry_entity.AssistantTelemetryProvider
		cnt       int64
	)
	qry := db.Model(internal_telemetry_entity.AssistantTelemetryProvider{}).
		Where("assistant_id = ? AND organization_id = ? AND project_id = ?",
			assistantId,
			*auth.GetCurrentOrganizationId(),
			*auth.GetCurrentProjectId(),
		)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			int(paginate.GetPage()),
			int(paginate.GetPageSize()),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&providers)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.GetAll", time.Since(start))
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("AssistantTelemetryProviderService.GetAll", time.Since(start))
	return cnt, providers, nil
}

func (eService *assistantTelemetryProviderService) Create(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	providerType string,
	enabled bool,
	options []*protos.Metadata,
) (*internal_telemetry_entity.AssistantTelemetryProvider, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	provider := &internal_telemetry_entity.AssistantTelemetryProvider{
		AssistantId:  assistantId,
		ProviderType: providerType,
		Enabled:      enabled,
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
	}
	tx := db.Create(provider)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Create", time.Since(start))
		return nil, tx.Error
	}
	if _, err := eService.createOptions(ctx, auth, provider.Id, options); err != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Create", time.Since(start))
		return nil, err
	}
	if tx = db.Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
		Where("id = ?", provider.Id).
		First(&provider); tx.Error != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Create", time.Since(start))
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantTelemetryProviderService.Create", time.Since(start))
	return provider, nil
}

func (eService *assistantTelemetryProviderService) Update(
	ctx context.Context,
	auth types.SimplePrinciple,
	telemetryProviderId uint64,
	assistantId uint64,
	providerType string,
	enabled bool,
	options []*protos.Metadata,
) (*internal_telemetry_entity.AssistantTelemetryProvider, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	provider := &internal_telemetry_entity.AssistantTelemetryProvider{
		ProviderType: providerType,
		Enabled:      enabled,
	}
	tx := db.Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ?",
		telemetryProviderId,
		assistantId,
		*auth.GetCurrentOrganizationId(),
		*auth.GetCurrentProjectId(),
	).Updates(provider)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Update", time.Since(start))
		return nil, tx.Error
	}
	if err := eService.archiveAllOptions(ctx, auth, telemetryProviderId); err != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Update", time.Since(start))
		return nil, err
	}
	if _, err := eService.createOptions(ctx, auth, telemetryProviderId, options); err != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Update", time.Since(start))
		return nil, err
	}
	var out *internal_telemetry_entity.AssistantTelemetryProvider
	if tx = db.Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
		Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ?",
			telemetryProviderId,
			assistantId,
			*auth.GetCurrentOrganizationId(),
			*auth.GetCurrentProjectId(),
		).
		First(&out); tx.Error != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Update", time.Since(start))
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantTelemetryProviderService.Update", time.Since(start))
	return out, nil
}

func (eService *assistantTelemetryProviderService) Delete(
	ctx context.Context,
	auth types.SimplePrinciple,
	telemetryProviderId uint64,
	assistantId uint64,
) (*internal_telemetry_entity.AssistantTelemetryProvider, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	provider := &internal_telemetry_entity.AssistantTelemetryProvider{}
	tx := db.Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
		Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ?",
			telemetryProviderId,
			assistantId,
			*auth.GetCurrentOrganizationId(),
			*auth.GetCurrentProjectId(),
		).First(provider)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Delete", time.Since(start))
		return nil, tx.Error
	}
	tx = db.Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ?",
		telemetryProviderId,
		assistantId,
		*auth.GetCurrentOrganizationId(),
		*auth.GetCurrentProjectId(),
	).Delete(&internal_telemetry_entity.AssistantTelemetryProvider{})
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantTelemetryProviderService.Delete", time.Since(start))
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantTelemetryProviderService.Delete", time.Since(start))
	return provider, nil
}

func (eService *assistantTelemetryProviderService) archiveAllOptions(
	ctx context.Context,
	auth types.SimplePrinciple,
	telemetryProviderId uint64,
) error {
	db := eService.postgres.DB(ctx)
	opt := &internal_telemetry_entity.AssistantTelemetryProviderOption{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("assistant_telemetry_provider_id = ? AND status = ?",
		telemetryProviderId, type_enums.RECORD_ACTIVE,
	).Updates(opt)
	return tx.Error
}

func (eService *assistantTelemetryProviderService) createOptions(
	ctx context.Context,
	auth types.SimplePrinciple,
	telemetryProviderId uint64,
	options []*protos.Metadata,
) ([]*internal_telemetry_entity.AssistantTelemetryProviderOption, error) {
	if len(options) == 0 {
		return []*internal_telemetry_entity.AssistantTelemetryProviderOption{}, nil
	}
	db := eService.postgres.DB(ctx)
	out := make([]*internal_telemetry_entity.AssistantTelemetryProviderOption, 0, len(options))
	for _, opt := range options {
		out = append(out, &internal_telemetry_entity.AssistantTelemetryProviderOption{
			AssistantTelemetryProviderId: telemetryProviderId,
			Metadata: gorm_models.Metadata{
				Key:   opt.GetKey(),
				Value: opt.GetValue(),
			},
			Mutable: gorm_models.Mutable{
				Status:    type_enums.RECORD_ACTIVE,
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
			},
		})
	}
	tx := db.Create(&out)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return out, nil
}
