// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observe_exporters

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/api/assistant-api/internal/observe"
)

// OTLPConfig holds configuration for connecting to an OTLP-compatible backend
// (Datadog APM, AWS X-Ray via ADOT, Google Cloud Trace, Azure Monitor, etc.).
type OTLPConfig struct {
	Endpoint string   // e.g. "localhost:4318" (HTTP) or "localhost:4317" (gRPC)
	Protocol string   // "http/protobuf" (default) or "grpc"
	Headers  []string // "Key=Value" pairs
	Insecure bool
}

// OTLPConfigFromOptions parses an OTLPConfig from a flat string key-value map
// (sourced from AssistantTelemetryProviderOption rows). The providerType
// ("otlp_http" or "otlp_grpc") is used to infer the default protocol.
func OTLPConfigFromOptions(opts map[string]interface{}, providerType string) OTLPConfig {
	cfg := OTLPConfig{}
	if v, ok := opts["endpoint"].(string); ok {
		cfg.Endpoint = v
	}
	if v, ok := opts["protocol"].(string); ok {
		cfg.Protocol = v
	} else if providerType == "otlp_grpc" {
		cfg.Protocol = "grpc"
	} else {
		cfg.Protocol = "http/protobuf"
	}
	if v, ok := opts["insecure"].(string); ok {
		cfg.Insecure = v == "true" || v == "1"
	}
	// Headers stored as comma-separated "Key=Value" pairs in a single option field,
	// or as individual header_0, header_1, … keys.
	if v, ok := opts["headers"].(string); ok && v != "" {
		cfg.Headers = strings.Split(v, ",")
	}
	return cfg
}

// OTLPExporter converts EventRecord and MetricRecord to OTEL spans and ships
// them to any OTLP-compatible backend via the configured endpoint.
//
// It implements both observe.EventExporter and observe.MetricExporter.
// A shared sdktrace.TracerProvider with BatchSpanProcessor handles internal
// batching and delivery. Shutdown() is idempotent (guarded by sync.Once).
type OTLPExporter struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	once     sync.Once
}

// NewOTLPExporter creates an OTLPExporter connected to the given OTLP endpoint.
func NewOTLPExporter(ctx context.Context, cfg OTLPConfig) (*OTLPExporter, error) {
	spanExporter, err := newOTLPSpanExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("observe/otlp: create span exporter: %w", err)
	}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("rapida-voice-assistant"),
			semconv.ServiceVersion("1.0"),
			semconv.TelemetrySDKLanguageGo,
		),
	)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	)

	return &OTLPExporter{
		provider: provider,
		tracer:   provider.Tracer("rapida.voice"),
	}, nil
}

// =============================================================================
// observe.EventExporter
// =============================================================================

func (e *OTLPExporter) ExportEvent(ctx context.Context, meta observe.SessionMeta, rec observe.EventRecord) error {
	t := rec.Time
	if t.IsZero() {
		t = time.Now()
	}
	attrs := []attribute.KeyValue{
		attribute.Int64("rapida.assistant.id", int64(meta.AssistantID)),
		attribute.Int64("rapida.conversation.id", int64(meta.AssistantConversationID)),
		attribute.Int64("rapida.project.id", int64(meta.ProjectID)),
		attribute.Int64("rapida.organization.id", int64(meta.OrganizationID)),
		attribute.String("rapida.event.message_id", rec.MessageID),
		attribute.String("rapida.event.name", rec.Name),
	}
	for k, v := range rec.Data {
		attrs = append(attrs, attribute.String("rapida.event.data."+k, v))
	}
	_, span := e.tracer.Start(ctx, "rapida.voice.event."+rec.Name,
		trace.WithTimestamp(t),
		trace.WithAttributes(attrs...),
	)
	span.End(trace.WithTimestamp(t))
	return nil
}

// =============================================================================
// observe.MetricExporter
// =============================================================================

func (e *OTLPExporter) ExportMetric(ctx context.Context, meta observe.SessionMeta, rec observe.MetricRecord) error {
	base := []attribute.KeyValue{
		attribute.Int64("rapida.assistant.id", int64(meta.AssistantID)),
		attribute.Int64("rapida.conversation.id", int64(meta.AssistantConversationID)),
		attribute.Int64("rapida.project.id", int64(meta.ProjectID)),
		attribute.Int64("rapida.organization.id", int64(meta.OrganizationID)),
	}
	switch m := rec.(type) {
	case observe.ConversationMetricRecord:
		t := m.Time
		if t.IsZero() {
			t = time.Now()
		}
		attrs := append(base, attribute.String("rapida.metric.conversation_id", m.ConversationID))
		for _, metric := range m.Metrics {
			attrs = append(attrs, attribute.String("rapida.metric."+metric.GetName(), metric.GetValue()))
		}
		_, span := e.tracer.Start(ctx, "rapida.voice.metric.conversation",
			trace.WithTimestamp(t),
			trace.WithAttributes(attrs...),
		)
		span.End(trace.WithTimestamp(t))

	case observe.MessageMetricRecord:
		t := m.Time
		if t.IsZero() {
			t = time.Now()
		}
		attrs := append(base,
			attribute.String("rapida.metric.message_id", m.MessageID),
			attribute.String("rapida.metric.conversation_id", m.ConversationID),
		)
		for _, metric := range m.Metrics {
			attrs = append(attrs, attribute.String("rapida.metric."+metric.GetName(), metric.GetValue()))
		}
		_, span := e.tracer.Start(ctx, "rapida.voice.metric.message",
			trace.WithTimestamp(t),
			trace.WithAttributes(attrs...),
		)
		span.End(trace.WithTimestamp(t))
	}
	return nil
}

// Shutdown flushes the batch processor and releases OTLP resources.
// Safe to call concurrently; executes only once.
func (e *OTLPExporter) Shutdown(ctx context.Context) error {
	var err error
	e.once.Do(func() {
		if ferr := e.provider.ForceFlush(ctx); ferr != nil {
			err = ferr
		}
		if serr := e.provider.Shutdown(ctx); serr != nil && err == nil {
			err = serr
		}
	})
	return err
}

// =============================================================================
// Internal helpers
// =============================================================================

func newOTLPSpanExporter(ctx context.Context, cfg OTLPConfig) (sdktrace.SpanExporter, error) {
	headers := parseOTLPHeaders(cfg.Headers)
	switch strings.ToLower(cfg.Protocol) {
	case "grpc":
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if len(headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(headers))
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			opts = append(opts, otlptracegrpc.WithInsecure()) //nolint:staticcheck
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		if len(headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)
	}
}

func parseOTLPHeaders(pairs []string) map[string]string {
	headers := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		if k, v, ok := strings.Cut(pair, "="); ok {
			headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return headers
}
