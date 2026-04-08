// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
)

func (d *Dispatcher) handleEventEmitted(ctx context.Context, v sip_infra.EventEmittedPipeline) {
	d.logger.Debugw("Pipeline: Event", "call_id", v.ID, "event", v.Event)
}

func (d *Dispatcher) handleMetricEmitted(ctx context.Context, v sip_infra.MetricEmittedPipeline) {
	d.logger.Debugw("Pipeline: Metric", "call_id", v.ID, "count", len(v.Metrics))
}

func (d *Dispatcher) handleDTMFReceived(ctx context.Context, v sip_infra.DTMFReceivedPipeline) {
	d.logger.Debugw("Pipeline: DTMFReceived", "call_id", v.ID, "digit", v.Digit)
}
