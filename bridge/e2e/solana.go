package e2e

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mr-tron/base58"
	"github.com/tendermint/tendermint/libs/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/ethereum/erc20"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

func getSPLBalance(ctx context.Context, c *kubernetes.Clientset, hexAddr string) (*big.Int, error) {
	b, err := executeCommandInPod(ctx, c, "solana-devnet-0", "setup",
		[]string{"cli", "balance", hexAddr})
	if err != nil {
		return nil, fmt.Errorf("error running 'cli balance': %w", err)
	}

	re := regexp.MustCompile("(?m)^amount: (.*)$")
	m := re.FindStringSubmatch(string(b))
	if len(m) == 0 {
		return nil, fmt.Errorf("invalid 'cli balance' output: %s", string(b))
	}

	n, ok := new(big.Int).SetString(m[1], 10)
	if !ok {
		return nil, fmt.Errorf("invalid int: %s", m[1])
	}

	return n, nil
}

func waitSPLBalance(t *testing.T, ctx context.Context, c *kubernetes.Clientset, hexAddr string, before *big.Int, target int64) {
	// Wait for target account balance to increase.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := wait.PollUntil(1*time.Second, func() (bool, error) {
		after, err := getSPLBalance(ctx, c, hexAddr)
		if err != nil {
			t.Fatal(err)
		}

		d := new(big.Int).Sub(after, before)
		t.Logf("SPL balance after: %d -> %d, delta %d", before, after, d)

		if after.Cmp(before) != 0 {
			if d.Cmp(new(big.Int).SetInt64(target)) != 0 {
				t.Errorf("expected SPL delta of %v, got: %v", target, d)
			}
			return true, nil
		}
		return false, nil
	}, ctx.Done())
	if err != nil {
		t.Error(err)
	}
}

func testSolanaLockup(t *testing.T, ctx context.Context, ec *ethclient.Client, c *kubernetes.Clientset,
	sourceAcct string, tokenAddr string, destination common.Address, amount int, precisionGain int) {
	token, err := erc20.NewErc20(destination, ec)
	if err != nil {
		panic(err)
	}

	// Store balance of wrapped destination token
	beforeErc20, err := token.BalanceOf(nil, devnet.GanacheClientDefaultAccountAddress)
	if err != nil {
		beforeErc20 = new(big.Int)
		t.Log(err) // account may not yet exist, defaults to 0
	}
	t.Logf("ERC20 balance: %v", beforeErc20)

	// Store balance of source SPL token
	beforeSPL, err := getSPLBalance(ctx, c, sourceAcct)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("SPL balance: %d", beforeSPL)

	_, err = executeCommandInPod(ctx, c, "solana-devnet-0", "setup",
		[]string{"cli", "lock",
			// Address of the Wormhole bridge.
			devnet.SolanaBridgeContract,
			// Account which holds the SPL tokens to be sent.
			sourceAcct,
			// The SPL token.
			tokenAddr,
			// Token amount.
			strconv.Itoa(amount),
			// Destination chain ID.
			strconv.Itoa(vaa.ChainIDEthereum),
			// Random nonce.
			strconv.Itoa(int(rand.Uint16())),
			// Destination account on Ethereum
			devnet.GanacheClientDefaultAccountAddress.Hex()[2:],
		})
	if err != nil {
		t.Fatal(err)
	}

	// Destination account increases by the full amount.
	waitEthBalance(t, ctx, token, beforeErc20, int64(float64(amount)*math.Pow10(precisionGain)))

	// Source account decreases by full amount.
	waitSPLBalance(t, ctx, c, sourceAcct, beforeSPL, -int64(amount))
}

func testSolanaToTerraLockup(t *testing.T, ctx context.Context, tc *TerraClient, c *kubernetes.Clientset,
	sourceAcct string, tokenAddr string, amount int, precisionGain int) {

	tokenSlice, err := base58.Decode(tokenAddr)
	if err != nil {
		t.Fatal(err)
	}
	terraToken, err := getAssetAddress(ctx, devnet.TerraBridgeAddress, vaa.ChainIDSolana, tokenSlice)

	// Get balance if deployed
	beforeCw20, err := getTerraBalance(ctx, terraToken)
	if err != nil {
		t.Log(err) // account may not yet exist, defaults to 0
	}
	t.Logf("CW20 balance: %v", beforeCw20)

	// Store balance of source SPL token
	beforeSPL, err := getSPLBalance(ctx, c, sourceAcct)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("SPL balance: %d", beforeSPL)

	_, err = executeCommandInPod(ctx, c, "solana-devnet-0", "setup",
		[]string{"cli", "lock",
			// Address of the Wormhole bridge.
			devnet.SolanaBridgeContract,
			// Account which holds the SPL tokens to be sent.
			sourceAcct,
			// The SPL token.
			tokenAddr,
			// Token amount.
			strconv.Itoa(amount),
			// Destination chain ID.
			strconv.Itoa(vaa.ChainIDTerra),
			// Random nonce.
			strconv.Itoa(int(rand.Uint16())),
			// Destination account on Terra
			devnet.TerraMainTestAddressHex,
		})
	if err != nil {
		t.Fatal(err)
	}

	// Source account decreases by full amount.
	waitSPLBalance(t, ctx, c, sourceAcct, beforeSPL, -int64(amount))

	// Destination account increases by the full amount.
	waitTerraUnknownBalance(t, ctx, devnet.TerraBridgeAddress, vaa.ChainIDSolana, tokenSlice, beforeCw20, int64(float64(amount)*math.Pow10(precisionGain)))
}
