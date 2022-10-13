package finalizers

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/interfaces"

	"go.uber.org/zap"
)

// OptimismFinalizer implements the finality check for Optimism.
// Optimism provides a special "rollup_getInfo" API call to determine the latest L2 (Optimism) block to be published on the L1 (Ethereum).
// This finalizer polls that API to determine if a block is finalized.

type FinalizerEntry struct {
	l2Block uint64
	l1Block uint64
}
type OptimismFinalizer struct {
	logger                 *zap.Logger
	connector              connectors.Connector
	l1Finalizer            interfaces.L1Finalizer
	latestFinalizedL2Block uint64

	// finalizerMapping is a array of FinalizerEntry structs with the L2 block number that has been verified and its corresponding L1 block number
	finalizerMapping []FinalizerEntry
}

func NewOptimismFinalizer(ctx context.Context, logger *zap.Logger, connector connectors.Connector, l1Finalizer interfaces.L1Finalizer) *OptimismFinalizer {
	return &OptimismFinalizer{
		logger:                 logger,
		connector:              connector,
		l1Finalizer:            l1Finalizer,
		latestFinalizedL2Block: 0,
		finalizerMapping:       make([]FinalizerEntry, 0),
	}
}

func (f *OptimismFinalizer) IsBlockFinalized(ctx context.Context, block *connectors.NewBlock) (bool, error) {
	finalizedL1Block := f.l1Finalizer.GetLatestFinalizedBlockNumber()
	if finalizedL1Block == 0 {
		// This happens on start up.
		return false, nil
	}

	// Result is the json information coming back from the Optimism node's rollup_getInfo() call
	type Result struct {
		Mode       string
		EthContext struct {
			BlockNumber uint64 `json:"blockNumber"`
			TimeStamp   uint64 `json:"timestamp"`
		} `json:"ethContext"`
		RollupContext struct {
			Index         uint64 `json:"index"`
			VerifiedIndex uint64 `json:"verifiedIndex"`
		} `json:"rollupContext"`
	}

	// Always call into the Optimism node to get the latest rollup information so we don't have to wait
	// any longer than is necessary for finality by skipping rollup info messages
	var info Result
	err := f.connector.RawCallContext(ctx, &info, "rollup_getInfo")
	if err != nil {
		// This is the error case where the RPC call fails
		f.logger.Error("failed to get rollup info", zap.String("eth_network", f.connector.NetworkName()), zap.Error(err))
		return false, err
	}
	if info.RollupContext.VerifiedIndex == 0 {
		// This is the error case where the RPC call is not working as expected.
		return false, fmt.Errorf("Received a verified index of 0.  Please check Optimism RPC parameter.")
	}

	f.logger.Debug("finalizerMapping", zap.Uint64("L2 verified index", info.RollupContext.VerifiedIndex), zap.String(" => ", ""), zap.Uint64("L1_blockNumber", info.EthContext.BlockNumber))
	// Look at the last element of the array and see if we need to add this entry
	// The assumption here is that every subsequent call moves forward (or stays the same).  It is an error if verifiedIndex goes backwards
	finalizerMappingSize := len(f.finalizerMapping)
	if finalizerMappingSize != 0 && f.finalizerMapping[finalizerMappingSize-1].l2Block > info.RollupContext.VerifiedIndex {
		// This is the error case where the RPC call is not working as expected.
		return false, fmt.Errorf("The received verified index just went backwards. Received %d. Last number in array is %d", info.RollupContext.VerifiedIndex, f.finalizerMapping[finalizerMappingSize-1].l2Block)
	}
	if finalizerMappingSize == 0 || f.finalizerMapping[finalizerMappingSize-1].l2Block < info.RollupContext.VerifiedIndex {
		// New information.  Append it to the array.
		f.finalizerMapping = append(f.finalizerMapping, FinalizerEntry{l2Block: info.RollupContext.VerifiedIndex, l1Block: info.EthContext.BlockNumber})
		f.logger.Info("Appending new entry.", zap.Int("finalizerMap size", len(f.finalizerMapping)), zap.Uint64("L2 verified index", info.RollupContext.VerifiedIndex), zap.Uint64("L1_blockNumber", info.EthContext.BlockNumber))
	}

	// Here we want to prune the known finalized entries from the mapping, while recording the latest finalized L2 block number
	pruneIdx := -1
	for idx, entry := range f.finalizerMapping {
		if entry.l1Block > finalizedL1Block {
			break
		}
		// The L1 block for this entry has been finalized so we can prune it.
		f.latestFinalizedL2Block = entry.l2Block
		pruneIdx = idx
	}
	if pruneIdx >= 0 {
		// Do the pruning here
		if pruneIdx+1 >= len(f.finalizerMapping) {
			f.finalizerMapping = nil
		} else {
			f.finalizerMapping = f.finalizerMapping[pruneIdx+1:]
		}
		f.logger.Info("Pruning finalizerMapping", zap.Int("Pruning from index", pruneIdx), zap.Int("new array size", len(f.finalizerMapping)))
	}

	isFinalized := block.Number.Uint64() <= f.latestFinalizedL2Block

	f.logger.Debug("got rollup info", zap.String("eth_network", f.connector.NetworkName()),
		zap.Bool("isFinalized", isFinalized),
		zap.String("mode", info.Mode),
		zap.Uint64("l1_blockNumber", info.EthContext.BlockNumber),
		zap.Uint64("l1_finalizedBlock", finalizedL1Block),
		zap.Uint64("l2_blockNumber", info.RollupContext.Index),
		zap.Uint64("verified_index", info.RollupContext.VerifiedIndex),
		zap.Uint64("latestFinalizedL2Block", f.latestFinalizedL2Block),
		zap.Stringer("desired_block", block.Number),
	)

	return isFinalized, nil
}

/*
curl -X POST --data '{"jsonrpc":"2.0","method":"rollup_getInfo","params":[],"id":1}' https://rpc.ankr.com/optimism_testnet
{
	"jsonrpc":"2.0","id":1,"result":{
		"mode":"verifier",
		"syncing":false,
		"ethContext":{
			"blockNumber":7763392,"timestamp":1665680949 // This is a few blocks behind the latest block on goerli.
			},
		"rollupContext":{
			"index":1952690,"queueIndex":13285,"verifiedIndex":0 // This is a few blocks behind the latest block on optimism.
		}
	}
}
*/
