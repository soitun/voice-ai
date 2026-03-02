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
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/storages"
)

type azureBlobStorage struct {
	config      configs.AssetStoreConfig
	logger      commons.Logger
	client      *azblob.Client
	accountName   string                       // resolved from config or connection string
	sharedKeyCred *azblob.SharedKeyCredential  // nil when using a connection string
}

// NewAzureBlobStorage creates an Azure Blob Storage backend.
// Auth priority: ConnectionString > AccountName+AccountKey.
func NewAzureBlobStorage(cfg configs.AssetStoreConfig, logger commons.Logger) storages.Storage {
	azCfg := cfg.AzureAuth

	if azCfg.ConnectionString != "" {
		client, err := azblob.NewClientFromConnectionString(azCfg.ConnectionString, nil)
		if err != nil {
			logger.Errorf("azure: unable to create client from connection string: %v", err)
		}
		return &azureBlobStorage{
			config:      cfg,
			logger:      logger,
			client:      client,
			accountName: accountNameFromConnString(azCfg.ConnectionString),
		}
	}

	cred, err := azblob.NewSharedKeyCredential(azCfg.AccountName, azCfg.AccountKey)
	if err != nil {
		logger.Errorf("azure: unable to create shared key credential: %v", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", azCfg.AccountName)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		logger.Errorf("azure: unable to create blob client: %v", err)
	}
	return &azureBlobStorage{
		config:        cfg,
		logger:        logger,
		client:        client,
		accountName:   azCfg.AccountName,
		sharedKeyCred: cred,
	}
}

// accountNameFromConnString extracts AccountName from an Azure connection string.
// Format: "...;AccountName=<name>;..."
func accountNameFromConnString(connStr string) string {
	for _, part := range strings.Split(connStr, ";") {
		if strings.HasPrefix(part, "AccountName=") {
			return strings.TrimPrefix(part, "AccountName=")
		}
	}
	return ""
}

func (az *azureBlobStorage) Name() string {
	return "azure"
}

// Store implements storages.Storage.
func (az *azureBlobStorage) Store(ctx context.Context, key string, fileContent []byte) storages.StorageOutput {
	az.logger.Debugf("azure.store key=%s container=%s", key, az.config.StoragePathPrefix)
	completePath := az.blobURL(key)
	_, err := az.client.UploadStream(ctx, az.config.StoragePathPrefix, key,
		bytes.NewReader(fileContent), nil)
	if err != nil {
		az.logger.Errorf("azure.store error: %v", err)
		return storages.StorageOutput{CompletePath: completePath, StorageType: configs.AZURE, Error: err}
	}
	return storages.StorageOutput{CompletePath: completePath, StorageType: configs.AZURE}
}

// Get implements storages.Storage.
func (az *azureBlobStorage) Get(ctx context.Context, key string) storages.GetStorageOutput {
	az.logger.Debugf("azure.get key=%s container=%s", key, az.config.StoragePathPrefix)
	resp, err := az.client.DownloadStream(ctx, az.config.StoragePathPrefix, key, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			az.logger.Errorf("azure.get blob not found: %s", key)
		} else {
			az.logger.Errorf("azure.get error: %v", err)
		}
		return storages.GetStorageOutput{Error: err}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		az.logger.Errorf("azure.get read error: %v", err)
		return storages.GetStorageOutput{Error: err}
	}
	return storages.GetStorageOutput{Data: data}
}

// GetUrl implements storages.Storage.
// Returns a 15-minute blob SAS URL when shared key credentials are available,
// otherwise returns the plain blob URL.
func (az *azureBlobStorage) GetUrl(ctx context.Context, key string) storages.StorageOutput {
	az.logger.Debugf("azure.getUrl key=%s container=%s", key, az.config.StoragePathPrefix)

	if az.sharedKeyCred == nil {
		// Connection-string auth: no shared key available for SAS signing.
		return storages.StorageOutput{CompletePath: az.blobURL(key), StorageType: configs.AZURE}
	}

	now := time.Now().UTC()
	sasParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     now.Add(-10 * time.Second),
		ExpiryTime:    now.Add(15 * time.Minute),
		Permissions:   (&sas.BlobPermissions{Read: true}).String(),
		ContainerName: az.config.StoragePathPrefix,
		BlobName:      key,
	}.SignWithSharedKey(az.sharedKeyCred)
	if err != nil {
		az.logger.Errorf("azure.getUrl SAS signing error: %v", err)
		return storages.StorageOutput{Error: err, StorageType: configs.AZURE}
	}

	return storages.StorageOutput{
		CompletePath: az.blobURL(key) + "?" + sasParams.Encode(),
		StorageType:  configs.AZURE,
	}
}

func (az *azureBlobStorage) blobURL(key string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		az.accountName, az.config.StoragePathPrefix, key)
}
