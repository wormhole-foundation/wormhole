package watchers

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestValidateObservationRequest(t *testing.T) {
	t.Run("rejects nil request", func(t *testing.T) {
		_, err := ValidateObservationRequest(nil, vaa.ChainIDSui)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("accepts expected chain", func(t *testing.T) {
		validated, err := ValidateObservationRequest(&gossipv1.ObservationRequest{
			ChainId:   uint32(vaa.ChainIDSui),
			TxHash:    []byte{1, 2, 3},
			Timestamp: 1234,
		}, vaa.ChainIDSui)
		require.NoError(t, err)
		assert.Equal(t, vaa.ChainIDSui, validated.ChainID())
		assert.Equal(t, []byte{1, 2, 3}, validated.TxHash())
		assert.Equal(t, int64(1234), validated.Timestamp())

		original := validated.TxHash()
		original[0] = 99
		assert.Equal(t, []byte{1, 2, 3}, validated.TxHash())
	})

	t.Run("rejects unknown chain number", func(t *testing.T) {
		_, err := ValidateObservationRequest(&gossipv1.ObservationRequest{
			ChainId: 999999,
			TxHash:  []byte{1, 2, 3},
		}, vaa.ChainIDSui)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid chain id")
	})

	t.Run("rejects unexpected chain", func(t *testing.T) {
		_, err := ValidateObservationRequest(&gossipv1.ObservationRequest{
			ChainId: uint32(vaa.ChainIDAptos),
			TxHash:  []byte{1, 2, 3},
		}, vaa.ChainIDSui)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected chain id")
	})

	t.Run("tx hash length helper", func(t *testing.T) {
		validated, err := ValidateObservationRequest(&gossipv1.ObservationRequest{
			ChainId: uint32(vaa.ChainIDSui),
			TxHash:  make([]byte, common.TxIDLenMin),
		}, vaa.ChainIDSui)
		require.NoError(t, err)
		require.NoError(t, validated.RequireTxHashLength(common.TxIDLenMin))
		require.Error(t, validated.RequireTxHashLength(64))
	})
}

func TestValidateReobservedMessage(t *testing.T) {
	validated, err := ValidateObservationRequest(&gossipv1.ObservationRequest{
		ChainId: uint32(vaa.ChainIDSui),
		TxHash:  make([]byte, common.TxIDLenMin),
	}, vaa.ChainIDSui)
	require.NoError(t, err)

	t.Run("rejects nil message", func(t *testing.T) {
		require.Error(t, ValidateReobservedMessage(validated, nil))
	})

	t.Run("rejects mismatched chain", func(t *testing.T) {
		msg := &common.MessagePublication{EmitterChain: vaa.ChainIDEthereum}
		require.Error(t, ValidateReobservedMessage(validated, msg))
	})

	msg := &common.MessagePublication{EmitterChain: vaa.ChainIDSui}
	require.NoError(t, ValidateReobservedMessage(validated, msg))
}

func TestValidObservationZapFields(t *testing.T) {
	validated, err := ValidateObservationRequest(&gossipv1.ObservationRequest{
		ChainId:   uint32(vaa.ChainIDSui),
		TxHash:    []byte{0xAA, 0xBB},
		Timestamp: 42,
	}, vaa.ChainIDSui)
	require.NoError(t, err)

	fields := validated.ZapFields(zap.String("extra", "value"))
	require.Len(t, fields, 5)
	assert.Equal(t, zap.String("extra", "value").Key, fields[0].Key)
	assert.Equal(t, "chainID", fields[1].Key)
	assert.Equal(t, "chain", fields[2].Key)
	assert.Equal(t, "txID", fields[3].Key)
	assert.Equal(t, "timestamp", fields[4].Key)
}

func TestLogInvalidObservationRequest(t *testing.T) {
	t.Run("logs raw request fields", func(t *testing.T) {
		core, logs := observer.New(zap.ErrorLevel)
		logger := zap.New(core)
		req := &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDSui), TxHash: []byte{0xAA}, Timestamp: 42}
		err := assert.AnError

		LogInvalidObservationRequest(logger, req, err, zap.String("extra", "value"))

		entries := logs.AllUntimed()
		require.Len(t, entries, 1)
		assert.Equal(t, "invalid observation request", entries[0].Message)
		ctx := entries[0].ContextMap()
		assert.Equal(t, uint32(vaa.ChainIDSui), ctx["chainID"])
		assert.Equal(t, "aa", ctx["txID"])
		assert.Equal(t, int64(42), ctx["timestamp"])
		assert.Equal(t, "value", ctx["extra"])
		require.Contains(t, ctx, "error")
	})

	t.Run("logs nil request", func(t *testing.T) {
		core, logs := observer.New(zap.ErrorLevel)
		logger := zap.New(core)

		LogInvalidObservationRequest(logger, nil, assert.AnError)

		entries := logs.AllUntimed()
		require.Len(t, entries, 1)
		ctx := entries[0].ContextMap()
		assert.Equal(t, true, ctx["nilRequest"])
	})
}
