package aptos

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
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
		chainID   vaa.ChainID
		networkID string

		aptosRPC     string
		aptosAccount string
		aptosHandle  string

		msgC          chan<- *common.MessagePublication
		obsvReqC      <-chan *gossipv1.ObservationRequest
		readinessSync readiness.Component
	}
)

var (
	//nolint:exhaustruct // Intentional design of CounterOpts to not include all items in the struct.
	aptosMessagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_aptos_observations_confirmed_total",
			Help: "Total number of verified observations found for the chain",
		}, []string{"chain_name"})
	//nolint:exhaustruct // Intentional design of GaugeOpts to not include all items in the struct.
	currentAptosHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_aptos_current_height",
			Help: "Current block height for the chain",
		}, []string{"chain_name"})
)

// NewWatcher creates a new Aptos appid watcher
func NewWatcher(
	chainID vaa.ChainID,
	networkID watchers.NetworkID,
	aptosRPC string,
	aptosAccount string,
	aptosHandle string,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		chainID:       chainID,
		networkID:     string(networkID),
		aptosRPC:      aptosRPC,
		aptosAccount:  aptosAccount,
		aptosHandle:   aptosHandle,
		msgC:          msgC,
		obsvReqC:      obsvReqC,
		readinessSync: common.MustConvertChainIdToReadinessSyncing(chainID),
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: e.aptosAccount,
	})

	logger := supervisor.Logger(ctx)

	logger.Info("Starting watcher",
		zap.String("watcher_name", e.networkID),
		zap.String("rpc", e.aptosRPC),
		zap.String("account", e.aptosAccount),
		zap.String("handle", e.aptosHandle),
	)

	// SECURITY: the API guarantees that we only get the events from the right
	// contract
	var eventsEndpoint = fmt.Sprintf(`%s/v1/accounts/%s/events/%s/event`, e.aptosRPC, e.aptosAccount, e.aptosHandle)
	var aptosHealth = fmt.Sprintf(`%s/v1`, e.aptosRPC)

	logger.Info("watcher connecting to RPC node ",
		zap.String("url", e.aptosRPC),
		zap.String("eventsQuery", eventsEndpoint),
		zap.String("healthQuery", aptosHealth),
	)

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
			// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
			// and only writes to the channel for this chain id.
			// If either of the below cases are true, something has gone wrong
			if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != e.chainID {
				panic("invalid chain ID")
			}

			// uint64 will read the *first* 8 bytes, but the sequence is stored in the *last* 8.
			nativeSeq := binary.BigEndian.Uint64(r.TxHash[24:])

			logger.Info("Received obsv request", zap.Uint64("tx_hash", nativeSeq))

			s := fmt.Sprintf(`%s?start=%d&limit=1`, eventsEndpoint, nativeSeq)

			body, err := e.retrievePayload(s)
			if err != nil {
				logger.Error("retrievePayload", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			if !gjson.Valid(string(body)) {
				logger.Error("InvalidJson: " + string(body))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
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
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
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
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
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

				eventSeq := eventSequence.Uint()
				if nextSequence == 0 && eventSeq != 0 {
					// Avoid publishing an old observation on startup. This does not block the first message on a new chain (when eventSeq would be zero).
					nextSequence = eventSeq + 1
					continue
				}

				// this is interesting in the last iteration, whereby we
				// find the next sequence that comes after the array
				nextSequence = eventSeq + 1

				data := event.Get("data")
				if !data.Exists() {
					continue
				}
				e.observeData(logger, data, eventSeq, false)
			}

			health, err := e.retrievePayload(aptosHealth)
			if err != nil {
				logger.Error("health", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			if !gjson.Valid(string(health)) {
				logger.Error("Invalid JSON in health response: " + string(health))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue

			}

			// TODO: Make this log more useful for humans
			logger.Debug(string(health) + string(eventsJson))

			pHealth := gjson.ParseBytes(health)

			blockHeight := pHealth.Get("block_height")

			if blockHeight.Uint() > math.MaxInt64 {
				logger.Error("Block height not a valid uint64: ", zap.Uint64("blockHeight", blockHeight.Uint()))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue
			}

			if blockHeight.Exists() {
				currentAptosHeight.WithLabelValues(e.networkID).Set(float64(blockHeight.Uint()))
				p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
					Height:          int64(blockHeight.Uint()), // #nosec G115 -- This is validated above
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
	defer res.Body.Close()
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

	if nonce.Uint() > math.MaxUint32 {
		logger.Error("nonce is larger than expected MaxUint32")
		return
	}

	if consistencyLevel.Uint() > math.MaxUint8 {
		logger.Error("consistency level is larger than expected MaxUint8")
		return
	}

	observation := &common.MessagePublication{
		TxID:             txHash.Bytes(),
		Timestamp:        time.Unix(int64(ts.Uint()), 0), // #nosec G115 -- This conversion is safe indefinitely
		Nonce:            uint32(nonce.Uint()),           // #nosec G115 -- This is validated above
		Sequence:         sequence.Uint(),
		EmitterChain:     e.chainID,
		EmitterAddress:   a,
		Payload:          pl,
		ConsistencyLevel: uint8(consistencyLevel.Uint()), // #nosec G115 -- This is validated above
		IsReobservation:  isReobservation,
	}

	aptosMessagesConfirmed.WithLabelValues(e.networkID).Inc()
	if isReobservation {
		watchers.ReobservationsByChain.WithLabelValues(e.chainID.String(), "std").Inc()
	}

	logger.Info("message observed",
		zap.String("txHash", observation.TxIDString()),
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
