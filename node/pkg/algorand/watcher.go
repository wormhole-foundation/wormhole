package algorand

import (
	"context"
	"encoding/hex"
//	"fmt"
//	"github.com/certusone/wormhole/node/pkg/p2p"
//	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
//	"github.com/prometheus/client_golang/prometheus/promauto"
//	"io/ioutil"
//	"net/http"
//	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"

//	"github.com/algorand/go-algorand-sdk/client/algod"
//	"github.com/algorand/go-algorand-sdk/client/kmd"

//	"github.com/gorilla/websocket"
//	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Algorand blockchain and reporting new transactions to the contract
	Watcher struct {
		urlRPC   string
		urlToken string
		contract string

		msgChan chan *common.MessagePublication
		setChan chan *common.GuardianSet
	}
)

type clientRequest struct {
	JSONRPC string `json:"jsonrpc"`
	// A String containing the name of the method to be invoked.
	Method string `json:"method"`
	// Object to pass as request parameter to the method.
	Params [1]string `json:"params"`
	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	ID uint64 `json:"id"`
}

// NewWatcher creates a new Algorand contract watcher
func NewWatcher(urlRPC string, urlToken string, contract string, lockEvents chan *common.MessagePublication, setEvents chan *common.GuardianSet) *Watcher {
	return &Watcher{urlRPC: urlToken, contract: contract, msgChan: lockEvents, setChan: setEvents}
}

func (e *Watcher) Run(ctx context.Context) error {
        return nil
}

