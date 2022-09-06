package guardiand

import (
	"context"

	"github.com/benbjohnson/clock"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"go.uber.org/zap"
)

// Multiplex batch requests to the appropriate chain
func handleBatchRequests(
	ctx context.Context,
	clock clock.Clock,
	logger *zap.Logger,
	batchReqC <-chan *common.BatchMessageID,
	chainBatchReqC map[vaa.ChainID]chan *common.BatchMessageID,
) {

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-batchReqC:
			if channel, ok := chainBatchReqC[req.EmitterChain]; ok {
				channel <- req
			} else {
				logger.Error("unknown chain ID for batch request",
					zap.Uint16("chain_id", uint16(req.EmitterChain)),
					zap.String("tx_hash", req.TransactionID.Hex()))
			}
		}
	}
}
