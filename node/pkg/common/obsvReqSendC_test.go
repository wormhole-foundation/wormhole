package common

import (
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
)

func TestObsvReqSendLimitEnforced(t *testing.T) {
	const ObsvReqChannelSize = 50

	obsvReqSendC := make(chan *gossipv1.ObservationRequest, ObsvReqChannelSize)

	// If the channel overflows, the write hangs, so use a go routine with a timeout.
	done := make(chan struct{})
	go func() {
		// Filling the queue up should work.
		for count := 1; count <= ObsvReqChannelSize; count++ {
			req := &gossipv1.ObservationRequest{
				ChainId: uint32(vaa.ChainIDSolana),
			}
			err := PostObservationRequest(obsvReqSendC, req)
			assert.Nil(t, err)
		}

		// But one more write should fail.
		req := &gossipv1.ObservationRequest{
			ChainId: uint32(vaa.ChainIDSolana),
		}
		err := PostObservationRequest(obsvReqSendC, req)
		assert.ErrorIs(t, err, ErrChanFull)

		done <- struct{}{}
	}()

	timeout := time.NewTimer(time.Second)
	select {
	case <-timeout.C:
		assert.Fail(t, "timed out")
	case <-done:
	}
}
