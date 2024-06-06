package p2p

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
)

// This is the definitive source for the default network parameters. Please reference these (or use the methods below), but avoid copying them!
const MainnetNetworkId = "/wormhole/mainnet/2"
const MainnetBootstrapPeers = "/dns4/wormhole-v2-mainnet-bootstrap.xlabs.xyz/udp/8999/quic/p2p/12D3KooWNQ9tVrcb64tw6bNs2CaNrUGPM7yRrKvBBheQ5yCyPHKC,/dns4/wormhole.mcf.rocks/udp/8999/quic/p2p/12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU,/dns4/wormhole-v2-mainnet-bootstrap.staking.fund/udp/8999/quic/p2p/12D3KooWG8obDX9DNi1KUwZNu9xkGwfKqTp2GFwuuHpWZ3nQruS1"
const MainnetCcqBootstrapPeers = "/dns4/wormhole-v2-mainnet-bootstrap.xlabs.xyz/udp/8996/quic/p2p/12D3KooWNQ9tVrcb64tw6bNs2CaNrUGPM7yRrKvBBheQ5yCyPHKC,/dns4/wormhole.mcf.rocks/udp/8996/quic/p2p/12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU,/dns4/wormhole-v2-mainnet-bootstrap.staking.fund/udp/8996/quic/p2p/12D3KooWG8obDX9DNi1KUwZNu9xkGwfKqTp2GFwuuHpWZ3nQruS1"

const TestnetNetworkId = "/wormhole/testnet/2/1"
const TestnetBootstrapPeers = "/dns4/t-guardian-01.testnet.xlabs.xyz/udp/8999/quic/p2p/12D3KooWCW3LGUtkCVkHZmVSZHzL3C4WRKWfqAiJPz1NR7dT9Bxh,/dns4/t-guardian-02.testnet.xlabs.xyz/udp/8999/quic/p2p/12D3KooWJXA6goBCiWM8ucjzc4jVUBSqL9Rri6UpjHbkMPErz5zK,/dns4/p2p-guardian-testnet-1.solana.p2p.org/udp/8999/quic/p2p/12D3KooWE4dmZwxhfjCKHLUqSaww96Cf7kmq1ZuKmzPz3MrJgZxp"
const TestnetCcqBootstrapPeers = "/dns4/t-guardian-01.testnet.xlabs.xyz/udp/8996/quic/p2p/12D3KooWCW3LGUtkCVkHZmVSZHzL3C4WRKWfqAiJPz1NR7dT9Bxh,/dns4/t-guardian-02.testnet.xlabs.xyz/udp/8996/quic/p2p/12D3KooWJXA6goBCiWM8ucjzc4jVUBSqL9Rri6UpjHbkMPErz5zK,/dns4/p2p-guardian-testnet-1.solana.p2p.org/udp/8996/quic/p2p/12D3KooWE4dmZwxhfjCKHLUqSaww96Cf7kmq1ZuKmzPz3MrJgZxp"

// The Devnet bootstrap peers are derived from the guardian key so we can't include them here.
const DevnetNetworkId = "/wormhole/dev"

// GetNetworkId returns the default network ID.
func GetNetworkId(env common.Environment) string {
	if env == common.MainNet {
		return MainnetNetworkId
	}
	if env == common.TestNet {
		return TestnetNetworkId
	}
	return DevnetNetworkId
}

// GetBootstrapPeers returns the default p2p bootstrap peers for mainnet and testnet. For any other environment, it returns an error.
func GetBootstrapPeers(env common.Environment) (string, error) {
	if env == common.MainNet {
		return MainnetBootstrapPeers, nil
	}
	if env == common.TestNet {
		return TestnetBootstrapPeers, nil
	}
	return "", fmt.Errorf("unsupported environment")
}

// GetCcqBootstrapPeers returns the default ccq bootstrap peers for mainnet and testnet. For any other environment, it returns an error.
func GetCcqBootstrapPeers(env common.Environment) (string, error) {
	if env == common.MainNet {
		return MainnetCcqBootstrapPeers, nil
	}
	if env == common.TestNet {
		return TestnetCcqBootstrapPeers, nil
	}
	return "", fmt.Errorf("unsupported environment")
}
