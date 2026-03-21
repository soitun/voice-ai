package telemetry_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/telemetry"
)

type fakeExporter struct {
	mu            sync.Mutex
	eventCalls    int
	metricCalls   int
	shutdownCalls int
}

func (f *fakeExporter) ExportEvent(_ context.Context, _ telemetry.SessionMeta, _ telemetry.EventRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.eventCalls++
	return nil
}

func (f *fakeExporter) ExportMetric(_ context.Context, _ telemetry.SessionMeta, _ telemetry.MetricRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.metricCalls++
	return nil
}

func (f *fakeExporter) Shutdown(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shutdownCalls++
	return nil
}

func testLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger(
		commons.Name("telemetry-collector-test"),
		commons.Level("error"),
		commons.EnableFile(false),
	)
	require.NoError(t, err)
	return logger
}

func TestCollectors_FanoutAndShutdown(t *testing.T) {
	logger := testLogger(t)
	meta := telemetry.SessionMeta{AssistantID: 1}

	evt1 := &fakeExporter{}
	evt2 := &fakeExporter{}
	eventCollector := telemetry.NewEventCollector(logger, meta, evt1, evt2)
	eventCollector.Collect(context.Background(), telemetry.EventRecord{Name: "session"})
	eventCollector.Shutdown(context.Background())

	assert.Equal(t, 1, evt1.eventCalls)
	assert.Equal(t, 1, evt2.eventCalls)
	assert.Equal(t, 1, evt1.shutdownCalls)
	assert.Equal(t, 1, evt2.shutdownCalls)

	met1 := &fakeExporter{}
	met2 := &fakeExporter{}
	metricCollector := telemetry.NewMetricCollector(logger, meta, met1, met2)
	metricCollector.Collect(context.Background(), telemetry.ConversationMetricRecord{})
	metricCollector.Shutdown(context.Background())

	assert.Equal(t, 1, met1.metricCalls)
	assert.Equal(t, 1, met2.metricCalls)
	assert.Equal(t, 1, met1.shutdownCalls)
	assert.Equal(t, 1, met2.shutdownCalls)
}

func TestCollectors_Noop(t *testing.T) {
	logger := testLogger(t)
	meta := telemetry.SessionMeta{}

	eventCollector := telemetry.NewEventCollector(logger, meta)
	metricCollector := telemetry.NewMetricCollector(logger, meta)

	assert.NotPanics(t, func() {
		eventCollector.Collect(context.Background(), telemetry.EventRecord{Name: "x"})
		metricCollector.Collect(context.Background(), telemetry.ConversationMetricRecord{})
		eventCollector.Shutdown(context.Background())
		metricCollector.Shutdown(context.Background())
	})
}
