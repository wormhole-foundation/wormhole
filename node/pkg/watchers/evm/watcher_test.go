package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethereum "github.com/ethereum/go-ethereum"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const (
	generatedPostMessageFixtureContract = "0x4a8bc80ed5a4067f1ccf107057b8270e0cc11a78"
	realPostMessageFixtureContract      = "0x98f3c9e6e3face36baad05fe09d375ef1464288b"
)

type postMessageFixture struct {
	name                       string
	fileName                   string
	receiptFileName            string
	contract                   eth_common.Address
	checkGeneratedDistribution bool
}

type postMessageFixtureCase struct {
	Comment          string             `json:"comment"`
	MessageSent      *bool              `json:"messageSent,omitempty"`
	Hash             string             `json:"hash,omitempty"`
	Sender           eth_common.Address `json:"Sender"`
	Sequence         uint64             `json:"Sequence"`
	Nonce            uint32             `json:"Nonce"`
	Payload          []byte             `json:"Payload"`
	ConsistencyLevel uint8              `json:"ConsistencyLevel"`
	BlockTime        uint64             `json:"BlockTime,omitempty"`
	Raw              types.Log          `json:"Raw"`
}

func (tc postMessageFixtureCase) event() *ethabi.AbiLogMessagePublished {
	return &ethabi.AbiLogMessagePublished{
		Sender:           tc.Sender,
		Sequence:         tc.Sequence,
		Nonce:            tc.Nonce,
		Payload:          tc.Payload,
		ConsistencyLevel: tc.ConsistencyLevel,
		Raw:              tc.Raw,
	}
}

func postMessageFixtureDir(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to locate watcher_test.go")

	return filepath.Join(filepath.Dir(file), "testdata")
}

func postMessageFixtures(t *testing.T) []postMessageFixture {
	t.Helper()

	return []postMessageFixture{
		{
			name:                       "generated",
			fileName:                   "generated_data.json",
			receiptFileName:            "generated_receipts.json",
			contract:                   eth_common.HexToAddress(generatedPostMessageFixtureContract),
			checkGeneratedDistribution: true,
		},
		{
			name:            "real",
			fileName:        "real_data.json",
			receiptFileName: "real_receipts.json",
			contract:        eth_common.HexToAddress(realPostMessageFixtureContract),
		},
	}
}

func postMessageFixturePath(t *testing.T, fixture postMessageFixture) string {
	t.Helper()
	return filepath.Join(postMessageFixtureDir(t), fixture.fileName)
}

func loadPostMessageFixture(t *testing.T, fixture postMessageFixture) []postMessageFixtureCase {
	t.Helper()

	fixturePath := postMessageFixturePath(t, fixture)
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	var cases []postMessageFixtureCase
	require.NoError(t, json.Unmarshal(data, &cases))
	return cases
}

func loadPostMessageReceiptFixture(t *testing.T, fileName string) []*types.Receipt {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(postMessageFixtureDir(t), fileName))
	require.NoError(t, err)

	var receipts []*types.Receipt
	require.NoError(t, json.Unmarshal(data, &receipts))
	return receipts
}

func postMessageReceiptFixtureByTx(t *testing.T, receipts []*types.Receipt) map[eth_common.Hash]*types.Receipt {
	t.Helper()

	receiptsByTx := map[eth_common.Hash]*types.Receipt{}
	for i, receipt := range receipts {
		require.NotNil(t, receipt, "receipt %d", i)
		_, exists := receiptsByTx[receipt.TxHash]
		require.False(t, exists, "duplicate receipt for tx %s", receipt.TxHash)
		receiptsByTx[receipt.TxHash] = receipt
	}
	return receiptsByTx
}

func fixtureLogsEqual(left, right types.Log) bool {
	if left.Address != right.Address ||
		left.BlockNumber != right.BlockNumber ||
		left.TxHash != right.TxHash ||
		left.TxIndex != right.TxIndex ||
		left.BlockHash != right.BlockHash ||
		left.Index != right.Index ||
		left.Removed != right.Removed ||
		!bytes.Equal(left.Data, right.Data) ||
		len(left.Topics) != len(right.Topics) {
		return false
	}

	for i := range left.Topics {
		if left.Topics[i] != right.Topics[i] {
			return false
		}
	}
	return true
}

func receiptContainsFixtureLog(receipt *types.Receipt, raw types.Log) bool {
	for _, log := range receipt.Logs {
		if log != nil && fixtureLogsEqual(*log, raw) {
			return true
		}
	}
	return false
}

func writePostMessageFixture(t *testing.T, fixture postMessageFixture, cases []postMessageFixtureCase) {
	t.Helper()

	data, err := json.MarshalIndent(cases, "", "  ")
	require.NoError(t, err)
	data = append(data, '\n')

	require.NoError(t, os.WriteFile(postMessageFixturePath(t, fixture), data, 0o644))
}

func fixtureValidationErrors(tc postMessageFixtureCase, contract eth_common.Address) []string {
	var reasons []string
	if tc.Raw.Removed {
		reasons = append(reasons, "removed log")
	}
	if tc.Raw.Address != contract {
		reasons = append(reasons, "wrong raw address")
	}
	if len(tc.Raw.Topics) == 0 {
		reasons = append(reasons, "missing event topic")
	} else if tc.Raw.Topics[0] != LogMessagePublishedTopic {
		reasons = append(reasons, "wrong event topic")
	}
	return reasons
}

func fixtureReceipt(ev *ethabi.AbiLogMessagePublished) *types.Receipt {
	log := ev.Raw
	return &types.Receipt{
		Status:      types.ReceiptStatusSuccessful,
		TxHash:      ev.Raw.TxHash,
		BlockHash:   ev.Raw.BlockHash,
		BlockNumber: new(big.Int).SetUint64(ev.Raw.BlockNumber),
		Logs:        []*types.Log{&log},
	}
}

func (tc postMessageFixtureCase) blockTime() uint64 {
	if tc.BlockTime != 0 {
		return tc.BlockTime
	}
	return testBlockTime
}

func fixtureBlock(ev *ethabi.AbiLogMessagePublished, blockTime uint64, finality connectors.FinalityLevel) *connectors.NewBlock {
	return &connectors.NewBlock{
		Number:   new(big.Int).SetUint64(ev.Raw.BlockNumber),
		Hash:     ev.Raw.BlockHash,
		Time:     blockTime,
		Finality: finality,
	}
}

func fixtureFinality(t *testing.T, consistencyLevel uint8) connectors.FinalityLevel {
	t.Helper()

	switch consistencyLevel {
	case vaa.ConsistencyLevelPublishImmediately:
		return connectors.Latest
	case vaa.ConsistencyLevelSafe:
		return connectors.Safe
	default:
		return connectors.Finalized
	}
}

