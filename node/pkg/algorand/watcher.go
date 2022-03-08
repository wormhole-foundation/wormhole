package algorand

import (
	"context"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"

	"encoding/json"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
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

// NewWatcher creates a new Algorand contract watcher
func NewWatcher(urlRPC string, urlToken string, contract string, lockEvents chan *common.MessagePublication, setEvents chan *common.GuardianSet) *Watcher {
	return &Watcher{urlRPC: urlRPC, urlToken: urlToken, contract: contract, msgChan: lockEvents, setChan: setEvents}
}

func lookAtTxn(e *Watcher, t models.Transaction) {
	var at = t.ApplicationTransaction
	if len(at.ApplicationArgs) == 0 {
		return
	}

	JSON, err := json.Marshal(t)
	_ = err
	fmt.Printf(string(JSON))

	fmt.Printf("%d\n", at.ApplicationId)
	if string(at.ApplicationArgs[0]) == "publishMessage" { // The note filter is effectively the same thing
		var vaa = at.ApplicationArgs[1]
		fmt.Printf(t.Sender + " -> " + string(vaa) + "\n")
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	readiness.SetReady(common.ReadinessAlgorandSyncing)

	go func() {
		indexerClient, err := indexer.MakeClient(indexerAddress, indexerToken)
		_ = err

		// Parameters
		var notePrefix = "publishMessage"
		var next_round uint64 = 0

		for true {
			var nextToken = ""
			for true {
				result, err := indexerClient.SearchForTransactions().NotePrefix([]byte(notePrefix)).MinRound(next_round).NextToken(nextToken).Do(context.Background())
				_ = err

				for i := 0; i < len(result.Transactions); i++ {
					var t = result.Transactions[i]
					if len(t.InnerTxns) > 0 {
						for q := 0; q < len(t.InnerTxns); q++ {
							lookAtTxn(t.InnerTxns[q])
						}
					} else {
						lookAtTxn(t)
					}
				}

				if result.NextToken != "" {
					nextToken = result.NextToken
				} else {
					next_round = result.CurrentRound + 1
					break
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
