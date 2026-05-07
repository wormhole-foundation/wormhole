package aptos

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func newTestWatcher(msgC chan<- *common.MessagePublication) *Watcher {
	return &Watcher{chainID: vaa.ChainIDAptos, networkID: "aptos", msgC: msgC}
}

func TestWatcherChainID(t *testing.T) {
	tests := []struct {
		name string
		want vaa.ChainID
	}{{name: "returns configured chain", want: vaa.ChainIDAptos}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher := watchers.Watcher(newTestWatcher(make(chan *common.MessagePublication, 1)))
			assert.Equal(t, tt.want, watcher.ChainID())
		})
	}
}

func TestWatcherValidate(t *testing.T) {
	nonZeroPrefixTxHash := make([]byte, 32)
	nonZeroPrefixTxHash[0] = 1

	tests := []struct {
		name      string
		req       *gossipv1.ObservationRequest
		wantErr   bool
		wantIsErr error
	}{
		{name: "accepts valid request", req: &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDAptos), TxHash: make([]byte, 32)}},
		{name: "rejects nil request", wantErr: true, wantIsErr: watchers.ErrNilObservationRequest},
		{name: "rejects wrong chain", req: &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDSui), TxHash: make([]byte, 32)}, wantErr: true},
		{name: "rejects too short aptos tx id", req: &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDAptos), TxHash: make([]byte, 31)}, wantErr: true},
		{name: "rejects non-zero aptos tx id prefix", req: &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDAptos), TxHash: nonZeroPrefixTxHash}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher := watchers.Watcher(newTestWatcher(make(chan *common.MessagePublication, 1)))
			validatedObservation, err := watcher.Validate(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantIsErr != nil {
					require.ErrorIs(t, err, tt.wantIsErr)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.req.GetTxHash(), validatedObservation.TxHash())
		})
	}
}

func TestWatcherPublishMessage(t *testing.T) {
	tests := []struct {
		name      string
		msg       *common.MessagePublication
		wantErr   bool
		wantIsErr error
	}{
		{name: "rejects nil message", wantErr: true, wantIsErr: watchers.ErrNilMessagePublication},
		{name: "rejects mismatched chain", msg: &common.MessagePublication{EmitterChain: vaa.ChainIDSui}, wantErr: true},
		{name: "publishes message", msg: &common.MessagePublication{EmitterChain: vaa.ChainIDAptos}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			watcher := watchers.Watcher(newTestWatcher(msgC))
			err := watcher.PublishMessage(tt.msg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantIsErr != nil {
					require.ErrorIs(t, err, tt.wantIsErr)
				}
				return
			}
			require.NoError(t, err)
			require.Same(t, tt.msg, <-msgC)
		})
	}
}

func TestWatcherPublishReobservation(t *testing.T) {
	txHash := make([]byte, 32)
	validatedObservation, err := newTestWatcher(make(chan *common.MessagePublication, 1)).Validate(&gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDAptos), TxHash: txHash})
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *common.MessagePublication
		wantErr   bool
		wantIsErr error
	}{
		{name: "rejects nil message", wantErr: true, wantIsErr: watchers.ErrNilMessagePublication},
		{name: "rejects mismatched chain", msg: &common.MessagePublication{EmitterChain: vaa.ChainIDSui}, wantErr: true},
		{name: "publishes reobservation", msg: &common.MessagePublication{TxID: txHash, EmitterChain: vaa.ChainIDAptos}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			watcher := watchers.Watcher(newTestWatcher(msgC))
			err := watcher.PublishReobservation(validatedObservation, tt.msg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantIsErr != nil {
					require.ErrorIs(t, err, tt.wantIsErr)
				}
				return
			}
			require.NoError(t, err)
			require.True(t, tt.msg.IsReobservation)
			require.Same(t, tt.msg, <-msgC)
		})
	}
}