func TestFixtureFinality(t *testing.T) {
	assert.Equal(t, connectors.Latest, fixtureFinality(t, vaa.ConsistencyLevelPublishImmediately))
	assert.Equal(t, connectors.Safe, fixtureFinality(t, vaa.ConsistencyLevelSafe))
	assert.Equal(t, connectors.Finalized, fixtureFinality(t, vaa.ConsistencyLevelFinalized))
	assert.Equal(t, connectors.Finalized, fixtureFinality(t, vaa.ConsistencyLevelCustom))
	assert.Equal(t, connectors.Finalized, fixtureFinality(t, 1))
}

func TestGeneratedReceiptFixtureMatchesGeneratedData(t *testing.T) {
	generatedFixture := postMessageFixture{
		name:     "generated",
		fileName: "generated_data.json",
		contract: eth_common.HexToAddress(generatedPostMessageFixtureContract),
	}
	cases := loadPostMessageFixture(t, generatedFixture)
	receipts := loadPostMessageReceiptFixture(t, "generated_receipts.json")
	require.Len(t, receipts, len(cases))

	receiptsByTx := postMessageReceiptFixtureByTx(t, receipts)
	for i, receipt := range receipts {
		require.Equal(t, uint8(types.DynamicFeeTxType), receipt.Type, "receipt %d", i)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "receipt %d", i)
		require.NotNil(t, receipt.BlockNumber, "receipt %d", i)
		require.Len(t, receipt.Logs, 1, "receipt %d", i)
		require.Equal(t, types.CreateBloom(types.Receipts{receipt}), receipt.Bloom, "receipt %d", i)

		log := receipt.Logs[0]
		require.NotNil(t, log, "receipt %d log", i)
		require.Equal(t, receipt.TxHash, log.TxHash, "receipt %d tx hash", i)
		require.Equal(t, receipt.BlockHash, log.BlockHash, "receipt %d block hash", i)
		require.Equal(t, receipt.BlockNumber.Uint64(), log.BlockNumber, "receipt %d block number", i)
		require.Equal(t, receipt.TransactionIndex, uint(log.TxIndex), "receipt %d transaction index", i)
	}

	for i, tc := range cases {
		receipt, exists := receiptsByTx[tc.Raw.TxHash]
		require.True(t, exists, "case %d missing receipt for tx %s", i, tc.Raw.TxHash)
		require.Equal(t, tc.Raw.BlockHash, receipt.BlockHash, "case %d block hash", i)
		require.Equal(t, tc.Raw.BlockNumber, receipt.BlockNumber.Uint64(), "case %d block number", i)
		require.Equal(t, uint(tc.Raw.TxIndex), receipt.TransactionIndex, "case %d transaction index", i)
		require.Equal(t, tc.Raw, *receipt.Logs[0], "case %d receipt log", i)
	}
}

func TestRealReceiptFixtureMatchesRealData(t *testing.T) {
	realFixture := postMessageFixture{
		name:     "real",
		fileName: "real_data.json",
		contract: eth_common.HexToAddress(realPostMessageFixtureContract),
	}
	cases := loadPostMessageFixture(t, realFixture)
	receipts := loadPostMessageReceiptFixture(t, "real_receipts.json")
	require.Len(t, receipts, len(cases))

	receiptsByTx := postMessageReceiptFixtureByTx(t, receipts)
	for i, receipt := range receipts {
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "receipt %d", i)
		require.NotNil(t, receipt.BlockNumber, "receipt %d", i)
		require.NotEmpty(t, receipt.Logs, "receipt %d", i)
		require.Equal(t, types.CreateBloom(types.Receipts{receipt}), receipt.Bloom, "receipt %d", i)
	}

	for i, tc := range cases {
		receipt, exists := receiptsByTx[tc.Raw.TxHash]
		require.True(t, exists, "case %d missing receipt for tx %s", i, tc.Raw.TxHash)
		require.Equal(t, tc.Raw.BlockHash, receipt.BlockHash, "case %d block hash", i)
		require.Equal(t, tc.Raw.BlockNumber, receipt.BlockNumber.Uint64(), "case %d block number", i)
		require.Equal(t, uint(tc.Raw.TxIndex), receipt.TransactionIndex, "case %d transaction index", i)
		require.True(t, receiptContainsFixtureLog(receipt, tc.Raw), "case %d receipt missing fixture log", i)
	}
}

func assertOrSeedFixtureMessageSent(t *testing.T, tc *postMessageFixtureCase, messageSent bool) bool {
	t.Helper()

	if tc.MessageSent == nil {
		tc.MessageSent = new(bool)
		*tc.MessageSent = messageSent
		return true
	}

	require.Equal(t, *tc.MessageSent, messageSent)
	return false
}

func assertOrSeedFixtureHash(t *testing.T, tc *postMessageFixtureCase, msg *common.MessagePublication) bool {
	t.Helper()

	hash := msg.CreateDigest()
	if tc.Hash == "" {
		tc.Hash = hash
		return true
	}

	require.Equal(t, tc.Hash, hash)
	return false
}

func TestMsgIdFromLogEvent(t *testing.T) {
	evJson := `
		{
		"Sender": "0x45c140dd2526e4bfd1c2a5bb0aa6aa1db00b1744",
		"Sequence": 3685,
		"Nonce": 0,
		"Payload": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJxMAwy+TX7P/UQKg5Siin3wZuTKLmUV0DFAtns2oZ5XBIkIUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWnn535GP/6Gswr9FgWgmmMr6lsBQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPd52OGwfx498UGoHE8ffWXAo4YRAAAAAAAAAAAAAAAHmBsAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAvVKf9zDa4Cn6hbONmNYEZyEhX6QUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAC9Up/3MNrgKfqFs42Y1gRnISFfpBQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACsblSHFxAb/NAsujjz79eA6AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHBmtsmDcAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA40KOnP4ABQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOZ6vaDUP3rI83h2u/ANHfrbuTqqAFU9+gAAAVQAAAArG5UhxcQG/zQLLo48+/XgOgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABgAAAAAAAAAAAAAAAAyL8tXA1r7IB8Ie9M7y8f078WlH4AAAAAAAAAAAAAAACUnABm1c8iBqanyHJ7Dwt3ceUclgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"ConsistencyLevel": 15,
		"Raw": {
			"address": "0x4a8bc80ed5a4067f1ccf107057b8270e0cc11a78",
			"topics": [
				"0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2",
				"0x00000000000000000000000045c140dd2526e4bfd1c2a5bb0aa6aa1db00b1744"
			],
			"data": "0x0000000000000000000000000000000000000000000000000000000000000e6500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000f0000000000000000000000000000000000000000000000000000000000000393000000000000000000000000000000000000000000000000000000000000271300c32f935fb3ff5102a0e528a29f7c19b9328b9945740c502d9ecda86795c12242140000000000000000000000000000000000000000000000000000000000000000000000000000000000000000169e7e77e463ffe86b30afd1605a09a632bea5b0140000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f779d8e1b07f1e3df141a81c4f1f7d65c0a38611000000000000000000000007981b00140000000000000000000000000000000000000000000000000000000000000000000000000000000000000000bd529ff730dae029fa85b38d98d6046721215fa4140000000000000000000000000000000000000000000000000000000000000000000000000000000000000000bd529ff730dae029fa85b38d98d6046721215fa41400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002b1b9521c5c406ff340b2e8e3cfbf5e03a000000000000000000000000000000000000000000000000000007066b6c983700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000038d0a3a73f800140000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e67abda0d43f7ac8f37876bbf00d1dfadbb93aaa00553dfa000001540000002b1b9521c5c406ff340b2e8e3cfbf5e03a0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000010400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000060000000000000000000000000c8bf2d5c0d6bec807c21ef4cef2f1fd3bf16947e000000000000000000000000949c0066d5cf2206a6a7c8727b0f0b7771e51c96000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			"blockNumber": "0x553dfa",
			"transactionHash": "0xb198a854efdae67684cd840795ddcadeabdfdba83bb1cbf14a3f2debac1fd1f6",
			"transactionIndex": "0x78",
			"blockHash": "0xfd4e19ca93de700470f2e6cdbd6fb67ba9e3e1508bd23289bc4f795ac641c375",
			"logIndex": "0x4d",
			"removed": false
		}
	}`

	var ev ethabi.AbiLogMessagePublished
	err := json.Unmarshal([]byte(evJson), &ev)
	require.NoError(t, err)
	msgId := msgIdFromLogEvent(vaa.ChainIDSepolia, &ev)
	assert.Equal(t, "10002/00000000000000000000000045c140dd2526e4bfd1c2a5bb0aa6aa1db00b1744/3685", msgId)
}

