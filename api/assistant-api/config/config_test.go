// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestInitConfig(t *testing.T) {
	// Mock Environment setup
	envPath := filepath.Join(os.TempDir(), ".assistant_test.env")
	err := os.WriteFile(envPath, []byte(`
		SERVICE_NAME="workflow-api"
		HOST="0.0.0.0"
		PORT=9007
		LOG_LEVEL="debug"
		SECRET="rpd_pks"
		ENV="development"
		
		POSTGRES__HOST="localhost"
		POSTGRES__DB_NAME="assistant_db"
		POSTGRES__AUTH__USER="rapida_user"
		POSTGRES__AUTH__PASSWORD="rapida_db_password"
		POSTGRES__PORT=5432
		POSTGRES__MAX_OPEN_CONNECTION=50
		POSTGRES__MAX_IDEAL_CONNECTION=25
		POSTGRES__SSL_MODE="disable"
		POSTGRES__SLC_CACHE__HOST=127.0.0.1
		POSTGRES__SLC_CACHE__PORT=6379
		POSTGRES__SLC_CACHE__MAX_CONNECTION=10
		POSTGRES__SLC_CACHE__MAX_DB=1
		
		REDIS__HOST=127.0.0.1
		REDIS__PORT=6379
		REDIS__MAX_CONNECTION=10
		REDIS__MAX_DB=0
		
		ASSET_STORE__STORAGE_TYPE="local"
		ASSET_STORE__STORAGE_PATH_PREFIX=${HOME}/rapida-data/assets/workflow
		
		OPENSEARCH__SCHEMA="http"
		OPENSEARCH__HOST="localhost"
		OPENSEARCH__PORT=9200
		OPENSEARCH__MAX_RETRIES=3
		OPENSEARCH__MAX_CONNECTION=10
		
		INTEGRATION_HOST=localhost:9004
		ENDPOINT_HOST=localhost:9005
		ASSISTANT_HOST=localhost:9007
		WEB_HOST=localhost:9001
		DOCUMENT_HOST=http://localhost:9010
		UI_HOST=http://localhost:3000
		PUBLIC_ASSISTANT_HOST=integral-presently-cub.ngrok-free.app
	`), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock env file: %v", err)
	}
	defer os.Remove(envPath) // Clean up after test

	os.Setenv("ENV_PATH", envPath)
	defer os.Unsetenv("ENV_PATH") // Reset ENV_PATH after test

	// Test initializing configuration
	vConfig, err := InitConfig()
	if err != nil {
		t.Fatalf("InitConfig returned an error: %v", err)
	}
	if vConfig == nil {
		t.Fatalf("vConfig is nil")
	}

	// Verify that viper has correctly read variables
	if vConfig.ConfigFileUsed() != envPath {
		t.Errorf("Expected config file used to be %v, but got %v", envPath, vConfig.ConfigFileUsed())
	}
	if vConfig.GetString("POSTGRES__DB_NAME") != "assistant_db" {
		t.Errorf("Expected POSTGRES__DB_NAME to be 'assistant_db', but got %v", vConfig.GetString("POSTGRES__DB_NAME"))
	}
	if vConfig.GetString("PUBLIC_ASSISTANT_HOST") != "integral-presently-cub.ngrok-free.app" {
		t.Errorf("Expected PUBLIC_ASSISTANT_HOST to be 'integral-presently-cub.ngrok-free.app', but got %v", vConfig.GetString("PUBLIC_ASSISTANT_HOST"))
	}
}

func TestGetApplicationConfig(t *testing.T) {
	// Mock viper setup
	vConfig := viper.NewWithOptions(viper.KeyDelimiter("__"))
	vConfig.Set("SERVICE_NAME", "workflow-api")
	vConfig.Set("HOST", "0.0.0.0")
	vConfig.Set("PORT", 9007)
	vConfig.Set("LOG_LEVEL", "debug")
	vConfig.Set("SECRET", "rpd_pks")
	vConfig.Set("ENV", "development")

	vConfig.Set("POSTGRES__HOST", "localhost")
	vConfig.Set("POSTGRES__DB_NAME", "assistant_db")
	vConfig.Set("POSTGRES__AUTH__USER", "rapida_user")
	vConfig.Set("POSTGRES__AUTH__PASSWORD", "rapida_db_password")
	vConfig.Set("POSTGRES__PORT", 5432)
	vConfig.Set("POSTGRES__MAX_OPEN_CONNECTION", 50)
	vConfig.Set("POSTGRES__MAX_IDEAL_CONNECTION", 25)
	vConfig.Set("POSTGRES__SSL_MODE", "disable")
	vConfig.Set("POSTGRES__SLC_CACHE__HOST", "127.0.0.1")
	vConfig.Set("POSTGRES__SLC_CACHE__PORT", 6379)
	vConfig.Set("POSTGRES__SLC_CACHE__MAX_CONNECTION", 10)
	vConfig.Set("POSTGRES__SLC_CACHE__MAX_DB", 1)

	vConfig.Set("REDIS__HOST", "127.0.0.1")
	vConfig.Set("REDIS__PORT", 6379)
	vConfig.Set("REDIS__MAX_CONNECTION", 10)
	vConfig.Set("REDIS__MAX_DB", 0)

	vConfig.Set("ASSET_STORE__STORAGE_TYPE", "local")
	vConfig.Set("ASSET_STORE__STORAGE_PATH_PREFIX", os.Getenv("HOME")+"/rapida-data/assets/workflow")

	vConfig.Set("OPENSEARCH__SCHEMA", "http")
	vConfig.Set("OPENSEARCH__HOST", "localhost")
	vConfig.Set("OPENSEARCH__PORT", 9200)
	vConfig.Set("OPENSEARCH__MAX_RETRIES", 3)
	vConfig.Set("OPENSEARCH__MAX_CONNECTION", 10)

	vConfig.Set("INTEGRATION_HOST", "localhost:9004")
	vConfig.Set("ENDPOINT_HOST", "localhost:9005")
	vConfig.Set("ASSISTANT_HOST", "localhost:9007")
	vConfig.Set("WEB_HOST", "localhost:9001")
	vConfig.Set("DOCUMENT_HOST", "http://localhost:9010")
	vConfig.Set("UI_HOST", "http://localhost:3000")
	vConfig.Set("PUBLIC_ASSISTANT_HOST", "integral-presently-cub.ngrok-free.app")

	appConfig, err := GetApplicationConfig(vConfig)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig == nil {
		t.Fatalf("appConfig is nil")
	}

	// Validate parsed configuration matches expectations
	if appConfig.PublicAssistantHost != "integral-presently-cub.ngrok-free.app" {
		t.Errorf("Expected PublicAssistantHost to be 'integral-presently-cub.ngrok-free.app', but got %v", appConfig.PublicAssistantHost)
	}
	if appConfig.PostgresConfig.DBName != "assistant_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'assistant_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.AssetStoreConfig.StorageType != "local" {
		t.Errorf("Expected AssetStoreConfig.StorageType to be 'local', but got %v", appConfig.AssetStoreConfig.StorageType)
	}
}

