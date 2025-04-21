package querystaking

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/query/queryratelimit"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// QueryType represents the different types of queries supported
type QueryType uint8

// Query type constants
const (
	EthCallQueryRequestType             QueryType = 1
	EthCallByTimestampQueryRequestType  QueryType = 2
	EthCallWithFinalityQueryRequestType QueryType = 3
	SolanaAccountQueryRequestType       QueryType = 4
	SolanaPdaQueryRequestType           QueryType = 5
)

// Metrics specific to staking module
var (
	stakingPolicyFetches = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_staking_policy_fetches_total",
			Help: "Total number of staking policy fetches by result",
		}, []string{"result", "pool"})

	stakingQueryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ccq_staking_query_duration_seconds",
			Help:    "Staking contract query latency by pool and operation",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}, []string{"pool", "operation"})

	stakingPolicyDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_staking_policy_decisions_total",
			Help: "Policy decisions made by staking provider by outcome and tier",
		}, []string{"outcome", "tier", "query_type"})

	stakingPoolErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_staking_pool_errors_total",
			Help: "Errors when querying staking pools by pool and error type",
		}, []string{"pool", "error_type"})
)

// QueryTypePool defines a query type pool that this node supports
// Rate limiting configuration (tranches/rates) is determined by the conversion table
// stored on-chain, not hardcoded here.
type QueryTypePool struct {
	QueryTypes []QueryType // Query types included in this pool
}

// queryTypeBits generates the bit field from the QueryTypes slice
// Note: This is only used for factory discovery. For direct pool configuration, use --stakingPoolAddresses instead.
func (p QueryTypePool) queryTypeBits() [32]byte {
	var bits [32]byte
	for _, qt := range p.QueryTypes {
		if qt == 0 {
			continue
		}
		qtUint8 := uint8(qt)
		byteIndex := 31 - (qtUint8-1)/8
		bitOffset := (qtUint8 - 1) % 8
		bits[byteIndex] |= 1 << bitOffset
	}
	return bits
}

// SupportedQueryPools defines the query type pools that this node supports
// Pools are discovered via the factory contract using the query type bits.
// Rate limiting (tranches/rates) is determined by each pool's conversion table.
var SupportedQueryPools = map[string]QueryTypePool{
	"evm": {
		QueryTypes: []QueryType{
			EthCallQueryRequestType,
			EthCallByTimestampQueryRequestType,
			EthCallWithFinalityQueryRequestType,
		},
	},
	"solana": {
		QueryTypes: []QueryType{
			SolanaAccountQueryRequestType,
			SolanaPdaQueryRequestType,
		},
	},
}

// QueryTypeToChain maps query types to their chain name for IPFS data lookup
// This mapping is used to extract chain-specific rates from conversion tables
var QueryTypeToChain = map[QueryType]string{
	EthCallQueryRequestType:             "EVM",
	EthCallByTimestampQueryRequestType:  "EVM",
	EthCallWithFinalityQueryRequestType: "EVM",
	SolanaAccountQueryRequestType:       "Solana",
	SolanaPdaQueryRequestType:           "Solana",
}

// getChainName returns the chain name for a query type
func getChainName(qt QueryType) (string, error) {
	chainName, ok := QueryTypeToChain[qt]
	if !ok {
		return "", fmt.Errorf("unknown query type: %d", qt)
	}
	return chainName, nil
}

// chainNameToQueryTypes returns the query types for a given chain name
func chainNameToQueryTypes(chainName string) []QueryType {
	var queryTypes []QueryType
	for qt, chain := range QueryTypeToChain {
		if chain == chainName {
			queryTypes = append(queryTypes, qt)
		}
	}
	return queryTypes
}

// PoolMetadata holds immutable pool data that can be safely cached.
// These values are set at pool deployment and never change.
type PoolMetadata struct {
	StakingTokenAddress common.Address
	TokenDecimals       uint8
}

// StakingClient wraps ethereum client for staking contract interactions
type StakingClient struct {
	client              *ethclient.Client
	logger              *zap.Logger
	factoryAddress      common.Address
	configuredPools     []common.Address // Direct pool addresses from config
	useDirectPoolConfig bool             // If true, use configuredPools instead of factory discovery
	ipfsClient          *IPFSClient
	cacheDuration       time.Duration // Duration for which policy results are cached

	// Single mutex protects all caches (accessed sequentially in FetchStakingPolicy)
	cacheMutex sync.RWMutex

	// Caches for immutable contract data
	conversionHistoryCache map[common.Address][][32]byte    // Pool -> CID array
	poolMetadataCache      map[common.Address]*PoolMetadata // Pool -> token metadata
	factoryPoolCache       map[[32]byte]common.Address      // QueryTypeBits -> pool address
}

