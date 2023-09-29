package aptos

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Aptos blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		aptosRPC     string
		aptosAccount string
		aptosHandle  string

		msgC          chan<- *common.MessagePublication
		obsvReqC      <-chan *gossipv1.ObservationRequest
		readinessSync readiness.Component
	}
)

var (
	aptosMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aptos_observations_confirmed_total",
			Help: "Total number of verified Aptos observations found",
		})
	currentAptosHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_aptos_current_height",
			Help: "Current Aptos block height",
		})
)

// NewWatcher creates a new Aptos appid watcher
func NewWatcher(
	aptosRPC string,
	aptosAccount string,
	aptosHandle string,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		aptosRPC:      aptosRPC,
		aptosAccount:  aptosAccount,
		aptosHandle:   aptosHandle,
		msgC:          msgC,
		obsvReqC:      obsvReqC,
		readinessSync: common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDAptos),
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAptos, &gossipv1.Heartbeat_Network{
		ContractAddress: e.aptosAccount,
	})

	logger := supervisor.Logger(ctx)

	logger.Info("Starting watcher",
		zap.String("watcher_name", "aptos"),
		zap.String("aptosRPC", e.aptosRPC),
		zap.String("aptosAccount", e.aptosAccount),
		zap.String("aptosHandle", e.aptosHandle),
	)

	logger.Info("Aptos watcher connecting to RPC node ", zap.String("url", e.aptosRPC))

	// SECURITY: the API guarantees that we only get the events from the right
	// contract
	var eventsEndpoint = fmt.Sprintf(`%s/v1/accounts/%s/events/%s/event`, e.aptosRPC, e.aptosAccount, e.aptosHandle)
	var aptosHealth = fmt.Sprintf(`%s/v1`, e.aptosRPC)

	// the events have sequence numbers associated with them in the aptos API
	// (NOTE: this is not the same as the wormhole sequence id). The event
	// endpoint is paginated, so we use this variable to keep track of which
	// sequence number to look up next.
	var nextSequence uint64 = 0

	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-e.obsvReqC:
			if vaa.ChainID(r.ChainId) != vaa.ChainIDAptos {
				panic("invalid chain ID")
			}

			// uint64 will read the *first* 8 bytes, but the sequence is stored in the *last* 8.
			nativeSeq := binary.BigEndian.Uint64(r.TxHash[24:])

			logger.Info("Received obsv request", zap.Uint64("tx_hash", nativeSeq))

			s := fmt.Sprintf(`%s?start=%d&limit=1`, eventsEndpoint, nativeSeq)

			body, err := e.retrievePayload(s)
			if err != nil {
				logger.Error("retrievePayload", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue
			}

			if !gjson.Valid(string(body)) {
				logger.Error("InvalidJson: " + string(body))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				break

			}

			outcomes := gjson.ParseBytes(body)

			for _, chunk := range outcomes.Array() {
				newSeq := chunk.Get("sequence_number")
				if !newSeq.Exists() {
					break
				}

				if newSeq.Uint() != nativeSeq {
					logger.Error("newSeq != nativeSeq")
					break

				}

				data := chunk.Get("data")
				if !data.Exists() {
					break
				}
				e.observeData(logger, data, nativeSeq, true)
			}

		case <-timer.C:
			s := ""

			if nextSequence == 0 {
				// if nextSequence is 0, we look up the most recent event
				s = fmt.Sprintf(`%s?limit=1`, eventsEndpoint)
			} else {
				// otherwise just look up events starting at nextSequence.
				// this will potentially return multiple events (whatever
				// the default limit is per page), so we'll handle all of them.
				s = fmt.Sprintf(`%s?start=%d`, eventsEndpoint, nextSequence)
			}

			eventsJson, err := e.retrievePayload(s)
			if err != nil {
				logger.Error("retrievePayload", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue
			}

			// data doesn't exist yet. skip, and try again later
			// this happens when the sequence id we're looking up hasn't
			// been used yet.
			if string(eventsJson) == "" {
				continue
			}

			if !gjson.Valid(string(eventsJson)) {
				logger.Error("InvalidJson: " + string(eventsJson))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue

			}

			events := gjson.ParseBytes(eventsJson)

			// the endpoint returns an array of events, ordered by sequence
			// id (ASC)
			for _, event := range events.Array() {
				eventSequence := event.Get("sequence_number")
				if !eventSequence.Exists() {
					continue
				}

				// this is interesting in the last iteration, whereby we
				// find the next sequence that comes after the array
				nextSequence = eventSequence.Uint() + 1

				data := event.Get("data")
				if !data.Exists() {
					continue
				}
				e.observeData(logger, data, eventSequence.Uint(), false)
			}

			health, err := e.retrievePayload(aptosHealth)
			if err != nil {
				logger.Error("health", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue
			}

			if !gjson.Valid(string(health)) {
				logger.Error("Invalid JSON in health response: " + string(health))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue

			}

			// TODO: Make this log more useful for humans
			logger.Debug(string(health) + string(eventsJson))

			pHealth := gjson.ParseBytes(health)

			blockHeight := pHealth.Get("block_height")

			if blockHeight.Exists() {
				currentAptosHeight.Set(float64(blockHeight.Uint()))
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAptos, &gossipv1.Heartbeat_Network{
					Height:          int64(blockHeight.Uint()),
					ContractAddress: e.aptosAccount,
				})

				readiness.SetReady(e.readinessSync)
			}
		}
	}
}

