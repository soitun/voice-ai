package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// optString
// =============================================================================

func TestOptString_NilMap(t *testing.T) {
	assert.Equal(t, "", optString(nil, "key"))
}

func TestOptString_MissingKey(t *testing.T) {
	assert.Equal(t, "", optString(map[string]interface{}{}, "key"))
}

func TestOptString_StringValue(t *testing.T) {
	assert.Equal(t, "hello", optString(map[string]interface{}{"k": "hello"}, "k"))
}

func TestOptString_TrimsWhitespace(t *testing.T) {
	assert.Equal(t, "val", optString(map[string]interface{}{"k": "  val  "}, "k"))
}

func TestOptString_NonStringValue(t *testing.T) {
	assert.Equal(t, "42", optString(map[string]interface{}{"k": 42}, "k"))
}

// =============================================================================
// optStringDefault
// =============================================================================

func TestOptStringDefault_ReturnsDefault(t *testing.T) {
	assert.Equal(t, "fallback", optStringDefault(nil, "k", "fallback"))
	assert.Equal(t, "fallback", optStringDefault(map[string]interface{}{}, "k", "fallback"))
	assert.Equal(t, "fallback", optStringDefault(map[string]interface{}{"k": ""}, "k", "fallback"))
}

func TestOptStringDefault_ReturnsValue(t *testing.T) {
	assert.Equal(t, "val", optStringDefault(map[string]interface{}{"k": "val"}, "k", "fallback"))
}

// =============================================================================
// optBool
// =============================================================================

func TestOptBool_NilMap(t *testing.T) {
	assert.False(t, optBool(nil, "key"))
}

func TestOptBool_MissingKey(t *testing.T) {
	assert.False(t, optBool(map[string]interface{}{}, "key"))
}

func TestOptBool_BoolTrue(t *testing.T) {
	assert.True(t, optBool(map[string]interface{}{"k": true}, "k"))
}

func TestOptBool_BoolFalse(t *testing.T) {
	assert.False(t, optBool(map[string]interface{}{"k": false}, "k"))
}

func TestOptBool_StringTrue(t *testing.T) {
	assert.True(t, optBool(map[string]interface{}{"k": "true"}, "k"))
	assert.True(t, optBool(map[string]interface{}{"k": "1"}, "k"))
}

func TestOptBool_StringFalse(t *testing.T) {
	assert.False(t, optBool(map[string]interface{}{"k": "false"}, "k"))
	assert.False(t, optBool(map[string]interface{}{"k": "0"}, "k"))
}

func TestOptBool_UnsupportedType(t *testing.T) {
	assert.False(t, optBool(map[string]interface{}{"k": 123}, "k"))
}

// =============================================================================
// optHeaders
// =============================================================================

func TestOptHeaders_NilMap(t *testing.T) {
	assert.Nil(t, optHeaders(nil, "headers"))
}

func TestOptHeaders_MissingKey(t *testing.T) {
	assert.Nil(t, optHeaders(map[string]interface{}{}, "headers"))
}

func TestOptHeaders_EmptyString(t *testing.T) {
	assert.Nil(t, optHeaders(map[string]interface{}{"headers": ""}, "headers"))
	assert.Nil(t, optHeaders(map[string]interface{}{"headers": "   "}, "headers"))
}

func TestOptHeaders_CommaSeparatedString(t *testing.T) {
	got := optHeaders(map[string]interface{}{"headers": "A=B, C=D"}, "headers")
	assert.Equal(t, []string{"A=B", "C=D"}, got)
}

func TestOptHeaders_SingleString(t *testing.T) {
	got := optHeaders(map[string]interface{}{"headers": "Authorization=Bearer token"}, "headers")
	assert.Equal(t, []string{"Authorization=Bearer token"}, got)
}

func TestOptHeaders_SliceInput(t *testing.T) {
	input := []string{"X-Key=val1", "X-Other=val2"}
	got := optHeaders(map[string]interface{}{"headers": input}, "headers")
	assert.Equal(t, input, got)
}

