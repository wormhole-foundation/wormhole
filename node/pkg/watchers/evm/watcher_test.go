package evm

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
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
	publishedMsg := <-msgC
	require.NotNil(t, publishedMsg)
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.NotVerified.String(), publishedMsg.VerificationState().String())

	tbAddr, byteErr := vaa.BytesToAddress([]byte{0x01})
	require.NoError(t, byteErr)

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
	publishedMsg = <-msgC
	require.Equal(t, common.NotVerified.String(), publishedMsg.VerificationState().String())

	// Check that message status is not changed if it didn't come from token bridge.
	// The NotVerified status is used when Transfer Verification is not enabled.
	msg = common.MessagePublication{}
	require.Nil(t, w.txVerifier)

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.Nil(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg = <-msgC
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
	publishedMsg = <-msgC
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
	publishedMsg = <-msgC
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
	publishedMsg = <-msgC
	require.NotNil(t, publishedMsg)
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.Valid.String(), publishedMsg.VerificationState().String())
}

// TestVerifyDoesNotMutateOriginalMessage checks that verify() does not modify
// the original MessagePublication passed to it.
func TestVerifyDoesNotMutateOriginalMessage(t *testing.T) {
	tbAddr, err := vaa.BytesToAddress([]byte{0x01})
	require.NoError(t, err)

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

			w.addPendingMsg(txHash, blockHash, 100, tc.cl, 0, 1)
			mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash}

			err := w.processNewBlock(context.TODO(), newBlock(tc.blockNumber, tc.finality))
			require.NoError(t, err)

			assert.Equal(t, tc.expectPending, len(w.pending))
			if tc.expectPublish {
				require.Equal(t, 1, len(msgC))
				assert.Equal(t, txHash.Bytes(), (<-msgC).TxID)
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

	w.addPendingMsg(txHash1, blockHash1, 100, vaa.ConsistencyLevelFinalized, 0, 1)
	w.addPendingMsg(txHash2, blockHash2, 100, vaa.ConsistencyLevelSafe, 0, 1)
	mock.receipts[txHash1] = &types.Receipt{Status: 1, BlockHash: blockHash1}
	mock.receipts[txHash2] = &types.Receipt{Status: 1, BlockHash: blockHash2}

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
	require.NoError(t, err)

	// Removed one from pending
	assert.Equal(t, 1, len(w.pending))

	// Published finalized message
	require.Equal(t, 1, len(msgC))
	assert.Equal(t, txHash1.Bytes(), (<-msgC).TxID)

	err = w.processNewBlock(context.TODO(), newBlock(105, connectors.Safe))
	require.NoError(t, err)

	// Removed both
	assert.Equal(t, 0, len(w.pending))

	// Published safe message
	require.Equal(t, 1, len(msgC))
	assert.Equal(t, txHash2.Bytes(), (<-msgC).TxID)
}

// Removal of the message without publication if the blockhash differs
func TestProcessBlockPendingWrongBlockHash(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	txHash := eth_common.HexToHash("0xd2d35ab0d18dd19e81a58dfe8d97ad8c68659bd81d7017bcdf4d9719b32119ef")
	blockHashBlock := eth_common.BigToHash(big.NewInt(111))
	blockHashMessage := eth_common.BigToHash(big.NewInt(222))

	w.addPendingMsg(txHash, blockHashMessage, 100, vaa.ConsistencyLevelFinalized, 0, 1)
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHashBlock}

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
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

	w.addPendingMsg(txHash, blockHash, 100, vaa.ConsistencyLevelFinalized, 0, 1)
	mock.receipts[txHash] = &types.Receipt{Status: 0, BlockHash: blockHash}

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
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

	w.addPendingMsg(txHash, blockHash, 100, vaa.ConsistencyLevelFinalized, 0, 1)
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash}
	mock.errors[txHash] = errors.New("not found")

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
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

	w.addPendingMsg(txHash, blockHash, 100, vaa.ConsistencyLevelFinalized, 0, 1)

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
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
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash}
	mock.errors[txHash] = errors.New("transient error")

	w.addPendingMsg(txHash, blockHash, 100, vaa.ConsistencyLevelFinalized, 0, 1)

	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
	require.NoError(t, err)

	// Not removed from pending
	assert.Equal(t, 1, len(w.pending))

	// Not published
	assert.Equal(t, 0, len(msgC))

	// Try the message again after the transient error has disappeared
	mock.errors[txHash] = nil
	err = w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
	require.NoError(t, err)

	// Removed from pending
	assert.Equal(t, 0, len(w.pending))

	// Published with correct TxID
	require.Equal(t, 1, len(msgC))
	assert.Equal(t, txHash.Bytes(), (<-msgC).TxID)
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

			w.addPendingMsg(txHash, blockHash, 100, tc.cl, tc.additionalBlocks, 1)
			mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash}

			err := w.processNewBlock(context.TODO(), newBlock(tc.blockNumber, tc.finality))
			require.NoError(t, err)

			assert.Equal(t, tc.expectPending, len(w.pending))
			if tc.expectPublish {
				require.Equal(t, 1, len(msgC))
				assert.Equal(t, txHash.Bytes(), (<-msgC).TxID)
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
	key := w.addPendingMsg(txHash, blockHash, 100, vaa.ConsistencyLevelFinalized, 5, 1)
	w.pending[key].message.ConsistencyLevel = vaa.ConsistencyLevelCustom
	mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash}

	// Finalized block at height+additionalBlocks: should confirm based on effectiveCL
	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
	require.NoError(t, err)

	// Removed from pending and published
	assert.Equal(t, 0, len(w.pending))
	require.Equal(t, 1, len(msgC))

	// The published message must still have ConsistencyLevelCustom for VAA hash consistency
	published := <-msgC
	assert.Equal(t, vaa.ConsistencyLevelCustom, published.ConsistencyLevel)
}

