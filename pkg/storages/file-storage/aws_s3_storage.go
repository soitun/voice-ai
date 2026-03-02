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
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	aws_session "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/utils"
)

type awsFileStorage struct {
	config   configs.AssetStoreConfig
	logger   commons.Logger
	s3Client *s3.S3
}

func NewAwsFileStorage(cfg configs.AssetStoreConfig, logger commons.Logger) storages.Storage {
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
		logger.Errorf("unable to create aws s3 session: %v", err)
	}
	return &awsFileStorage{
		config:   cfg,
		logger:   logger,
		s3Client: s3.New(sess),
	}
}

func (storage *awsFileStorage) contentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return "application/json"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".m4a":
		return "audio/mp4"
	default:
		return "application/octet-stream"
	}
}

// Store implements storages.Storage.
func (storage *awsFileStorage) Store(ctx context.Context, key string, fileContent []byte) storages.StorageOutput {
	storage.logger.Debugf("s3.store with file path name %s storage path prefix %s", key, storage.config.StoragePathPrefix)
	completePath := fmt.Sprintf("s3://%s/%s", storage.config.StoragePathPrefix, key)
	reader := bytes.NewReader(fileContent)
	_, err := storage.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(storage.config.StoragePathPrefix),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(storage.contentType(key)),
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

func (lfs *awsFileStorage) Name() string {
	return "aws"
}

func (storage *awsFileStorage) Get(ctx context.Context, key string) storages.GetStorageOutput {
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

func (aws *awsFileStorage) GetUrl(ctx context.Context, key string) storages.StorageOutput {
	aws.logger.Debugf("awsFileStorage.getUrl with file path name %s", key)
	req, _ := aws.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: utils.Ptr(aws.config.StoragePathPrefix),
		Key:    utils.Ptr(key),
	})

	urlStr, err := req.Presign(15 * time.Minute) // URL valid for 15 minutes
	if err != nil {
		aws.logger.Errorf("Error getting pre-signed URL: %v", err)
		return storages.StorageOutput{Error: err, StorageType: configs.S3}
	}

	return storages.StorageOutput{
		CompletePath: urlStr,
		StorageType:  configs.S3,
	}
}
