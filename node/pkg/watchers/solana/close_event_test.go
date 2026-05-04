package solana

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Captured from: cargo test test_close_posted_message_capture_fixture -- --nocapture
// See solana/bridge/program/tests/integration.rs
const (
	// The CPI event inner instruction data emitted by close_posted_message.
	// Layout: EVENT_IX_TAG_LE (8) + DISCRIMINATOR (8) + account prefix "msg" (3) + borsh(MessageData).
	capturedCPIEventHex = "e445a52e51cb9a1d9ef61bc2241428b96d736700010000000000000000000000000000000000000000000000000000000000000000000000008782b469a98487e200000000000000000100e4af35976c379d3028ca489e977e4efaf0ca40fc7f5e19909d673f15a5a42370200000000202020202020202020202020202020202020202020202020202020202020202"

	// Real account keys from the captured transaction.
	capturedPayer          = "5TSpbYV3ZfuHciTgLqZJu5Sm77vbym6yoUkUPi1Hq5MZ"
	capturedMessage        = "o6XiTkG4cBwKxKDTA3cxmP5kXRFWNrJ82XCG5CFf8Y7"
	capturedBridge         = "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
	capturedFeeCollector   = "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
	capturedProgram        = "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
	capturedClock          = "SysvarC1ock11111111111111111111111111111111"
	capturedEventAuthority = "GvMcVyV5xsijxCayHJXJyNvbcbqKnLTJF1LJ7BR5Phvw"
	capturedSignature      = "26YFK4phBtW3daXKfF3gjBJok2bBUY9mzrZuY5YovTxK5CnTmAkBQh5kZA9YtUTdLi3HNTCpddMtJGDwzpcwYJQR"
	capturedBlockhash      = "9cfBkPsoQ2NPHYPi7b69bcQG8FKfNc33k2UfRxiPFyd9"
)

func closeEventNewWatcherForTest(t *testing.T, msgC chan<- *common.MessagePublication) *SolanaWatcher {
	t.Helper()

	contractAddress, err := solana.PublicKeyFromBase58(capturedProgram)
	require.NoError(t, err)

	return &SolanaWatcher{
		contract:            contractAddress,
		rawContract:         capturedProgram,
		chainID:             vaa.ChainIDSolana,
		commitment:          rpc.CommitmentFinalized,
		msgC:                msgC,
		networkName:         "solana-test",
		msgObservedLogLevel: zapcore.InfoLevel,
		logger:              zap.NewNop(),
	}
}

func TestParseCloseEventAccountData(t *testing.T) {
	eventData, err := hex.DecodeString(capturedCPIEventHex)
	require.NoError(t, err)

	// Validate header.
	require.True(t, bytes.Equal(eventData[:8], closeEventTag))
	require.True(t, bytes.Equal(eventData[8:16], closeEventDiscriminator))

	// After the 16-byte header, the payload is full account data (prefix + borsh)
	// which ParseMessagePublicationAccount knows how to parse.
	msg, err := ParseMessagePublicationAccount(eventData[16:])
	require.NoError(t, err)
	require.NotNil(t, msg)

	assert.Equal(t, uint8(0), msg.VaaVersion)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.Equal(t, uint16(1), msg.EmitterChain) // Solana
	assert.Equal(t, uint64(0), msg.Sequence)

	// Payload is [2u8; 32] from the Rust test.
	expectedPayload := bytes.Repeat([]byte{0x02}, 32)
	assert.True(t, bytes.Equal(expectedPayload, msg.Payload))

	// Emitter address from the captured test run.
	expectedEmitter, err := hex.DecodeString("e4af35976c379d3028ca489e977e4efaf0ca40fc7f5e19909d673f15a5a42370")
	require.NoError(t, err)
	assert.True(t, bytes.Equal(expectedEmitter, msg.EmitterAddress[:]))

	assert.True(t, msg.SubmissionTime > 0)
	assert.True(t, msg.Nonce > 0)
}