// Pending messages with different finalizations and block times
func TestProcessBlockCCLMultiplePendingDifferentAdditionalBlocks(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	txHashA := eth_common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	txHashB := eth_common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	blockHash := eth_common.BigToHash(big.NewInt(100))

	// Message A: effectiveCL=Finalized, additionalBlocks=0, height=100 -> ready at block 100
	w.addPendingMsg(txHashA, blockHash, 100, vaa.ConsistencyLevelFinalized, 0, 1)
	mock.receipts[txHashA] = &types.Receipt{Status: 1, BlockHash: blockHash}

	// Message B: effectiveCL=Finalized, additionalBlocks=10, height=100 -> ready at block 110
	w.addPendingMsg(txHashB, blockHash, 100, vaa.ConsistencyLevelFinalized, 10, 2)
	mock.receipts[txHashB] = &types.Receipt{Status: 1, BlockHash: blockHash}

	// Send finalized block at 105: A should confirm, B should stay pending
	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
	require.NoError(t, err)

	assert.Equal(t, 1, len(w.pending), "only message B should remain pending")
	require.Equal(t, 1, len(msgC), "only message A should be published")
	assert.Equal(t, txHashA.Bytes(), (<-msgC).TxID)

	// Send finalized block at 110: B should now confirm
	err = w.processNewBlock(context.TODO(), newBlock(110, connectors.Finalized))
	require.NoError(t, err)

	assert.Equal(t, 0, len(w.pending), "no messages should remain pending")
	require.Equal(t, 1, len(msgC), "message B should now be published")
	assert.Equal(t, txHashB.Bytes(), (<-msgC).TxID)
}

// Orphaned tx handling
func TestProcessBlockOneConfirmedOneOrphaned(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)
	blockHash := eth_common.BigToHash(big.NewInt(100))

	txHashGood := eth_common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	txHashOrphaned := eth_common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	w.addPendingMsg(txHashGood, blockHash, 100, vaa.ConsistencyLevelFinalized, 0, 1)
	w.addPendingMsg(txHashOrphaned, blockHash, 100, vaa.ConsistencyLevelFinalized, 0, 2)

	// Good tx has a valid receipt
	mock.receipts[txHashGood] = &types.Receipt{Status: 1, BlockHash: blockHash}

	// Orphaned tx returns nil receipt (not in mock.receipts, so defaults to nil)
	err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
	require.NoError(t, err)

	// Both removed from pending
	assert.Equal(t, 0, len(w.pending))

	// Only the valid message was published
	assert.Equal(t, 1, len(msgC))

	published := <-msgC
	assert.Equal(t, txHashGood.Bytes(), published.TxID)
}

// Invalid finality level process. Should return an error.
func TestProcessBlockUnexpectedFinality(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	err := w.processNewBlock(context.TODO(), newBlock(100, connectors.FinalityLevel(99)))
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

			key := w.addPendingMsg(txHash, blockHash, 100, vaa.ConsistencyLevelFinalized, 0, 1)
			if tc.useTokenBridge {
				w.pending[key].message.EmitterAddress = PadAddress(testTokenBridge)
			}
			mock.receipts[txHash] = &types.Receipt{Status: 1, BlockHash: blockHash}

			err := w.processNewBlock(context.TODO(), newBlock(105, connectors.Finalized))
			require.NoError(t, err)

			assert.Equal(t, 0, len(w.pending))
			require.Equal(t, 1, len(msgC))

			msg := <-msgC
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

			ev := newTestLogEvent(100, tc.cl)
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

			assertMessageMatchesEvent(t, pe.message, ev, 1234)
			assertPendingMetadata(t, pe, tc.cl, 100, 0)
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

			ev := newTestLogEvent(100, vaa.ConsistencyLevelCustom)
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
			assertMessageMatchesEvent(t, pe.message, ev, 1234)
			assertPendingMetadata(t, pe, vaa.ConsistencyLevelFinalized, 100, 0)
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
			ev := newTestLogEvent(100, vaa.ConsistencyLevelCustom)
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
			assertMessageMatchesEvent(t, pe.message, ev, 1234)
			assertPendingMetadata(t, pe, tc.effectiveCL, 100, uint64(tc.additionalBlocks))
		})
	}
}

