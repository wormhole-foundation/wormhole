package mock

import (
	"context"
	"math/rand"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func NewWatcherRunnable(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) supervisor.Runnable {
	return func(ctx context.Context) error {

		msgStream := createMessageStream(ctx)

		var obsvReqCounter uint64 = 0

		for {
			select {
			case <-ctx.Done():
				return nil
			case m := <-msgStream:
				msgC <- m
			case o := <-obsvReqC:
				msgC <- createMessageFromObsvReq(o, obsvReqCounter)
				obsvReqCounter++
			}
		}
	}
}

func createMessageFromObsvReq(o *gossipv1.ObservationRequest, sequence uint64) *common.MessagePublication {
	return &common.MessagePublication{
		TxHash:           eth_common.BytesToHash(o.TxHash),
		Timestamp:        time.Now(),
		Nonce:            rand.Uint32(), //#nosec G404 test code
		Sequence:         sequence,
		ConsistencyLevel: 1,
		EmitterChain:     vaa.ChainID(o.ChainId),
		EmitterAddress:   [32]byte{1, 0, 1},
		Payload:          []byte{},
		Unreliable:       false,
	}
}

func createMessageStream(ctx context.Context) chan *common.MessagePublication {
	c := make(chan *common.MessagePublication)

	go func() {
		timer := time.NewTimer(time.Second)

		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			case t := <-timer.C:
				c <- &common.MessagePublication{
					TxHash:           [32]byte{1, 2, 3},
					Timestamp:        t,
					Nonce:            rand.Uint32(), //#nosec G404 test code
					Sequence:         uint64(i),
					ConsistencyLevel: 1,
					EmitterChain:     vaa.ChainIDEthereum,
					EmitterAddress:   [32]byte{1, 2, 3},
					Payload:          []byte{},
					Unreliable:       false,
				}
			}
		}
	}()

	return c
}
