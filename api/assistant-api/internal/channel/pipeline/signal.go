// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"fmt"
)

func (d *Dispatcher) handleDisconnectRequested(ctx context.Context, v DisconnectRequestedPipeline) {
	d.logger.Infow("Pipeline: DisconnectRequested", "call_id", v.ID, "reason", v.Reason)
}

func (d *Dispatcher) handleCallCompleted(ctx context.Context, v CallCompletedPipeline) {
	d.logger.Infow("Pipeline: CallCompleted",
		"call_id", v.ID,
		"duration", v.Duration,
		"messages", v.Messages,
		"reason", v.Reason)
}

func (d *Dispatcher) handleCallFailed(ctx context.Context, v CallFailedPipeline) {
	d.logger.Warnw("Pipeline: CallFailed",
		"call_id", v.ID,
		"stage", v.Stage,
		"error", fmt.Sprintf("%v", v.Error))
}
