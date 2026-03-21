package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTLPTelemetryConfigFromOptions_Defaults(t *testing.T) {
	cfgHTTP := OTLPTelemetryConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4318",
	}, TelemetryOTLPHTTP.String())
	assert.Equal(t, "localhost:4318", cfgHTTP.Endpoint)
	assert.Equal(t, "http/protobuf", cfgHTTP.Protocol)

	cfgGRPC := OTLPTelemetryConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4317",
	}, TelemetryOTLPGRPC.String())
	assert.Equal(t, "localhost:4317", cfgGRPC.Endpoint)
	assert.Equal(t, "grpc", cfgGRPC.Protocol)
}

func TestOTLPTelemetryConfigFromOptions_HeadersAndBool(t *testing.T) {
	cfg := OTLPTelemetryConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4318",
		"headers":  "A=B, C=D",
		"insecure": "true",
	}, TelemetryOTLPHTTP.String())
	assert.True(t, cfg.Insecure)
	assert.Equal(t, []string{"A=B", "C=D"}, cfg.Headers)
}

func TestDatadogTelemetryConfigFromOptions(t *testing.T) {
	_, err := DatadogTelemetryConfigFromOptions(map[string]interface{}{})
	require.Error(t, err)

	cfg, err := DatadogTelemetryConfigFromOptions(map[string]interface{}{
		"endpoint": "https://trace.agent.datadoghq.com",
		"api_key":  "secret",
		"headers":  "X-Env=prod",
		"insecure": true,
	})
	require.NoError(t, err)
	assert.Equal(t, "http/protobuf", cfg.Protocol)
	assert.True(t, cfg.Insecure)
	assert.Equal(t, []string{"DD-API-KEY=secret", "X-Env=prod"}, cfg.Headers)
}

func TestGoogleTraceTelemetryConfigFromOptions(t *testing.T) {
	_, err := GoogleTraceTelemetryConfigFromOptions(map[string]interface{}{})
	require.Error(t, err)

	cfg, err := GoogleTraceTelemetryConfigFromOptions(map[string]interface{}{
		"endpoint":     "cloudtrace.googleapis.com",
		"access_token": "token",
		"api_key":      "key",
		"headers":      "X-Foo=bar",
		"insecure":     "1",
	})
	require.NoError(t, err)
	assert.Equal(t, "cloudtrace.googleapis.com", cfg.Endpoint)
	assert.Equal(t, "token", cfg.AccessToken)
	assert.Equal(t, "key", cfg.APIKey)
	assert.Equal(t, []string{"X-Foo=bar"}, cfg.Headers)
	assert.True(t, cfg.Insecure)
}

func TestAzureMonitorTelemetryConfigFromOptions(t *testing.T) {
	_, err := AzureMonitorTelemetryConfigFromOptions(map[string]interface{}{})
	require.Error(t, err)

	cfg, err := AzureMonitorTelemetryConfigFromOptions(map[string]interface{}{
		"endpoint": "eastus.in.applicationinsights.azure.com",
		"api_key":  "abc",
		"headers":  "X-Test=yes",
		"insecure": "false",
	})
	require.NoError(t, err)
	assert.Equal(t, "eastus.in.applicationinsights.azure.com", cfg.Endpoint)
	assert.Equal(t, "abc", cfg.APIKey)
	assert.Equal(t, []string{"X-Test=yes"}, cfg.Headers)
	assert.False(t, cfg.Insecure)
}

func TestXRayTelemetryConfigFromOptions(t *testing.T) {
	_, err := XRayTelemetryConfigFromOptions(map[string]interface{}{})
	require.Error(t, err)

	cfg, err := XRayTelemetryConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4318",
		"region":   "us-east-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "localhost:4318", cfg.Endpoint)
	assert.Equal(t, "http/protobuf", cfg.Protocol)
	assert.Equal(t, "us-east-1", cfg.Region)
}
