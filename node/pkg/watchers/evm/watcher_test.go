package evm

import (
	"context"
	"encoding/json"
	"errors"
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

func TestVerifyAndPublish(t *testing.T) {

	msgC := make(chan *common.MessagePublication, 1)
	w := NewWatcherForTest(t, msgC)

	// Contents of the message don't matter for the sake of these tests.
	msg := common.MessagePublication{}
	ctx := context.TODO()

	// Check preconditions
	require.Equal(t, 0, len(w.msgC))
	require.Equal(t, common.NotVerified, msg.VerificationState())

	// Check nil message
	err := w.verifyAndPublish(nil, ctx, eth_common.Hash{}, &types.Receipt{})
	require.ErrorContains(t, err, "message publication cannot be nil")
	require.Equal(t, common.NotVerified, msg.VerificationState())

	// Check transfer verifier not enabled case. The message should be published normally
	msg = common.MessagePublication{}
	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.NoError(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg := <-msgC
	require.NotNil(t, publishedMsg)
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.NotApplicable, publishedMsg.VerificationState())

	// Check scenario where transfer verifier is enabled but isn't initialized.
	msg = common.MessagePublication{}
	w.txVerifierEnabled = true

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.ErrorContains(t, err, "transfer verifier should be instantiated but is nil")
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.NotVerified, publishedMsg.VerificationState())

	// Check scenario where the message already has a verification status.
	msg = common.MessagePublication{}
	setErr := msg.SetVerificationState(common.Anomalous)
	require.NoError(t, setErr)

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.ErrorContains(t, err, "message publication already has a verification status")
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.Anomalous, msg.VerificationState())

	// Check case where Transfer Verifier finds a dangerous transaction. Note that this case does
	// not return an error, but the published message should be marked as Rejected.
	msg = common.MessagePublication{}
	failMock := &MockTransferVerifier[ethclient.Client, connectors.Connector]{false}
	w.txVerifier = failMock

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.Nil(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg = <-msgC
	require.NotNil(t, publishedMsg)
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.Rejected, publishedMsg.VerificationState())

	// Check happy path where txverifier is enabled and initialized
	msg = common.MessagePublication{}
	successMock := &MockTransferVerifier[ethclient.Client, connectors.Connector]{true}
	w.txVerifier = successMock

	err = w.verifyAndPublish(&msg, ctx, eth_common.Hash{}, &types.Receipt{})
	require.NoError(t, err)
	require.Equal(t, 1, len(msgC))
	publishedMsg = <-msgC
	require.NotNil(t, publishedMsg)
	require.Equal(t, 0, len(msgC))
	require.Equal(t, common.Valid, publishedMsg.VerificationState())
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

// Mock ProcessEvent function that simulates the evaluation made by the Transfer Verifier.
func (m *MockTransferVerifier[E, C]) ProcessEvent(ctx context.Context, txHash eth_common.Hash, receipt *types.Receipt) bool {
	return m.success
}
