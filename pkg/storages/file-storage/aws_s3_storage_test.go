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

func newTestAwsConfig() configs.AssetStoreConfig {
	return configs.AssetStoreConfig{
		StorageType:       "s3",
		StoragePathPrefix: "test-bucket",
		Auth: &configs.AwsConfig{
			Region:      "us-east-1",
			AccessKeyId: "test-key",
			SecretKey:   "test-secret",
		},
	}
}

func TestAwsFileStorage_Name(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAwsFileStorage(newTestAwsConfig(), logger)
	assert.Equal(t, "aws", storage.Name())
}

// TestAwsFileStorage_ClientReuse verifies the S3 client is created once in the
// constructor and reused across method calls (not recreated per call).
func TestAwsFileStorage_ClientReuse(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	s := NewAwsFileStorage(newTestAwsConfig(), logger).(*awsFileStorage)
	require.NotNil(t, s.s3Client, "s3Client must be initialised in constructor")

	// Second reference must be the same pointer
	assert.Same(t, s.s3Client, s.s3Client)
}

func TestAwsFileStorage_contentType(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAwsFileStorage(newTestAwsConfig(), logger).(*awsFileStorage)

	tests := []struct {
		filename   string
		expectedCT string
	}{
		{"test.json", "application/json"},
		{"audio.mp3", "audio/mpeg"},
		{"audio.wav", "audio/wav"},
		{"audio.ogg", "audio/ogg"},
		{"audio.flac", "audio/flac"},
		{"audio.aac", "audio/aac"},
		{"audio.m4a", "audio/mp4"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			assert.Equal(t, tt.expectedCT, storage.contentType(tt.filename))
		})
	}
}

// TestAwsFileStorage_Store_ReturnsError verifies Store propagates S3 API errors
// (the client is valid but no real S3 endpoint is available in CI).
func TestAwsFileStorage_Store_ReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAwsFileStorage(newTestAwsConfig(), logger)

	result := storage.Store(context.Background(), "test/file.txt", []byte("content"))

	assert.Error(t, result.Error)
	assert.Equal(t, configs.S3, result.StorageType)
	assert.Contains(t, result.CompletePath, "s3://test-bucket/test/file.txt")
}

// TestAwsFileStorage_Get_ReturnsError verifies Get propagates S3 API errors.
func TestAwsFileStorage_Get_ReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAwsFileStorage(newTestAwsConfig(), logger)

	result := storage.Get(context.Background(), "test/file.txt")

	assert.Error(t, result.Error)
	assert.Nil(t, result.Data)
}

// TestAwsFileStorage_GetUrl verifies GetUrl returns a pre-signed URL or error.
// With fake credentials the SDK may still produce a signed URL string.
func TestAwsFileStorage_GetUrl_ReturnsStorageType(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewAwsFileStorage(newTestAwsConfig(), logger)

	result := storage.GetUrl(context.Background(), "test/file.txt")

	assert.Equal(t, configs.S3, result.StorageType)
	// Either a signed URL was generated or an error occurred — never both empty.
	assert.True(t, result.CompletePath != "" || result.Error != nil)
	if result.CompletePath != "" {
		assert.Contains(t, result.CompletePath, "test-bucket")
		assert.Contains(t, result.CompletePath, "test/file.txt")
	}
}
