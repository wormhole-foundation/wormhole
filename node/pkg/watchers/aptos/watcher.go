package aptos

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
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

		msgChan  chan *common.MessagePublication
		obsvReqC chan *gossipv1.ObservationRequest
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
	messageEvents chan *common.MessagePublication,
	obsvReqC chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		aptosRPC:     aptosRPC,
		aptosAccount: aptosAccount,
		aptosHandle:  aptosHandle,
		msgChan:      messageEvents,
		obsvReqC:     obsvReqC,
	}
}

func (e *Watcher) retrievePayload(s string) ([]byte, error) {
	res, err := http.Get(s) // nolint
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

func (e *Watcher) observeData(logger *zap.Logger, data gjson.Result, native_seq uint64) {
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
	binary.BigEndian.PutUint64(id, native_seq)

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

	consistency_level := data.Get("consistency_level")
	if !consistency_level.Exists() {
		logger.Error("consistency_level field missing")
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
		ConsistencyLevel: uint8(consistency_level.Uint()),
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
		zap.Uint8("consistency_level", observation.ConsistencyLevel),
	)

	e.msgChan <- observation
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAptos, &gossipv1.Heartbeat_Network{
		ContractAddress: e.aptosAccount,
	})

	logger := supervisor.Logger(ctx)
	errC := make(chan error)

	logger.Info("Aptos watcher connecting to RPC node ", zap.String("url", e.aptosRPC))

	// SECURITY: the API guarantees that we only get the events from the right
	// contract
	var eventsEndpoint string = fmt.Sprintf(`%s/v1/accounts/%s/events/%s/event`, e.aptosRPC, e.aptosAccount, e.aptosHandle)
	var aptosHealth string = fmt.Sprintf(`%s/v1`, e.aptosRPC)

	// the events have sequence numbers associated with them in the aptos API
	// (NOTE: this is not the same as the wormhole sequence id). The event
	// endpoint is paginated, so we use this variable to keep track of which
	// sequence number to look up next.
	var next_sequence uint64 = 0

	go func() {
		timer := time.NewTicker(time.Second * 1)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case r := <-e.obsvReqC:
				if vaa.ChainID(r.ChainId) != vaa.ChainIDAptos {
					panic("invalid chain ID")
				}

				native_seq := binary.BigEndian.Uint64(r.TxHash)

				logger.Info("Received obsv request", zap.Uint64("tx_hash", native_seq))

				s := fmt.Sprintf(`%s?start=%d&limit=1`, eventsEndpoint, native_seq)

				body, err := e.retrievePayload(s)
				if err != nil {
					logger.Error("retrievePayload", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
					errC <- err
					break
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

					if newSeq.Uint() != native_seq {
						logger.Error("newSeq != native_seq")
						break

					}

					data := chunk.Get("data")
					if !data.Exists() {
						break
					}
					e.observeData(logger, data, native_seq)
				}

			case <-timer.C:
				s := ""

				if next_sequence == 0 {
					// if next_sequence is 0, we look up the most recent event
					s = fmt.Sprintf(`%s?limit=1`, eventsEndpoint)
				} else {
					// otherwise just look up events starting at next_sequence.
					// this will potentially return multiple events (whatever
					// the default limit is per page), so we'll handle all of them.
					s = fmt.Sprintf(`%s?start=%d`, eventsEndpoint, next_sequence)
				}

				events_json, err := e.retrievePayload(s)
				if err != nil {
					logger.Error("retrievePayload", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
					errC <- err
					break
				}

				// data doesn't exist yet. skip, and try again later
				// this happens when the sequence id we're looking up hasn't
				// been used yet.
				if string(events_json) == "" {
					continue
				}

				if !gjson.Valid(string(events_json)) {
					logger.Error("InvalidJson: " + string(events_json))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
					break

				}

				events := gjson.ParseBytes(events_json)

				// the endpoint returns an array of events, ordered by sequence
				// id (ASC)
				for _, event := range events.Array() {
					event_sequence := event.Get("sequence_number")
					if !event_sequence.Exists() {
						continue
					}

					// this is interesting in the last iteration, whereby we
					// find the next sequence that comes after the array
					next_sequence = event_sequence.Uint() + 1

					data := event.Get("data")
					if !data.Exists() {
						continue
					}
					e.observeData(logger, data, event_sequence.Uint())
				}

				health, err := e.retrievePayload(aptosHealth)
				if err != nil {
					logger.Error("health", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
					errC <- err
					break
				}

				if !gjson.Valid(string(health)) {
					logger.Error("Invalid JSON in health response: " + string(health))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
					continue

				}

				logger.Info(string(health) + string(events_json))

				phealth := gjson.ParseBytes(health)

				block_height := phealth.Get("block_height")

				if block_height.Exists() {
					currentAptosHeight.Set(float64(block_height.Uint()))
					p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDAptos, &gossipv1.Heartbeat_Network{
						Height:          int64(block_height.Uint()),
						ContractAddress: e.aptosAccount,
					})

					readiness.SetReady(common.ReadinessAptosSyncing)
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
