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
	"github.com/stretchr/testify/require"

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
