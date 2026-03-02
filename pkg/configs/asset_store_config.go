// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type StorageType string

const (
	S3    StorageType = "s3"
	LOCAL StorageType = "local"
	CDN   StorageType = "cdn"
	AZURE StorageType = "azure"
	GCS   StorageType = "gcs"
)

// asset_upload_bucket

type AssetStoreConfig struct {
	StorageType       string        `mapstructure:"storage_type" validate:"required"`
	StoragePathPrefix string        `mapstructure:"storage_path_prefix"`
	PublicUrlPrefix   *string       `mapstructure:"public_url_prefix"`
	Auth              *AwsConfig    `mapstructure:"auth"`
	AzureAuth         *AzureConfig  `mapstructure:"azure_auth"`
	GCSAuth           *GCSConfig    `mapstructure:"gcs_auth"`
}

func (cfg *AssetStoreConfig) Type() StorageType {
	switch cfg.StorageType {
	case string(S3):
		return S3
	case string(CDN):
		return CDN
	case string(AZURE):
		return AZURE
	case string(GCS):
		return GCS
	default:
		return LOCAL
	}
}

func (cfg *AssetStoreConfig) IsLocal() bool {
	return cfg.Type() != S3
}

func (cfg *AssetStoreConfig) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"storage_type":        cfg.StorageType,
		"storage_path_prefix": cfg.StoragePathPrefix,
	}

	if cfg.Auth != nil {
		result["auth"] = cfg.Auth.ToMap()
	} else {
		result["auth"] = nil
	}

	return result
}