// NewStakingClient creates a new staking client
func NewStakingClient(client *ethclient.Client, logger *zap.Logger, factoryAddress common.Address, poolAddresses []common.Address, ipfsClient *IPFSClient, cacheDuration time.Duration) *StakingClient {
	useDirectConfig := len(poolAddresses) > 0
	if useDirectConfig {
		logger.Info("Staking client configured with direct pool addresses",
			zap.Int("poolCount", len(poolAddresses)))
	} else {
		logger.Info("Staking client configured for factory-based discovery",
			zap.String("factoryAddress", factoryAddress.Hex()))
	}

	return &StakingClient{
		client:                 client,
		logger:                 logger.With(zap.String("component", "staking-client")),
		factoryAddress:         factoryAddress,
		configuredPools:        poolAddresses,
		useDirectPoolConfig:    useDirectConfig,
		ipfsClient:             ipfsClient,
		cacheDuration:          cacheDuration,
		conversionHistoryCache: make(map[common.Address][][32]byte),
		poolMetadataCache:      make(map[common.Address]*PoolMetadata),
		factoryPoolCache:       make(map[[32]byte]common.Address),
	}
}

// GetStakeInfo queries a pool for a staker's information with comprehensive error handling
func (sc *StakingClient) GetStakeInfo(ctx context.Context, poolAddress, stakerAddress common.Address, poolName string) (*StakeInfo, error) {
	start := time.Now()

	sc.logger.Debug("querying stake info",
		zap.String("pool", poolName),
		zap.String("poolAddress", poolAddress.Hex()),
		zap.String("staker", stakerAddress.Hex()))

	callData := PackStakesCall(stakerAddress)

	// Measure RPC call latency
	rpcStart := time.Now()
	result, err := sc.client.CallContract(ctx, ethereum.CallMsg{
		To:   &poolAddress,
		Data: callData,
	}, nil)

	stakingQueryLatency.WithLabelValues(poolName, "rpc_call").Observe(time.Since(rpcStart).Seconds())

	if err != nil {
		stakingPoolErrors.WithLabelValues(poolName, "rpc_error").Inc()

		sc.logger.Error("failed to call staking contract",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()),
			zap.String("staker", stakerAddress.Hex()),
			zap.Error(err))

		return nil, fmt.Errorf("failed to call getStakeInfo on pool %s: %w", poolName, err)
	}

	// Measure parsing latency
	parseStart := time.Now()
	stakeInfo, err := ParseStakeInfo(result)
	stakingQueryLatency.WithLabelValues(poolName, "parse_result").Observe(time.Since(parseStart).Seconds())

	if err != nil {
		stakingPoolErrors.WithLabelValues(poolName, "parse_error").Inc()
		sc.logger.Error("failed to parse stake info",
			zap.String("pool", poolName),
			zap.String("staker", stakerAddress.Hex()),
			zap.Int("resultLength", len(result)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse stake info from pool %s: %w", poolName, err)
	}

	// Log successful query with stake details
	sc.logger.Debug("successfully queried stake info",
		zap.String("pool", poolName),
		zap.String("staker", stakerAddress.Hex()),
		zap.String("stakeAmount", stakeInfo.Amount.String()),
		zap.Uint64("lockupEnd", stakeInfo.LockupEnd),
		zap.Uint64("accessEnd", stakeInfo.AccessEnd),
		zap.Duration("totalLatency", time.Since(start)))

	return stakeInfo, nil
}

// GetSignerAddress queries the stakerSigners mapping to find the designated signer for a staker
func (sc *StakingClient) GetSignerAddress(ctx context.Context, poolAddress, stakerAddress common.Address, poolName string) (common.Address, error) {
	callData := PackStakerSignersCall(stakerAddress)

	result, err := sc.client.CallContract(ctx, ethereum.CallMsg{
		To:   &poolAddress,
		Data: callData,
	}, nil)
	if err != nil {
		sc.logger.Debug("failed to call stakerSigners",
			zap.String("pool", poolName),
			zap.String("staker", stakerAddress.Hex()),
			zap.Error(err))
		return common.Address{}, fmt.Errorf("failed to call stakerSigners on pool %s: %w", poolName, err)
	}

	signerAddress, err := ParseAddress(result)
	if err != nil {
		sc.logger.Error("failed to parse signer address",
			zap.String("pool", poolName),
			zap.String("staker", stakerAddress.Hex()),
			zap.Error(err))
		return common.Address{}, fmt.Errorf("failed to parse signer address from pool %s: %w", poolName, err)
	}

	return signerAddress, nil
}

// IsSignerAuthorizedForStaker checks if signerAddress is the designated signer for stakerAddress
// by querying the stakerSigners forward mapping.
func (sc *StakingClient) IsSignerAuthorizedForStaker(ctx context.Context, poolAddress, signerAddress, stakerAddress common.Address, poolName string) (bool, error) {
	// Query the forward mapping: stakerSigners[staker] => address
	registeredSigner, err := sc.GetSignerAddress(ctx, poolAddress, stakerAddress, poolName)
	if err != nil {
		sc.logger.Debug("failed to get registered signer",
			zap.String("pool", poolName),
			zap.String("staker", stakerAddress.Hex()),
			zap.Error(err))
		return false, fmt.Errorf("failed to get registered signer for staker %s: %w", stakerAddress.Hex(), err)
	}

	// The signer is authorized if it matches the registered signer AND is not zero address
	isAuthorized := registeredSigner != (common.Address{}) && registeredSigner == signerAddress

	if isAuthorized {
		sc.logger.Debug("signer is authorized via stakerSigners mapping",
			zap.String("pool", poolName),
			zap.String("staker", stakerAddress.Hex()),
			zap.String("signer", signerAddress.Hex()))
	} else {
		// Log details about why authorization failed
		if registeredSigner == (common.Address{}) {
			sc.logger.Debug("staker has not delegated to any signer (zero address registered)",
				zap.String("pool", poolName),
				zap.String("staker", stakerAddress.Hex()),
				zap.String("attemptedSigner", signerAddress.Hex()))
		} else {
			sc.logger.Debug("signer does not match registered signer",
				zap.String("pool", poolName),
				zap.String("staker", stakerAddress.Hex()),
				zap.String("attemptedSigner", signerAddress.Hex()),
				zap.String("registeredSigner", registeredSigner.Hex()))
		}
	}

	return isAuthorized, nil
}

// VerifySignerAuthorization verifies that signerAddr is authorized to act on behalf of stakerAddr
// Returns nil if authorized, error otherwise
// This function supports both self-staking (signer == staker) and delegated signing.
func (sc *StakingClient) VerifySignerAuthorization(ctx context.Context, poolAddress, stakerAddr, signerAddr common.Address, poolName string) error {
	if stakerAddr == signerAddr {
		sc.logger.Debug("signer is staker (self-staking)",
			zap.String("pool", poolName),
			zap.String("address", stakerAddr.Hex()))
		return nil
	}

	// Check if the registered signer for the staker matches the provided signer
	isAuthorized, err := sc.IsSignerAuthorizedForStaker(ctx, poolAddress, signerAddr, stakerAddr, poolName)
	if err != nil {
		return fmt.Errorf("failed to check signer authorization for staker %s: %w", stakerAddr.Hex(), err)
	}

	if !isAuthorized {
		return fmt.Errorf("signer %s is not authorized to act on behalf of staker %s",
			signerAddr.Hex(), stakerAddr.Hex())
	}

	sc.logger.Debug("verified delegated signer authorization via stakerSigners mapping",
		zap.String("pool", poolName),
		zap.String("staker", stakerAddr.Hex()),
		zap.String("signer", signerAddr.Hex()))

	return nil
}

// IsBlocklisted checks if an address is blocklisted for a specific pool
func (sc *StakingClient) IsBlocklisted(ctx context.Context, poolAddress, userAddress common.Address, poolName string) (bool, error) {
	callData := PackIsBlocklistedCall(userAddress)

	result, err := sc.client.CallContract(ctx, ethereum.CallMsg{
		To:   &poolAddress,
		Data: callData,
	}, nil)
	if err != nil {
		sc.logger.Debug("failed to call isBlocklisted",
			zap.String("pool", poolName),
			zap.String("user", userAddress.Hex()),
			zap.Error(err))
		return false, fmt.Errorf("failed to call isBlocklisted on pool %s: %w", poolName, err)
	}

	isBlocked, err := ParseBoolResult(result)
	if err != nil {
		sc.logger.Error("failed to parse blocklist result",
			zap.String("pool", poolName),
			zap.String("user", userAddress.Hex()),
			zap.Error(err))
		return false, fmt.Errorf("failed to parse blocklist result from pool %s: %w", poolName, err)
	}

	return isBlocked, nil
}

// GetPoolMetadata fetches and caches immutable pool metadata (staking token address and decimals).
// This eliminates repeated contract calls for data that never changes.
func (sc *StakingClient) GetPoolMetadata(ctx context.Context, poolAddress common.Address, poolName string) (*PoolMetadata, error) {
	// Check cache first (with read lock)
	sc.cacheMutex.RLock()
	cached, exists := sc.poolMetadataCache[poolAddress]
	sc.cacheMutex.RUnlock()

	if exists {
		sc.logger.Debug("using cached pool metadata",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()),
			zap.Uint8("decimals", cached.TokenDecimals))
		return cached, nil
	}

	// Not in cache, acquire write lock and fetch
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	// Double-check cache in case another goroutine filled it while we waited
	if cached, exists := sc.poolMetadataCache[poolAddress]; exists {
		return cached, nil
	}

	sc.logger.Debug("fetching pool metadata from contract",
		zap.String("pool", poolName),
		zap.String("poolAddress", poolAddress.Hex()))

	// Get STAKING_TOKEN address from the pool
	tokenResult, err := sc.client.CallContract(ctx, ethereum.CallMsg{
		To:   &poolAddress,
		Data: PackStakingTokenCall(),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get STAKING_TOKEN from pool %s: %w", poolName, err)
	}

	tokenAddress, err := ParseAddress(tokenResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse STAKING_TOKEN address from pool %s: %w", poolName, err)
	}

	// Get decimals from the token contract
	decimalsResult, err := sc.client.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddress,
		Data: PackDecimalsCall(),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals from token %s: %w", tokenAddress.Hex(), err)
	}

	decimals, err := ParseUint8Result(decimalsResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decimals from token %s: %w", tokenAddress.Hex(), err)
	}

	// Store in cache
	metadata := &PoolMetadata{
		StakingTokenAddress: tokenAddress,
		TokenDecimals:       decimals,
	}
	sc.poolMetadataCache[poolAddress] = metadata

	sc.logger.Info("cached pool metadata",
		zap.String("pool", poolName),
		zap.String("poolAddress", poolAddress.Hex()),
		zap.String("tokenAddress", tokenAddress.Hex()),
		zap.Uint8("decimals", decimals))

	return metadata, nil
}

