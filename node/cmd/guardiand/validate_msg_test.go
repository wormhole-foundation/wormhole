package guardiand

import (
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eth_common "github.com/ethereum/go-ethereum/common"

	"go.uber.org/zap"
)

func TestValidateMessageSuccess(t *testing.T) {
	logger := zap.NewNop()

	emitterAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	msg := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddress,
		Payload:          []byte("Hello"),
		ConsistencyLevel: 32,
	}

	shouldPublish, err := validateMessage(logger, msg, vaa.ChainIDEthereum)
	require.NoError(t, err)
	assert.Equal(t, true, shouldPublish)
}

func TestValidateMessageWrongChainShouldReturnError(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	emitterAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	msg := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddress,
		Payload:          []byte("Hello"),
		ConsistencyLevel: 32,
	}

	shouldPublish, err := validateMessage(logger, msg, vaa.ChainIDBSC)

	assert.Error(t, err)
	assert.Equal(t, "Received observation from a chain that was not marked as originating from that chain", err.Error())
	assert.Equal(t, false, shouldPublish)
}

func TestValidateMessageZeroAddressShouldNotPublish(t *testing.T) {
	logger := zap.NewNop()

	msg := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   vaa.Address{},
		Payload:          []byte("Hello"),
		ConsistencyLevel: 32,
	}

	shouldPublish, err := validateMessage(logger, msg, vaa.ChainIDEthereum)
	require.NoError(t, err)
	assert.Equal(t, false, shouldPublish)
}
