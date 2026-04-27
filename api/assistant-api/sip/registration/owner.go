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

// ownerKey scopes the Redis distribution key to the DID. The stored value is
// the owning instance's externalIP (each server has a distinct externalIP, so
// it doubles as instance identity).
func ownerKey(did string) string { return OwnerKeyPrefix + did }

// stageClaimOwnership implements "Check if owner is not there -> create owner".
// SETNX claims unowned DIDs; if already owned by us, refresh TTL; if owned by
// a peer, the pipeline stops with errPeerOwned (silently — not an error).
func (m *Manager) stageClaimOwnership(ctx context.Context, rec *Record) error {
	ok, err := m.claimOwner(ctx, rec.DID)
	if err != nil {
		rec.Outcome = OutcomeClaimError
		m.logger.Warnw("Ownership claim failed", "did", rec.DID, "error", err)
		return err
	}
	if !ok {
		rec.Outcome = OutcomePeerOwned
		m.logger.Debugw("DID owned by peer instance — skipping",
			"did", rec.DID, "self", m.externalIP)
		return errPeerOwned
	}
	return nil
}

// claimOwner returns true if this instance owns the DID after the call.
// New claims happen via SETNX; existing self-owned claims have their TTL
// refreshed; peer-owned DIDs return false.
func (m *Manager) claimOwner(ctx context.Context, did string) (bool, error) {
	key := ownerKey(did)

	claimed, err := m.redis.SetNX(ctx, key, m.externalIP, OwnershipTTL).Result()
	if err != nil {
		return false, err
	}
	if claimed {
		m.logger.Debugw("DID ownership claimed",
			"did", did, "owner", m.externalIP, "ttl", OwnershipTTL)
		return true, nil
	}

	cur, err := m.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		// Race: key expired between SETNX and GET. One more attempt.
		again, _ := m.redis.SetNX(ctx, key, m.externalIP, OwnershipTTL).Result()
		return again, nil
	}
	if err != nil {
		return false, err
	}
	if cur == m.externalIP {
		// Already ours — extend the lease.
		m.redis.Expire(ctx, key, OwnershipTTL)
		m.logger.Debugw("DID ownership refreshed",
			"did", did, "owner", m.externalIP, "ttl", OwnershipTTL)
		return true, nil
	}
	return false, nil
}

// releaseOwner deletes the ownership key if (and only if) we still own it.
// Used after Unregister so a peer can claim immediately rather than waiting
// for the TTL to expire.
func (m *Manager) releaseOwner(ctx context.Context, did string) {
	key := ownerKey(did)
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
