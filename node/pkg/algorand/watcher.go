package algorand

import (
	"context"
	"encoding/json"
        "fmt"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"go.uber.org/zap"
	"time"
)

type (
	// Watcher is responsible for looking over Algorand blockchain and reporting new transactions to the appid
	Watcher struct {
		urlRPC       string
		urlToken     string
		indexerRPC   string
		indexerToken string
		appid        uint64

		msgChan chan *common.MessagePublication
		setChan chan *common.GuardianSet
	}
)

// NewWatcher creates a new Algorand appid watcher
func NewWatcher(urlRPC string, urlToken string, indexerRPC string, indexerToken string, appid uint64, lockEvents chan *common.MessagePublication, setEvents chan *common.GuardianSet) *Watcher {
	return &Watcher{urlRPC: urlRPC, urlToken: urlToken, indexerRPC: indexerRPC, indexerToken: indexerToken, appid: appid, msgChan: lockEvents, setChan: setEvents}
}

func (e *Watcher) Run(ctx context.Context) error {
	readiness.SetReady(common.ReadinessAlgorandSyncing)

	logger := supervisor.Logger(ctx)
	errC := make(chan error)

	logger.Info("Algorand watcher connecting", zap.String("url", e.indexerRPC))

	go func() {
		timer := time.NewTicker(time.Second * 1)
		defer timer.Stop()

		indexerClient, err := indexer.MakeClient(e.indexerRPC, e.indexerToken)
		_ = err

		// Parameters
		var notePrefix = "publishMessage"
		var next_round uint64 = 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				logger.Info("Algorand tick")

				var nextToken = ""
				for true {
					result, err := indexerClient.SearchForTransactions().NotePrefix([]byte(notePrefix)).MinRound(next_round).NextToken(nextToken).Do(context.Background())
					if err != nil {
						logger.Info(err.Error())
						break
					}

					for i := 0; i < len(result.Transactions); i++ {
						var t = result.Transactions[i]
						if len(t.InnerTxns) > 0 {
							for q := 0; q < len(t.InnerTxns); q++ {
								var it = t.InnerTxns[q]
								var at = it.ApplicationTransaction

//	                                                        logger.Info(fmt.Sprintf("%d", at.ApplicationId))
								if (len(at.ApplicationArgs) == 0) || (at.ApplicationId != e.appid) {
									continue
								}


								if string(at.ApplicationArgs[0]) != "publishMessage" { 
                                                                        continue
                                                                }
                                                                

								JSON, err := json.Marshal(it)
								_ = err
								logger.Info(string(JSON))

	                                                        logger.Info(fmt.Sprintf("%d", at.ApplicationId))
									var vaa = at.ApplicationArgs[1]
									logger.Info(fmt.Sprintf(t.Sender + " -> " + string(vaa)))
							}
						}
					}

					if result.NextToken != "" {
						nextToken = result.NextToken
					} else {
						next_round = result.CurrentRound + 1
						break
					}
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
