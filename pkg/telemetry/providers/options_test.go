package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTLPConfigFromOptions_Defaults(t *testing.T) {
	cfgHTTP := OTLPConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4318",
	}, "otlp_http")
	assert.Equal(t, "localhost:4318", cfgHTTP.Endpoint)
	assert.Equal(t, "http/protobuf", cfgHTTP.Protocol)

	cfgGRPC := OTLPConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4317",
	}, "otlp_grpc")
	assert.Equal(t, "localhost:4317", cfgGRPC.Endpoint)
	assert.Equal(t, "grpc", cfgGRPC.Protocol)
}

func TestOTLPConfigFromOptions_ParsesHeadersAndBool(t *testing.T) {
	cfg := OTLPConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4318",
		"headers":  "A=B, C=D",
		"insecure": "true",
	}, "otlp_http")
	assert.True(t, cfg.Insecure)
	assert.Equal(t, []string{"A=B", "C=D"}, cfg.Headers)
}

func TestDatadogConfigFromOptions(t *testing.T) {
	_, err := DatadogConfigFromOptions(map[string]interface{}{})
	require.Error(t, err)

	cfg, err := DatadogConfigFromOptions(map[string]interface{}{
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

func TestGoogleTraceConfigFromOptions(t *testing.T) {
	_, err := GoogleTraceConfigFromOptions(map[string]interface{}{})
	require.Error(t, err)

	cfg, err := GoogleTraceConfigFromOptions(map[string]interface{}{
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

func TestAzureMonitorConfigFromOptions(t *testing.T) {
	_, err := AzureMonitorConfigFromOptions(map[string]interface{}{})
	require.Error(t, err)

	cfg, err := AzureMonitorConfigFromOptions(map[string]interface{}{
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

func TestXRayConfigFromOptions(t *testing.T) {
	_, err := XRayConfigFromOptions(map[string]interface{}{})
	require.Error(t, err)

	cfg, err := XRayConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4318",
		"region":   "us-east-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "localhost:4318", cfg.Endpoint)
	assert.Equal(t, "http/protobuf", cfg.Protocol)
	assert.Equal(t, "us-east-1", cfg.Region)
}