// DiscoverPoolFromFactory queries the factory contract to find the pool address for a query type
func (sc *StakingClient) DiscoverPoolFromFactory(ctx context.Context, factoryAddress common.Address, queryType [32]byte) (common.Address, error) {
	if factoryAddress == (common.Address{}) {
		return common.Address{}, fmt.Errorf("factory address not configured")
	}

	callData := PackQueryTypePoolsCall(queryType)

	result, err := sc.client.CallContract(ctx, ethereum.CallMsg{
		To:   &factoryAddress,
		Data: callData,
	}, nil)
	if err != nil {
		sc.logger.Debug("failed to call queryTypePools on factory",
			zap.String("factory", factoryAddress.Hex()),
			zap.String("queryType", fmt.Sprintf("%x", queryType)),
			zap.Error(err))
		return common.Address{}, fmt.Errorf("failed to call queryTypePools on factory: %w", err)
	}

	poolAddress, err := ParseAddress(result) // Reuse the address parsing logic
	if err != nil {
		sc.logger.Error("failed to parse pool address from factory",
			zap.String("factory", factoryAddress.Hex()),
			zap.String("queryType", fmt.Sprintf("%x", queryType)),
			zap.Error(err))
		return common.Address{}, fmt.Errorf("failed to parse pool address from factory: %w", err)
	}

	return poolAddress, nil
}

