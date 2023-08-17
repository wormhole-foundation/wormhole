package governor

import (
	"context"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestIsVAAEnqueuedNilMessageID(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	enqueued, err := gov.IsVAAEnqueued(nil)
	require.EqualError(t, err, "no message ID specified")
	assert.Equal(t, false, enqueued)
}

func TestIsTransactionEnqueued(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)
	ce := gov.chains[vaa.ChainIDEthereum]
	require.NotNil(t, ce)

	txHash1 := hashFromString("06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063")
	txHash2 := hashFromString("06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064")

	// When the pending map is empty, it should return false.
	assert.False(t, gov.IsTransactionEnqueued(vaa.ChainIDEthereum, txHash1))

	// When we enqueue the transfer, it should return true.
	ce.pending = append(ce.pending, &pendingEntry{dbData: db.PendingTransfer{Msg: common.MessagePublication{TxHash: txHash1}}})
	assert.True(t, gov.IsTransactionEnqueued(vaa.ChainIDEthereum, txHash1))

	// Some other transfer should still return false.
	assert.False(t, gov.IsTransactionEnqueued(vaa.ChainIDEthereum, txHash2))

	// Looking for the same txHash on a different chain should return false.
	assert.False(t, gov.IsTransactionEnqueued(vaa.ChainIDPolygon, txHash1))

	// Looking for a non-existent chain should return false.
	assert.False(t, gov.IsTransactionEnqueued(vaa.ChainIDUnset, txHash1))
}
