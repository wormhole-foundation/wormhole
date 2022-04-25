package algorand

import (
	"context"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		indexerRPC   string
		indexerToken string
		appid        uint64

		msgChan  chan *common.MessagePublication
		setChan  chan *common.GuardianSet
		obsvReqC chan *gossipv1.ObservationRequest

		next_round   uint64
		debug        bool
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
	indexerRPC   string,
	indexerToken string,
	appid        uint64,
	lockEvents   chan *common.MessagePublication,
	setEvents    chan *common.GuardianSet,
	obsvReqC     chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		indexerRPC:   indexerRPC,
		indexerToken: indexerToken,
		appid:        appid,
		msgChan:      lockEvents,
		setChan:      setEvents,
		obsvReqC:     obsvReqC,
		next_round:   0,
		debug:        true,
	}
}

func lookAtTxn(e *Watcher, t models.Transaction, logger *zap.Logger) {
	for q := 0; q < len(t.InnerTxns); q++ {
		var it = t.InnerTxns[q]
		var at = it.ApplicationTransaction

		if (len(at.ApplicationArgs) != 3) || (at.ApplicationId != e.appid) || (len(it.Logs) == 0) {
			continue
		}

		if string(at.ApplicationArgs[0]) != "publishMessage" {
			continue

		}

		if e.debug {
			JSON, err := json.Marshal(it)
			if err != nil {
				logger.Error("Cannot JSON marshal transaction", zap.Error(err))
			} else {
				logger.Info(string(JSON))
			}
		}

		emitter, err := types.DecodeAddress(it.Sender)
		if err != nil {
			logger.Error("DecodeAddress", zap.Error(err))
			continue;
		}

		var a vaa.Address
		copy(a[:], emitter[:]) // 32 bytes = 8edf5b0e108c3a1a0a4b704cc89591f2ad8d50df24e991567e640ed720a94be2
                                                             
		if e.debug {
			logger.Error("emitter: " + hex.EncodeToString(emitter[:]))
		}

		id, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(t.Id)
		if err != nil {
			logger.Error("Base32 DecodeString", zap.Error(err))
			continue;
		}

		if e.debug {
			logger.Info("id: " + hex.EncodeToString(id) + " " + t.Id)
		}

		var txHash = eth_common.BytesToHash(id)   // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

		observation := &common.MessagePublication{
			TxHash:           txHash,
			Timestamp:        time.Unix(int64(it.RoundTime), 0),
			Nonce:            uint32(binary.BigEndian.Uint64(at.ApplicationArgs[2])),
			Sequence:         binary.BigEndian.Uint64(it.Logs[0]),
			EmitterChain:     vaa.ChainIDAlgorand,
			EmitterAddress:   a,
			Payload:          at.ApplicationArgs[1],
			ConsistencyLevel: 0,
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

func (e *Watcher) Run(ctx context.Context) error {
	// an odd thing to broadcast... 
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAlgorand, &gossipv1.Heartbeat_Network{
		ContractAddress: fmt.Sprintf("%d", e.appid),
	})

	logger := supervisor.Logger(ctx)
	errC := make(chan error)

	logger.Info("Algorand watcher connecting", zap.String("url", e.indexerRPC))

	go func() {
		timer := time.NewTicker(time.Second * 1)
		defer timer.Stop()

		indexerClient, err := indexer.MakeClient(e.indexerRPC, e.indexerToken)
		if err != nil {
			logger.Error("indexer make client", zap.Error(err))
			p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
			errC <- err
			return
		}

		// Parameters
		var notePrefix = "publishMessage"

		if e.next_round == 0 {
			// What is the latest round...
			result, err := indexerClient.SearchForTransactions().Limit(0).Do(context.Background())
			if err != nil {
				logger.Error("SearchForTransaction", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
				errC <- err
				return
			}
			e.next_round = result.CurrentRound + 1
			logger.Info(fmt.Sprintf("Algorand next_round set to %d", e.next_round))
		}

		for {
			select {
			case <-ctx.Done():
				return
			case r := <-e.obsvReqC:
				if vaa.ChainID(r.ChainId) != vaa.ChainIDAlgorand {
					panic("invalid chain ID")
				}

				logger.Info("Received obsv request",
					zap.String("tx_hash", hex.EncodeToString(r.TxHash)),
					zap.String("base32_tx_hash", base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(r.TxHash)))

				result, err := indexerClient.SearchForTransactions().TXID(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(r.TxHash)).Do(context.Background())
				if err != nil {
					logger.Error("SearchForTransactions", zap.Error(err))
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
				for {
					result, err := indexerClient.SearchForTransactions().NotePrefix([]byte(notePrefix)).MinRound(e.next_round).NextToken(nextToken).Do(context.Background())
					if err != nil {
						logger.Error("SearchForTransaction", zap.Error(err))
						p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
						errC <- err
						return
					}

					for i := 0; i < len(result.Transactions); i++ {
						var t = result.Transactions[i]
						lookAtTxn(e, t, logger)
					}

					if result.NextToken != "" {
						nextToken = result.NextToken
					} else {
						e.next_round = result.CurrentRound + 1
						break
					}
				}
				readiness.SetReady(common.ReadinessAlgorandSyncing)
				currentAlgorandHeight.Set(float64(e.next_round - 1))
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAlgorand, &gossipv1.Heartbeat_Network{
					Height:          int64(e.next_round - 1),
					ContractAddress: fmt.Sprintf("%d", e.appid),
				})
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
