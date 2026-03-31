// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_caller_metrics

import (
	"testing"

	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findMetric(metrics []*protos.Metric, name string) *protos.Metric {
	for _, m := range metrics {
		if m.GetName() == name {
			return m
		}
	}
	return nil
}

func TestNewMetricBuilder(t *testing.T) {
	mb := NewMetricBuilder(42)
	require.NotNil(t, mb)
	assert.Equal(t, uint64(42), mb.requestId)
	assert.Empty(t, mb.Build(), "no metrics before OnStart")
}

func TestOnStart_InitializesMetrics(t *testing.T) {
	mb := NewMetricBuilder(99)
	mb.OnStart()

	metrics := mb.Build()
	assert.Len(t, metrics, 3, "OnStart should produce TIME_TAKEN, LLM_REQUEST_ID, STATUS")

	timeTaken := findMetric(metrics, "time_taken")
	require.NotNil(t, timeTaken)

	requestID := findMetric(metrics, "llm_request_id")
	require.NotNil(t, requestID)
	assert.Equal(t, "99", requestID.Value)

	status := findMetric(metrics, "status")
	require.NotNil(t, status)
	assert.Equal(t, "FAILED", status.Value, "initial status should be FAILED")
}

func TestOnSuccess_UpdatesStatus(t *testing.T) {
	mb := NewMetricBuilder(1)
	mb.OnStart().OnSuccess()

	metrics := mb.Build()
	status := findMetric(metrics, "status")
	require.NotNil(t, status)
	assert.Equal(t, "SUCCESS", status.Value)
}

func TestOnFailure_KeepsFailedStatus(t *testing.T) {
	mb := NewMetricBuilder(1)
	mb.OnStart().OnFailure()

	metrics := mb.Build()
	status := findMetric(metrics, "status")
	require.NotNil(t, status)
	assert.Equal(t, "FAILED", status.Value)
}

func TestOnAddMetrics_AddsCustomMetrics(t *testing.T) {
	mb := NewMetricBuilder(1)
	mb.OnStart()
	mb.OnAddMetrics(
		&protos.Metric{Name: "INPUT_TOKEN", Value: "100"},
		&protos.Metric{Name: "OUTPUT_TOKEN", Value: "50"},
	)

	metrics := mb.Build()
	assert.Len(t, metrics, 5, "3 base + 2 custom")

	input := findMetric(metrics, "INPUT_TOKEN")
	require.NotNil(t, input)
	assert.Equal(t, "100", input.Value)

	output := findMetric(metrics, "OUTPUT_TOKEN")
	require.NotNil(t, output)
	assert.Equal(t, "50", output.Value)
}

func TestOnAddMetrics_OverwritesDuplicates(t *testing.T) {
	mb := NewMetricBuilder(1)
	mb.OnStart()
	mb.OnAddMetrics(&protos.Metric{Name: "INPUT_TOKEN", Value: "100"})
	mb.OnAddMetrics(&protos.Metric{Name: "INPUT_TOKEN", Value: "200"})

	metrics := mb.Build()
	input := findMetric(metrics, "INPUT_TOKEN")
	require.NotNil(t, input)
	assert.Equal(t, "200", input.Value, "later value should overwrite earlier")
}

func TestChaining(t *testing.T) {
	metrics := NewMetricBuilder(1).
		OnStart().
		OnAddMetrics(&protos.Metric{Name: "CUSTOM", Value: "v"}).
		OnSuccess().
		Build()

	assert.Len(t, metrics, 4)
	status := findMetric(metrics, "status")
	require.NotNil(t, status)
	assert.Equal(t, "SUCCESS", status.Value)
}
