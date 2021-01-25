package e2e

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"k8s.io/client-go/kubernetes"
	"testing"
	"time"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/ethereum/go-ethereum/ethclient"
)

func setup(t *testing.T) (*kubernetes.Clientset, *ethclient.Client, *bind.TransactOpts) {
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
	kt := devnet.GetKeyedTransactor(context.Background())

	return c, ec, kt
}

// Careful about parallel tests - accounts on some chains like Ethereum cannot be
// used concurrently as they have monotonically increasing nonces that would conflict.
// Either use different Ethereum account, or do not run Ethereum tests in parallel.

func TestEndToEnd_SOL_ETH(t *testing.T) {
	c, ec, kt := setup(t)

	t.Run("[SOL] Native -> [ETH] Wrapped", func(t *testing.T) {
		testSolanaLockup(t, context.Background(), ec, c,
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
		testEthereumLockup(t, context.Background(), ec, kt, c,
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
		testEthereumLockup(t, context.Background(), ec, kt, c,
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
		testSolanaLockup(t, context.Background(), ec, c,
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

func TestEndToEnd_SOL_Terra(t *testing.T) {
	c, _, _ := setup(t)

	tc, err := NewTerraClient()
	if err != nil {
		t.Fatalf("creating devnet terra client failed: %v", err)
	}

	t.Run("[Terra] Native -> [SOL] Wrapped", func(t *testing.T) {
		testTerraLockup(t, context.Background(), tc, c,
			// Source CW20 token
			devnet.TerraTokenAddress,
			// Destination SPL token account
			devnet.SolanaExampleWrappedCWTokenOwningAccount,
			// Amount
			2*devnet.TerraDefaultPrecision,
			// Same precision - same amount, no precision gained.
			0,
		)
	})

	// TODO(https://github.com/certusone/wormhole/issues/164): [SOL] Wrapped -> [Terra] Native

	t.Run("[SOL] Native -> [Terra] Wrapped", func(t *testing.T) {
		testSolanaToTerraLockup(t, context.Background(), tc, c,
			// Source SPL account
			devnet.SolanaExampleTokenOwningAccount,
			// Source SPL token
			devnet.SolanaExampleToken,
			// Amount of SPL token value to transfer.
			50*devnet.SolanaDefaultPrecision,
			// Same precision - same amount, no precision gained.
			0,
		)
	})

	// TODO(https://github.com/certusone/wormhole/issues/164): [Terra] Wrapped -> [SOL] Native
}

// TODO(https://github.com/certusone/wormhole/issues/164): TestEndToEnd_ETH_Terra