func Test_canRetryGetBlockTime(t *testing.T) {
	assert.True(t, canRetryGetBlockTime(ethereum.NotFound))
	assert.True(t, canRetryGetBlockTime(errors.New("not found")))
	assert.True(t, canRetryGetBlockTime(errors.New("Unknown block")))
	assert.True(t, canRetryGetBlockTime(errors.New("cannot query unfinalized data")))
	assert.False(t, canRetryGetBlockTime(errors.New("Hello, World!")))
}

// TestVerifyAndPublish checks the operation of the verifyAndPublish method of the watcher in
// scenarios where the Transfer Verifier is disabled and when it's enabled. It covers much of
// the behaviour of the verify() function.
func TestVerifyAndPublish(t *testing.T) {

	msgC := make(chan *common.MessagePublication, 1)
	w := NewWatcherForTest(t, msgC)

	// Contents of the message don't matter for the sake of these tests.
	msg := common.MessagePublication{}
	ctx := context.TODO()

	// Check preconditions for the Transfer Verifier disabled case.
	require.Equal(t, 0, len(w.msgC))
	require.Equal(t, common.NotVerified.String(), msg.VerificationState().String())
	require.Nil(t, w.txVerifier)

	// Check nil message
	err := w.verifyAndPublish(nil, ctx, eth_common.Hash{}, &types.Receipt{})
	require.ErrorContains(t, err, "message publication cannot be nil")
	require.Equal(t, common.NotVerified.String(), msg.VerificationState().String())

	// Check transfer verifier not enabled case. The message should be published normally.
	msg = common.MessagePublication{}
	require.Nil(t, w.txVerifier)

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.NoError(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg := recvMsg(t, msgC)
	require.NotNil(t, publishedMsg)
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.NotVerified.String(), publishedMsg.VerificationState().String())

	tbAddr := PadAddress(testTokenBridge)

	// Check scenario where transfer verifier is enabled on the watcher level but
	// there is no Transfer Verifier instantiated. In this case, fail open and continue
	// to process messages. This shouldn't be possible in practice as the constructor
	// should return an error on startup if the Transfer Verifier can't be instantiated
	// when txVerifierEnabled is true.
	w.txVerifierEnabled = true
	msg = common.MessagePublication{}
	require.Nil(t, w.txVerifier)

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.NoError(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg = recvMsg(t, msgC)
	require.Equal(t, common.NotVerified.String(), publishedMsg.VerificationState().String())

	// Check that message status is not changed if it didn't come from token bridge.
	// The NotVerified status is used when Transfer Verification is not enabled.
	msg = common.MessagePublication{}
	require.Nil(t, w.txVerifier)

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.Nil(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg = recvMsg(t, msgC)
	require.Equal(t, common.NotVerified.String(), publishedMsg.VerificationState().String())

	// Check scenario where the message already has a verification status.
	failMock := &MockTransferVerifier[ethclient.Client, connectors.Connector]{false}
	w.txVerifier = failMock
	msg = common.MessagePublication{}
	setErr := msg.SetVerificationState(common.Anomalous)
	require.NoError(t, setErr)
	require.NotNil(t, w.txVerifier)

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.ErrorContains(t, err, "MessagePublication already has a non-default verification state")
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.Anomalous.String(), msg.VerificationState().String())

	// Check case where Transfer Verifier finds a dangerous transaction. Note that this case does
	// not return an error, but the published message should be marked as Rejected.
	failMock = &MockTransferVerifier[ethclient.Client, connectors.Connector]{false}
	w.txVerifier = failMock
	require.NotNil(t, w.txVerifier)
	msg = common.MessagePublication{
		EmitterAddress: tbAddr,
	}

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.Nil(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg = recvMsg(t, msgC)
	require.NotNil(t, publishedMsg)
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.Rejected.String(), publishedMsg.VerificationState().String())

	// Check that message status is not changed if it didn't come from token bridge.
	// The NotApplicable status is used when Transfer Verification is enabled.
	msg = common.MessagePublication{}
	require.NotNil(t, w.txVerifier)

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.Nil(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg = recvMsg(t, msgC)
	require.Equal(t, common.NotApplicable.String(), publishedMsg.VerificationState().String())

	// Check happy path where txverifier is enabled, initialized, and the message is from the token bridge.
	successMock := &MockTransferVerifier[ethclient.Client, connectors.Connector]{true}
	w.txVerifier = successMock
	require.NotNil(t, w.txVerifier)
	msg = common.MessagePublication{
		EmitterAddress: tbAddr,
	}

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.NoError(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg = recvMsg(t, msgC)
	require.NotNil(t, publishedMsg)
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.Valid.String(), publishedMsg.VerificationState().String())
}

// TestVerifyDoesNotMutateOriginalMessage checks that verify() does not modify
// the original MessagePublication passed to it.
func TestVerifyDoesNotMutateOriginalMessage(t *testing.T) {
	tbAddr := PadAddress(testTokenBridge)

	msg := &common.MessagePublication{
		EmitterAddress: tbAddr,
	}
	require.Equal(t, common.NotVerified.String(), msg.VerificationState().String())

	successMock := &MockTransferVerifier[ethclient.Client, connectors.Connector]{true}
	ctx := context.TODO()

	result, err := verify(ctx, msg, eth_common.Hash{}, &types.Receipt{}, successMock)
	require.NoError(t, err)

	// The returned copy should have the updated verification state.
	require.Equal(t, common.Valid.String(), result.VerificationState().String())

	// The original message must remain unmodified.
	require.Equal(t, common.NotVerified.String(), msg.VerificationState().String())
}

// Several test cases for a pending message getting processed depending on the block number
func TestProcessBlockPendingByFinality(t *testing.T) {
	tests := []struct {
		name          string
		cl            uint8
		finality      connectors.FinalityLevel
		blockNumber   uint64
		expectPending int
		expectPublish bool
	}{
		{"finalized", vaa.ConsistencyLevelFinalized, connectors.Finalized, 105, 0, true},
		{"safe", vaa.ConsistencyLevelSafe, connectors.Safe, 105, 0, true},
		{"instant", vaa.ConsistencyLevelPublishImmediately, connectors.Latest, 105, 0, true},
		{"finalized_before_block", vaa.ConsistencyLevelFinalized, connectors.Finalized, 99, 1, false},
		{"finalized_with_safe_block", vaa.ConsistencyLevelFinalized, connectors.Safe, 105, 1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, mock, msgC := newTestWatcher(t)
			txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
			blockHash := eth_common.BigToHash(big.NewInt(100))

			w.addPendingMsg(txHash, blockHash, tc.cl, 0, 1)
			mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHash}

			err := w.processNewBlock(context.TODO(), newBlock(tc.blockNumber, tc.finality), &gossipv1.Heartbeat_Network{})
			require.NoError(t, err)

			assert.Equal(t, tc.expectPending, len(w.pending))
			if tc.expectPublish {
				require.Equal(t, 1, len(msgC))
				assert.Equal(t, txHash.Bytes(), recvMsg(t, msgC).TxID)
			} else {
				assert.Equal(t, 0, len(msgC))
			}
		})
	}
}

