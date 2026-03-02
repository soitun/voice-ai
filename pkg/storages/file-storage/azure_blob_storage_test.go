// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package storage_files

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
)

// newTestAzureConfig returns a config with a syntactically valid (but fake)
// AccountKey so credential parsing succeeds in unit tests.
// The base64 value decodes to 32 zero bytes, which satisfies the SDK length check.
func newTestAzureConfig() configs.AssetStoreConfig {
	return configs.AssetStoreConfig{
		StorageType:       "azure",
		StoragePathPrefix: "test-container",
		AzureAuth: &configs.AzureConfig{
			AccountName: "testaccount",
			AccountKey:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", // 32-byte base64
		},
	}
}

func newTestAzureConnStringConfig() configs.AssetStoreConfig {
	return configs.AssetStoreConfig{
		StorageType:       "azure",
		StoragePathPrefix: "test-container",
		AzureAuth: &configs.AzureConfig{
			ConnectionString: "DefaultEndpointsProtocol=https;AccountName=testaccount;" +
				"AccountKey=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=;" +
				"EndpointSuffix=core.windows.net",
		},
	}
}

func TestAzureBlobStorage_Name(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAzureBlobStorage(newTestAzureConfig(), logger)
	assert.Equal(t, "azure", storage.Name())
}

// TestAzureBlobStorage_ClientReuse verifies the client is created once in the
// constructor and not re-created on every method call.
func TestAzureBlobStorage_ClientReuse(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	s := NewAzureBlobStorage(newTestAzureConfig(), logger).(*azureBlobStorage)
	require.NotNil(t, s.client, "azure client must be initialised in constructor")
	require.NotNil(t, s.sharedKeyCred, "sharedKeyCred must be set when using AccountKey auth")
}

func TestAzureBlobStorage_ConnectionString_NoSharedKeyCred(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	s := NewAzureBlobStorage(newTestAzureConnStringConfig(), logger).(*azureBlobStorage)
	require.NotNil(t, s.client)
	assert.Nil(t, s.sharedKeyCred, "sharedKeyCred must be nil when using connection string auth")
}

// TestAzureBlobStorage_Store_ReturnsError verifies Store propagates Azure API errors
// (fake credentials, no real Azure endpoint reachable).
func TestAzureBlobStorage_Store_ReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAzureBlobStorage(newTestAzureConfig(), logger)

	result := storage.Store(context.Background(), "test/file.txt", []byte("content"))

	assert.Error(t, result.Error)
	assert.Equal(t, configs.AZURE, result.StorageType)
	assert.Contains(t, result.CompletePath, "testaccount")
	assert.Contains(t, result.CompletePath, "test-container")
	assert.Contains(t, result.CompletePath, "test/file.txt")
}

// TestAzureBlobStorage_Get_ReturnsError verifies Get propagates Azure API errors.
func TestAzureBlobStorage_Get_ReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAzureBlobStorage(newTestAzureConfig(), logger)

	result := storage.Get(context.Background(), "test/file.txt")

	assert.Error(t, result.Error)
	assert.Nil(t, result.Data)
}

// TestAzureBlobStorage_GetUrl_WithSharedKey verifies SAS URL generation with a
// shared key credential — expects either a signed URL or an error.
func TestAzureBlobStorage_GetUrl_WithSharedKey(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAzureBlobStorage(newTestAzureConfig(), logger)

	result := storage.GetUrl(context.Background(), "test/file.txt")

	assert.Equal(t, configs.AZURE, result.StorageType)
	assert.True(t, result.CompletePath != "" || result.Error != nil)
	if result.CompletePath != "" {
		assert.Contains(t, result.CompletePath, "testaccount")
		assert.Contains(t, result.CompletePath, "test/file.txt")
	}
}

// TestAzureBlobStorage_GetUrl_ConnectionString verifies GetUrl returns a plain
// blob URL when no shared key is available (connection-string auth path).
func TestAzureBlobStorage_GetUrl_ConnectionString(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAzureBlobStorage(newTestAzureConnStringConfig(), logger)

	result := storage.GetUrl(context.Background(), "test/file.txt")

	assert.NoError(t, result.Error)
	assert.Equal(t, configs.AZURE, result.StorageType)
	assert.Contains(t, result.CompletePath, "testaccount")
	assert.Contains(t, result.CompletePath, "test-container")
	assert.Contains(t, result.CompletePath, "test/file.txt")
}

func TestAzureBlobStorage_blobURL(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	s := NewAzureBlobStorage(newTestAzureConfig(), logger).(*azureBlobStorage)

	url := s.blobURL("some/path/file.mp3")
	assert.Equal(t,
		"https://testaccount.blob.core.windows.net/test-container/some/path/file.mp3",
		url,
	)
}