// Instant message is published instead of being added to the pending queue
func TestPostMessageInstantPublishes(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	ev := newTestLogEvent(100, vaa.ConsistencyLevelPublishImmediately)
	w.postMessage(context.TODO(), ev, 1234)

	// Should be published instead of being added to the pending queue
	require.Equal(t, 0, len(w.pending))
	assert.Equal(t, 1, len(msgC))

	msg := <-msgC

	assertMessageMatchesEvent(t, msg, ev, 1234)
	assert.Equal(t, common.NotVerified, msg.VerificationState())
}

// Multiple instant publishes are handled properly
func TestPostMessageTwoInstantPublishes(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	ev1 := newTestLogEvent(100, vaa.ConsistencyLevelPublishImmediately)

	ev2 := newTestLogEvent(100, vaa.ConsistencyLevelPublishImmediately)
	ev2.Sender = eth_common.HexToAddress("0x388C818CA8B9251b393131C08a736A67ccB19297")
	ev2.Nonce = 20
	ev2.Sequence = 2

	w.postMessage(context.TODO(), ev1, 1234)
	w.postMessage(context.TODO(), ev2, 1234)

	// Should be added to pending, not published immediately
	require.Equal(t, 0, len(w.pending))
	assert.Equal(t, 2, len(msgC))

	msg1 := <-msgC
	msg2 := <-msgC

	assertMessageMatchesEvent(t, msg1, ev1, 1234)
	assert.Equal(t, common.NotVerified, msg1.VerificationState())

	assertMessageMatchesEvent(t, msg2, ev2, 1234)
	assert.Equal(t, common.NotVerified, msg2.VerificationState())
}

// An instant and a final message go to the proper spots (msgC and pending)
func TestPostMessageInstantAndFinalized(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	// ev1: instant publish
	ev1 := newTestLogEvent(100, vaa.ConsistencyLevelPublishImmediately)

	// ev2: finalized (goes to pending) — use a different emitter and sequence
	ev2 := newTestLogEventFromParams(testLogEventParams{
		sender:           eth_common.HexToAddress("0x388C818CA8B9251b393131C08a736A67ccB19297"),
		sequence:         2,
		blockNumber:      100,
		consistencyLevel: vaa.ConsistencyLevelFinalized,
	})

	w.postMessage(context.TODO(), ev1, 1234)
	w.postMessage(context.TODO(), ev2, 1234)

	// One message published immediately, one added to pending.
	require.Equal(t, 1, len(w.pending))
	assert.Equal(t, 1, len(msgC))

	// Verify the instant-published message
	msg := <-msgC
	assertMessageMatchesEvent(t, msg, ev1, 1234)
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

	assertMessageMatchesEvent(t, pe.message, ev2, 1234)
	assertPendingMetadata(t, pe, vaa.ConsistencyLevelFinalized, 100, 0)
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

	ev := newTestLogEvent(100, vaa.ConsistencyLevelPublishImmediately)
	ev.Raw.Removed = true

	w.postMessage(context.TODO(), ev, 1234)

	assert.Equal(t, 0, len(msgC), "removed log should not be published to msgC")
	assert.Equal(t, 0, len(w.pending), "removed log should not be added to pending")
}

func TestPostMessageWrongContractAddressIsIgnored(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	ev := newTestLogEvent(100, vaa.ConsistencyLevelPublishImmediately)
	ev.Raw.Address = eth_common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	w.postMessage(context.TODO(), ev, 1234)

	assert.Equal(t, 0, len(msgC), "log from wrong contract should not be published to msgC")
	assert.Equal(t, 0, len(w.pending), "log from wrong contract should not be added to pending")
}

func TestPostMessageWrongEventSignatureIsIgnored(t *testing.T) {
	w, _, msgC := newTestWatcher(t)

	ev := newTestLogEvent(100, vaa.ConsistencyLevelPublishImmediately)
	ev.Raw.Topics = []eth_common.Hash{
		eth_common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		eth_common.BytesToHash(testEmitter.Bytes()),
	}

	w.postMessage(context.TODO(), ev, 1234)

	assert.Equal(t, 0, len(msgC), "log with wrong event signature should not be published to msgC")
	assert.Equal(t, 0, len(w.pending), "log with wrong event signature should not be added to pending")
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
			w, _, msgC := newTestWatcher(t)
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
			w.postMessage(context.TODO(), ev, 1234)

			require.Equal(t, 0, len(w.pending))
			require.Equal(t, 1, len(msgC))

			msg := <-msgC
			assertMessageMatchesEvent(t, msg, ev, 1234)
			assert.Equal(t, tc.expectState, msg.VerificationState())
		})
	}
}

/*
TODO - add failed transaction status to the flow.
- Requires mocking the receipt gathering in every RPC call.
*/

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