// GetCachedPoolAddress fetches and caches pool addresses from the factory contract.
// Pool addresses for a given query type are immutable once deployed.
func (sc *StakingClient) GetCachedPoolAddress(ctx context.Context, queryTypeBits [32]byte, poolName string) (common.Address, error) {
	// Check cache first (with read lock)
	sc.cacheMutex.RLock()
	cached, exists := sc.factoryPoolCache[queryTypeBits]
	sc.cacheMutex.RUnlock()

	if exists {
		sc.logger.Debug("using cached pool address",
			zap.String("pool", poolName),
			zap.String("poolAddress", cached.Hex()))
		return cached, nil
	}

	// Not in cache, acquire write lock and fetch
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	// Double-check cache in case another goroutine filled it while we waited
	if cached, exists := sc.factoryPoolCache[queryTypeBits]; exists {
		return cached, nil
	}

	sc.logger.Debug("fetching pool address from factory",
		zap.String("pool", poolName),
		zap.String("queryTypeBits", fmt.Sprintf("%x", queryTypeBits)))

	// Use existing method to discover pool
	poolAddress, err := sc.DiscoverPoolFromFactory(ctx, sc.factoryAddress, queryTypeBits)
	if err != nil {
		return common.Address{}, err
	}

	// Only cache actual pool addresses, not zero addresses (non-existent pools).
	// This allows newly deployed pools to be discovered when the policy cache refreshes,
	// without requiring CCQ server restart.
	if poolAddress != (common.Address{}) {
		sc.factoryPoolCache[queryTypeBits] = poolAddress
		sc.logger.Info("cached pool address",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()))
	} else {
		sc.logger.Debug("no pool deployed for query type (not caching to allow future discovery)",
			zap.String("pool", poolName))
	}

	return poolAddress, nil
}

