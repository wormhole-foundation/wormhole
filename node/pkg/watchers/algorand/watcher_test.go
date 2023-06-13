package algorand

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/algorand/go-algorand-sdk/types"
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"go.uber.org/zap"
)

const APP_ID = 86525623

// Tests for nested inner transactions calling the core bridge
func TestLookAtTxnInnerTxn(t *testing.T) {
	// Setup a watcher
	msgC := make(chan *common.MessagePublication)
	obsvReqC := make(chan *gossipv1.ObservationRequest, 50)
	w := NewWatcher("", "", "", "", APP_ID, msgC, obsvReqC)

	var expectedSequence uint64 = 993

	// read in test block for inner transactions
	b, err := os.ReadFile("test_nested_inner.block.json")
	if err != nil {
		t.Fatalf("failed to read block file: %s", err)
	}

	txn := types.SignedTxnInBlock{}
	err = json.Unmarshal(b, &txn)
	if err != nil {
		t.Fatalf("failed to unmarshal block: %s", err)
	}

	// Because we are using a json blob and the type of logs array is []string
	// and because go json package will refuse to properly encode/decode
	// invalid utf8 characters, the json blob has the relevant log encoded as base64
	// and we base64 decode it and convert it to a string _manually_ so we can
	// make sure we got the right sequence number
	b64Data := txn.EvalDelta.InnerTxns[2].EvalDelta.InnerTxns[0].EvalDelta.Logs[0]
	bb, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		t.Fatalf("Cant decode: %s", err)
	}
	txn.EvalDelta.InnerTxns[2].EvalDelta.InnerTxns[0].EvalDelta.Logs[0] = string(bb)

	// for each tx in the block, check to see if its a valid
	// wh emitted message
	logger, _ := zap.NewProduction()
	observations := gatherObservations(w, txn.SignedTxnWithAD, 0, logger)

	if len(observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(observations))
	}

	if observations[0].sequence != expectedSequence {
		t.Fatalf("expected sequence observed to be %d, got %d", expectedSequence, observations[0].sequence)
	}
}
