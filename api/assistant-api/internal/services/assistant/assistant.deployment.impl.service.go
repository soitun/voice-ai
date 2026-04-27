// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"errors"

	"github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type assistantDeploymentService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	cfg      *config.AssistantConfig
}

func NewAssistantDeploymentService(cfg *config.AssistantConfig,
	logger commons.Logger,
	postgres connectors.PostgresConnector) internal_services.AssistantDeploymentService {
	return &assistantDeploymentService{
		logger:   logger,
		postgres: postgres,
		cfg:      cfg,
	}
}

func (eService assistantDeploymentService) CreateWebPluginDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	suggestion []string,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantWebPluginDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        IdleTimeout,
			IdleTimeoutBackoff: IdleTimeoutBackoff,
			IdleTimeoutMessage: IdleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
		Suggestion: suggestion,
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	//
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) createAssistantDeploymentAudio(
	ctx context.Context,
	auth types.SimplePrinciple, deploymentId uint64,
	audioType string,
	audioConfig *protos.DeploymentAudioProvider) (*internal_assistant_entity.AssistantDeploymentAudio, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantDeploymentAudio{
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			Status:    type_enums.RecordState(audioConfig.GetStatus()),
		},
		AudioType:             audioType,
		AssistantDeploymentId: deploymentId,
		AudioProvider:         audioConfig.GetAudioProvider(),
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create deployment audio config for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if len(audioConfig.GetAudioOptions()) == 0 {
		return deployment, nil
	}
	audioDeploymentOptions := make([]*internal_assistant_entity.AssistantDeploymentAudioOption, 0)
	for _, v := range audioConfig.GetAudioOptions() {
		audioDeploymentOptions = append(audioDeploymentOptions, &internal_assistant_entity.AssistantDeploymentAudioOption{
			AssistantDeploymentAudioId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RecordState(audioConfig.GetStatus()),
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_audio_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(audioDeploymentOptions)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create deployment audio config metadata for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	return deployment, nil
}

func (eService assistantDeploymentService) CreateDebuggerDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantDebuggerDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        IdleTimeout,
			IdleTimeoutBackoff: IdleTimeoutBackoff,
			IdleTimeoutMessage: IdleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) CreateApiDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantApiDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        IdleTimeout,
			IdleTimeoutBackoff: IdleTimeoutBackoff,
			IdleTimeoutMessage: IdleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) CreateWhatsappDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	idleTimeout *uint64,
	idleTimeoutBackoff *uint64,
	idleTimeoutMessage *string, maxSessionDuration *uint64,
	whatsappProvider string,
	whatsappOptions []*protos.Metadata,
) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantWhatsappDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        idleTimeout,
			IdleTimeoutBackoff: idleTimeoutBackoff,
			IdleTimeoutMessage: idleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
		AssistantDeploymentWhatsapp: internal_assistant_entity.AssistantDeploymentWhatsapp{
			WhatsappProvider: whatsappProvider,
		},
	}

	// TODO: Persist the deployment to the database
	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if len(whatsappOptions) == 0 {
		return deployment, nil
	}

	whatsappOpts := make([]*internal_assistant_entity.AssistantDeploymentWhatsappOption, 0)
	for _, v := range whatsappOptions {
		whatsappOpts = append(whatsappOpts, &internal_assistant_entity.AssistantDeploymentWhatsappOption{
			AssistantDeploymentWhatsappId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RECORD_ACTIVE,
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_whatsapp_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(whatsappOpts)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create whatsapp options for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	return deployment, nil
}

func (eService assistantDeploymentService) CreatePhoneDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	phoneProvider string,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
	opts []*protos.Metadata,
) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantPhoneDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        IdleTimeout,
			IdleTimeoutBackoff: IdleTimeoutBackoff,
			IdleTimeoutMessage: IdleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
		AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
			TelephonyProvider: phoneProvider,
		},
	}

	// Today we support one active phone deployment per assistant. So we archive prior deployments before creating a new one. In future if we want to support multiple active phone deployments, we can remove this logic and add necessary filters while fetching the deployment for a call.
	if err := db.Model(&internal_assistant_entity.AssistantPhoneDeployment{}).
		Where("assistant_id = ? AND status = ?", assistantId, type_enums.RECORD_ACTIVE).
		Updates(map[string]interface{}{
			"status":     type_enums.RECORD_ARCHIEVE,
			"updated_by": *auth.GetUserId(),
		}).Error; err != nil {
		eService.logger.Errorf("unable to archive prior phone deployments for assistant %d: %v", assistantId, err)
		return nil, err
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	if len(opts) == 0 {
		eService.logger.Warnf("no options for the telephony provider.")
		return deployment, nil
	}

	phoneOpts := make([]*internal_assistant_entity.AssistantDeploymentTelephonyOption, 0)
	for _, v := range opts {
		phoneOpts = append(phoneOpts, &internal_assistant_entity.AssistantDeploymentTelephonyOption{
			AssistantDeploymentTelephonyId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RECORD_ACTIVE,
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}

	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_telephony_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(phoneOpts)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create telephony options for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	return deployment, nil
}

func (eService assistantDeploymentService) GetAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	var apiDeployment *internal_assistant_entity.AssistantApiDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&apiDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return apiDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	var debuggerDeployment *internal_assistant_entity.AssistantDebuggerDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&debuggerDeployment)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return debuggerDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	var phoneDeployment *internal_assistant_entity.AssistantPhoneDeployment
	qry := db.
		Preload("TelephonyOption").
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&phoneDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return phoneDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	var webPluginDeployment *internal_assistant_entity.AssistantWebPluginDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&webPluginDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find web plugin deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return webPluginDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	var whatsappDeployment *internal_assistant_entity.AssistantWhatsappDeployment
	qry := db.
		Preload("WhatsappOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&whatsappDeployment)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find whatsapp deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return whatsappDeployment, nil
}