// getConversionTableHistory fetches and caches the full conversion table history for a pool.
// This eliminates repeated contract calls for individual entries.
// The cache is populated lazily on first access and is thread-safe.
// If new entries are added to the conversion table on-chain, they will be fetched and appended to the cache.
func (sc *StakingClient) getConversionTableHistory(ctx context.Context, poolAddress common.Address, poolName string) ([][32]byte, error) {
	// Get the current length from the contract first to detect new entries
	lengthCallData := PackGetConversionTableHistoryLengthCall()
	lengthResult, err := sc.client.CallContract(ctx, ethereum.CallMsg{
		To:   &poolAddress,
		Data: lengthCallData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversion table history length: %w", err)
	}

	historyLength, err := ParseUint256Result(lengthResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse history length: %w", err)
	}

	currentLength := historyLength.Uint64()
	if currentLength == 0 {
		sc.logger.Warn("conversion table history is empty",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()))
		return nil, fmt.Errorf("conversion table history is empty for pool %s", poolName)
	}

	// Check cache (with read lock)
	sc.cacheMutex.RLock()
	cached, exists := sc.conversionHistoryCache[poolAddress]
	cachedLength := uint64(len(cached))
	sc.cacheMutex.RUnlock()

	// If cache exists and is up-to-date, return it
	if exists && cachedLength == currentLength {
		sc.logger.Debug("using cached conversion table history",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()),
			zap.Int("entries", len(cached)))
		return cached, nil
	}

	// Cache is stale or doesn't exist, acquire write lock and update
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	// Double-check cache in case another goroutine updated it while we waited
	cached, exists = sc.conversionHistoryCache[poolAddress]
	cachedLength = uint64(len(cached))
	if exists && cachedLength == currentLength {
		return cached, nil
	}

	// Determine what to fetch
	var startIndex uint64
	var history [][32]byte

	if exists && cachedLength < currentLength {
		// Cache exists but is outdated - fetch only new entries
		sc.logger.Info("detected new conversion table entries, fetching updates",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()),
			zap.Uint64("cachedEntries", cachedLength),
			zap.Uint64("currentEntries", currentLength),
			zap.Uint64("newEntries", currentLength-cachedLength))

		startIndex = cachedLength
		history = make([][32]byte, currentLength)
		copy(history, cached) // Copy existing cached entries
	} else {
		// No cache or cache is invalid - fetch all entries
		sc.logger.Info("fetching full conversion table history from contract",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()),
			zap.Uint64("entries", currentLength))

		startIndex = 0
		history = make([][32]byte, currentLength)
	}

	// Fetch new entries starting from startIndex
	sc.logger.Debug("fetching conversion table entries",
		zap.String("pool", poolName),
		zap.Uint64("startIndex", startIndex),
		zap.Uint64("endIndex", currentLength-1))

	for i := startIndex; i < currentLength; i++ {
		index := uint256.NewInt(i)
		callData := PackConversionTableHistoryCall(index)

		result, err := sc.client.CallContract(ctx, ethereum.CallMsg{
			To:   &poolAddress,
			Data: callData,
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch conversion table entry %d: %w", i, err)
		}

		entry, err := ParseConversionTableEntry(result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse conversion table entry %d: %w", i, err)
		}

		history[i] = entry
	}

	// Store updated cache
	sc.conversionHistoryCache[poolAddress] = history

	if startIndex > 0 {
		sc.logger.Info("updated conversion table history cache with new entries",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()),
			zap.Int("totalEntries", len(history)),
			zap.Uint64("newEntries", currentLength-startIndex))
	} else {
		sc.logger.Info("cached conversion table history",
			zap.String("pool", poolName),
			zap.String("poolAddress", poolAddress.Hex()),
			zap.Int("entries", len(history)))
	}

	return history, nil
}

// CalculateRates calculates rate limits based on stake amount and conversion tranches.
// The tranches define rate/tranche pairs locked in at stake time and must be sorted
// Each tranche can specify both QPS (queries per second) and QPM (queries per minute).
// The decimals parameter is used to convert the stake amount from wei to token units.
func CalculateRates(stakeAmount *uint256.Int, tranches []ConversionTranche, decimals uint8) queryratelimit.Rule {
	if stakeAmount == nil || stakeAmount.Cmp(uint256.NewInt(0)) == 0 {
		return queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0}
	}

	if len(tranches) == 0 {
		return queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0}
	}

	// Convert stake amount from wei to token units using decimals
	// This allows the JSON config to use human-readable token amounts (e.g., 5000 tokens)
	// instead of wei amounts (e.g., 5000000000000000000000)
	divisor := new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(uint64(decimals)))
	normalizedStake := new(uint256.Int).Div(stakeAmount, divisor)
	stakeAmountUint64 := normalizedStake.Uint64()

	// Find the highest tranche that the stake qualifies for
	// Tranches are expected to be in ascending order by stake amount
	var selectedTranche *ConversionTranche

	for i := range tranches {
		if stakeAmountUint64 >= tranches[i].Tranche {
			selectedTranche = &tranches[i]
		} else {
			break // Once we exceed stake amount, stop searching
		}
	}

	// If stake doesn't meet any tranche, return zero rate
	if selectedTranche == nil {
		return queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0}
	}

	// Check for misconfigured tranche (tranche value of 0 would cause division by zero)
	// Tranches should be configured with positive values (e.g., {"EVM":{"0":{"qpm":1}} is invalid)
	if selectedTranche.Tranche == 0 {
		return queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0}
	}

	// Calculate rate: (stakeAmount * rate) / tranche
	maxPerSecond := (stakeAmountUint64 * selectedTranche.RatePerSecond) / selectedTranche.Tranche
	maxPerMinute := (stakeAmountUint64 * selectedTranche.RatePerMinute) / selectedTranche.Tranche

	// If QPS was not explicitly set (is 0), derive it from QPM
	// This maintains backward compatibility with QPM-only configurations
	if selectedTranche.RatePerSecond == 0 && maxPerMinute >= 60 {
		maxPerSecond = maxPerMinute / 60
	}

	return queryratelimit.Rule{
		MaxPerSecond: maxPerSecond,
		MaxPerMinute: maxPerMinute,
	}
}