// buildCloseEventTransactionJSON builds a Solana RPC getTransaction-style JSON
// fixture from data captured by the Rust test:
//
//	cargo test-sbf ... test_close_posted_message_capture_fixture -- --nocapture
func buildCloseEventTransactionJSON(t *testing.T) string {
	t.Helper()

	eventData, err := hex.DecodeString(capturedCPIEventHex)
	require.NoError(t, err)
	eventDataB58 := base58.Encode(eventData)
	closeIxDataB58 := base58.Encode([]byte{closePostedMessageInstructionID})

	// Account layout and instruction accounts from the real transaction:
	//   0: payer (writable signer)
	//   1: message (writable non-signer)
	//   2: bridge (writable non-signer)
	//   3: fee_collector (writable non-signer)
	//   4: program (readonly non-signer)
	//   5: clock (readonly non-signer)
	//   6: event_authority (readonly non-signer)
	//
	// instruction accounts=[2, 1, 3, 5, 6, 4]  program_id_index=4
	// inner CPI: accounts=[6]  program_id_index=4
	return fmt.Sprintf(`{
		"blockTime": 1736530812,
		"meta": {
			"computeUnitsConsumed": 50000,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 0,
					"instructions": [
						{
							"accounts": [6],
							"data": "%s",
							"programIdIndex": 4,
							"stackHeight": 2
						}
					]
				}
			],
			"loadedAddresses": { "readonly": [], "writable": [] },
			"logMessages": [],
			"postBalances": [1000000000, 1000000, 1000000, 1000000, 1000000, 1000000, 1000000],
			"postTokenBalances": [],
			"preBalances": [1000000000, 1000000, 1000000, 1000000, 1000000, 1000000, 1000000],
			"preTokenBalances": [],
			"rewards": [],
			"status": { "Ok": null }
		},
		"slot": 100,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 3,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"%s",
					"%s",
					"%s",
					"%s",
					"%s",
					"%s",
					"%s"
				],
				"recentBlockhash": "%s",
				"instructions": [
					{
						"accounts": [2, 1, 3, 5, 6, 4],
						"data": "%s",
						"programIdIndex": 4,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"%s"
			]
		},
		"version": "legacy"
	}`,
		eventDataB58,
		capturedPayer, capturedMessage, capturedBridge, capturedFeeCollector,
		capturedProgram, capturedClock, capturedEventAuthority,
		capturedBlockhash,
		closeIxDataB58,
		capturedSignature,
	)
}

func TestClosePostedMessageDirect(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 10)
	s := closeEventNewWatcherForTest(t, msgC)

	eventJson := buildCloseEventTransactionJSON(t)

	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 1, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	// Find program index.
	var programIndex uint16
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			programIndex = uint16(n) // #nosec G115 -- test code, bounded by account keys length
		}
	}
	require.Equal(t, uint16(4), programIndex)

	// Call processClosePostedMessageEvent directly.
	logger := zap.NewNop()
	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.processClosePostedMessageEvent(
		logger,
		programIndex,
		tx,
		txRpc.Meta.InnerInstructions,
		0,
		tx.Message.Instructions[0],
		alreadyProcessed,
		tx.Signatures[0],
	)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, len(msgC))

	// The CPI event inner instruction (outerIdx=0, innerIdx=0) should be
	// recorded so the outer transaction loop does not re-process it.
	require.Equal(t, 1, len(alreadyProcessed))
	assert.True(t, alreadyProcessed.exists(0, 0))

	msg := <-msgC
	require.NotNil(t, msg)

	// TxID should be the Solana transaction signature.
	expectedSignature := solana.MustSignatureFromBase58(capturedSignature)
	assert.True(t, bytes.Equal(expectedSignature[:], msg.TxID))

	assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain)
	assert.Equal(t, uint64(0), msg.Sequence)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.True(t, msg.IsReobservation)
	assert.False(t, msg.Unreliable)

	// Payload is [2u8; 32] from the Rust test.
	expectedPayload := bytes.Repeat([]byte{0x02}, 32)
	assert.True(t, bytes.Equal(expectedPayload, msg.Payload))

	// Emitter address from the captured test run.
	expectedEmitterAddress, err := vaa.StringToAddress("e4af35976c379d3028ca489e977e4efaf0ca40fc7f5e19909d673f15a5a42370")
	require.NoError(t, err)
	assert.Equal(t, expectedEmitterAddress, msg.EmitterAddress)

	assert.True(t, msg.Timestamp.After(time.Unix(0, 0)))
	assert.True(t, msg.Nonce > 0)
}

func TestClosePostedMessageSkippedDuringNormalProcessing(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 10)
	s := closeEventNewWatcherForTest(t, msgC)

	eventJson := buildCloseEventTransactionJSON(t)

	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	// Process as normal (non-reobservation) — should produce NO observation.
	numObs := s.processTransaction(context.Background(), nil, tx, txRpc.Meta, 100, false)
	assert.Equal(t, uint32(0), numObs)
	assert.Equal(t, 0, len(msgC), "normal processing must not generate observation for close events")

	// Process as reobservation — should produce an observation.
	numObs = s.processTransaction(context.Background(), nil, tx, txRpc.Meta, 100, true)
	assert.Equal(t, uint32(1), numObs)
	assert.Equal(t, 1, len(msgC), "reobservation should generate observation for close events")

	msg := <-msgC
	assert.Equal(t, uint64(0), msg.Sequence)
	assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain)
	assert.True(t, msg.IsReobservation)
}
