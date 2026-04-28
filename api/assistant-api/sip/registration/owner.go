// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// handleClaimOwnership implements "Check if owner is not there -> create
// owner". On a fresh claim or self-owned refresh returns RegisterPipeline; on
// peer-owned or claim error returns nil to stop the chain. Mirrors the
// per-type handler signature of sip/pipeline/registration.go.
func (m *Manager) handleClaimOwnership(ctx context.Context, s ClaimOwnershipPipeline) Pipeline {
	rec := s.Record
	key := OwnerKeyPrefix + rec.DID

	claimed, err := m.redis.SetNX(ctx, key, m.externalIP, OwnershipTTL).Result()
	if err != nil {
		rec.Outcome = OutcomeClaimError
		m.logger.Warnw("Ownership claim failed", "did", rec.DID, "error", err)
		return nil
	}
	if claimed {
		m.logger.Debugw("DID ownership claimed",
			"did", rec.DID, "owner", m.externalIP, "ttl", OwnershipTTL)
		return RegisterPipeline{Record: rec}
	}

	cur, err := m.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		// Race: key expired between SETNX and GET. One more attempt.
		again, _ := m.redis.SetNX(ctx, key, m.externalIP, OwnershipTTL).Result()
		if again {
			m.logger.Debugw("DID ownership claimed (post-race)",
				"did", rec.DID, "owner", m.externalIP)
			return RegisterPipeline{Record: rec}
		}
		rec.Outcome = OutcomePeerOwned
		return nil
	}
	if err != nil {
		rec.Outcome = OutcomeClaimError
		m.logger.Warnw("Ownership claim failed", "did", rec.DID, "error", err)
		return nil
	}
	if cur == m.externalIP {
		// Already ours — extend the lease.
		m.redis.Expire(ctx, key, OwnershipTTL)
		m.logger.Debugw("DID ownership refreshed",
			"did", rec.DID, "owner", m.externalIP, "ttl", OwnershipTTL)
		return RegisterPipeline{Record: rec}
	}
	rec.Outcome = OutcomePeerOwned
	m.logger.Debugw("DID owned by peer instance — skipping",
		"did", rec.DID, "owner", cur, "self", m.externalIP)
	return nil
}

// releaseOwner deletes the ownership key if (and only if) we still own it.
// Used by the reconcile cleanup branch and by ReleaseAll on graceful
// shutdown so a peer can claim immediately rather than waiting for the TTL.
func (m *Manager) releaseOwner(ctx context.Context, did string) {
	key := OwnerKeyPrefix + did
	cur, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		return
	}
	if cur != m.externalIP {
		return
	}
	m.redis.Del(ctx, key)
	m.logger.Debugw("DID ownership released", "did", did, "owner", m.externalIP)
}
