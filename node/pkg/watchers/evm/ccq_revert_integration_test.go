//go:build integration

package evm

// Integration test for ccqBatchHasRevert against a real Ethereum node.
//
// Run with:
//   go test -tags integration -v -run TestCcqBatchHasRevertIntegration ./pkg/watchers/evm/
//
// Requires: anvil in PATH (ships with Foundry).
//
// Verifies that a reverted eth_call produces a BatchElem.Error containing
// "execution reverted", and that ccqBatchHasRevert correctly detects it.

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	eth_hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCcqBatchHasRevertIntegration spins up a local anvil node, deploys an
// always-reverting contract, runs the same RPC batch call the guardian uses,
// and asserts that ccqBatchHasRevert detects the revert.
func TestCcqBatchHasRevertIntegration(t *testing.T) {
	anvilPath, err := exec.LookPath("anvil")
	if err != nil {
		t.Skip("anvil not found in PATH — install Foundry to run this test")
	}

	port, err := freePort()
	require.NoError(t, err)
	rpcURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// ── 1. Start anvil ───────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	anvilCmd := exec.CommandContext(ctx, anvilPath,
		"--port", fmt.Sprintf("%d", port),
		"--silent", // suppress anvil banner / log noise
	)
	anvilCmd.Stdout = os.Stderr // route to test stderr so -v shows it
	anvilCmd.Stderr = os.Stderr
	require.NoError(t, anvilCmd.Start())
	defer anvilCmd.Process.Kill() //nolint:errcheck

	require.NoError(t, waitForAnvil(rpcURL, 10*time.Second), "anvil did not start in time")
	t.Logf("anvil listening on %s", rpcURL)

	// ── 2. Deploy an always-reverting contract ───────────────────────────────
	//
	// Init bytecode:  6460006000fd6000526005601bf3
	//   PUSH5 0x60006000fd  (runtime code)
	//   PUSH1 0x00          (memory offset)
	//   MSTORE              (store 32 bytes; runtime code lands at offset 27)
	//   PUSH1 0x05          (size = 5)
	//   PUSH1 0x1b          (offset = 27)
	//   RETURN              → deploys runtime code
	//
	// Runtime bytecode: 60006000fd
	//   PUSH1 0x00
	//   PUSH1 0x00
	//   REVERT              → reverts with empty data on every call
	initBytecode := common.FromHex("6460006000fd6000526005601bf3")

	ec, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	defer ec.Close()

	// Anvil test account #0 (mnemonic: "test test … junk")
	privKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(privKey.PublicKey)

	chainID, err := ec.ChainID(ctx)
	require.NoError(t, err)
	nonce, err := ec.PendingNonceAt(ctx, from)
	require.NoError(t, err)
	gasPrice, err := ec.SuggestGasPrice(ctx)
	require.NoError(t, err)

	deployTx, err := types.SignTx(
		types.NewContractCreation(nonce, big.NewInt(0), 100_000, gasPrice, initBytecode),
		types.NewEIP155Signer(chainID),
		privKey,
	)
	require.NoError(t, err)
	require.NoError(t, ec.SendTransaction(ctx, deployTx))

	var contractAddr common.Address
	require.Eventually(t, func() bool {
		receipt, err := ec.TransactionReceipt(ctx, deployTx.Hash())
		if err != nil || receipt == nil {
			return false
		}
		require.EqualValues(t, 1, receipt.Status, "contract deployment tx failed")
		contractAddr = receipt.ContractAddress
		return true
	}, 10*time.Second, 200*time.Millisecond)
	t.Logf("always-revert contract deployed at %s", contractAddr.Hex())

	// ── 3. Make an RPC batch call — same format the guardian uses ────────────
	rpcClient, err := rpc.Dial(rpcURL)
	require.NoError(t, err)
	defer rpcClient.Close()

	var callResult eth_hexutil.Bytes
	batch := []rpc.BatchElem{
		{
			// Mirrors ccqBuildBatchFromCallData:
			//   Method: "eth_call", Args: [{to, data}, blockArg]
			Method: "eth_call",
			Args: []interface{}{
				map[string]interface{}{
					"to":   contractAddr,
					"data": "0x06fdde03", // name() — doesn't matter, contract always reverts
				},
				"latest",
			},
			Result: &callResult,
		},
	}

	// BatchCallContext is what EthereumBaseConnector.RawBatchCallContext calls.
	err = rpcClient.BatchCallContext(ctx, batch)
	require.NoError(t, err, "BatchCallContext transport error (individual call errors go in batch[i].Error)")

	// ── 4. Assert the revert is visible in BatchElem.Error ───────────────────
	t.Logf("batch[0].Error = %v", batch[0].Error)
	require.NotNil(t, batch[0].Error,
		"expected a per-call revert error; got nil (contract may not have deployed correctly)")

	errStr := strings.ToLower(batch[0].Error.Error())
	assert.True(t,
		strings.Contains(errStr, "execution reverted"),
		"error should contain 'execution reverted', got: %q", batch[0].Error.Error(),
	)

	// ── 5. The key assertion: ccqBatchHasRevert must detect it ───────────────
	assert.True(t,
		ccqBatchHasRevert(batch, 1),
		"ccqBatchHasRevert(batch, 1) should be true — this is what triggers QueryFatalError in the PR",
	)
	assert.False(t,
		ccqBatchHasRevert(batch, 0),
		"ccqBatchHasRevert(batch, 0) should always be false (numCalls=0 guards against empty batches)",
	)

	t.Logf("batch[0].Error = %q, ccqBatchHasRevert → true", batch[0].Error.Error())
}