// FetchStakingPolicy creates a policy based on staking contract state using factory discovery
// stakerAddr is the address that holds the stake, signerAddr is the address that signed the request
// For self-staking, both addresses will be the same
func (sc *StakingClient) FetchStakingPolicy(ctx context.Context, stakerAddr, signerAddr common.Address) (*queryratelimit.Policy, error) {
	start := time.Now()

	// Log whether this is self-staking or delegated query
	isDelegated := stakerAddr != signerAddr
	if isDelegated {
		sc.logger.Info("fetching staking policy for delegated query",
			zap.String("staker", stakerAddr.Hex()),
			zap.String("signer", signerAddr.Hex()))
	} else {
		sc.logger.Info("fetching staking policy for self-staking query",
			zap.String("address", stakerAddr.Hex()))
	}

	policy := &queryratelimit.Policy{
		Limits: queryratelimit.Limits{
			Types: make(map[uint8]queryratelimit.Rule),
		},
	}

	currentTime := uint64(time.Now().Unix()) // #nosec G115 -- Unix timestamp is always positive since epoch
	poolsChecked := 0
	poolsWithStakes := 0
	totalErrors := 0

	// Track failure reasons for better diagnostics
	failureReasons := make(map[string]int)
	poolsSkipped := 0

	// Determine which pools to check
	var poolsToCheck []struct {
		address common.Address
		name    string
	}

	if sc.useDirectPoolConfig {
		// Use directly configured pool addresses
		for _, poolAddr := range sc.configuredPools {
			poolsToCheck = append(poolsToCheck, struct {
				address common.Address
				name    string
			}{
				address: poolAddr,
				name:    poolAddr.Hex(),
			})
		}
		sc.logger.Debug("using direct pool configuration",
			zap.Int("poolCount", len(poolsToCheck)))
	} else {
		// Use factory discovery (legacy path)
		for poolName, pool := range SupportedQueryPools {
			poolAddress, err := sc.GetCachedPoolAddress(ctx, pool.queryTypeBits(), poolName)
			if err != nil {
				totalErrors++
				failureReasons["factory_error"]++
				stakingPolicyFetches.WithLabelValues("factory_error", poolName).Inc()
				sc.logger.Warn("failed to discover pool from factory",
					zap.String("poolName", poolName),
					zap.String("staker", stakerAddr.Hex()),
					zap.Error(err))
				continue
			}

			// Skip if no pool exists for this query type
			if poolAddress == (common.Address{}) {
				poolsSkipped++
				failureReasons["no_pool_deployed"]++
				sc.logger.Debug("no pool found for query type", zap.String("queryType", poolName))
				continue
			}

			poolsToCheck = append(poolsToCheck, struct {
				address common.Address
				name    string
			}{
				address: poolAddress,
				name:    poolName,
			})
		}
		sc.logger.Debug("using factory-based pool discovery",
			zap.Int("poolCount", len(poolsToCheck)))
	}

	// Query each pool for stakes
	for _, poolInfo := range poolsToCheck {
		poolAddress := poolInfo.address
		poolName := poolInfo.name
		poolsChecked++

		// Verify signer is authorized to act on behalf of staker
		if err := sc.VerifySignerAuthorization(ctx, poolAddress, stakerAddr, signerAddr, poolName); err != nil {
			totalErrors++
			failureReasons["unauthorized_signer"]++
			stakingPolicyFetches.WithLabelValues("unauthorized_signer", poolName).Inc()
			sc.logger.Warn("signer not authorized for staker",
				zap.String("poolName", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.String("signer", signerAddr.Hex()),
				zap.Error(err))
			continue
		}

		stakeInfo, err := sc.GetStakeInfo(ctx, poolAddress, stakerAddr, poolName)
		if err != nil {
			totalErrors++
			failureReasons["contract_call_error"]++
			stakingPolicyFetches.WithLabelValues("error", poolName).Inc()
			sc.logger.Warn("failed to query staking pool during policy fetch",
				zap.String("poolName", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Error(err))
			continue
		}

		// Check stake status and record metrics
		if !stakeInfo.HasStake() {
			failureReasons["no_stake"]++
			stakingPolicyFetches.WithLabelValues("no_stake", poolName).Inc()
			sc.logger.Debug("no stake found in pool",
				zap.String("queryType", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.String("stakeAmount", stakeInfo.Amount.String()))
			continue
		}

		// Check if the staker is blocklisted
		// SECURITY: Fail closed - deny access if we cannot verify blocklist status
		stakerBlocked, err := sc.IsBlocklisted(ctx, poolAddress, stakerAddr, poolName)
		if err != nil {
			totalErrors++
			failureReasons["blocklist_check_error"]++
			stakingPolicyFetches.WithLabelValues("blocklist_error", poolName).Inc()
			sc.logger.Warn("failed to check staker blocklist status, denying access (fail-closed)",
				zap.String("queryType", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Error(err))
			continue
		}
		if stakerBlocked {
			failureReasons["staker_blocklisted"]++
			stakingPolicyFetches.WithLabelValues("blocklisted", poolName).Inc()
			sc.logger.Info("staker is blocklisted in pool",
				zap.String("queryType", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()))
			continue
		}

		// Check if the signer is blocklisted (if different from staker)
		// SECURITY: Fail closed - deny access if we cannot verify blocklist status
		if stakerAddr != signerAddr {
			signerBlocked, err := sc.IsBlocklisted(ctx, poolAddress, signerAddr, poolName)
			if err != nil {
				totalErrors++
				failureReasons["blocklist_check_error"]++
				stakingPolicyFetches.WithLabelValues("blocklist_error", poolName).Inc()
				sc.logger.Warn("failed to check signer blocklist status, denying access (fail-closed)",
					zap.String("queryType", poolName),
					zap.String("poolAddress", poolAddress.Hex()),
					zap.String("signer", signerAddr.Hex()),
					zap.Error(err))
				continue
			}
			if signerBlocked {
				failureReasons["signer_blocklisted"]++
				stakingPolicyFetches.WithLabelValues("blocklisted", poolName).Inc()
				sc.logger.Info("signer is blocklisted in pool",
					zap.String("queryType", poolName),
					zap.String("poolAddress", poolAddress.Hex()),
					zap.String("signer", signerAddr.Hex()))
				continue
			}
		}

		cacheDurationSeconds := uint64(sc.cacheDuration.Seconds()) // #nosec G115 -- cache duration is always positive
		if stakeInfo.HasExpired(currentTime, cacheDurationSeconds) {
			failureReasons["stake_expired"]++
			stakingPolicyFetches.WithLabelValues("expired", poolName).Inc()
			sc.logger.Info("stake has expired or will expire within cache duration",
				zap.String("queryType", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Uint64("accessEnd", stakeInfo.AccessEnd),
				zap.Uint64("currentTime", currentTime),
				zap.Uint64("cacheDurationSeconds", cacheDurationSeconds))
			continue
		}

		// Valid stake found
		poolsWithStakes++
		stakingPolicyFetches.WithLabelValues("success", poolName).Inc()

		// Get cached conversion table history for this pool (lazy-loads if needed)
		conversionHistory, err := sc.getConversionTableHistory(ctx, poolAddress, poolName)
		if err != nil {
			totalErrors++
			stakingPolicyFetches.WithLabelValues("conversion_history_error", poolName).Inc()
			sc.logger.Warn("failed to get conversion table history during policy fetch",
				zap.String("poolName", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Error(err))
			continue
		}

		// Validate index is within bounds
		if stakeInfo.ConversionTableIndex.Uint64() >= uint64(len(conversionHistory)) {
			totalErrors++
			stakingPolicyFetches.WithLabelValues("invalid_index", poolName).Inc()
			sc.logger.Error("conversion table index out of bounds",
				zap.String("poolName", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Uint64("index", stakeInfo.ConversionTableIndex.Uint64()),
				zap.Int("historyLength", len(conversionHistory)))
			continue
		}

		// Get CID from cached history using staker's index
		conversionCID := conversionHistory[stakeInfo.ConversionTableIndex.Uint64()]

		// Fetch and parse IPFS JSON (cached per CID)
		conversionTable, err := sc.ipfsClient.FetchConversionTable(ctx, conversionCID)
		if err != nil {
			totalErrors++
			stakingPolicyFetches.WithLabelValues("ipfs_fetch_error", poolName).Inc()
			sc.logger.Warn("failed to fetch conversion table from IPFS during policy fetch",
				zap.String("poolName", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Error(err))
			continue
		}

		// Fetch pool metadata for proper stake amount conversion (cached)
		poolMetadata, err := sc.GetPoolMetadata(ctx, poolAddress, poolName)
		var decimals uint8 = 18 // Default to 18 decimals (standard ERC20)
		if err != nil {
			sc.logger.Warn("failed to get pool metadata, defaulting to 18 decimals",
				zap.String("poolName", poolName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.Error(err))
		} else {
			decimals = poolMetadata.TokenDecimals
		}

		// Process rates for each chain in the conversion table
		// In direct config mode, we discover supported chains from IPFS
		// In factory mode, we use the predefined query types
		var chainsToProcess []string

		if sc.useDirectPoolConfig {
			// Get all chains from the conversion table
			chainsToProcess = conversionTable.GetSupportedChains()
			sc.logger.Debug("discovered chains from conversion table",
				zap.String("poolName", poolName),
				zap.Strings("chains", chainsToProcess))
		} else {
			// Use predefined query types from SupportedQueryPools
			poolConfig := SupportedQueryPools[poolName]
			if len(poolConfig.QueryTypes) == 0 {
				sc.logger.Error("pool has no query types",
					zap.String("poolName", poolName))
				continue
			}

			chainName, err := getChainName(poolConfig.QueryTypes[0])
			if err != nil {
				totalErrors++
				sc.logger.Error("unknown query type in pool",
					zap.String("poolName", poolName),
					zap.Uint8("queryType", uint8(poolConfig.QueryTypes[0])),
					zap.Error(err))
				continue
			}
			chainsToProcess = []string{chainName}
		}

		// Calculate and apply rates for each chain
		for _, chainName := range chainsToProcess {
			// Extract chain-specific rates from IPFS data
			tranches, err := conversionTable.GetTranchesByChain(chainName)
			if err != nil {
				totalErrors++
				stakingPolicyFetches.WithLabelValues("chain_parse_error", poolName).Inc()
				sc.logger.Warn("failed to get tranches for chain during policy fetch",
					zap.String("poolName", poolName),
					zap.String("chainName", chainName),
					zap.String("staker", stakerAddr.Hex()),
					zap.Error(err))
				continue
			}

			// Calculate rate limits using tranches
			rates := CalculateRates(stakeInfo.Amount, tranches, decimals)

			sc.logger.Info("rate calculation details",
				zap.String("pool", poolName),
				zap.String("chainName", chainName),
				zap.String("stakeAmount", stakeInfo.Amount.String()),
				zap.Uint8("decimals", decimals),
				zap.Int("trancheCount", len(tranches)),
				zap.Uint64("maxPerSecond", rates.MaxPerSecond),
				zap.Uint64("maxPerMinute", rates.MaxPerMinute))

			// Determine tier for metrics
			tier := "none"
			if rates.MaxPerSecond > 0 {
				tier = "qps"
			} else if rates.MaxPerMinute > 0 {
				tier = "qpm"
			}

			// Skip if no rates calculated
			if rates.MaxPerSecond == 0 && rates.MaxPerMinute == 0 {
				stakingPolicyDecisions.WithLabelValues("denied", tier, chainName).Inc()
				continue
			}

			sc.logger.Debug("calculated rates for pool",
				zap.String("poolName", poolName),
				zap.String("chainName", chainName),
				zap.String("poolAddress", poolAddress.Hex()),
				zap.String("signer", signerAddr.Hex()),
				zap.String("tier", tier),
				zap.Uint64("maxPerSecond", rates.MaxPerSecond),
				zap.Uint64("maxPerMinute", rates.MaxPerMinute),
				zap.String("stakeAmount", stakeInfo.Amount.String()))

			// Map chain name to query types and apply rates
			queryTypes := chainNameToQueryTypes(chainName)
			for _, queryType := range queryTypes {
				queryTypeStr := fmt.Sprintf("%d", queryType)
				qt := uint8(queryType)

				// Record policy decision
				stakingPolicyDecisions.WithLabelValues("allowed", tier, queryTypeStr).Inc()

				// If multiple pools grant access to the same query type, take the maximum
				if existingRule, exists := policy.Limits.Types[qt]; exists {
					updated := false
					if rates.MaxPerSecond > existingRule.MaxPerSecond {
						existingRule.MaxPerSecond = rates.MaxPerSecond
						updated = true
					}
					if rates.MaxPerMinute > existingRule.MaxPerMinute {
						existingRule.MaxPerMinute = rates.MaxPerMinute
						updated = true
					}
					policy.Limits.Types[qt] = existingRule

					if updated {
						sc.logger.Debug("updated existing policy with higher limits",
							zap.String("queryType", poolName),
							zap.Uint8("queryType", qt),
							zap.Uint64("newMaxPerSecond", existingRule.MaxPerSecond),
							zap.Uint64("newMaxPerMinute", existingRule.MaxPerMinute))
					}
				} else {
					policy.Limits.Types[qt] = rates
					sc.logger.Debug("added new policy",
						zap.String("queryType", poolName),
						zap.Uint8("queryType", qt),
						zap.Uint64("maxPerSecond", rates.MaxPerSecond),
						zap.Uint64("maxPerMinute", rates.MaxPerMinute))
				}
			}
		}
	}

	// Final logging and metrics
	totalQueryTypes := len(policy.Limits.Types)
	policyFetchDuration := time.Since(start)

	// Build detailed failure reason summary
	failureReasonFields := []zap.Field{
		zap.String("staker", stakerAddr.Hex()),
		zap.String("signer", signerAddr.Hex()),
		zap.Bool("isDelegated", isDelegated),
		zap.Int("queryTypesChecked", poolsChecked),
		zap.Int("poolsSkipped", poolsSkipped),
		zap.Int("poolsWithStakes", poolsWithStakes),
		zap.Int("totalErrors", totalErrors),
		zap.Int("allowedQueryTypes", totalQueryTypes),
		zap.Duration("fetchDuration", policyFetchDuration),
	}

	// Add failure reason breakdown if any
	if len(failureReasons) > 0 {
		for reason, count := range failureReasons {
			failureReasonFields = append(failureReasonFields, zap.Int("failures_"+reason, count))
		}
	}

	if totalQueryTypes > 0 {
		sc.logger.Info("completed staking policy fetch - access granted", failureReasonFields...)
		stakingPolicyFetches.WithLabelValues("policy_created", "all").Inc()
	} else {
		// Policy is empty - provide detailed diagnostic summary
		sc.logger.Warn("completed staking policy fetch - NO ACCESS GRANTED", failureReasonFields...)

		// Log specific diagnostic message based on failure reasons
		if failureReasons["unauthorized_signer"] > 0 {
			if isDelegated {
				sc.logger.Warn("delegation failure: signer not authorized by staker",
					zap.String("staker", stakerAddr.Hex()),
					zap.String("signer", signerAddr.Hex()),
					zap.String("hint", "staker must call setSigner() to authorize this signer"))
			}
		} else if failureReasons["no_stake"] > 0 {
			sc.logger.Warn("no access: staker has no stake in any pools",
				zap.String("staker", stakerAddr.Hex()),
				zap.String("hint", "staker must stake tokens to gain CCQ access"))
		} else if failureReasons["stake_expired"] > 0 {
			sc.logger.Warn("no access: all stakes have expired",
				zap.String("staker", stakerAddr.Hex()),
				zap.String("hint", "staker needs to renew stakes"))
		} else if failureReasons["staker_blocklisted"] > 0 || failureReasons["signer_blocklisted"] > 0 {
			sc.logger.Warn("no access: address is blocklisted",
				zap.String("staker", stakerAddr.Hex()),
				zap.String("signer", signerAddr.Hex()))
		}

		stakingPolicyFetches.WithLabelValues("no_access", "all").Inc()
	}

	return policy, nil
}
