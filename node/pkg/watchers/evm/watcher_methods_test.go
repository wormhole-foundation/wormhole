package evm

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func newMethodTestWatcher(msgC chan<- *common.MessagePublication) *Watcher {
	return &Watcher{chainID: vaa.ChainIDEthereum, networkName: "ethereum", msgC: msgC, logger: zap.NewNop()}
}

func TestWatcherMethodChainID(t *testing.T) {
	tests := []struct {
		name string
		want vaa.ChainID
	}{{name: "returns configured chain", want: vaa.ChainIDEthereum}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, newMethodTestWatcher(make(chan *common.MessagePublication, 1)).ChainID())
		})
	}
}

func TestWatcherMethodValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     *gossipv1.ObservationRequest
		wantErr bool
	}{
		{name: "accepts valid request", req: &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDEthereum), TxHash: make([]byte, common.TxIDLenMin)}},
		{name: "rejects wrong chain", req: &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDSui), TxHash: make([]byte, common.TxIDLenMin)}, wantErr: true},
		{name: "accepts empty tx hash", req: &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDEthereum)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validated, err := newMethodTestWatcher(make(chan *common.MessagePublication, 1)).Validate(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.req.GetTxHash(), validated.TxHash())
		})
	}
}

func TestWatcherMethodPublishMessage(t *testing.T) {
	tests := []struct {
		name    string
		msg     *common.MessagePublication
		wantErr bool
	}{
		{name: "rejects nil message", wantErr: true},
		{name: "publishes message", msg: &common.MessagePublication{EmitterChain: vaa.ChainIDEthereum}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			err := newMethodTestWatcher(msgC).PublishMessage(tt.msg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Same(t, tt.msg, <-msgC)
		})
	}
}

func TestWatcherMethodPublishReobservation(t *testing.T) {
	validated, err := newMethodTestWatcher(make(chan *common.MessagePublication, 1)).Validate(&gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDEthereum), TxHash: make([]byte, common.TxIDLenMin)})
	require.NoError(t, err)
	tests := []struct {
		name    string
		msg     *common.MessagePublication
		wantErr bool
	}{
		{name: "rejects mismatched chain", msg: &common.MessagePublication{EmitterChain: vaa.ChainIDSui}, wantErr: true},
		{name: "publishes reobservation", msg: &common.MessagePublication{EmitterChain: vaa.ChainIDEthereum}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			err := newMethodTestWatcher(msgC).PublishReobservation(validated, tt.msg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.True(t, tt.msg.IsReobservation)
			require.Same(t, tt.msg, <-msgC)
		})
	}
}
