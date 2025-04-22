package node

import (
	"context"
	"encoding/hex"
	"math"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const (
	CachePurgeInterval = 7 * time.Minute
	CacheFor           = 11 * time.Minute
)

type Clock interface {
	Now() time.Time
	NewTicker(d time.Duration) *time.Ticker
}

// CacheClock is a wrapper for some of the standard library's time functions.
// It exists to allow for dependency injection while testing [handleReobservationRequests].
type CacheClock struct{}

func (c *CacheClock) Now() time.Time {
	return time.Now()
}
func (c *CacheClock) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

// Multiplex observation requests to the appropriate chain
func handleReobservationRequests(
	ctx context.Context,
	clock Clock,
	logger *zap.Logger,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	chainObsvReqC map[vaa.ChainID]chan *gossipv1.ObservationRequest,
) {
	// Due to the automatic re-observation requests sent out by the processor we may end
	// up getting multiple requests to re-observe the same tx. Keep a cache of the
	// requests received in the last 11 minutes so that we don't end up repeatedly
	// re-observing the same transactions.
	type cachedRequest struct {
		chainId vaa.ChainID
		txHash  string
	}

	cache := make(map[cachedRequest]time.Time)
	ticker := clock.NewTicker(CachePurgeInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := clock.Now()
			for r, t := range cache {
				if now.Sub(t) > CacheFor {
					delete(cache, r)
				}
			}
		case req := <-obsvReqC:
			if req.ChainId > math.MaxUint16 {
				logger.Error("chain id is larger than MaxUint16",
					zap.Uint32("chain_id", req.ChainId),
				)
				continue
			}

			r := cachedRequest{
				chainId: vaa.ChainID(req.ChainId),
				txHash:  hex.EncodeToString(req.TxHash),
			}

			if _, ok := cache[r]; ok {
				// We've recently seen a re-observation request for this tx
				// so skip this one.
				logger.Info("skipping duplicate re-observation request",
					zap.Stringer("chain", r.chainId),
					zap.String("tx_hash", r.txHash),
				)
				continue
			}

			if channel, ok := chainObsvReqC[r.chainId]; ok {
				select {
				case channel <- req:
					cache[r] = clock.Now()

				default:
					logger.Warn("failed to send reobservation request to watcher",
						zap.Stringer("chain_id", r.chainId),
						zap.String("tx_hash", r.txHash))
				}
			} else {
				logger.Error("unknown chain ID for reobservation request",
					zap.Uint16("chain_id", uint16(r.chainId)),
					zap.String("tx_hash", r.txHash))
			}
		}
	}
}
