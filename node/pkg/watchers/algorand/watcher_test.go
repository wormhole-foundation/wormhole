package algorand

import (
	"context"
	"sync"
	"testing"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"go.uber.org/zap"
)

// TODO: grab these from const file somewhere?
const (
	IDX_URL   = "https://testnet-idx.algonode.cloud"
	ALGOD_URL = "https://testnet-api.algonode.cloud"

	APP_ID = 86525623
)

type testStruct struct {
	block uint64
	seq   uint64
}

var testCases = []testStruct{
	{block: 30453935, seq: 993},
}

func TestLookAtTxn(t *testing.T) {

	logger, _ := zap.NewProduction()
	for _, tc := range testCases {
		// Setup a watcher
		msgC := make(chan *common.MessagePublication)
		obsvReqC := make(chan *gossipv1.ObservationRequest, 50)
		w := NewWatcher(IDX_URL, "", ALGOD_URL, "", APP_ID, msgC, obsvReqC)

		// grab client
		algodClient, err := algod.MakeClient(w.algodRPC, w.algodToken)
		if err != nil {
			t.Fatalf("Failed to create client: %s", err)
		}

		// spin up goroutine to receive msgpub
		var foundMsg *common.MessagePublication
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for x := range msgC {
				foundMsg = x
			}
		}()

		// grab the block
		block, err := algodClient.Block(tc.block).Do(context.Background())
		if err != nil {
			t.Fatalf("Failed to get block: %s", err)
		}

		// for each tx in the block, check to see if its a valid
		// wh emitted message
		for _, element := range block.Payset {
			lookAtTxn(w, element, block, logger)
		}

		// close the channel so the goroutine returns
		close(msgC)
		// wait for the goroutine to finish
		wg.Wait()

		// assertions
		if foundMsg == nil {
			t.Fatal("no message found")
		}

		if foundMsg.Sequence != tc.seq {
			t.Fatalf("sequence did not match: %d vs %d", foundMsg.Sequence, tc.seq)
		}
	}
}
