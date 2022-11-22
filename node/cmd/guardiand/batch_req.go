package guardiand

import (
	"context"

	"github.com/benbjohnson/clock"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Multiplex batch requests to the appropriate chain
func handleBatchRequests(
	ctx context.Context,
	clock clock.Clock,
	logger *zap.Logger,
	batchReqC <-chan *common.TransactionQuery,
	chainBatchReqC map[vaa.ChainID]chan *common.TransactionQuery,
) {

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-batchReqC:
			if channel, ok := chainBatchReqC[req.EmitterChain]; ok {
				channel <- req
			} else {
				logger.Warn("batch query request received for unsupported chain ID",
					zap.Uint16("chain_id", uint16(req.EmitterChain)),
					zap.Stringer("tx_id", req.TransactionID))
			}
		}
	}
}

// for when the batchVAA feature flag is disabled.
func disregardBatchRequests(
	ctx context.Context,
	batchReqC <-chan *common.TransactionQuery,
	chainBatchReqC map[vaa.ChainID]chan *common.TransactionQuery,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-batchReqC:
		}
	}
}
