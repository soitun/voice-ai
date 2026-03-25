// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package providers

import (
	"fmt"
	"strings"
)

// OTLPConfig holds configuration for connecting to an OTLP-compatible backend.
type OTLPConfig struct {
	Endpoint string
	Protocol string
	Headers  []string
	Insecure bool
}

// DatadogConfig configures Datadog OTLP export.
type DatadogConfig struct {
	Endpoint string
	Protocol string
	Headers  []string
	APIKey   string
	Insecure bool
}

// XRayConfig configures AWS X-Ray export via ADOT/OTLP.
type XRayConfig struct {
	Endpoint string
	Protocol string
	Insecure bool
	Region   string
}

// GoogleTraceConfig configures Google Cloud Trace export via OTLP HTTP.
type GoogleTraceConfig struct {
	Endpoint    string
	Headers     []string
	APIKey      string
	AccessToken string
	Insecure    bool
}

// AzureMonitorConfig configures Azure Monitor OTLP HTTP export.
type AzureMonitorConfig struct {
	Endpoint string
	Headers  []string
	APIKey   string
	Insecure bool
}

// OpenSearchConfig configures the OpenSearch telemetry exporter.
type OpenSearchConfig struct {
	IndexPrefix string
}

// LoggingConfig configures the logging telemetry exporter.
type LoggingConfig struct{}

// =============================================================================
// FromOptions parsers — parse from map[string]interface{} (DB options or ToMap)
// =============================================================================

// OTLPConfigFromOptions parses an OTLPConfig from a flat option map.
// providerType ("otlp_http" or "otlp_grpc") infers the default protocol.
func OTLPConfigFromOptions(opts map[string]interface{}, providerType string) OTLPConfig {
	protocol := optString(opts, "protocol")
	if protocol == "" {
		if providerType == "otlp_grpc" {
			protocol = "grpc"
		} else {
			protocol = "http/protobuf"
		}
	}
	return OTLPConfig{
		Endpoint: optString(opts, "endpoint"),
		Protocol: protocol,
		Headers:  optHeaders(opts, "headers"),
		Insecure: optBool(opts, "insecure"),
	}
}

// DatadogConfigFromOptions parses Datadog options. Requires endpoint.
func DatadogConfigFromOptions(opts map[string]interface{}) (DatadogConfig, error) {
	endpoint := optString(opts, "endpoint")
	if endpoint == "" {
		return DatadogConfig{}, fmt.Errorf("telemetry/datadog: endpoint is required")
	}
	apiKey := optString(opts, "api_key")
	headers := optHeaders(opts, "headers")
	if apiKey != "" {
		headers = append([]string{"DD-API-KEY=" + apiKey}, headers...)
	}
	return DatadogConfig{
		Endpoint: endpoint,
		Protocol: optStringDefault(opts, "protocol", "http/protobuf"),
		Headers:  headers,
		APIKey:   apiKey,
		Insecure: optBool(opts, "insecure"),
	}, nil
}

// XRayConfigFromOptions parses X-Ray options. Requires endpoint.
func XRayConfigFromOptions(opts map[string]interface{}) (XRayConfig, error) {
	endpoint := optString(opts, "endpoint")
	if endpoint == "" {
		return XRayConfig{}, fmt.Errorf("telemetry/xray: endpoint is required")
	}
	return XRayConfig{
		Endpoint: endpoint,
		Protocol: optStringDefault(opts, "protocol", "http/protobuf"),
		Insecure: optBool(opts, "insecure"),
		Region:   optString(opts, "region"),
	}, nil
}

// GoogleTraceConfigFromOptions parses Google Trace options. Requires endpoint.
func GoogleTraceConfigFromOptions(opts map[string]interface{}) (GoogleTraceConfig, error) {
	endpoint := optString(opts, "endpoint")
	if endpoint == "" {
		return GoogleTraceConfig{}, fmt.Errorf("telemetry/google_trace: endpoint is required")
	}
	return GoogleTraceConfig{
		Endpoint:    endpoint,
		Headers:     optHeaders(opts, "headers"),
		APIKey:      optString(opts, "api_key"),
		AccessToken: optString(opts, "access_token"),
		Insecure:    optBool(opts, "insecure"),
	}, nil
}

// AzureMonitorConfigFromOptions parses Azure Monitor options. Requires endpoint.
func AzureMonitorConfigFromOptions(opts map[string]interface{}) (AzureMonitorConfig, error) {
	endpoint := optString(opts, "endpoint")
	if endpoint == "" {
		return AzureMonitorConfig{}, fmt.Errorf("telemetry/azure_monitor: endpoint is required")
	}
	return AzureMonitorConfig{
		Endpoint: endpoint,
		Headers:  optHeaders(opts, "headers"),
		APIKey:   optString(opts, "api_key"),
		Insecure: optBool(opts, "insecure"),
	}, nil
}

// OpenSearchConfigFromOptions parses OpenSearch options.
func OpenSearchConfigFromOptions(opts map[string]interface{}) (OpenSearchConfig, error) {
	return OpenSearchConfig{
		IndexPrefix: optString(opts, "index_prefix"),
	}, nil
}

// LoggingConfigFromOptions parses logging options.
func LoggingConfigFromOptions(opts map[string]interface{}) (LoggingConfig, error) {
	return LoggingConfig{}, nil
}

// =============================================================================
// option helpers
// =============================================================================

func optString(opts map[string]interface{}, key string) string {
	if opts == nil {
		return ""
	}
	v, ok := opts[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func optStringDefault(opts map[string]interface{}, key, def string) string {
	s := optString(opts, key)
	if s == "" {
		return def
	}
	return s
}

func optBool(opts map[string]interface{}, key string) bool {
	if opts == nil {
		return false
	}
	v, ok := opts[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true" || b == "1"
	default:
		return false
	}
}

// optHeaders parses a headers value that may be a single comma-separated string
// or already a []string.
func optHeaders(opts map[string]interface{}, key string) []string {
	if opts == nil {
		return nil
	}
	v, ok := opts[key]
	if !ok {
		return nil
	}
	switch h := v.(type) {
	case []string:
		return h
	case string:
		if strings.TrimSpace(h) == "" {
			return nil
		}
		parts := strings.Split(h, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	default:
		return nil
	}
}
