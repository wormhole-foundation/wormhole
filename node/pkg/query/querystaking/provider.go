package querystaking

import (
	"context"
	"time"

	"github.com/certusone/wormhole/node/pkg/query/queryratelimit"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// CreateStakingPolicyProvider creates a PolicyProvider configured for staking-based rate limits
func CreateStakingPolicyProvider(ethClient *ethclient.Client, logger *zap.Logger, parentContext context.Context, factoryAddress common.Address, poolAddresses []common.Address, ipfsGateway string, cacheDuration time.Duration) (*queryratelimit.PolicyProvider, error) {
	// Create IPFS client with hardcoded timeout (30s)
	ipfsClient := NewIPFSClient(ipfsGateway, 30*time.Second, logger)

	stakingClient := NewStakingClient(ethClient, logger, factoryAddress, poolAddresses, ipfsClient, cacheDuration)

	// Create the fetcher function that queries staking contracts
	// This supports both self-staking and delegated signing:
	// - signerAddr: The address that signed the query request
	// - stakerAddr: The address that holds the stake
	// For self-staking, both addresses are the same.
	// For delegated signing, the signer is authorized by the staker.
	fetcher := func(ctx context.Context, signerAddr, stakerAddr common.Address) (*queryratelimit.Policy, error) {
		return stakingClient.FetchStakingPolicy(ctx, stakerAddr, signerAddr)
	}

	// Create PolicyProvider with staking fetcher
	return queryratelimit.NewPolicyProvider(
		queryratelimit.WithPolicyProviderFetcher(fetcher),
		queryratelimit.WithPolicyProviderLogger(logger.With(zap.String("component", "staking-policy-provider"))),
		queryratelimit.WithPolicyProviderParentContext(parentContext),
		queryratelimit.WithPolicyProviderOptimistic(true), // Enable background cache refresh
		queryratelimit.WithPolicyProviderCacheDuration(cacheDuration),
	)
}
