package algorand

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"time"
	eth_common "github.com/ethereum/go-ethereum/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
)

type (
	// Watcher is responsible for looking over Algorand blockchain and reporting new transactions to the appid
	Watcher struct {
		urlRPC       string
		urlToken     string
		indexerRPC   string
		indexerToken string
		appid        uint64

		msgChan  chan *common.MessagePublication
		setChan  chan *common.GuardianSet
		obsvReqC chan *gossipv1.ObservationRequest
	}
)

var (
	algorandMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_algorand_observations_confirmed_total",
			Help: "Total number of verified Algorand observations found",
		})
	currentAlgorandHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_algorand_current_height",
			Help: "Current Algorand block height",
		})
)


// NewWatcher creates a new Algorand appid watcher
func NewWatcher(
	urlRPC      string,
	urlToken     string,
	indexerRPC   string,
	indexerToken string,
	appid        uint64,
	lockEvents   chan *common.MessagePublication,
	setEvents    chan *common.GuardianSet,
	obsvReqC     chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		urlRPC:       urlRPC,
		urlToken:     urlToken,
		indexerRPC:   indexerRPC,
		indexerToken: indexerToken,
		appid:        appid,
		msgChan:      lockEvents,
		setChan:      setEvents,
		obsvReqC:     obsvReqC,
	}
}

func lookAtTxn(e *Watcher, t models.Transaction, logger *zap.Logger) {
	if len(t.InnerTxns) > 0 {
		for q := 0; q < len(t.InnerTxns); q++ {
			var it = t.InnerTxns[q]
			var at = it.ApplicationTransaction

			if (len(at.ApplicationArgs) == 0) || (at.ApplicationId != e.appid) || (len(it.Logs) == 0) {
				continue
			}

			if string(at.ApplicationArgs[0]) != "publishMessage" {
				continue
			}

			JSON, err := json.Marshal(it)
			_ = err
			logger.Info(string(JSON))

			var seq = binary.BigEndian.Uint64(it.Logs[0])

			emitter, err := types.DecodeAddress(it.Sender)
			var a vaa.Address
			copy(a[:], emitter[:])

			var txHash eth_common.Hash
			copy(txHash[:], it.Id)

			observation := &common.MessagePublication{
				TxHash:           txHash,
				Timestamp:        time.Unix(int64(it.RoundTime), 0),
				Nonce:            0,
				Sequence:         seq,
				EmitterChain:     vaa.ChainIDAlgorand,
				EmitterAddress:   a,
				Payload:          at.ApplicationArgs[1],
				ConsistencyLevel: 32,
			}
			
			algorandMessagesConfirmed.Inc()

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

func (e *Watcher) Run(ctx context.Context) error {
	// an odd thing to broadcast... 
	contractAddr := string(e.appid)
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAlgorand, &gossipv1.Heartbeat_Network{
		ContractAddress: contractAddr,
	})

	logger := supervisor.Logger(ctx)
	errC := make(chan error)

	logger.Info("Algorand watcher connecting", zap.String("url", e.indexerRPC))

	go func() {
		timer := time.NewTicker(time.Second * 1)
		defer timer.Stop()

		indexerClient, err := indexer.MakeClient(e.indexerRPC, e.indexerToken)
		if err != nil {
			logger.Info(err.Error())
			p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
			errC <- err
			return
		}

		// Parameters
		var notePrefix = "publishMessage"
		var next_round uint64 = 0

		for {
			select {
			case <-ctx.Done():
				return
			case r := <-e.obsvReqC:
				// Can somebody tell me how to test this?
				if vaa.ChainID(r.ChainId) != vaa.ChainIDAlgorand {
					panic("invalid chain ID")
				}

				result, err := indexerClient.SearchForTransactions().TXID(string(r.TxHash)).Do(context.Background())
				if err != nil {
					logger.Info(err.Error())
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
					errC <- err
					return
				}
				for i := 0; i < len(result.Transactions); i++ {
					var t = result.Transactions[i]
					lookAtTxn(e, t, logger)
				}

			case <-timer.C:
				var nextToken = ""
				for true {
					result, err := indexerClient.SearchForTransactions().NotePrefix([]byte(notePrefix)).MinRound(next_round).NextToken(nextToken).Do(context.Background())
					if err != nil {
						logger.Info(err.Error())
						p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
						errC <- err
						return
					}

					// Indexer returned "stuff"...  the world is good
					readiness.SetReady(common.ReadinessAlgorandSyncing)
					currentAlgorandHeight.Set(float64(result.CurrentRound))

					for i := 0; i < len(result.Transactions); i++ {
						var t = result.Transactions[i]
						lookAtTxn(e, t, logger)
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