func TestGetApplicationConfig_TelemetryEnvParsing(t *testing.T) {
	vConfig := viper.NewWithOptions(viper.KeyDelimiter("__"))
	vConfig.Set("SERVICE_NAME", "workflow-api")
	vConfig.Set("HOST", "0.0.0.0")
	vConfig.Set("PORT", 9007)
	vConfig.Set("LOG_LEVEL", "debug")
	vConfig.Set("SECRET", "rpd_pks")
	vConfig.Set("ENV", "development")

	vConfig.Set("POSTGRES__HOST", "localhost")
	vConfig.Set("POSTGRES__DB_NAME", "assistant_db")
	vConfig.Set("POSTGRES__AUTH__USER", "rapida_user")
	vConfig.Set("POSTGRES__AUTH__PASSWORD", "rapida_db_password")
	vConfig.Set("POSTGRES__PORT", 5432)
	vConfig.Set("POSTGRES__MAX_OPEN_CONNECTION", 50)
	vConfig.Set("POSTGRES__MAX_IDEAL_CONNECTION", 25)
	vConfig.Set("POSTGRES__SSL_MODE", "disable")

	vConfig.Set("REDIS__HOST", "127.0.0.1")
	vConfig.Set("REDIS__PORT", 6379)
	vConfig.Set("REDIS__MAX_CONNECTION", 10)
	vConfig.Set("REDIS__MAX_DB", 0)

	vConfig.Set("ASSET_STORE__STORAGE_TYPE", "local")
	vConfig.Set("ASSET_STORE__STORAGE_PATH_PREFIX", os.Getenv("HOME")+"/rapida-data/assets/workflow")

	vConfig.Set("INTEGRATION_HOST", "localhost:9004")
	vConfig.Set("ENDPOINT_HOST", "localhost:9005")
	vConfig.Set("ASSISTANT_HOST", "localhost:9007")
	vConfig.Set("WEB_HOST", "localhost:9001")
	vConfig.Set("DOCUMENT_HOST", "http://localhost:9010")
	vConfig.Set("UI_HOST", "http://localhost:3000")
	vConfig.Set("PUBLIC_ASSISTANT_HOST", "integral-presently-cub.ngrok-free.app")

	vConfig.Set("TELEMETRY__TYPE", "otlp_http")
	vConfig.Set("TELEMETRY__OTLP_HTTP__ENDPOINT", "otel-collector:4318")
	vConfig.Set("TELEMETRY__OTLP_HTTP__PROTOCOL", "http/protobuf")
	vConfig.Set("TELEMETRY__OTLP_HTTP__HEADERS", "Authorization=Bearer test-token")
	vConfig.Set("TELEMETRY__OTLP_HTTP__INSECURE", true)

	appConfig, err := GetApplicationConfig(vConfig)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig == nil || appConfig.TelemetryConfig == nil {
		t.Fatalf("telemetry config is nil")
	}
	if got := string(appConfig.TelemetryConfig.Type()); got != "otlp_http" {
		t.Fatalf("TelemetryConfig.Type() = %v, want otlp_http", got)
	}
	if appConfig.TelemetryConfig.OTLPHTTP == nil {
		t.Fatalf("TelemetryConfig.OTLPHTTP is nil")
	}
	if appConfig.TelemetryConfig.OTLPHTTP.Endpoint != "otel-collector:4318" {
		t.Fatalf("TelemetryConfig.OTLPHTTP.Endpoint = %q, want %q", appConfig.TelemetryConfig.OTLPHTTP.Endpoint, "otel-collector:4318")
	}
}
