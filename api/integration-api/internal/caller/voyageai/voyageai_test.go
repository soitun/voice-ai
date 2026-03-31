// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_voyageai_callers

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func TestEndpoint(t *testing.T) {
	vg := &Voyageai{logger: newTestLogger()}
	assert.Equal(t, "https://api.voyageai.com/v1/embeddings", vg.Endpoint("embeddings"))
	assert.Equal(t, "https://api.voyageai.com/v1/rerank", vg.Endpoint("rerank"))
}

func TestUsageMetrics_WithUsage(t *testing.T) {
	vg := &Voyageai{logger: newTestLogger()}
	usage := &VoyageaiUsage{TotalTokens: 150}
	metrics := vg.UsageMetrics(usage)
	require.Len(t, metrics, 1)
	assert.Equal(t, "total_token", metrics[0].Name)
	assert.Equal(t, "150", metrics[0].Value)
}

func TestUsageMetrics_NilUsage(t *testing.T) {
	vg := &Voyageai{logger: newTestLogger()}
	metrics := vg.UsageMetrics(nil)
	assert.Empty(t, metrics)
}

func TestVoyageaiError_Format(t *testing.T) {
	err := VoyageaiError{Detail: "bad request", StatusCode: 400}
	assert.Contains(t, err.Error(), "bad request")
	assert.Contains(t, err.Error(), "400")
}