// TestCcqBatchHasRevertIntegration_NoRevert is a negative control: a valid
// view call must NOT be flagged as a revert.
func TestCcqBatchHasRevertIntegration_NoRevert(t *testing.T) {
	anvilPath, err := exec.LookPath("anvil")
	if err != nil {
		t.Skip("anvil not found in PATH — install Foundry to run this test")
	}

	port, err := freePort()
	require.NoError(t, err)
	rpcURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	anvilCmd := exec.CommandContext(ctx, anvilPath, "--port", fmt.Sprintf("%d", port), "--silent")
	anvilCmd.Stderr = os.Stderr
	require.NoError(t, anvilCmd.Start())
	defer anvilCmd.Process.Kill() //nolint:errcheck

	require.NoError(t, waitForAnvil(rpcURL, 10*time.Second))

	rpcClient, err := rpc.Dial(rpcURL)
	require.NoError(t, err)
	defer rpcClient.Close()

	// Call eth_getBalance on account #0 — always succeeds, never reverts.
	// We use eth_call with no data to a normal account → returns 0x (empty, success).
	var callResult eth_hexutil.Bytes
	batch := []rpc.BatchElem{
		{
			Method: "eth_call",
			Args: []interface{}{
				map[string]interface{}{
					"to":   "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", // anvil account #0 (EOA, no code)
					"data": "0x",
				},
				"latest",
			},
			Result: &callResult,
		},
	}

	require.NoError(t, rpcClient.BatchCallContext(ctx, batch))

	// EOA call: no error, no revert
	assert.Nil(t, batch[0].Error, "call to EOA should not error")
	assert.False(t,
		ccqBatchHasRevert(batch, 1),
		"ccqBatchHasRevert should be false for a successful call",
	)
	t.Logf("negative control passed — successful call correctly not flagged as revert")
}

// ─── helpers ────────────────────────────────────────────────────────────────

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitForAnvil(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c, err := rpc.Dial(url)
		if err == nil {
			var n string
			if callErr := c.CallContext(context.Background(), &n, "eth_blockNumber"); callErr == nil {
				c.Close()
				return nil
			}
			c.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("anvil at %s not ready after %s", url, timeout)
}
