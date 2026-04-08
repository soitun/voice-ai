// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"fmt"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
)

func (d *Dispatcher) handleByeReceived(ctx context.Context, v sip_infra.ByeReceivedPipeline) {
	d.logger.Infow("Pipeline: ByeReceived", "call_id", v.ID, "reason", v.Reason)
}

func (d *Dispatcher) handleCancelReceived(ctx context.Context, v sip_infra.CancelReceivedPipeline) {
	d.logger.Infow("Pipeline: CancelReceived", "call_id", v.ID)
}

func (d *Dispatcher) handleTransferRequested(ctx context.Context, v sip_infra.TransferRequestedPipeline) {
	d.logger.Warnw("Pipeline: TransferRequested (not supported)", "call_id", v.ID, "target", v.TargetURI)
}

func (d *Dispatcher) handleCallEnded(ctx context.Context, v sip_infra.CallEndedPipeline) {
	d.logger.Infow("Pipeline: CallEnded",
		"call_id", v.ID,
		"duration", v.Duration,
		"reason", v.Reason)
}

func (d *Dispatcher) handleCallFailed(ctx context.Context, v sip_infra.CallFailedPipeline) {
	d.logger.Warnw("Pipeline: CallFailed",
		"call_id", v.ID,
		"error", fmt.Sprintf("%v", v.Error),
		"sip_code", v.SIPCode)
}
