// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

// GCSConfig holds credentials for Google Cloud Storage.
// If CredentialsJSON is empty, Application Default Credentials (ADC) are used.
type GCSConfig struct {
	ProjectID       string `mapstructure:"project_id"`
	CredentialsJSON string `mapstructure:"credentials_json"`
}

func (cfg *GCSConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"project_id":       cfg.ProjectID,
		"credentials_json": cfg.CredentialsJSON,
	}
}
