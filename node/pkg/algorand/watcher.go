package algorand

import (
	"context"
	"encoding/base64"
	"encoding/binary"
//	"encoding/hex"
	"encoding/json"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"go.uber.org/zap"
	eth_common "github.com/ethereum/go-ethereum/common"	
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
;
								if (len(at.ApplicationArgs) == 0) || (at.ApplicationId != e.appid) || (len(it.Logs) == 0) {
									continue
								}

								if string(at.ApplicationArgs[0]) != "publishMessage" { 
                                                                        continue
                                                                }

								payload, err := base64.StdEncoding.DecodeString(string(at.ApplicationArgs[1]))
								seqStr, err := base64.StdEncoding.DecodeString(string(it.Logs[0]))
								var seq = binary.BigEndian.Uint64(seqStr)

								emitter, err := types.DecodeAddress(it.Sender)
								var a vaa.Address
								copy(a[:], emitter[:])

								JSON, err := json.Marshal(it)
								_ = err
								logger.Info(string(JSON))

								var txHash eth_common.Hash
//								copy(txHash[:], acc[:])
//
								observation := &common.MessagePublication{
									TxHash:           txHash,
									Timestamp:        time.Unix(0, 0),
									Nonce:            0,
									Sequence:         seq,
									EmitterChain:     vaa.ChainIDAlgorand,
									EmitterAddress:   a,
									Payload:          payload,
									ConsistencyLevel: 32,
								}
//
//								solanaMessagesConfirmed.Inc()

								logger.Info("message observed",
									zap.Time("timestamp", observation.Timestamp),
									zap.Uint32("nonce", observation.Nonce),
									zap.Uint64("sequence", observation.Sequence),
									zap.Stringer("emitter_chain", observation.EmitterChain),
									zap.Stringer("emitter_address", observation.EmitterAddress),
									zap.Binary("payload", observation.Payload),
									zap.Uint8("consistency_level", observation.ConsistencyLevel),
								)

								e.msgChan <- observation
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