// Handling both a finalized and a safe message
func TestProcessBlockPendingFinalizedAndSafe(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	txHash1 := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	txHash2 := eth_common.HexToHash("0xe2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ee")
	blockHash1 := eth_common.BigToHash(big.NewInt(111))
	blockHash2 := eth_common.BigToHash(big.NewInt(222))

	w.addPendingMsg(txHash1, blockHash1, vaa.ConsistencyLevelFinalized, 0, 1)
	w.addPendingMsg(txHash2, blockHash2, vaa.ConsistencyLevelSafe, 0, 1)
	mock.receipts[txHash1] = &types.Receipt{Status: 1, BlockHash: blockHash1, TxHash: txHash1}
	mock.receipts[txHash2] = &types.Receipt{Status: 1, BlockHash: blockHash2, TxHash: txHash2}

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed one from pending
	assert.Equal(t, 1, len(w.pending))

	// Published finalized message
	require.Equal(t, 1, len(msgC))
	assert.Equal(t, txHash1.Bytes(), recvMsg(t, msgC).TxID)

	err = w.processNewBlock(context.TODO(), newBlock(105, connectors.Safe), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed both
	assert.Equal(t, 0, len(w.pending))

	// Published safe message
	require.Equal(t, 1, len(msgC))
	assert.Equal(t, txHash2.Bytes(), recvMsg(t, msgC).TxID)
}

// Removal of the message without publication if the blockhash differs
func TestProcessBlockPendingWrongBlockHash(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHashBlock := eth_common.BigToHash(big.NewInt(111))
	blockHashMessage := eth_common.BigToHash(big.NewInt(222))

	w.addPendingMsg(txHash, blockHashMessage, vaa.ConsistencyLevelFinalized, 0, 1)
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHashBlock, TxHash: txHash}

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed from pending
	assert.Equal(t, 0, len(w.pending))

	// Not published
	assert.Equal(t, 0, len(msgC))
}

// Failed transaction status gets rejected
func TestProcessBlockPendingFailedTx(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHash := eth_common.BigToHash(big.NewInt(100))

	w.addPendingMsg(txHash, blockHash, vaa.ConsistencyLevelFinalized, 0, 1)
	mock.receipts[txHash] = &types.Receipt{Status: 0, BlockHash: blockHash, TxHash: txHash}

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed from pending
	assert.Equal(t, 0, len(w.pending))

	// Not published
	assert.Equal(t, 0, len(msgC))
}

// Failed receipt test case
func TestProcessBlockValidReceiptWithError(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHash := eth_common.BigToHash(big.NewInt(100))

	w.addPendingMsg(txHash, blockHash, vaa.ConsistencyLevelFinalized, 0, 1)
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHash}
	mock.errors[txHash] = errors.New("not found")

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed from pending
	assert.Equal(t, 0, len(w.pending))

	// Not published
	assert.Equal(t, 0, len(msgC))
}

// No receipt is found. This should remove from the pending list.
func TestProcessBlockInValidReceiptNoError(t *testing.T) {
	w, _, msgC := newTestWatcher(t)
	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHash := eth_common.BigToHash(big.NewInt(100))

	w.addPendingMsg(txHash, blockHash, vaa.ConsistencyLevelFinalized, 0, 1)

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed from pending
	assert.Equal(t, 0, len(w.pending))

	// Not published
	assert.Equal(t, 0, len(msgC))
}

// Transient errors on receipt RPC requests should be retried
func TestProcessBlockTransientError(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHash := eth_common.BigToHash(big.NewInt(100))
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHash}
	mock.errors[txHash] = errors.New("rate limit 429 error")

	w.addPendingMsg(txHash, blockHash, vaa.ConsistencyLevelFinalized, 0, 1)

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Not removed from pending
	assert.Equal(t, 1, len(w.pending))

	// Not published
	assert.Equal(t, 0, len(msgC))

	// Try the message again after the transient error has disappeared
	mock.errors[txHash] = nil
	err = w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed from pending
	assert.Equal(t, 0, len(w.pending))

	// Published with correct TxID
	require.Equal(t, 1, len(msgC))
	assert.Equal(t, txHash.Bytes(), recvMsg(t, msgC).TxID)
}