func TestOptHeaders_UnsupportedType(t *testing.T) {
	assert.Nil(t, optHeaders(map[string]interface{}{"headers": 42}, "headers"))
}

// =============================================================================
// OpenSearchConfigFromOptions
// =============================================================================

func TestOpenSearchConfigFromOptions_Defaults(t *testing.T) {
	cfg, err := OpenSearchConfigFromOptions(nil)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.IndexPrefix)
}

func TestOpenSearchConfigFromOptions_WithPrefix(t *testing.T) {
	cfg, err := OpenSearchConfigFromOptions(map[string]interface{}{
		"index_prefix": "myapp",
	})
	require.NoError(t, err)
	assert.Equal(t, "myapp", cfg.IndexPrefix)
}

// =============================================================================
// LoggingConfigFromOptions
// =============================================================================

func TestLoggingConfigFromOptions_NilOpts(t *testing.T) {
	cfg, err := LoggingConfigFromOptions(nil)
	require.NoError(t, err)
	assert.Equal(t, LoggingConfig{}, cfg)
}

func TestLoggingConfigFromOptions_EmptyOpts(t *testing.T) {
	cfg, err := LoggingConfigFromOptions(map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, LoggingConfig{}, cfg)
}

// =============================================================================
// OTLPConfigFromOptions — additional edge cases
// =============================================================================

func TestOTLPConfigFromOptions_NilOpts(t *testing.T) {
	cfg := OTLPConfigFromOptions(nil, "otlp_http")
	assert.Equal(t, "", cfg.Endpoint)
	assert.Equal(t, "http/protobuf", cfg.Protocol)
	assert.Nil(t, cfg.Headers)
	assert.False(t, cfg.Insecure)
}

func TestOTLPConfigFromOptions_ExplicitProtocol(t *testing.T) {
	cfg := OTLPConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4317",
		"protocol": "grpc",
	}, "otlp_http") // providerType says http, but explicit protocol wins
	assert.Equal(t, "grpc", cfg.Protocol)
}

// =============================================================================
// DatadogConfigFromOptions — additional edge cases
// =============================================================================

func TestDatadogConfigFromOptions_NoAPIKey(t *testing.T) {
	cfg, err := DatadogConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4317",
	})
	require.NoError(t, err)
	assert.Equal(t, "localhost:4317", cfg.Endpoint)
	assert.Empty(t, cfg.APIKey)
	assert.Nil(t, cfg.Headers)
}

// =============================================================================
// XRayConfigFromOptions — additional edge cases
// =============================================================================

func TestXRayConfigFromOptions_InsecureFlag(t *testing.T) {
	cfg, err := XRayConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4318",
		"insecure": true,
	})
	require.NoError(t, err)
	assert.True(t, cfg.Insecure)
}

func TestXRayConfigFromOptions_CustomProtocol(t *testing.T) {
	cfg, err := XRayConfigFromOptions(map[string]interface{}{
		"endpoint": "localhost:4317",
		"protocol": "grpc",
	})
	require.NoError(t, err)
	assert.Equal(t, "grpc", cfg.Protocol)
}

// =============================================================================
// GoogleTraceConfigFromOptions — additional edge cases
// =============================================================================

func TestGoogleTraceConfigFromOptions_NoCredentials(t *testing.T) {
	cfg, err := GoogleTraceConfigFromOptions(map[string]interface{}{
		"endpoint": "cloudtrace.googleapis.com",
	})
	require.NoError(t, err)
	assert.Empty(t, cfg.APIKey)
	assert.Empty(t, cfg.AccessToken)
	assert.Nil(t, cfg.Headers)
}

// =============================================================================
// AzureMonitorConfigFromOptions — additional edge cases
// =============================================================================

func TestAzureMonitorConfigFromOptions_NoAPIKey(t *testing.T) {
	cfg, err := AzureMonitorConfigFromOptions(map[string]interface{}{
		"endpoint": "eastus.in.applicationinsights.azure.com",
	})
	require.NoError(t, err)
	assert.Empty(t, cfg.APIKey)
	assert.Nil(t, cfg.Headers)
}
