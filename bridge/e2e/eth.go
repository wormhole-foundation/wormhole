package e2e

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tendermint/tendermint/libs/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/ethereum/abi"
	"github.com/certusone/wormhole/bridge/pkg/ethereum/erc20"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

// waitEthBalance waits for target account before to increase.
func waitEthBalance(t *testing.T, ctx context.Context, token *erc20.Erc20, before *big.Int, target int64) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := wait.PollUntil(1*time.Second, func() (bool, error) {
		after, err := token.BalanceOf(nil, devnet.GanacheClientDefaultAccountAddress)
		if err != nil {
			t.Log(err)
			return false, nil
		}

		d := new(big.Int).Sub(after, before)
		t.Logf("ERC20 balance after: %d -> %d, delta %d", before, after, d)

		if after.Cmp(before) != 0 {
			if d.Cmp(new(big.Int).SetInt64(target)) != 0 {
				t.Errorf("expected ERC20 delta of %v, got: %v", target, d)
			}
			return true, nil
		}
		return false, nil
	}, ctx.Done())

	if err != nil {
		t.Error(err)
	}
}

func testEthereumLockup(t *testing.T, ctx context.Context, ec *ethclient.Client, kt *bind.TransactOpts,
	c *kubernetes.Clientset, tokenAddr common.Address, destination string, amount int64, precisionLoss int) {

	// Bridge client
	ethBridge, err := abi.NewAbi(devnet.GanacheBridgeContractAddress, ec)
	if err != nil {
		panic(err)
	}

	// Source token client
	token, err := erc20.NewErc20(tokenAddr, ec)
	if err != nil {
		panic(err)
	}

	// Store balance of source ERC20 token
	beforeErc20, err := token.BalanceOf(nil, devnet.GanacheClientDefaultAccountAddress)
	if err != nil {
		t.Log(err) // account may not yet exist, defaults to 0
	}
	t.Logf("ERC20 balance: %v", beforeErc20)

	// Store balance of destination SPL token
	beforeSPL, err := getSPLBalance(ctx, c, destination)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("SPL balance: %d", beforeSPL)

	// Send lockup
	tx, err := ethBridge.LockAssets(kt,
		// asset address
		tokenAddr,
		// token amount
		new(big.Int).SetInt64(amount),
		// recipient address on target chain
		devnet.MustBase58ToEthAddress(destination),
		// target chain
		vaa.ChainIDSolana,
		// random nonce
		rand.Uint32(),
		// refund dust?
		false,
	)
	if err != nil {
		t.Error(err)
	}

	t.Logf("sent lockup tx: %v", tx.Hash().Hex())

	// Destination account increases by full amount.
	waitSPLBalance(t, ctx, c, destination, beforeSPL, int64(float64(amount)/math.Pow10(precisionLoss)))

	// Source account decreases by the full amount.
	waitEthBalance(t, ctx, token, beforeErc20, -int64(amount))
}