func (e *Watcher) retrievePayload(s string) ([]byte, error) {
	res, err := http.Get(s) // nolint
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

func (e *Watcher) observeData(logger *zap.Logger, data gjson.Result, nativeSeq uint64, isReobservation bool) {
	em := data.Get("sender")
	if !em.Exists() {
		logger.Error("sender field missing")
		return
	}

	emitter := make([]byte, 8)
	binary.BigEndian.PutUint64(emitter, em.Uint())

	var a vaa.Address
	copy(a[24:], emitter)

	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, nativeSeq)

	var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

	v := data.Get("payload")
	if !v.Exists() {
		logger.Error("payload field missing")
		return
	}

	pl, err := hex.DecodeString(v.String()[2:])
	if err != nil {
		logger.Error("payload decode")
		return
	}

	ts := data.Get("timestamp")
	if !ts.Exists() {
		logger.Error("timestamp field missing")
		return
	}

	nonce := data.Get("nonce")
	if !nonce.Exists() {
		logger.Error("nonce field missing")
		return
	}

	sequence := data.Get("sequence")
	if !sequence.Exists() {
		logger.Error("sequence field missing")
		return
	}

	consistencyLevel := data.Get("consistency_level")
	if !consistencyLevel.Exists() {
		logger.Error("consistencyLevel field missing")
		return
	}

	observation := &common.MessagePublication{
		TxHash:           txHash,
		Timestamp:        time.Unix(int64(ts.Uint()), 0),
		Nonce:            uint32(nonce.Uint()), // uint32
		Sequence:         sequence.Uint(),
		EmitterChain:     vaa.ChainIDAptos,
		EmitterAddress:   a,
		Payload:          pl,
		ConsistencyLevel: uint8(consistencyLevel.Uint()),
		IsReobservation:  isReobservation,
	}

	aptosMessagesConfirmed.Inc()

	logger.Info("message observed",
		zap.Stringer("txHash", observation.TxHash),
		zap.Time("timestamp", observation.Timestamp),
		zap.Uint32("nonce", observation.Nonce),
		zap.Uint64("sequence", observation.Sequence),
		zap.Stringer("emitter_chain", observation.EmitterChain),
		zap.Stringer("emitter_address", observation.EmitterAddress),
		zap.Binary("payload", observation.Payload),
		zap.Uint8("consistencyLevel", observation.ConsistencyLevel),
	)

	e.msgC <- observation
}
