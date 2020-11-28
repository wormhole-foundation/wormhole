package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
)

func TestEndToEnd(t *testing.T) {
	// List of pods we need in a ready state before we can run tests.
	want := []string{
		// Our test guardian set.
		"guardian-0",
		//"guardian-1",
		//"guardian-2",
		//"guardian-3",
		//"guardian-4",
		//"guardian-5",

		// Connected chains
		"solana-devnet-0",

		"terra-terrad-0",
		"terra-lcd-0",

		"eth-devnet-0",
	}

	c := getk8sClient()

	// Wait for all pods to be ready. This blocks until the bridge is ready to receive lockups.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	waitForPods(ctx, c, want)
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}

	// Ethereum client.
	ec, err := ethclient.Dial(devnet.GanacheRPCURL)
	if err != nil {
		t.Fatalf("dialing devnet eth rpc failed: %v", err)
	}
	kt := devnet.GetKeyedTransactor(ctx)

	// Generic context for tests.
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	t.Run("[SOL] Native -> [ETH] Wrapped", func(t *testing.T) {
		testSolanaLockup(t, ctx, ec, c,
			// Source SPL account
			devnet.SolanaExampleTokenOwningAccount,
			// Source SPL token
			devnet.SolanaExampleToken,
			// Our wrapped destination token on Ethereum
			devnet.GanacheExampleERC20WrappedSOL,
			// Amount of SPL token value to transfer.
			50*devnet.SolanaDefaultPrecision,
			// Same precision - same amount, no precision gained.
			0,
		)
	})

	t.Run("[ETH] Wrapped -> [SOL] Native", func(t *testing.T) {
		testEthereumLockup(t, ctx, ec, kt, c,
			// Source ERC20 token
			devnet.GanacheExampleERC20WrappedSOL,
			// Destination SPL token account
			devnet.SolanaExampleTokenOwningAccount,
			// Amount (the reverse of what the previous test did, with the same precision because
			// the wrapped ERC20 is set to the original asset's 10**9 precision).
			50*devnet.SolanaDefaultPrecision,
			// No precision loss
			0,
		)
	})

	t.Run("[ETH] Native -> [SOL] Wrapped", func(t *testing.T) {
		testEthereumLockup(t, ctx, ec, kt, c,
			// Source ERC20 token
			devnet.GanacheExampleERC20Token,
			// Destination SPL token account
			devnet.SolanaExampleWrappedERCTokenOwningAccount,
			// Amount
			0.000000012*devnet.ERC20DefaultPrecision,
			// We lose 9 digits of precision on this path, as the default ERC20 token has 10**18 precision.
			9,
		)
	})

	t.Run("[SOL] Wrapped -> [ETH] Native", func(t *testing.T) {
		testSolanaLockup(t, ctx, ec, c,
			// Source SPL account
			devnet.SolanaExampleWrappedERCTokenOwningAccount,
			// Source SPL token
			devnet.SolanaExampleWrappedERCToken,
			// Our wrapped destination token on Ethereum
			devnet.GanacheExampleERC20Token,
			// Amount of SPL token value to transfer.
			0.000000012*devnet.SolanaDefaultPrecision,
			// We gain 9 digits of precision on Eth.
			9,
		)
	})
}