// A transient error that persists past MaxWaitConfirmations should remove the pending entry.
func TestProcessBlockTransientErrorTimeout(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHash := eth_common.BigToHash(big.NewInt(100))
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHash}
	mock.errors[txHash] = errors.New("transient error")

	w.addPendingMsg(txHash, blockHash, vaa.ConsistencyLevelFinalized, 0, 1)

	// blockNumberU = pLock.height + MaxWaitConfirmations exactly hits the timeout branch.
	err := w.processNewBlock(context.TODO(), newBlock(100+MaxWaitConfirmations, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed from pending after timeout
	assert.Equal(t, 0, len(w.pending))

	// Not published
	assert.Equal(t, 0, len(msgC))
}

// AdditionalBlocks test cases for waiting the proper amount of time before publication
func TestProcessBlockAdditionalBlocks(t *testing.T) {
	tests := []struct {
		name             string
		cl               uint8
		finality         connectors.FinalityLevel
		additionalBlocks uint64
		blockNumber      uint64
		expectPending    int
		expectPublish    bool
	}{
		{"before", vaa.ConsistencyLevelFinalized, connectors.Finalized, 20, 100, 1, false},
		{"before_by_one", vaa.ConsistencyLevelFinalized, connectors.Finalized, 20, 119, 1, false},
		{"exact", vaa.ConsistencyLevelFinalized, connectors.Finalized, 20, 120, 0, true},
		{"none", vaa.ConsistencyLevelPublishImmediately, connectors.Latest, 0, 100, 0, true},
		{"maximum", vaa.ConsistencyLevelPublishImmediately, connectors.Latest, 65535, 100 + 65535, 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, mock, msgC := newTestWatcher(t)
			txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
			blockHash := eth_common.BigToHash(big.NewInt(100))

			w.addPendingMsg(txHash, blockHash, tc.cl, tc.additionalBlocks, 1)
			mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHash}

			err := w.processNewBlock(context.TODO(), newBlock(tc.blockNumber, tc.finality), &gossipv1.Heartbeat_Network{})
			require.NoError(t, err)

			assert.Equal(t, tc.expectPending, len(w.pending))
			if tc.expectPublish {
				require.Equal(t, 1, len(msgC))
				assert.Equal(t, txHash.Bytes(), recvMsg(t, msgC).TxID)
			} else {
				assert.Equal(t, 0, len(msgC))
			}
		})
	}
}

// Effective consistency level (CL) and the VAA CL should differ.
func TestProcessBlockCCLEffectiveCLDiffersFromMessageCL(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHash := eth_common.BigToHash(big.NewInt(100))

	// Simulate CCL: message.ConsistencyLevel = Custom, but effectiveCL = Finalized
	key := w.addPendingMsg(txHash, blockHash, vaa.ConsistencyLevelFinalized, 5, 1)
	w.pending[key].message.ConsistencyLevel = vaa.ConsistencyLevelCustom
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHash}

	// Finalized block at height+additionalBlocks: should confirm based on effectiveCL
	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Removed from pending and published
	assert.Equal(t, 0, len(w.pending))
	require.Equal(t, 1, len(msgC))

	// The published message must still have ConsistencyLevelCustom for VAA hash consistency
	published := recvMsg(t, msgC)
	assert.Equal(t, vaa.ConsistencyLevelCustom, published.ConsistencyLevel)
}

// Pending messages with different finalizations and block times
func TestProcessBlockCCLMultiplePendingDifferentAdditionalBlocks(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	txHashA := eth_common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	txHashB := eth_common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	blockHash := eth_common.BigToHash(big.NewInt(100))

	// Message A: effectiveCL=Finalized, additionalBlocks=0, height=100 -> ready at block 100
	w.addPendingMsg(txHashA, blockHash, vaa.ConsistencyLevelFinalized, 0, 1)
	mock.receipts[txHashA] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHashA}

	// Message B: effectiveCL=Finalized, additionalBlocks=10, height=100 -> ready at block 110
	w.addPendingMsg(txHashB, blockHash, vaa.ConsistencyLevelFinalized, 10, 2)
	mock.receipts[txHashB] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHashB}

	// Send finalized block at 105: A should confirm, B should stay pending
	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	assert.Equal(t, 1, len(w.pending), "only message B should remain pending")
	require.Equal(t, 1, len(msgC), "only message A should be published")
	assert.Equal(t, txHashA.Bytes(), recvMsg(t, msgC).TxID)

	// Send finalized block at 110: B should now confirm
	err = w.processNewBlock(context.TODO(), newBlock(110, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	assert.Equal(t, 0, len(w.pending), "no messages should remain pending")
	require.Equal(t, 1, len(msgC), "message B should now be published")
	assert.Equal(t, txHashB.Bytes(), recvMsg(t, msgC).TxID)
}

// Orphaned tx handling
func TestProcessBlockOneConfirmedOneOrphaned(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	blockHash := eth_common.BigToHash(big.NewInt(100))

	txHashGood := eth_common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	txHashOrphaned := eth_common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	w.addPendingMsg(txHashGood, blockHash, vaa.ConsistencyLevelFinalized, 0, 1)
	w.addPendingMsg(txHashOrphaned, blockHash, vaa.ConsistencyLevelFinalized, 0, 2)

	// Good tx has a valid receipt
	mock.receipts[txHashGood] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHashGood}

	// Orphaned tx returns nil receipt (not in mock.receipts, so defaults to nil)
	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
	require.NoError(t, err)

	// Both removed from pending
	assert.Equal(t, 0, len(w.pending))

	// Only the valid message was published
	assert.Equal(t, 1, len(msgC))

	published := recvMsg(t, msgC)
	assert.Equal(t, txHashGood.Bytes(), published.TxID)
}

// Invalid finality level process. Should return an error.
func TestProcessBlockUnexpectedFinality(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	err := w.processNewBlock(context.TODO(), newBlock(100, connectors.FinalityLevel(99)), &gossipv1.Heartbeat_Network{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected finality in block")

	// Nothing published
	assert.Equal(t, 0, len(msgC))
}

// TxVerifier support in processBlock
func TestProcessBlockTxVerifier(t *testing.T) {
	tests := []struct {
		name           string
		success        bool
		useTokenBridge bool
		expectState    common.VerificationState
	}{
		{"success", true, true, common.Valid},
		{"failure", false, true, common.Rejected},
		{"not_applicable", false, false, common.NotApplicable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, mock, msgC := newTestWatcher(t)
			w.txVerifier = &MockTransferVerifier[ethclient.Client, connectors.Connector]{success: tc.success}

			txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
			blockHash := eth_common.BigToHash(big.NewInt(100))

			key := w.addPendingMsg(txHash, blockHash, vaa.ConsistencyLevelFinalized, 0, 1)
			if tc.useTokenBridge {
				w.pending[key].message.EmitterAddress = PadAddress(testTokenBridge)
			}
			mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash, TxHash: txHash}

			err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized), &gossipv1.Heartbeat_Network{})
			require.NoError(t, err)

			assert.Equal(t, 0, len(w.pending))
			require.Equal(t, 1, len(msgC))

			msg := recvMsg(t, msgC)
			assert.Equal(t, txHash.Bytes(), msg.TxID)
			assert.Equal(t, tc.expectState, msg.VerificationState())
		})
	}
}

// Safe and finalized add to pending in postMessage
func TestPostMessageAddsToPending(t *testing.T) {
	tests := []struct {
		name string
		cl   uint8
	}{
		{"finalized", vaa.ConsistencyLevelFinalized},
		{"safe", vaa.ConsistencyLevelSafe},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, _, msgC := newTestWatcher(t)

			ev := newTestLogEvent(tc.cl)
			w.postMessage(context.TODO(), ev, 1234)

			require.Equal(t, 1, len(w.pending))
			assert.Equal(t, 0, len(msgC))

			key := pendingKey{
				TxHash:         ev.Raw.TxHash,
				BlockHash:      ev.Raw.BlockHash,
				EmitterAddress: PadAddress(ev.Sender),
				Sequence:       ev.Sequence,
			}
			pe := w.pending[key]
			require.NotNil(t, pe)

			assertMessageMatchesEvent(t, pe.message, ev)
			assertPendingMetadata(t, pe, tc.cl, 0)
		})
	}
}

