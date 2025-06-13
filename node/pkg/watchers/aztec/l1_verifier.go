package aztec

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"
)

// L1Verifier defines the interface for verifying finality
type L1Verifier interface {
	// Include interfaces.L1Finalizer as an embedded interface
	interfaces.L1Finalizer

	// Get the latest finalized block from Aztec
	GetFinalizedBlock(ctx context.Context) (*FinalizedBlock, error)

	// Check if a block is finalized
	IsBlockFinalized(ctx context.Context, blockNumber int) (bool, error)
}

// aztecFinalityVerifier is a simplified L1Verifier that queries Aztec directly
type aztecFinalityVerifier struct {
	rpcClient *rpc.Client
	logger    *zap.Logger

	// Cache for finalized blocks
	finalizedBlockCache     *FinalizedBlock
	finalizedBlockCacheTime time.Time
	finalizedBlockCacheMu   sync.RWMutex
	finalizedBlockCacheTTL  time.Duration
}

// NewAztecFinalityVerifier creates a new L1 verifier
func NewAztecFinalityVerifier(
	rpcURL string,
	logger *zap.Logger,
) (L1Verifier, error) {
	// Create a new RPC client
	client, err := rpc.DialContext(context.Background(), rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %v", err)
	}

	return &aztecFinalityVerifier{
		rpcClient:              client,
		logger:                 logger,
		finalizedBlockCacheTTL: 30 * time.Second,
	}, nil
}

// GetLatestFinalizedBlockNumber implements the interfaces.L1Finalizer interface
func (v *aztecFinalityVerifier) GetLatestFinalizedBlockNumber() uint64 {
	// Check the cache first
	if block, found := v.getFromCache(); found {
		return uint64(block.Number)
	}

	// If no cache, fetch the latest finalized block
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	block, err := v.GetFinalizedBlock(ctx)
	if err != nil {
		v.logger.Warn("Failed to get finalized block for L1Finalizer", zap.Error(err))
		return 0
	}

	return uint64(block.Number)
}

// Helper function to safely get from the finalized block cache
func (v *aztecFinalityVerifier) getFromCache() (*FinalizedBlock, bool) {
	v.finalizedBlockCacheMu.RLock()
	defer v.finalizedBlockCacheMu.RUnlock()

	if v.finalizedBlockCache != nil && time.Since(v.finalizedBlockCacheTime) < v.finalizedBlockCacheTTL {
		return v.finalizedBlockCache, true
	}

	return nil, false // Cache miss or expired
}

// Helper function to safely update the finalized block cache
func (v *aztecFinalityVerifier) updateCache(block *FinalizedBlock) {
	v.finalizedBlockCacheMu.Lock()
	defer v.finalizedBlockCacheMu.Unlock()

	v.finalizedBlockCache = block
	v.finalizedBlockCacheTime = time.Now()
}

// GetFinalizedBlock gets the latest finalized block from Aztec
func (v *aztecFinalityVerifier) GetFinalizedBlock(ctx context.Context) (*FinalizedBlock, error) {
	// Check cache first
	if block, found := v.getFromCache(); found {
		v.logger.Debug("Using cached finalized block",
			zap.Int("number", block.Number),
			zap.String("hash", block.Hash))
		return block, nil
	}

	// Cache miss, fetch from network
	v.logger.Debug("Fetching L2 tips")
	var l2Tips L2Tips
	err := v.rpcClient.CallContext(ctx, &l2Tips, "node_getL2Tips")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 tips: %v", err)
	}

	// Create finalized block info
	block := &FinalizedBlock{
		Number: l2Tips.Finalized.Number,
		Hash:   l2Tips.Finalized.Hash,
	}

	// Update the cache
	v.updateCache(block)

	v.logger.Info("Updated finalized block",
		zap.Int("number", block.Number))

	return block, nil
}

// IsBlockFinalized checks if a specific block number is finalized
func (v *aztecFinalityVerifier) IsBlockFinalized(ctx context.Context, blockNumber int) (bool, error) {
	finalizedBlock, err := v.GetFinalizedBlock(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get finalized block: %v", err)
	}

	isFinalized := blockNumber <= finalizedBlock.Number
	v.logger.Debug("Block finality check",
		zap.Int("block", blockNumber),
		zap.Int("finalized_block", finalizedBlock.Number),
		zap.Bool("is_finalized", isFinalized))

	return isFinalized, nil
}
