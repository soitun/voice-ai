package providers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/telemetry"
)

func factoryTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger(
		commons.Name("telemetry-factory-test"),
		commons.Level("error"),
		commons.EnableFile(false),
	)
	require.NoError(t, err)
	return logger
}

func TestNewExporterFromOptions_UnknownProvider(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), "unknown", nil, FactoryDependencies{})
	require.Error(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_Logging(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.LOGGING.String(), nil, FactoryDependencies{
		Logger: factoryTestLogger(t),
	})
	require.NoError(t, err)
	require.NotNil(t, exp)
}

func TestNewExporterFromOptions_OpenSearchMissingConnector(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.OPENSEARCH.String(), nil, FactoryDependencies{
		Logger: factoryTestLogger(t),
	})
	require.Error(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_OTLPSkipsWhenNoEndpoint(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.OTLP_HTTP.String(), map[string]interface{}{}, FactoryDependencies{
		Logger: factoryTestLogger(t),
	})
	require.NoError(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_DatadogMissingEndpoint(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.DATADOG.String(), map[string]interface{}{}, FactoryDependencies{
		Logger: factoryTestLogger(t),
	})
	require.Error(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_LoggingMissingLogger(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.LOGGING.String(), nil, FactoryDependencies{})
	require.Error(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_OpenSearchMissingLogger(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.OPENSEARCH.String(), nil, FactoryDependencies{})
	require.Error(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_OTLPGRPCSkipsWhenNoEndpoint(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.OTLP_GRPC.String(), map[string]interface{}{}, FactoryDependencies{
		Logger: factoryTestLogger(t),
	})
	require.NoError(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_XRayMissingEndpoint(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.XRAY.String(), map[string]interface{}{}, FactoryDependencies{
		Logger: factoryTestLogger(t),
	})
	require.Error(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_GoogleTraceMissingEndpoint(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.GOOGLE_TRACE.String(), map[string]interface{}{}, FactoryDependencies{
		Logger: factoryTestLogger(t),
	})
	require.Error(t, err)
	assert.Nil(t, exp)
}

func TestNewExporterFromOptions_AzureMonitorMissingEndpoint(t *testing.T) {
	exp, err := NewExporterFromOptions(context.Background(), telemetry.AZURE_MONITOR.String(), map[string]interface{}{}, FactoryDependencies{
		Logger: factoryTestLogger(t),
	})
	require.Error(t, err)
	assert.Nil(t, exp)
}
