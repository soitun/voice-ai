// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package storage_files

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
)

func newTestCDNConfig() configs.AssetStoreConfig {
	return configs.AssetStoreConfig{
		StorageType:       "cdn",
		StoragePathPrefix: "https://cdn.example.com",
		Auth: &configs.AwsConfig{
			Region:      "us-east-1",
			AccessKeyId: "test-key",
			SecretKey:   "test-secret",
		},
	}
}

func TestCDNStorage_Name(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewCDNStorage(newTestCDNConfig(), logger)
	assert.Equal(t, "cdn", storage.Name())
}

// TestCDNStorage_ClientReuse verifies the S3 client is created once in the
// constructor and reused across calls.
func TestCDNStorage_ClientReuse(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	s := NewCDNStorage(newTestCDNConfig(), logger).(*cdnStorage)
	require.NotNil(t, s.s3Client, "s3Client must be initialised in constructor")
	assert.Same(t, s.s3Client, s.s3Client)
}

// TestCDNStorage_Store_ReturnsError verifies Store propagates S3 API errors.
func TestCDNStorage_Store_ReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewCDNStorage(newTestCDNConfig(), logger)

	result := storage.Store(context.Background(), "test/file.txt", []byte("content"))

	assert.Error(t, result.Error)
	assert.Equal(t, configs.S3, result.StorageType)
	assert.Contains(t, result.CompletePath, "https://cdn.example.com/cdn/")
}

// TestCDNStorage_Get_ReturnsError verifies Get propagates S3 API errors.
func TestCDNStorage_Get_ReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewCDNStorage(newTestCDNConfig(), logger)

	result := storage.Get(context.Background(), "test/file.txt")

	assert.Error(t, result.Error)
	assert.Nil(t, result.Data)
}

func TestCDNStorage_GetUrl(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewCDNStorage(newTestCDNConfig(), logger)

	result := storage.GetUrl(context.Background(), "test/file.txt")

	assert.NoError(t, result.Error)
	assert.Equal(t, configs.S3, result.StorageType)
	assert.Equal(t, "https://cdn.example.com/test/file.txt", result.CompletePath)
}

func TestCDNStorage_prefix(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewCDNStorage(newTestCDNConfig(), logger).(*cdnStorage)

	prefixed := storage.prefix(context.Background(), "file.txt")

	assert.True(t, strings.HasPrefix(prefixed, "cdn/"), "prefix must start with cdn/")
	assert.True(t, strings.HasSuffix(prefixed, "_file.txt"), "prefix must end with _<key>")
	// Format: cdn/{snowflake-id}_file.txt — exactly one underscore separating id and key
	withoutCdn := strings.TrimPrefix(prefixed, "cdn/")
	parts := strings.SplitN(withoutCdn, "_", 2)
	assert.Len(t, parts, 2)
	assert.NotEmpty(t, parts[0], "snowflake id must not be empty")
	assert.Equal(t, "file.txt", parts[1])
}
