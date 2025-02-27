package algorand

import (
	"context"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
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
	"github.com/certusone/wormhole/node/pkg/watchers"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Algorand allows max depth of 8 inner transactions
const MAX_DEPTH = 8

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

	algorandObservation struct {
		emitterAddress vaa.Address
		nonce          uint32
		sequence       uint64
		payload        []byte
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

// gatherObservations recurses through a given transactions inner-transactions
// to find any messages emitted from the core wormhole contract.
// Algorand allows up to 8 levels of inner transactions.
func gatherObservations(e *Watcher, t types.SignedTxnWithAD, depth int, logger *zap.Logger) (obs []algorandObservation) {

	// SECURITY defense-in-depth: don't recurse > max depth allowed by Algorand
	if depth >= MAX_DEPTH {
		logger.Error("algod client", zap.Error(fmt.Errorf("exceeded max depth of %d", MAX_DEPTH)))
		return
	}

	// recurse through nested inner transactions
	for _, itxn := range t.EvalDelta.InnerTxns {
		obs = append(obs, gatherObservations(e, itxn, depth+1, logger)...)
	}

	var at = t.Txn
	var ed = t.EvalDelta

	// check if the current transaction meets what we expect
	// for an emitted message
	if (len(at.ApplicationArgs) != 3) || (uint64(at.ApplicationID) != e.appid) || string(at.ApplicationArgs[0]) != "publishMessage" || len(ed.Logs) == 0 {
		return
	}

	logger.Info("emitter: " + hex.EncodeToString(at.Sender[:]))

	var a vaa.Address
	copy(a[:], at.Sender[:]) // 32 bytes = 8edf5b0e108c3a1a0a4b704cc89591f2ad8d50df24e991567e640ed720a94be2

	obs = append(obs, algorandObservation{
		nonce:          uint32(binary.BigEndian.Uint64(at.ApplicationArgs[2])), // #nosec G115 -- Nonce is 32 bits on chain
		sequence:       binary.BigEndian.Uint64([]byte(ed.Logs[0])),
		emitterAddress: a,
		payload:        at.ApplicationArgs[1],
	})

	return
}

// lookAtTxn takes an outer transaction from the block.payset and gathers
// observations from messages emitted in nested inner transactions
// then passes them on the relevant channels
func lookAtTxn(e *Watcher, t types.SignedTxnInBlock, b types.Block, logger *zap.Logger, isReobservation bool) {

	observations := gatherObservations(e, t.SignedTxnWithAD, 0, logger)

	// We use the outermost transaction id in the observation message
	// so we can apply the same logic to gather any messages emitted
	// by inner transactions
	var txHash eth_common.Hash
	if len(observations) > 0 {
		// Repopulate the genesis id/hash for the transaction
		// since in the block encoding, it's omitted to save space
		t.Txn.GenesisID = b.GenesisID
		t.Txn.GenesisHash = b.GenesisHash
		Id := crypto.GetTxID(t.Txn)

		id, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(Id)
		if err != nil {
			logger.Error("Base32 DecodeString", zap.Error(err))
			return
		}
		logger.Info("id: " + hex.EncodeToString(id) + " " + Id)

		txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b
	}

	for _, obs := range observations {
		observation := &common.MessagePublication{
			TxID:             txHash.Bytes(),
			Timestamp:        time.Unix(b.TimeStamp, 0),
			Nonce:            obs.nonce,
			Sequence:         obs.sequence,
			EmitterChain:     vaa.ChainIDAlgorand,
			EmitterAddress:   obs.emitterAddress,
			Payload:          obs.payload,
			ConsistencyLevel: 0,
			IsReobservation:  isReobservation,
		}

		algorandMessagesConfirmed.Inc()
		if isReobservation {
			watchers.ReobservationsByChain.WithLabelValues("algorand", "std").Inc()
		}

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

	logger.Info("Starting watcher",
		zap.String("watcher_name", "algorand"),
		zap.String("indexerRPC", e.indexerRPC),
		zap.String("indexerToken", e.indexerToken),
		zap.String("algodRPC", e.algodRPC),
		zap.String("algodToken", e.algodToken),
		zap.Uint64("appid", e.appid),
	)

	logger.Info("Algorand watcher connecting to indexer  ", zap.String("url", e.indexerRPC))
	logger.Info("Algorand watcher connecting to RPC node ", zap.String("url", e.algodRPC))

	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

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

	status, err := algodClient.StatusAfterBlock(0).Do(ctx)
	if err != nil {
		logger.Error("StatusAfterBlock", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
		return err
	}

	e.next_round = status.LastRound + 1

	logger.Info(fmt.Sprintf("first block %d", e.next_round))

	for {
		select {
		case <-ctx.Done():
			return nil
		case r := <-e.obsvReqC:
			// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
			// and only writes to the channel for this chain id.
			// If either of the below cases are true, something has gone wrong
			if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != vaa.ChainIDAlgorand {
				panic("invalid chain ID")
			}

			logger.Info("Received obsv request",
				zap.String("tx_hash", hex.EncodeToString(r.TxHash)),
				zap.String("base32_tx_hash", base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(r.TxHash)))

			result, err := indexerClient.SearchForTransactions().TXID(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(r.TxHash)).Do(ctx)
			if err != nil {
				logger.Error("SearchForTransactions", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
				break
			}
			for _, t := range result.Transactions {
				r := t.ConfirmedRound

				block, err := algodClient.Block(r).Do(ctx)
				if err != nil {
					logger.Error("SearchForTransactions", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
					break
				}

				for _, element := range block.Payset {
					lookAtTxn(e, element, block, logger, true)
				}
			}

		case <-timer.C:
			status, err := algodClient.Status().Do(ctx)
			if err != nil {
				logger.Error(fmt.Sprintf("algodClient.Status: %s", err.Error()))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
				continue
			}

			if e.next_round <= status.LastRound {
				for {
					block, err := algodClient.Block(e.next_round).Do(ctx)
					if err != nil {
						logger.Error(fmt.Sprintf("algodClient.Block %d: %s", e.next_round, err.Error()))
						p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
						break
					}

					if block.Round == 0 {
						break
					}

					for _, element := range block.Payset {
						lookAtTxn(e, element, block, logger, false)
					}
					e.next_round = e.next_round + 1

					if e.next_round > status.LastRound {
						break
					}
				}
			}

			if status.LastRound > math.MaxInt64 {
				logger.Error("Last round not a valid int64: ", zap.Uint64("lastRound", status.LastRound))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAlgorand, 1)
				continue
			}

			currentAlgorandHeight.Set(float64(status.LastRound))
			p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAlgorand, &gossipv1.Heartbeat_Network{
				Height:          int64(status.LastRound), // #nosec G115 -- This is validated above
				ContractAddress: fmt.Sprintf("%d", e.appid),
			})

			readiness.SetReady(e.readinessSync)
		}
	}
}