// Custom consistency level (CCL) edge cases lead to finalized by default
func TestPostMessageCustomDefaultToFinalized(t *testing.T) {
	tests := []struct {
		name     string
		setupCCL func(w *Watcher)
	}{
		{"ccl_disabled", nil},
		{"ccl_enabled_nothing_special", func(w *Watcher) {
			w.enableCCL()
			w.seedCCLNothingSpecial(testEmitter)
		}},
		{"invalid_additional_blocks_config", func(w *Watcher) {
			w.enableCCL()
			w.seedCCLAdditionalBlocks(testEmitter, 1, 1)
		}},
		{"invalid_type", func(w *Watcher) {
			w.enableCCL()
			w.seedCCLInvalidType(testEmitter)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, _, msgC := newTestWatcher(t)

			if tc.setupCCL != nil {
				tc.setupCCL(w)
			}

			ev := newTestLogEvent(vaa.ConsistencyLevelCustom)
			w.postMessage(context.TODO(), ev, 1234)

			require.Equal(t, 1, len(w.pending))
			assert.Equal(t, 0, len(msgC))

			key := pendingKey{
				TxHash:         ev.Raw.TxHash,
				BlockHash:      ev.Raw.BlockHash,
				EmitterAddress: PadAddress(ev.Sender),
				Sequence:       ev.Sequence,
			}
			pe := w.pending[key]
			require.NotNil(t, pe)
			assertMessageMatchesEvent(t, pe.message, ev)
			assertPendingMetadata(t, pe, vaa.ConsistencyLevelFinalized, 0)
		})
	}
}

// AdditionalBlocks basic testing
func TestPostMessageCustomAdditionalBlocks(t *testing.T) {
	tests := []struct {
		name             string
		effectiveCL      uint8
		additionalBlocks uint16
	}{
		{"finalized", vaa.ConsistencyLevelFinalized, 101},
		{"safe", vaa.ConsistencyLevelSafe, 50},
		{"instant", vaa.ConsistencyLevelPublishImmediately, 10},
		{"zero_blocks", vaa.ConsistencyLevelFinalized, 0},
		{"one_block", vaa.ConsistencyLevelFinalized, 1},
		{"small_blocks", vaa.ConsistencyLevelFinalized, 5},
		{"medium_blocks", vaa.ConsistencyLevelFinalized, 500},
		{"large_blocks", vaa.ConsistencyLevelFinalized, 10000},
		{"max_uint16", vaa.ConsistencyLevelFinalized, 0xFFFF},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, _, msgC := newTestWatcher(t)

			w.enableCCL()
			w.seedCCLAdditionalBlocks(testEmitter, tc.effectiveCL, tc.additionalBlocks)
			ev := newTestLogEvent(vaa.ConsistencyLevelCustom)
			w.postMessage(context.TODO(), ev, 1234)

			require.Equal(t, 1, len(w.pending))
			assert.Equal(t, 0, len(msgC))

			key := pendingKey{
				TxHash:         ev.Raw.TxHash,
				BlockHash:      ev.Raw.BlockHash,
				EmitterAddress: PadAddress(ev.Sender),
				Sequence:       ev.Sequence,
			}
			pe := w.pending[key]
			require.NotNil(t, pe)
			assertMessageMatchesEvent(t, pe.message, ev)
			assertPendingMetadata(t, pe, tc.effectiveCL, uint64(tc.additionalBlocks))
		})
	}
}

// Instant message is published instead of being added to the pending queue
func TestPostMessageInstantPublishes(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	ev := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)
	mock.receipts[ev.Raw.TxHash] = newTestReceipt(ev.Raw.BlockNumber, nil)
	w.postMessage(context.TODO(), ev, 1234)

	// Should be published instead of being added to the pending queue
	require.Equal(t, 0, len(w.pending))
	assert.Equal(t, 1, len(msgC))

	msg := recvMsg(t, msgC)

	assertMessageMatchesEvent(t, msg, ev)
	assert.Equal(t, common.NotVerified, msg.VerificationState())
}

// Multiple instant publishes are handled properly
func TestPostMessageTwoInstantPublishes(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	ev1 := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)

	ev2 := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)
	ev2.Sender = eth_common.HexToAddress("0x388C818CA8B9251b393131C08a736A67ccB19297")
	ev2.Nonce = 20
	ev2.Sequence = 2

	mock.receipts[ev1.Raw.TxHash] = newTestReceipt(ev1.Raw.BlockNumber, nil)
	mock.receipts[ev2.Raw.TxHash] = newTestReceipt(ev2.Raw.BlockNumber, nil)
	w.postMessage(context.TODO(), ev1, 1234)
	w.postMessage(context.TODO(), ev2, 1234)

	// Should be added to pending, not published immediately
	require.Equal(t, 0, len(w.pending))
	assert.Equal(t, 2, len(msgC))

	msg1 := recvMsg(t, msgC)
	msg2 := recvMsg(t, msgC)

	assertMessageMatchesEvent(t, msg1, ev1)
	assert.Equal(t, common.NotVerified, msg1.VerificationState())

	assertMessageMatchesEvent(t, msg2, ev2)
	assert.Equal(t, common.NotVerified, msg2.VerificationState())
}

// An instant and a final message go to the proper spots (msgC and pending)
func TestPostMessageInstantAndFinalized(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	// ev1: instant publish
	ev1 := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)

	// ev2: finalized (goes to pending) — use a different emitter and sequence
	ev2 := newTestLogEventFromParams(testLogEventParams{
		sender:           eth_common.HexToAddress("0x388C818CA8B9251b393131C08a736A67ccB19297"),
		sequence:         2,
		blockNumber:      100,
		consistencyLevel: vaa.ConsistencyLevelFinalized,
	})

	mock.receipts[ev1.Raw.TxHash] = newTestReceipt(ev1.Raw.BlockNumber, nil)
	w.postMessage(context.TODO(), ev1, 1234)
	w.postMessage(context.TODO(), ev2, 1234)

	// One message published immediately, one added to pending.
	require.Equal(t, 1, len(w.pending))
	assert.Equal(t, 1, len(msgC))

	// Verify the instant-published message
	msg := recvMsg(t, msgC)
	assertMessageMatchesEvent(t, msg, ev1)
	assert.Equal(t, common.NotVerified, msg.VerificationState())

	// Verify the finalized pending entry
	key := pendingKey{
		TxHash:         ev2.Raw.TxHash,
		BlockHash:      ev2.Raw.BlockHash,
		EmitterAddress: PadAddress(ev2.Sender),
		Sequence:       ev2.Sequence,
	}
	pe := w.pending[key]
	require.NotNil(t, pe)

	assertMessageMatchesEvent(t, pe.message, ev2)
	assertPendingMetadata(t, pe, vaa.ConsistencyLevelFinalized, 0)
}

