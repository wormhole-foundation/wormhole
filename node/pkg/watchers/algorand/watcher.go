package algorand

import (
	"context"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Algorand blockchain and reporting new transactions to the appid
	Watcher struct {
		indexerRPC   string
		indexerToken string
		algodRPC     string
		algodToken   string
		appid        uint64

		msgC          chan<- *common.MessagePublication
		obsvReqC      <-chan *gossipv1.ObservationRequest
		readinessSync readiness.Component

		next_round uint64
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
	indexerRPC string,
	indexerToken string,
	algodRPC string,
	algodToken string,
	appid uint64,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		indexerRPC:    indexerRPC,
		indexerToken:  indexerToken,
		algodRPC:      algodRPC,
		algodToken:    algodToken,
		appid:         appid,
		msgC:          msgC,
		obsvReqC:      obsvReqC,
		readinessSync: common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDAlgorand),
		next_round:    0,
	}
}

func lookAtTxn(e *Watcher, t types.SignedTxnInBlock, b types.Block, logger *zap.Logger) {
	for q := 0; q < len(t.EvalDelta.InnerTxns); q++ {
		var it = t.EvalDelta.InnerTxns[q]
		var at = it.Txn

		if (len(at.ApplicationArgs) != 3) || (uint64(at.ApplicationID) != e.appid) {
			continue
		}

		if string(at.ApplicationArgs[0]) != "publishMessage" {
			continue
		}

		var ed = it.EvalDelta
		if len(ed.Logs) == 0 {
			continue
		}

		emitter := at.Sender

		var a vaa.Address
		copy(a[:], emitter[:]) // 32 bytes = 8edf5b0e108c3a1a0a4b704cc89591f2ad8d50df24e991567e640ed720a94be2

		logger.Info("emitter: " + hex.EncodeToString(emitter[:]))

		t.Txn.GenesisID = b.GenesisID
		t.Txn.GenesisHash = b.GenesisHash
		Id := crypto.GetTxID(t.Txn)

		id, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(Id)
		if err != nil {
			logger.Error("Base32 DecodeString", zap.Error(err))
			continue
		}

		logger.Info("id: " + hex.EncodeToString(id) + " " + Id)

		var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

		observation := &common.MessagePublication{
			TxHash:           txHash,
			Timestamp:        time.Unix(int64(b.TimeStamp), 0),
			Nonce:            uint32(binary.BigEndian.Uint64(at.ApplicationArgs[2])),
			Sequence:         binary.BigEndian.Uint64([]byte(ed.Logs[0])),
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

		e.msgC <- observation
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	// an odd thing to broadcast...
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAlgorand, &gossipv1.Heartbeat_Network{
		ContractAddress: fmt.Sprintf("%d", e.appid),
	})

	logger := supervisor.Logger(ctx)

	logger.Info("Algorand watcher connecting to indexer  ", zap.String("url", e.indexerRPC))
	logger.Info("Algorand watcher connecting to RPC node ", zap.String("url", e.algodRPC))

	// The indexerClient is used to get the transaction by hash for a re-observation
	indexerClient, err := indexer.MakeClient(e.indexerRPC, e.indexerToken)
	if err != nil {
		logger.Error("indexer make client", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
		return err
	}

	algodClient, err := algod.MakeClient(e.algodRPC, e.algodToken)
	if err != nil {
		logger.Error("algod client", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
		return err
	}

	status, err := algodClient.StatusAfterBlock(0).Do(context.Background())
	if err != nil {
		logger.Error("StatusAfterBlock", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
		return err
	}

	e.next_round = status.LastRound + 1

	logger.Info(fmt.Sprintf("first block %d", e.next_round))

	// Create the timer for the getting the block height
	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	// Create an error channel
	errC := make(chan error)
	defer close(errC)

	// Signal that basic initialization is complete
	readiness.SetReady(e.readinessSync)

	// Signal to the supervisor that this runnable has finished initialization
	supervisor.Signal(ctx, supervisor.SignalHealthy)

	// Create the go routine to handle events from core contract
	common.RunWithScissors(ctx, errC, "core_events_and_block_height", func(ctx context.Context) error {
		logger.Info("Entering core_events_and_block_height...")

		for {
			select {
			case err := <-errC:
				logger.Error("core_events_and_block_height died", zap.Error(err))
				return fmt.Errorf("core_events_and_block_height died: %w", err)
			case <-ctx.Done():
				logger.Error("core_events_and_block_height context done")
				return ctx.Err()

			case <-timer.C:
				status, err := algodClient.Status().Do(context.Background())
				if err != nil {
					logger.Error("algodClient.Status", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
					continue
				}

				if e.next_round <= status.LastRound {
					for {
						block, err := algodClient.Block(e.next_round).Do(context.Background())
						if err != nil {
							logger.Error("algodClient.Block %d: %s", zap.Uint64("next_round", e.next_round), zap.Error(err))
							p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
							break
						}

						if block.Round == 0 {
							break
						}

						for _, element := range block.Payset {
							lookAtTxn(e, element, block, logger)
						}
						e.next_round = e.next_round + 1

						if e.next_round > status.LastRound {
							break
						}
					}
				}

				currentAlgorandHeight.Set(float64(status.LastRound))
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAlgorand, &gossipv1.Heartbeat_Network{
					Height:          int64(status.LastRound),
					ContractAddress: fmt.Sprintf("%d", e.appid),
				})

				readiness.SetReady(e.readinessSync)
			}
		}
	})

	// Create the go routine to listen for re-observation requests
	common.RunWithScissors(ctx, errC, "fetch_obvs_req", func(ctx context.Context) error {
		logger.Info("Entering fetch_obvs_req...")

		for {
			select {
			case err := <-errC:
				logger.Error("fetch_obvs_req died", zap.Error(err))
				return fmt.Errorf("fetch_obvs_req died: %w", err)
			case <-ctx.Done():
				logger.Error("fetch_obvs_req context done")
				return ctx.Err()

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
					break
				}
				for _, t := range result.Transactions {
					r := t.ConfirmedRound

					block, err := algodClient.Block(r).Do(context.Background())
					if err != nil {
						logger.Error("SearchForTransactions", zap.Error(err))
						p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
						break
					}

					for _, element := range block.Payset {
						lookAtTxn(e, element, block, logger)
					}
				}
			}
		}
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
