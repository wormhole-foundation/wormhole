package manager

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/manager/delegatedmanagersetabi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// ManagerSetReader reads manager sets from the DelegatedManagerSet contract on Ethereum.
// It provides thread-safe caching of manager sets to minimize RPC calls.
type ManagerSetReader struct {
	mu           sync.RWMutex
	logger       *zap.Logger
	rpcURL       string
	contractAddr string
	env          common.Environment

	// cache stores fetched manager sets: chainId -> index -> ManagerSetConfig
	cache map[vaa.ChainID]map[uint32]*ManagerSetConfig
}

// NewManagerSetReader creates a new reader using the provided RPC URL.
// The RPC URL should be from the Ethereum watcher config.
func NewManagerSetReader(
	logger *zap.Logger,
	env common.Environment,
	rpcURL string,
) (*ManagerSetReader, error) {
	contractAddr, ok := DelegatedManagerSetContracts[env]
	if !ok {
		return nil, fmt.Errorf("no DelegatedManagerSet contract configured for environment %s", env)
	}

	logger.Info("initialized manager set reader",
		zap.String("contract", contractAddr),
		zap.String("rpc", rpcURL),
		zap.String("env", string(env)),
	)

	return &ManagerSetReader{
		logger:       logger.With(zap.String("component", "manager-reader")),
		rpcURL:       rpcURL,
		contractAddr: contractAddr,
		env:          env,
		cache:        make(map[vaa.ChainID]map[uint32]*ManagerSetConfig),
	}, nil
}

// GetManagerSet retrieves a manager set by chain ID and index.
// It first checks the cache; on cache miss, it fetches from the contract and caches the result.
// The signer parameter is used to determine if this node is part of the manager set.
func (r *ManagerSetReader) GetManagerSet(
	ctx context.Context,
	chainID vaa.ChainID,
	index uint32,
	signer guardiansigner.GuardianSigner,
) (*ManagerSetConfig, error) {
	// Check cache first
	r.mu.RLock()
	if chainSets, ok := r.cache[chainID]; ok {
		if set, ok := chainSets[index]; ok {
			r.mu.RUnlock()
			return set, nil
		}
	}
	r.mu.RUnlock()

	// Cache miss - fetch from contract
	r.logger.Debug("cache miss, fetching manager set from contract",
		zap.Stringer("chain", chainID),
		zap.Uint32("index", index),
	)

	// Create a fresh connection for this call
	client, err := ethclient.DialContext(ctx, r.rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC %s: %w", r.rpcURL, err)
	}
	defer client.Close()

	caller, err := delegatedmanagersetabi.NewDelegatedManagerSetCaller(
		ethCommon.HexToAddress(r.contractAddr),
		client,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DelegatedManagerSet caller: %w", err)
	}

	// #nosec G115 -- ChainID is uint16
	data, err := caller.GetManagerSet(nil, uint16(chainID), index)
	if err != nil {
		return nil, fmt.Errorf("failed to call getManagerSet(%d, %d): %w", chainID, index, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("manager set not found for chain %s index %d", chainID, index)
	}

	// Parse the manager set bytes
	set, err := r.parseManagerSetBytes(ctx, data, index, signer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manager set bytes: %w", err)
	}

	// Cache the result
	r.mu.Lock()
	if _, ok := r.cache[chainID]; !ok {
		r.cache[chainID] = make(map[uint32]*ManagerSetConfig)
	}
	r.cache[chainID][index] = set
	r.mu.Unlock()

	r.logger.Info("fetched manager set from contract",
		zap.Stringer("chain", chainID),
		zap.Uint32("index", index),
		zap.Uint8("m", set.M),
		zap.Uint8("n", set.N),
		zap.Bool("is_signer", set.IsSigner),
	)

	return set, nil
}

// parseManagerSetBytes parses the raw bytes from the contract into a ManagerSetConfig.
// Format: Type (1 byte) | M (1 byte) | N (1 byte) | PublicKeys (N * 33 bytes)
func (r *ManagerSetReader) parseManagerSetBytes(
	ctx context.Context,
	data []byte,
	index uint32,
	signer guardiansigner.GuardianSigner,
) (*ManagerSetConfig, error) {
	var set vaa.Secp256k1MultisigManagerSet
	if err := set.Deserialize(data); err != nil {
		return nil, fmt.Errorf("failed to deserialize manager set: %w", err)
	}

	// Convert [33]byte arrays to []byte slices
	pubKeys := make([][]byte, set.N)
	for i, pk := range set.PublicKeys {
		pubKeys[i] = pk[:]
	}

	// Determine if this node is a signer
	var isSigner bool
	var signerIndex uint8
	if signer != nil && ctx != nil {
		signerPubKey := signer.PublicKey(ctx)
		signerCompressed := compressPublicKey(&signerPubKey)
		for i, pk := range pubKeys {
			if bytes.Equal(signerCompressed, pk) {
				isSigner = true
				signerIndex = uint8(i) // #nosec G115 -- i < n which is uint8
				break
			}
		}
	}

	return &ManagerSetConfig{
		Index:       index,
		M:           set.M,
		N:           set.N,
		PublicKeys:  pubKeys,
		IsSigner:    isSigner,
		SignerIndex: signerIndex,
	}, nil
}