// Transaction contains multiple events
func TestPostMessageMultipleEventsFromSameTransaction(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHash := eth_common.HexToHash("0xa1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1")

	ev1 := newTestLogEventFromParams(testLogEventParams{
		sender:           testEmitter,
		sequence:         1,
		blockNumber:      100,
		consistencyLevel: vaa.ConsistencyLevelFinalized,
		txHash:           txHash,
		blockHash:        blockHash,
	})
	ev2 := newTestLogEventFromParams(testLogEventParams{
		sender:           testEmitter,
		sequence:         2,
		blockNumber:      100,
		consistencyLevel: vaa.ConsistencyLevelFinalized,
		txHash:           txHash,
		blockHash:        blockHash,
	})

	w.postMessage(context.TODO(), ev1, 1234)
	w.postMessage(context.TODO(), ev2, 1234)

	assert.Equal(t, 0, len(msgC))
	require.Equal(t, 2, len(w.pending))

	key1 := pendingKey{
		TxHash:         txHash,
		BlockHash:      blockHash,
		EmitterAddress: PadAddress(testEmitter),
		Sequence:       1,
	}
	key2 := pendingKey{
		TxHash:         txHash,
		BlockHash:      blockHash,
		EmitterAddress: PadAddress(testEmitter),
		Sequence:       2,
	}

	pe1 := w.pending[key1]
	pe2 := w.pending[key2]
	require.NotNil(t, pe1)
	require.NotNil(t, pe2)

	assert.Equal(t, txHash.Bytes(), pe1.message.TxID)
	assert.Equal(t, uint64(1), pe1.message.Sequence)
	assert.Equal(t, txHash.Bytes(), pe2.message.TxID)
	assert.Equal(t, uint64(2), pe2.message.Sequence)
}

func TestPostMessageRemovedLogIsIgnored(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	ev := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)
	ev.Raw.Removed = true

	w.postMessage(context.TODO(), ev, 1234)

	assert.Equal(t, 0, len(msgC), "removed log should not be published to msgC")
	assert.Equal(t, 0, len(w.pending), "removed log should not be added to pending")
}

func TestPostMessageWrongContractAddressIsIgnored(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	ev := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)
	ev.Raw.Address = eth_common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	w.postMessage(context.TODO(), ev, 1234)

	assert.Equal(t, 0, len(msgC), "log from wrong contract should not be published to msgC")
	assert.Equal(t, 0, len(w.pending), "log from wrong contract should not be added to pending")
}

func TestPostMessageWrongEventSignatureIsIgnored(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	ev := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)
	ev.Raw.Topics = []eth_common.Hash{
		eth_common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		eth_common.BytesToHash(testEmitter.Bytes()),
	}

	w.postMessage(context.TODO(), ev, 1234)

	assert.Equal(t, 0, len(msgC), "log with wrong event signature should not be published to msgC")
	assert.Equal(t, 0, len(w.pending), "log with wrong event signature should not be added to pending")
}

func TestPostMessageGeneratedFixture(t *testing.T) {
	for _, fixture := range postMessageFixtures(t) {
		fixture := fixture

		t.Run(fixture.name, func(t *testing.T) {
			cases := loadPostMessageFixture(t, fixture)
			require.Len(t, cases, 200)
			var correctAddress, correctTopic, removed uint
			wrongCounts := map[int]int{}
			fixtureUpdated := false

			for i := range cases {
				tc := &cases[i]
				reasons := fixtureValidationErrors(*tc, fixture.contract)
				wrongCounts[len(reasons)]++
				if tc.Raw.Address == fixture.contract {
					correctAddress++
				}
				if len(tc.Raw.Topics) > 0 && tc.Raw.Topics[0] == LogMessagePublishedTopic {
					correctTopic++
				}
				if tc.Raw.Removed {
					removed++
				}
				require.LessOrEqual(t, len(reasons), 3, "case %d has too many validation errors: %v", i, reasons)

				i := i
				tcForCase := tc
				t.Run(fmt.Sprintf("case_%03d", i), func(t *testing.T) {
					w, mock, msgC := newTestWatcher(t)
					w.contract = fixture.contract

					ev := tcForCase.event()
					mock.receipts[ev.Raw.TxHash] = fixtureReceipt(ev)
					blockTime := tcForCase.blockTime()

					w.postMessage(context.TODO(), ev, blockTime)

					if ev.ConsistencyLevel != vaa.ConsistencyLevelPublishImmediately {
						err := w.processNewBlock(
							context.TODO(),
							fixtureBlock(ev, blockTime, fixtureFinality(t, ev.ConsistencyLevel)),
							&gossipv1.Heartbeat_Network{},
						)
						require.NoError(t, err)
					}

					require.LessOrEqual(t, len(msgC), 1)
					messageSent := len(msgC) == 1
					if assertOrSeedFixtureMessageSent(t, tcForCase, messageSent) {
						fixtureUpdated = true
					}

					assert.Equal(t, 0, len(w.pending), "fixture %q should not leave pending messages", tcForCase.Comment)
					if !messageSent {
						assert.Empty(t, tcForCase.Hash, "fixture %q should not have an output hash when no message is sent", tcForCase.Comment)
						return
					}

					msg := <-msgC
					assertMessageMatchesEventAtTime(t, msg, ev, blockTime)
					assert.Equal(t, common.NotVerified, msg.VerificationState())
					if assertOrSeedFixtureHash(t, tcForCase, msg) {
						fixtureUpdated = true
					}
				})
			}

			if fixture.checkGeneratedDistribution {
				assert.Equal(t, uint(180), correctAddress)
				assert.Equal(t, uint(180), correctTopic)
				assert.Equal(t, uint(10), removed)
				assert.Equal(t, 160, wrongCounts[0])
				assert.Equal(t, 30, wrongCounts[1])
				assert.Equal(t, 10, wrongCounts[2])
				assert.Equal(t, 0, wrongCounts[3])
			}

			if fixtureUpdated && !t.Failed() {
				writePostMessageFixture(t, fixture, cases)
			}
		})
	}
}

