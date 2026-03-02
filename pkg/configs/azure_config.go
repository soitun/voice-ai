// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

// AzureConfig holds credentials for Azure Blob Storage.
// Either AccountKey or ConnectionString may be provided for authentication.
type AzureConfig struct {
	AccountName      string `mapstructure:"account_name"`
	AccountKey       string `mapstructure:"account_key"`
	ConnectionString string `mapstructure:"connection_string"`
}

func (cfg *AzureConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"account_name":      cfg.AccountName,
		"account_key":       cfg.AccountKey,
		"connection_string": cfg.ConnectionString,
	}
}
