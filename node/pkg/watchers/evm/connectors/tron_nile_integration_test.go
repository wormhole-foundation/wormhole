package connectors

import (
	"context"
	"os"
	"testing"
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
	eth_hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
)

// TestTronNileLogMessagePublished is a live-network integration test against
// Tron Nile testnet's eth-compat JSON-RPC. It is skipped in CI; opt in with:
//
//	TRON_INTEGRATION=1 go test ./pkg/watchers/evm/connectors -run TestTronNileLogMessagePublished -v
//
// It probes whether the eth_* surface on Tron is sufficient for the EVM
// watcher to consume LogMessagePublished events from the core bridge deployed
// at TDjYx6vjKPmmiNvgj47YUntbVM1UcpVsGF, using a known publishMessage tx.
func TestTronNileLogMessagePublished(t *testing.T) {
	if os.Getenv("TRON_INTEGRATION") == "" {
		t.Skip("set TRON_INTEGRATION=1 to run (hits live Tron Nile testnet)")
	}

	const (
		rpcURL          = "https://nile.trongrid.io/jsonrpc"
		coreAddrHex     = "0x294b5510a771111df96acbc08515678edf0f83e0" // TDjYx6vjKPmmiNvgj47YUntbVM1UcpVsGF, last 20 bytes
		txHash          = "0xe4a8afa5c1d02a4839c5a97227e414b51bbb1ea9974a29df24da4dc58cde61fd"
		wantEvmChainID  = uint64(3448148188) // 0xcd8690dc, matches INIT_EVM_CHAIN_ID for Nile
		wantNonce       = uint32(1)
		wantConsistency = uint8(202)
		wantPayload     = "hello world"
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethClient.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, wantEvmChainID, chainID.Uint64(), "eth_chainId mismatch")

	coreAddr := ethCommon.HexToAddress(coreAddrHex)

	receipt, err := client.TransactionReceipt(ctx, ethCommon.HexToHash(txHash))
	require.NoError(t, err)
	require.Equal(t, uint64(1), receipt.Status, "tx must be successful")

	filterer, err := ethAbi.NewAbiFilterer(coreAddr, client)
	require.NoError(t, err)

	var found *ethAbi.AbiLogMessagePublished
	for _, log := range receipt.Logs {
		if log.Address != coreAddr {
			continue
		}
		ev, perr := filterer.ParseLogMessagePublished(*log)
		if perr != nil {
			continue
		}
		found = ev
		break
	}
	require.NotNil(t, found, "expected a LogMessagePublished event from %s in tx %s", coreAddrHex, txHash)

	require.Equal(t, wantNonce, found.Nonce, "nonce")
	require.Equal(t, wantConsistency, found.ConsistencyLevel, "consistencyLevel")
	require.Equal(t, wantPayload, string(found.Payload), "payload")

	// Confirms the chain config we want: Finalized must work, Safe must not.
	rpcClient, err := ethRpc.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer rpcClient.Close()

	type marshaller struct {
		Number *eth_hexutil.Big
	}
	var finalized marshaller
	require.NoError(t, rpcClient.CallContext(ctx, &finalized, "eth_getBlockByNumber", "finalized", false))
	require.NotNil(t, finalized.Number, `expected "finalized" tag to return a block`)

	var safe marshaller
	safeErr := rpcClient.CallContext(ctx, &safe, "eth_getBlockByNumber", "safe", false)
	require.Error(t, safeErr, `expected "safe" tag to be unsupported on Tron`)

	t.Logf("sender=%s sequence=%d nonce=%d consistency=%d payload=%q",
		found.Sender.Hex(), found.Sequence, found.Nonce, found.ConsistencyLevel, string(found.Payload))
	t.Logf("finalized block=%s; safe unsupported (err=%v) — chain config should be Finalized:true, Safe:false",
		finalized.Number.String(), safeErr)
}

// TestTronNilePollConnectorLogPolling drives PollConnector's log polling
// end-to-end against the live Tron Nile testnet, starting from a known block
// that contains a publishMessage tx, and verifies the parsed event is
// delivered to the sink.
func TestTronNilePollConnectorLogPolling(t *testing.T) {
	if os.Getenv("TRON_INTEGRATION") == "" {
		t.Skip("set TRON_INTEGRATION=1 to run (hits live Tron Nile testnet)")
	}

	const (
		rpcURL      = "https://nile.trongrid.io/jsonrpc"
		coreAddrHex = "0x294b5510a771111df96acbc08515678edf0f83e0"
		// The known tx is in block 0x3fe4cc8 = 66997448.
		txBlockNum = uint64(66997448)
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()
	coreAddr := ethCommon.HexToAddress(coreAddrHex)

	base, err := NewEthereumBaseConnector(ctx, "tron-nile-test", rpcURL, coreAddr, nil, logger)
	require.NoError(t, err)

	// Cap each scan to a single block so the test only inspects the known
	// block range instead of scanning the (very large) gap up to current latest.
	poll := NewPollConnector(ctx, logger, base, false, 50*time.Millisecond, 1)

	sink := make(chan *ethAbi.AbiLogMessagePublished, 4)
	errC := make(chan error, 1)

	sub, err := poll.watchLogMessagePublishedFrom(ctx, errC, sink, txBlockNum)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	select {
	case ev := <-sink:
		assert.Equal(t, uint32(1), ev.Nonce)
		assert.Equal(t, uint8(202), ev.ConsistencyLevel)
		assert.Equal(t, "hello world", string(ev.Payload))
		t.Logf("received event from poller: nonce=%d seq=%d payload=%q",
			ev.Nonce, ev.Sequence, string(ev.Payload))
	case err := <-errC:
		t.Fatalf("watcher reported error: %v", err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for LogMessagePublished event")
	}
}
