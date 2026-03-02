// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package storage_files

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	rapida_storages "github.com/rapidaai/pkg/storages"
)

type gcsStorage struct {
	config configs.AssetStoreConfig
	logger commons.Logger
	client *storage.Client
}

func NewGCSStorage(cfg configs.AssetStoreConfig, logger commons.Logger) rapida_storages.Storage {
	var (
		client *storage.Client
		err    error
	)

	gcsCfg := cfg.GCSAuth
	if gcsCfg != nil && gcsCfg.CredentialsJSON != "" {
		client, err = storage.NewClient(context.Background(),
			option.WithCredentialsJSON([]byte(gcsCfg.CredentialsJSON)))
	} else {
		// Fall back to Application Default Credentials
		client, err = storage.NewClient(context.Background())
	}

	if err != nil {
		logger.Errorf("unable to create gcs client: %v", err)
	}

	return &gcsStorage{
		config: cfg,
		logger: logger,
		client: client,
	}
}

func (g *gcsStorage) Name() string {
	return "gcs"
}

// Store implements rapida_storages.Storage.
func (g *gcsStorage) Store(ctx context.Context, key string, fileContent []byte) rapida_storages.StorageOutput {
	g.logger.Debugf("gcs.store object key=%s bucket=%s", key, g.config.StoragePathPrefix)
	completePath := fmt.Sprintf("gs://%s/%s", g.config.StoragePathPrefix, key)

	wc := g.client.Bucket(g.config.StoragePathPrefix).Object(key).NewWriter(ctx)
	if _, err := wc.Write(fileContent); err != nil {
		g.logger.Errorf("gcs.store write error: %v", err)
		return rapida_storages.StorageOutput{CompletePath: completePath, StorageType: configs.GCS, Error: err}
	}
	if err := wc.Close(); err != nil {
		g.logger.Errorf("gcs.store close error: %v", err)
		return rapida_storages.StorageOutput{CompletePath: completePath, StorageType: configs.GCS, Error: err}
	}
	return rapida_storages.StorageOutput{CompletePath: completePath, StorageType: configs.GCS}
}

// Get implements rapida_storages.Storage.
func (g *gcsStorage) Get(ctx context.Context, key string) rapida_storages.GetStorageOutput {
	g.logger.Debugf("gcs.get object key=%s bucket=%s", key, g.config.StoragePathPrefix)
	rc, err := g.client.Bucket(g.config.StoragePathPrefix).Object(key).NewReader(ctx)
	if err != nil {
		g.logger.Errorf("gcs.get error: %v", err)
		return rapida_storages.GetStorageOutput{Error: err}
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		g.logger.Errorf("gcs.get read error: %v", err)
		return rapida_storages.GetStorageOutput{Error: err}
	}
	return rapida_storages.GetStorageOutput{Data: data}
}

// GetUrl implements rapida_storages.Storage — returns a 15-minute signed URL.
func (g *gcsStorage) GetUrl(ctx context.Context, key string) rapida_storages.StorageOutput {
	g.logger.Debugf("gcs.getUrl object key=%s bucket=%s", key, g.config.StoragePathPrefix)

	// V4 signed URL — requires HMAC credentials or service account key.
	// Falls back to a plain GCS URL if signing is unavailable.
	url, err := g.client.Bucket(g.config.StoragePathPrefix).SignedURL(key, &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(15 * time.Minute),
		Scheme:  storage.SigningSchemeV4,
	})
	if err != nil {
		// Signing not available (e.g. ADC without a service account key) —
		// return the authenticated GCS URI so callers still have a path.
		g.logger.Warnf("gcs.getUrl signing unavailable, returning gs:// URI: %v", err)
		return rapida_storages.StorageOutput{
			CompletePath: fmt.Sprintf("gs://%s/%s", g.config.StoragePathPrefix, key),
			StorageType:  configs.GCS,
		}
	}
	return rapida_storages.StorageOutput{CompletePath: url, StorageType: configs.GCS}
}
