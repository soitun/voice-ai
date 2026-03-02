// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package storage_files

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	aws_session "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/storages"
)

type cdnStorage struct {
	config   configs.AssetStoreConfig
	logger   commons.Logger
	s3Client *s3.S3
}

func NewCDNStorage(cfg configs.AssetStoreConfig, logger commons.Logger) storages.Storage {
	awsConfig := aws.Config{
		Region: aws.String(cfg.Auth.Region),
	}
	if cfg.Auth.AccessKeyId != "" && cfg.Auth.SecretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			cfg.Auth.AccessKeyId,
			cfg.Auth.SecretKey,
			"",
		)
	}
	sess, err := aws_session.NewSessionWithOptions(aws_session.Options{
		Config:            awsConfig,
		SharedConfigState: aws_session.SharedConfigEnable,
	})
	if err != nil {
		logger.Errorf("unable to create cdn s3 session: %v", err)
	}
	return &cdnStorage{
		config:   cfg,
		logger:   logger,
		s3Client: s3.New(sess),
	}
}

func (lfs *cdnStorage) Name() string {
	return "cdn"
}
func (storage *cdnStorage) prefix(ctx context.Context, key string) string {
	return fmt.Sprintf("cdn/%d_%s", gorm_generator.ID(), key)
}

// Store implements storages.Storage.
func (storage *cdnStorage) Store(ctx context.Context, key string, fileContent []byte) storages.StorageOutput {
	storage.logger.Debugf("s3.store with file path name %s storage path prefix %s", key, storage.config.StoragePathPrefix)
	key = storage.prefix(ctx, key)
	completePath := fmt.Sprintf("%s/%s", storage.config.StoragePathPrefix, key)
	reader := bytes.NewReader(fileContent)
	_, err := storage.s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(storage.config.StoragePathPrefix),
		Key:    aws.String(key),
		Body:   reader,
	})

	if err != nil {
		storage.logger.Errorf("Error uploading data to S3: %v", err)
		return storages.StorageOutput{
			CompletePath: completePath,
			Error:        err,
			StorageType:  configs.S3}
	}
	return storages.StorageOutput{
		CompletePath: completePath,
		StorageType:  configs.S3,
	}
}

func (storage *cdnStorage) Get(ctx context.Context, key string) storages.GetStorageOutput {
	resp, err := storage.s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.config.StoragePathPrefix),
		Key:    aws.String(key),
	})
	if err != nil {
		storage.logger.Errorf("Error downloading object: %v", err)
		return storages.GetStorageOutput{Error: err}
	}
	defer resp.Body.Close()
	jsonData, err := io.ReadAll(resp.Body)
	if err != nil {
		storage.logger.Errorf("Error reading JSON data: %v", err)
		return storages.GetStorageOutput{Error: err}
	}
	return storages.GetStorageOutput{Data: jsonData}
}

func (cdn *cdnStorage) GetUrl(ctx context.Context, key string) storages.StorageOutput {
	cdn.logger.Debugf("localstorage.getUrl with file path name %s", key)
	return storages.StorageOutput{
		CompletePath: fmt.Sprintf("%s/%s", cdn.config.StoragePathPrefix, key),
		StorageType:  configs.S3}
}
