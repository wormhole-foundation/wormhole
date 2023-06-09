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

const (
	IDX_URL   = "https://testnet-idx.algonode.cloud"
	ALGOD_URL = "https://testnet-api.algonode.cloud"

	APP_ID = 86525623
	BLOCK  = 30453935
)

func TestLookAtTxn(t *testing.T) {
	// Setup a watcher
	msgC := make(chan *common.MessagePublication)
	obsvReqC := make(chan *gossipv1.ObservationRequest, 50)
	w := NewWatcher(IDX_URL, "", ALGOD_URL, "", APP_ID, msgC, obsvReqC)

	var foundMsg *common.MessagePublication

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for x := range msgC {
			foundMsg = x
		}
	}()

	algodClient, err := algod.MakeClient(w.algodRPC, w.algodToken)
	if err != nil {
		t.Fatalf("Failed to create client: %s", err)
	}

	block, err := algodClient.Block(BLOCK).Do(context.Background())
	if err != nil {
		t.Fatalf("Failed to get block: %s", err)
	}

	logger, _ := zap.NewProduction()
	for _, element := range block.Payset {
		lookAtTxn(w, element, block, logger)
	}

	close(msgC)
	wg.Wait()

	if foundMsg == nil {
		t.Fatal("No message found :(")
	}

	t.Logf("Found message: %s %d", foundMsg.EmitterAddress, foundMsg.Sequence)
}
