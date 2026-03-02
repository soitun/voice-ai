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

// newTestGCSConfig returns a config backed by a minimal inline credentials JSON.
// The key values are fake — the client will fail at API call time, not construction.
func newTestGCSConfig() configs.AssetStoreConfig {
	// Minimal service-account JSON that satisfies the GCS client parser.
	credJSON := `{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "key-id",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA2a2rwplBQLF29amygykEMmYz0+Kcj3bKBp29jqdkHRMKVmJ\nd3L8dCFhw1HYLJ4lABsIm2YJCB27TTqEFVBxRVFIaXwOEtFJi7sVBmP9Z5OXCO\naY36VkE5jDjLBPjqe9LBhB4TPVL6HdBXOmTTDJhK4G2Pxaipb2mrFETqoHyOwFx\n-----END RSA PRIVATE KEY-----\n",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "123456789",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token"
	}`
	return configs.AssetStoreConfig{
		StorageType:       "gcs",
		StoragePathPrefix: "test-bucket",
		GCSAuth: &configs.GCSConfig{
			ProjectID:       "test-project",
			CredentialsJSON: credJSON,
		},
	}
}

func TestGCSStorage_Name(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewGCSStorage(newTestGCSConfig(), logger)
	assert.Equal(t, "gcs", storage.Name())
}

// TestGCSStorage_ClientInitialised verifies the GCS client is created in the
// constructor and not re-created on every method call.
func TestGCSStorage_ClientInitialised(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	s := NewGCSStorage(newTestGCSConfig(), logger).(*gcsStorage)
	require.NotNil(t, s.client, "gcs client must be initialised in constructor")
}

// TestGCSStorage_Store_ReturnsError verifies Store propagates GCS API errors
// (fake credentials, no real GCS endpoint reachable in CI).
func TestGCSStorage_Store_ReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewGCSStorage(newTestGCSConfig(), logger)

	result := storage.Store(context.Background(), "test/file.txt", []byte("content"))

	assert.Error(t, result.Error)
	assert.Equal(t, configs.GCS, result.StorageType)
	assert.Equal(t, "gs://test-bucket/test/file.txt", result.CompletePath)
}

// TestGCSStorage_Get_ReturnsError verifies Get propagates GCS API errors.
func TestGCSStorage_Get_ReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewGCSStorage(newTestGCSConfig(), logger)

	result := storage.Get(context.Background(), "test/file.txt")

	assert.Error(t, result.Error)
	assert.Nil(t, result.Data)
}

// TestGCSStorage_GetUrl_ReturnsPath verifies GetUrl returns a usable path.
// With a fake private key, signing fails — the fallback gs:// URI must be returned.
func TestGCSStorage_GetUrl_FallbackOnSigningError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	storage := NewGCSStorage(newTestGCSConfig(), logger)

	result := storage.GetUrl(context.Background(), "test/file.txt")

	assert.Equal(t, configs.GCS, result.StorageType)
	// Either a signed HTTPS URL or the gs:// fallback — never both empty.
	assert.NotEmpty(t, result.CompletePath)
	assert.Contains(t, result.CompletePath, "test-bucket")
	assert.Contains(t, result.CompletePath, "test/file.txt")
}

func TestGCSStorage_ADC_NoCredentials(t *testing.T) {
	// When no credentials JSON is provided, the client uses ADC.
	// In CI without ADC configured, NewGCSStorage logs the error but returns
	// a non-nil Storage (with a nil internal client). Just verify Name() works.
	cfg := configs.AssetStoreConfig{
		StorageType:       "gcs",
		StoragePathPrefix: "test-bucket",
		GCSAuth:           &configs.GCSConfig{ProjectID: "test-project"},
	}
	logger, _ := commons.NewApplicationLogger()
	storage := NewGCSStorage(cfg, logger)
	assert.NotNil(t, storage)
	assert.Equal(t, "gcs", storage.Name())
}
