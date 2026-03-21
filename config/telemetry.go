package config

import (
	"fmt"
	"strconv"
	"strings"
)

// TelemetryProviderType enumerates supported telemetry exporter backends.
type TelemetryProviderType string

const (
	TelemetryOTLPHTTP     TelemetryProviderType = "otlp_http"
	TelemetryOTLPGRPC     TelemetryProviderType = "otlp_grpc"
	TelemetryXRay         TelemetryProviderType = "xray"
	TelemetryGoogleTrace  TelemetryProviderType = "google_trace"
	TelemetryAzureMonitor TelemetryProviderType = "azure_monitor"
	TelemetryDatadog      TelemetryProviderType = "datadog"
	TelemetryOpenSearch   TelemetryProviderType = "opensearch"
	TelemetryLogging      TelemetryProviderType = "logging"
)

// TelemetryOTLPConfig configures generic OTLP exporter behavior.
type TelemetryOTLPConfig struct {
	Endpoint string
	Protocol string
	Headers  []string
	Insecure bool
}

// TelemetryDatadogConfig configures Datadog OTLP export.
type TelemetryDatadogConfig struct {
	Endpoint string
	Protocol string
	Headers  []string
	Insecure bool
}

// TelemetryGoogleTraceConfig configures Google Cloud Trace export via OTLP HTTP.
type TelemetryGoogleTraceConfig struct {
	Endpoint    string
	AccessToken string
	APIKey      string
	Headers     []string
	Insecure    bool
}

// TelemetryAzureMonitorConfig configures Azure Monitor export via OTLP HTTP.
type TelemetryAzureMonitorConfig struct {
	Endpoint string
	APIKey   string
	Headers  []string
	Insecure bool
}

// TelemetryXRayConfig configures AWS X-Ray export via ADOT/OTLP.
type TelemetryXRayConfig struct {
	Endpoint string
	Protocol string
	Insecure bool
	Region   string
}

// TelemetryOpenSearchConfig exists for uniform provider configuration.
type TelemetryOpenSearchConfig struct{}

// TelemetryLoggingConfig exists for uniform provider configuration.
type TelemetryLoggingConfig struct{}

// OTLPTelemetryConfigFromOptions parses OTLP config from a flat option map.
func OTLPTelemetryConfigFromOptions(opts map[string]interface{}, providerType string) TelemetryOTLPConfig {
	cfg := TelemetryOTLPConfig{
		Endpoint: optionString(opts, "endpoint"),
		Insecure: optionBool(opts, "insecure"),
		Headers:  optionHeaders(opts, "headers"),
	}
	cfg.Protocol = optionString(opts, "protocol")
	if cfg.Protocol == "" {
		if providerType == TelemetryOTLPGRPC.String() {
			cfg.Protocol = "grpc"
		} else {
			cfg.Protocol = "http/protobuf"
		}
	}
	return cfg
}

// DatadogTelemetryConfigFromOptions parses Datadog options.
func DatadogTelemetryConfigFromOptions(opts map[string]interface{}) (TelemetryDatadogConfig, error) {
	cfg := TelemetryDatadogConfig{
		Endpoint: optionString(opts, "endpoint"),
		Protocol: optionString(opts, "protocol"),
		Insecure: optionBool(opts, "insecure"),
		Headers:  optionHeaders(opts, "headers"),
	}
	if cfg.Endpoint == "" {
		return TelemetryDatadogConfig{}, fmt.Errorf("telemetry/datadog: missing required option 'endpoint'")
	}
	if cfg.Protocol == "" {
		if strings.HasPrefix(cfg.Endpoint, "http") {
			cfg.Protocol = "http/protobuf"
		} else {
			cfg.Protocol = "grpc"
		}
	}
	if apiKey := optionString(opts, "api_key"); apiKey != "" {
		cfg.Headers = append([]string{"DD-API-KEY=" + apiKey}, cfg.Headers...)
	}
	return cfg, nil
}

// GoogleTraceTelemetryConfigFromOptions parses Google Trace options.
func GoogleTraceTelemetryConfigFromOptions(opts map[string]interface{}) (TelemetryGoogleTraceConfig, error) {
	cfg := TelemetryGoogleTraceConfig{
		Endpoint:    optionString(opts, "endpoint"),
		AccessToken: optionString(opts, "access_token"),
		APIKey:      optionString(opts, "api_key"),
		Headers:     optionHeaders(opts, "headers"),
		Insecure:    optionBool(opts, "insecure"),
	}
	if cfg.Endpoint == "" {
		return TelemetryGoogleTraceConfig{}, fmt.Errorf("telemetry/google_trace: missing required option 'endpoint'")
	}
	return cfg, nil
}

// AzureMonitorTelemetryConfigFromOptions parses Azure Monitor options.
func AzureMonitorTelemetryConfigFromOptions(opts map[string]interface{}) (TelemetryAzureMonitorConfig, error) {
	cfg := TelemetryAzureMonitorConfig{
		Endpoint: optionString(opts, "endpoint"),
		APIKey:   optionString(opts, "api_key"),
		Headers:  optionHeaders(opts, "headers"),
		Insecure: optionBool(opts, "insecure"),
	}
	if cfg.Endpoint == "" {
		return TelemetryAzureMonitorConfig{}, fmt.Errorf("telemetry/azure_monitor: missing required option 'endpoint'")
	}
	return cfg, nil
}

// XRayTelemetryConfigFromOptions parses X-Ray options.
func XRayTelemetryConfigFromOptions(opts map[string]interface{}) (TelemetryXRayConfig, error) {
	cfg := TelemetryXRayConfig{
		Endpoint: optionString(opts, "endpoint"),
		Protocol: optionString(opts, "protocol"),
		Insecure: optionBool(opts, "insecure"),
		Region:   optionString(opts, "region"),
	}
	if cfg.Endpoint == "" {
		return TelemetryXRayConfig{}, fmt.Errorf("telemetry/xray: missing required option 'endpoint'")
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "http/protobuf"
	}
	return cfg, nil
}

// OpenSearchTelemetryConfigFromOptions parses OpenSearch options.
func OpenSearchTelemetryConfigFromOptions(_ map[string]interface{}) (TelemetryOpenSearchConfig, error) {
	return TelemetryOpenSearchConfig{}, nil
}

// LoggingTelemetryConfigFromOptions parses logging options.
func LoggingTelemetryConfigFromOptions(_ map[string]interface{}) (TelemetryLoggingConfig, error) {
	return TelemetryLoggingConfig{}, nil
}

func (t TelemetryProviderType) String() string {
	return string(t)
}

func optionString(opts map[string]interface{}, key string) string {
	if opts == nil {
		return ""
	}
	v, ok := opts[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []byte:
		return strings.TrimSpace(string(t))
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func optionBool(opts map[string]interface{}, key string) bool {
	if opts == nil {
		return false
	}
	v, ok := opts[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(t))
		return err == nil && b
	case int:
		return t == 1
	case int8:
		return t == 1
	case int16:
		return t == 1
	case int32:
		return t == 1
	case int64:
		return t == 1
	case uint:
		return t == 1
	case uint8:
		return t == 1
	case uint16:
		return t == 1
	case uint32:
		return t == 1
	case uint64:
		return t == 1
	case float32:
		return t == 1
	case float64:
		return t == 1
	default:
		return false
	}
}

func optionHeaders(opts map[string]interface{}, key string) []string {
	raw := optionString(opts, key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