func TestFixtureObservationMatchesReobservation(t *testing.T) {
	for _, fixture := range postMessageFixtures(t) {
		fixture := fixture

		t.Run(fixture.name, func(t *testing.T) {
			cases := loadPostMessageFixture(t, fixture)
			receiptsByTx := postMessageReceiptFixtureByTx(t, loadPostMessageReceiptFixture(t, fixture.receiptFileName))

			for i := range cases {
				tc := cases[i]
				require.NotNil(t, tc.MessageSent, "case %d missing messageSent", i)
				if *tc.MessageSent {
					require.NotEmpty(t, tc.Hash, "case %d missing hash", i)
				} else {
					require.Empty(t, tc.Hash, "case %d should not have hash when no message is sent", i)
				}

				t.Run(fmt.Sprintf("case_%03d", i), func(t *testing.T) {
					receipt, exists := receiptsByTx[tc.Raw.TxHash]
					require.True(t, exists, "missing receipt for tx %s", tc.Raw.TxHash)
					require.NotNil(t, receipt.BlockNumber, "receipt for tx %s missing block number", tc.Raw.TxHash)
					require.True(t, receiptContainsFixtureLog(receipt, tc.Raw), "receipt for tx %s missing fixture log", tc.Raw.TxHash)

					blockTime := tc.blockTime()
					ev := tc.event()

					obsWatcher, obsMock, obsC := newTestWatcher(t)
					obsWatcher.contract = fixture.contract
					obsMock.receipts[tc.Raw.TxHash] = receipt
					obsMock.blockTimes[receipt.BlockHash] = blockTime

					obsWatcher.postMessage(context.TODO(), ev, blockTime)
					if ev.ConsistencyLevel != vaa.ConsistencyLevelPublishImmediately {
						err := obsWatcher.processNewBlock(
							context.TODO(),
							fixtureBlock(ev, blockTime, fixtureFinality(t, ev.ConsistencyLevel)),
							&gossipv1.Heartbeat_Network{},
						)
						require.NoError(t, err)
					}

					require.LessOrEqual(t, len(obsC), 1, "observation flow should publish at most the fixture log")
					require.Equal(t, *tc.MessageSent, len(obsC) == 1, "observation sent state must match fixture")

					var observedMsg *common.MessagePublication
					if len(obsC) == 1 {
						observedMsg = recvMsg(t, obsC)
						require.False(t, observedMsg.IsReobservation)
						require.Equal(t, tc.Hash, observedMsg.CreateDigest(), "observation hash must match fixture")
					}

					reobsWatcher, reobsMock, reobsC := newTestWatcher(t)
					reobsWatcher.contract = fixture.contract
					reobsMock.receipts[tc.Raw.TxHash] = receipt
					reobsMock.blockTimes[receipt.BlockHash] = blockTime

					blockNumber := receipt.BlockNumber.Uint64()
					numObs, err := reobsWatcher.handleReobservationRequest(
						context.TODO(),
						reobsWatcher.chainID,
						tc.Raw.TxHash.Bytes(),
						reobsMock,
						blockNumber,
						blockNumber,
					)
					require.NoError(t, err)
					require.Equal(t, int(numObs), len(reobsC), "reobservation count should match published messages")

					if !*tc.MessageSent {
						require.Empty(t, reobsC, "reobservation should not publish when observation does not publish")
						return
					}

					require.GreaterOrEqual(t, len(reobsC), 1, "reobservation should publish at least the fixture log")
					var reobservedMsg *common.MessagePublication
					for len(reobsC) > 0 {
						msg := recvMsg(t, reobsC)
						require.True(t, msg.IsReobservation)
						if msg.MessageIDString() == observedMsg.MessageIDString() {
							reobservedMsg = msg
						}
					}

					require.NotNil(t, reobservedMsg, "reobservation did not publish fixture message %s", observedMsg.MessageIDString())
					require.Equal(t, tc.Hash, reobservedMsg.CreateDigest(), "reobservation hash must match fixture")
					require.Equal(t, observedMsg.CreateDigest(), reobservedMsg.CreateDigest(), "observation and reobservation hashes must match")
				})
			}
		})
	}
}

// TxVerifier is used on postMessage
func TestPostMessageTxVerifier(t *testing.T) {
	tests := []struct {
		name           string
		success        bool
		useTokenBridge bool
		expectState    common.VerificationState
	}{
		{"success", true, true, common.Valid},
		{"failure", false, true, common.Rejected},
		{"not_applicable", false, false, common.NotApplicable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, mock, msgC := newTestWatcher(t)
			w.txVerifier = &MockTransferVerifier[ethclient.Client, connectors.Connector]{success: tc.success}

			sender := testEmitter
			if tc.useTokenBridge {
				sender = testTokenBridge
			}
			ev := newTestLogEventFromParams(testLogEventParams{
				sender:           sender,
				sequence:         1,
				blockNumber:      100,
				consistencyLevel: vaa.ConsistencyLevelPublishImmediately,
			})
			mock.receipts[ev.Raw.TxHash] = newTestReceipt(ev.Raw.BlockNumber, nil)
			w.postMessage(context.TODO(), ev, 1234)

			require.Equal(t, 0, len(w.pending))
			require.Equal(t, 1, len(msgC))

			msg := recvMsg(t, msgC)
			assertMessageMatchesEvent(t, msg, ev)
			assert.Equal(t, tc.expectState, msg.VerificationState())
		})
	}
}

// Instant publish must drop the message when the receipt cannot be fetched.
func TestPostMessageInstantReceiptError(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	ev := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)
	mock.errors[ev.Raw.TxHash] = errors.New("rpc failure")

	w.postMessage(context.TODO(), ev, 1234)

	assert.Equal(t, 0, len(w.pending))
	assert.Equal(t, 0, len(msgC))
}

// Instant publish must drop the message when the receipt reports a non-success status.
func TestPostMessageInstantTxFailed(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	ev := newTestLogEvent(vaa.ConsistencyLevelPublishImmediately)
	receipt := newTestReceipt(ev.Raw.BlockNumber, nil)
	receipt.Status = 0
	mock.receipts[ev.Raw.TxHash] = receipt

	w.postMessage(context.TODO(), ev, 1234)

	assert.Equal(t, 0, len(w.pending))
	assert.Equal(t, 0, len(msgC))
}

func TestConsistencyLevelMatches(t *testing.T) {
	// Success cases.
	assert.True(t, consistencyLevelMatches(vaa.ConsistencyLevelPublishImmediately, vaa.ConsistencyLevelPublishImmediately))
	assert.True(t, consistencyLevelMatches(vaa.ConsistencyLevelSafe, vaa.ConsistencyLevelSafe))
	assert.True(t, consistencyLevelMatches(vaa.ConsistencyLevelFinalized, vaa.ConsistencyLevelFinalized))
	assert.True(t, consistencyLevelMatches(vaa.ConsistencyLevelFinalized, 0))
	assert.True(t, consistencyLevelMatches(vaa.ConsistencyLevelFinalized, 42))

	// Failure cases.
	assert.False(t, consistencyLevelMatches(vaa.ConsistencyLevelPublishImmediately, vaa.ConsistencyLevelSafe))
	assert.False(t, consistencyLevelMatches(vaa.ConsistencyLevelSafe, vaa.ConsistencyLevelFinalized))
	assert.False(t, consistencyLevelMatches(vaa.ConsistencyLevelFinalized, vaa.ConsistencyLevelPublishImmediately))
	assert.False(t, consistencyLevelMatches(vaa.ConsistencyLevelPublishImmediately, 0))
	assert.False(t, consistencyLevelMatches(vaa.ConsistencyLevelSafe, 0))
}
