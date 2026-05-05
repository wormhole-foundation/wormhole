package evm

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// TestTronNileReobservation exercises the same code path as a guardian
// reobservation request: it creates a PollConnector (the HTTP-only connector
// used for Tron), fetches a known publishMessage transaction via
// MessageEventsForTransaction, and asserts the parsed MessagePublication
// matches expected values.
//
// Skipped in CI. Run with:
//
//	TRON_INTEGRATION=1 go test ./pkg/watchers/evm -run TestTronNileReobservation -v
func TestTronNileReobservation(t *testing.T) {
	if os.Getenv("TRON_INTEGRATION") == "" {
		t.Skip("set TRON_INTEGRATION=1 to run (hits live Tron Nile testnet)")
	}

	const (
		rpcURL      = "https://nile.trongrid.io/jsonrpc"
		coreAddrHex = "0x294b5510a771111df96acbc08515678edf0f83e0"
		txHash      = "0xe4a8afa5c1d02a4839c5a97227e414b51bbb1ea9974a29df24da4dc58cde61fd"
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()
	coreAddr := ethCommon.HexToAddress(coreAddrHex)

	// Create the base connector then wrap in PollConnector — same path the
	// watcher takes for HTTP URLs.
	base, err := connectors.NewEthereumBaseConnector(ctx, "tron-nile-test", rpcURL, coreAddr, nil, logger)
	require.NoError(t, err)

	poll := connectors.NewPollConnector(ctx, logger, base, false, time.Second)

	// MessageEventsForTransaction is the exact function used by
	// handleReobservationRequest (reobserve.go).
	receipt, blockNum, msgs, err := MessageEventsForTransaction(
		ctx, poll, coreAddr, vaa.ChainIDTron,
		ethCommon.HexToHash(txHash),
	)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Greater(t, blockNum, uint64(0))
	require.Len(t, msgs, 1, "expected exactly one LogMessagePublished event")

	msg := msgs[0]
	assert.Equal(t, vaa.ChainIDTron, msg.EmitterChain)
	assert.Equal(t, uint32(1), msg.Nonce, "nonce")
	assert.Equal(t, uint8(202), msg.ConsistencyLevel, "consistencyLevel")
	assert.Equal(t, "hello world", string(msg.Payload), "payload")
	assert.Equal(t, uint64(0), msg.Sequence, "sequence")
	assert.False(t, msg.IsReobservation)

	// Verify the emitter address is the 32-byte left-padded sender.
	sender := ethCommon.HexToAddress("0x8F26A0025dcCc6Cfc07A7d38756280a10E295ad7")
	expectedEmitter := PadAddress(sender)
	assert.Equal(t, expectedEmitter, msg.EmitterAddress, "emitter address")

	t.Logf("block=%d emitter=%x nonce=%d seq=%d consistency=%d payload=%q",
		blockNum, msg.EmitterAddress, msg.Nonce, msg.Sequence, msg.ConsistencyLevel, string(msg.Payload))
}

// TestTronNilePollConnectorBlocks verifies that the PollConnector can fetch
// latest and finalized blocks over HTTP.
func TestTronNilePollConnectorBlocks(t *testing.T) {
	if os.Getenv("TRON_INTEGRATION") == "" {
		t.Skip("set TRON_INTEGRATION=1 to run (hits live Tron Nile testnet)")
	}

	const (
		rpcURL      = "https://nile.trongrid.io/jsonrpc"
		coreAddrHex = "0x294b5510a771111df96acbc08515678edf0f83e0"
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()
	coreAddr := ethCommon.HexToAddress(coreAddrHex)

	base, err := connectors.NewEthereumBaseConnector(ctx, "tron-nile-test", rpcURL, coreAddr, nil, logger)
	require.NoError(t, err)

	poll := connectors.NewPollConnector(ctx, logger, base, false, time.Second)

	latest, finalized, safe, err := poll.GetLatest(ctx)
	require.NoError(t, err)

	assert.Greater(t, latest, uint64(0), "latest block should be nonzero")
	assert.Greater(t, finalized, uint64(0), "finalized block should be nonzero")
	assert.Equal(t, finalized, safe, "safe should equal finalized when safe is unsupported")
	assert.GreaterOrEqual(t, latest, finalized, "latest should be >= finalized")

	t.Logf("latest=%d finalized=%d safe=%d gap=%d", latest, finalized, safe, latest-finalized)
}

// TestTronNilePollConnectorLogPolling verifies that the PollConnector's
// WatchLogMessagePublished can discover events via eth_getLogs polling.
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

	base, err := connectors.NewEthereumBaseConnector(ctx, "tron-nile-test", rpcURL, coreAddr, nil, logger)
	require.NoError(t, err)

	// Build a PollConnector but we won't use SubscribeForBlocks — we'll
	// directly test that FilterLogs over HTTP works for the known block range.
	poll := connectors.NewPollConnector(ctx, logger, base, false, time.Second)

	// Use the connector's Client to call FilterLogs for the block containing our tx.
	logs, err := poll.Client().FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(txBlockNum),
		ToBlock:   new(big.Int).SetUint64(txBlockNum),
		Addresses: []ethCommon.Address{coreAddr},
		Topics:    [][]ethCommon.Hash{{LogMessagePublishedTopic}},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(logs), 1, "expected at least one log in the tx block")

	// Parse the first log as LogMessagePublished.
	ev, err := poll.ParseLogMessagePublished(logs[0])
	require.NoError(t, err)
	assert.Equal(t, uint32(1), ev.Nonce)
	assert.Equal(t, uint8(202), ev.ConsistencyLevel)
	assert.Equal(t, "hello world", string(ev.Payload))

	t.Logf("found %d log(s) in block %d; first event: nonce=%d payload=%q", len(logs), txBlockNum, ev.Nonce, string(ev.Payload))
}
