// Optimism has a Canonical Transaction Chain (CTC) contract running on the L1.
// This contract contains information about the mapping of L2 => L1 blocks.
// The Finalizer queries that information and places it in an array.
// This allows the finalizer to map an L2 block to an L1 block.
// Then the finalizer can query the L1 chain directly to see if the related L1 block is finalized.

// CTC mainnet contract: 0x5E4e65926BA27467555EB562121fac00D24E9dD2
// CTC testnet contract: 0x607F755149cFEB3a14E1Dc3A4E2450Cde7dfb04D

package finalizers

import (
	"context"
	"fmt"
	"math/big"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	ctcAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/finalizers/optimismctcabi"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"

	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	"go.uber.org/zap"
)

// OptimismFinalizer implements the finality check for Optimism.
// Optimism provides a special "rollup_getInfo" API call to determine the latest L2 (Optimism) block to be published on the L1 (Ethereum).
// This finalizer polls that API to determine if a block is finalized.

type OptimismFinalizer struct {
	logger                 *zap.Logger
	connector              connectors.Connector
	l1Finalizer            interfaces.L1Finalizer
	latestFinalizedL2Block *big.Int

	// finalizerMapping is a array of RollupInfo structs with the L2 block number that has been verified and its corresponding L1 block number
	finalizerMapping []RollupInfo

	// These are used for querying the ctc contract.
	ctcRawClient *ethRpc.Client
	ctcClient    *ethClient.Client

	// This is used to grab the rollup information from the ctc contract
	ctcCaller *ctcAbi.OptimismCtcAbiCaller
}

func NewOptimismFinalizer(
	ctx context.Context,
	logger *zap.Logger,
	connector connectors.Connector,
	l1Finalizer interfaces.L1Finalizer,
	ctcChainUrl string,
	ctcChainAddress string,
) (*OptimismFinalizer, error) {

	ctcRawClient, err := ethRpc.DialContext(ctx, ctcChainUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create raw client for url %s: %w", ctcChainUrl, err)
	}

	ctcClient := ethClient.NewClient(ctcRawClient)

	addr := ethCommon.HexToAddress(ctcChainAddress)

	ctcCaller, err := ctcAbi.NewOptimismCtcAbiCaller(addr, ctcClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create ctc caller for url %s: %w", ctcChainUrl, err)
	}

	finalizer := &OptimismFinalizer{
		logger:                 logger,
		connector:              connector,
		l1Finalizer:            l1Finalizer,
		latestFinalizedL2Block: big.NewInt(0),
		finalizerMapping:       make([]RollupInfo, 0),
		ctcRawClient:           ctcRawClient,
		ctcClient:              ctcClient,
		ctcCaller:              ctcCaller,
	}

	return finalizer, nil
}

// Both types are big.Int because that is what the abi returns.
// However, there are 2 points to note:
// The L1 block number is a uint40 in the abi.  So, safe to convert to Uint64.
// The L2 block number is a uint256 in the abi.  So, this is never converted to anything else.
type RollupInfo struct {
	l2Block *big.Int
	l1Block *big.Int
}

func (f *OptimismFinalizer) GetRollupInfo(ctx context.Context) (RollupInfo, error) {
	// Get the current latest blocks.
	opts := &ethBind.CallOpts{Context: ctx}
	var entry RollupInfo
	l2Block, err := f.ctcCaller.GetTotalElements(opts)
	if err != nil {
		return entry, fmt.Errorf("failed to get L2 block: %w", err)
	}
	l1Block, err := f.ctcCaller.GetLastBlockNumber(opts)
	if err != nil {
		return entry, fmt.Errorf("failed to get L1 block: %w", err)
	}
	entry.l1Block = l1Block
	entry.l2Block = l2Block
	f.logger.Debug("GetRollupInfo", zap.Stringer("l1Block", entry.l1Block), zap.Stringer("l2Block", entry.l2Block))

	return entry, nil
}

func (f *OptimismFinalizer) IsBlockFinalized(ctx context.Context, block *connectors.NewBlock) (bool, error) {
	finalizedL1Block := f.l1Finalizer.GetLatestFinalizedBlockNumber() // Uint64
	if finalizedL1Block == 0 {
		// This happens on start up.
		return false, nil
	}

	// Always call into the Optimism node to get the latest rollup information so we don't have to wait
	// any longer than is necessary for finality by skipping rollup info messages
	rInfo, err := f.GetRollupInfo(ctx)
	if err != nil {
		// This is the error case where the contract call fails
		f.logger.Error("failed to get rollup info", zap.Error(err))
		return false, err
	}

	f.logger.Debug("finalizerMapping", zap.String("L2 block", rInfo.l2Block.String()), zap.String("=> L1 block", rInfo.l1Block.String()))
	// Look at the last element of the array and see if we need to add this entry
	// The assumption here is that every subsequent call moves forward (or stays the same).  It is an error if verifiedIndex goes backwards
	finalizerMappingSize := len(f.finalizerMapping)
	if finalizerMappingSize != 0 && f.finalizerMapping[finalizerMappingSize-1].l2Block.Cmp(rInfo.l2Block) > 0 {
		// This is the error case where the RPC call is not working as expected.
		return false, fmt.Errorf("The received verified index just went backwards. Received %s. Last number in array is %s", rInfo.l2Block.String(), f.finalizerMapping[finalizerMappingSize-1].l2Block.String())
	}
	if finalizerMappingSize == 0 || f.finalizerMapping[finalizerMappingSize-1].l2Block.Cmp(rInfo.l2Block) < 0 {
		// New information.  Append it to the array.
		f.finalizerMapping = append(f.finalizerMapping, rInfo)
		f.logger.Info("Appending new entry.", zap.Int("finalizerMap size", len(f.finalizerMapping)), zap.String("L2 block number", rInfo.l2Block.String()), zap.String("L1 block number", rInfo.l1Block.String()))
	}

	// Here we want to prune the known finalized entries from the mapping, while recording the latest finalized L2 block number
	pruneIdx := -1
	for idx, entry := range f.finalizerMapping {
		if entry.l1Block.Uint64() > finalizedL1Block {
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
		f.logger.Debug("Pruning finalizerMapping", zap.Int("Pruning from index", pruneIdx), zap.Int("new array size", len(f.finalizerMapping)))
	}

	isFinalized := block.Number.Cmp(f.latestFinalizedL2Block) <= 0

	f.logger.Debug("got rollup info",
		zap.Uint64("l1_blockNumber", rInfo.l1Block.Uint64()),
		zap.Uint64("l1_finalizedBlock", finalizedL1Block),
		zap.String("l2_blockNumber", rInfo.l2Block.String()),
		zap.String("latestFinalizedL2Block", f.latestFinalizedL2Block.String()),
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
