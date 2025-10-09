package evm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/txverifier"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethereum "github.com/ethereum/go-ethereum"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
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

// Helper function to set up a test Ethereum Watcher
func NewWatcherForTest(t *testing.T, msgC chan<- *common.MessagePublication) *Watcher {
	t.Helper()
	logger := zap.NewNop()

	w := &Watcher{
		// this is implicit but added here for clarity
		txVerifierEnabled: false,
		msgC:              msgC,
		logger:            logger,
	}

	return w
}

type MockTransferVerifier[E ethclient.Client, C connectors.Connector] struct {
	success bool
}

// TransferIsValid simulates the evaluation made by the Transfer Verifier.
// Always returns nil. The error should be non-nil only when a parsing or RPC error occurs.
// For now, these are not included in the unit tests.
func (m *MockTransferVerifier[E, C]) TransferIsValid(_ context.Context, _ string, _ eth_common.Hash, _ *types.Receipt) (bool, error) {
	return m.success, nil
}
func (m *MockTransferVerifier[E, C]) Addrs() *txverifier.TVAddresses {
	return &txverifier.TVAddresses{
		TokenBridgeAddr: eth_common.BytesToAddress([]byte{0x01}),
	}
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
