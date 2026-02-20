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

	// Fetcher computes rate limit policy from the staker's on-chain stake.
	// Signer authorization is handled separately via AuthorizeSigner.
	fetcher := func(ctx context.Context, stakerAddr common.Address) (*queryratelimit.Policy, error) {
		return stakingClient.FetchStakingPolicy(ctx, stakerAddr)
	}

	// Create PolicyProvider with staking fetcher and authorizer
	return queryratelimit.NewPolicyProvider(
		queryratelimit.WithPolicyProviderFetcher(fetcher),
		queryratelimit.WithPolicyProviderAuthorizer(stakingClient.AuthorizeSigner),
		queryratelimit.WithPolicyProviderLogger(logger.With(zap.String("component", "staking-policy-provider"))),
		queryratelimit.WithPolicyProviderParentContext(parentContext),
		queryratelimit.WithPolicyProviderOptimistic(true), // Enable background cache refresh
		queryratelimit.WithPolicyProviderCacheDuration(cacheDuration),
	)
}
