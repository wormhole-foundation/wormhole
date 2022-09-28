package near

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Near blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		mainnet bool

		nearRPC          string
		wormholeContract string

		msgChan  chan *common.MessagePublication
		obsvReqC chan *gossipv1.ObservationRequest

		next_round  uint64
		final_round uint64

		pending   map[pendingKey]*pendingMessage
		pendingMu sync.Mutex
	}

	pendingKey struct {
		hash string
	}

	pendingMessage struct {
		height uint64
	}
)

var (
	nearMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_near_observations_confirmed_total",
			Help: "Total number of verified Near observations found",
		})
	currentNearHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_near_current_height",
			Help: "Current Near block height",
		})
)

// NewWatcher creates a new Near appid watcher
func NewWatcher(
	nearRPC string,
	wormholeContract string,
	lockEvents chan *common.MessagePublication,
	obsvReqC chan *gossipv1.ObservationRequest,
	mainnet bool,
) *Watcher {
	return &Watcher{
		nearRPC:          nearRPC,
		wormholeContract: wormholeContract,
		msgChan:          lockEvents,
		obsvReqC:         obsvReqC,
		next_round:       0,
		final_round:      0,
		pending:          map[pendingKey]*pendingMessage{},
		mainnet:          mainnet,
	}
}

func (e *Watcher) getBlock(block uint64) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": %d}}`, block)
	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (e *Watcher) getBlockHash(block_id string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": "%s"}}`, block_id)
	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		// TODO: We should look at the specifics of the error before we try twice
		resp, err = http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (e *Watcher) getFinalBlock() ([]byte, error) {
	s := `{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"finality": "final"}}`
	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (e *Watcher) getChunk(chunk string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "chunk", "params": {"chunk_id": "%s"}}`, chunk)

	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (e *Watcher) getTxStatus(logger *zap.Logger, tx string, src string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "EXPERIMENTAL_tx_status", "params": ["%s", "%s"]}`, tx, src)

	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (e *Watcher) parseStatus(logger *zap.Logger, t []byte, hash string) error {
	outcomes := gjson.ParseBytes(t).Get("result.receipts_outcome")

	if !outcomes.Exists() {
		return nil
	}

	for _, o := range outcomes.Array() {
		outcome := o.Get("outcome")
		if !outcome.Exists() {
			continue
		}

		executor_id := outcome.Get("executor_id")
		if !executor_id.Exists() {
			continue
		}

		if executor_id.String() == e.wormholeContract {
			l := outcome.Get("logs")
			if !l.Exists() {
				continue
			}
			block_hash := o.Get("block_hash")
			if !block_hash.Exists() {
				logger.Error("block_hash key not found")
				continue
			}
			for _, log := range l.Array() {
				event := log.String()
				if !strings.HasPrefix(event, "EVENT_JSON:") {
					continue
				}
				logger.Info("event", zap.String("event", event[11:]))

				event_json := gjson.ParseBytes([]byte(event[11:]))

				standard := event_json.Get("standard")
				if !standard.Exists() || standard.String() != "wormhole" {
					continue
				}
				event_type := event_json.Get("event")
				if !event_type.Exists() || event_type.String() != "publish" {
					continue
				}

				em := event_json.Get("emitter")
				if !em.Exists() {
					continue
				}

				emitter, err := hex.DecodeString(em.String())
				if err != nil {
					return err
				}

				var a vaa.Address
				copy(a[:], emitter)

				id, err := base58.Decode(hash)
				if err != nil {
					return err
				}

				var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

				v := event_json.Get("data")
				if !v.Exists() {
					logger.Info("data")
					return nil
				}

				pl, err := hex.DecodeString(v.String())
				if err != nil {
					return err
				}

				block_hash_str := block_hash.String()

				txBlock, err := e.getBlockHash(block_hash_str)
				if err != nil {
					return err
				}
				body := gjson.ParseBytes(txBlock)
				if !body.Exists() {
					return errors.New("block parse error")
				}
				ts_nanosec := body.Get("result.header.timestamp")
				if !ts_nanosec.Exists() {
					return errors.New("block parse error, missing timestamp")
				}
				ts := uint64(ts_nanosec.Uint()) / 1000000000

				if e.mainnet {
					height := body.Get("result.header.height")
					if height.Exists() && height.Uint() < 74473147 {
						return errors.New("test missing observe")
					}
				}

				observation := &common.MessagePublication{
					TxHash:           txHash,
					Timestamp:        time.Unix(int64(ts), 0),
					Nonce:            uint32(event_json.Get("nonce").Uint()), // uint32
					Sequence:         event_json.Get("seq").Uint(),
					EmitterChain:     vaa.ChainIDNear,
					EmitterAddress:   a,
					Payload:          pl,
					ConsistencyLevel: 0,
				}

				nearMessagesConfirmed.Inc()

				logger.Info("message observed",
					zap.Uint64("ts", ts),
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

	return nil
}

func (e *Watcher) inspectStatus(logger *zap.Logger, hash string, receiver_id string) error {
	t, err := e.getTxStatus(logger, hash, receiver_id)

	if err != nil {
		return err
	}

	return e.parseStatus(logger, t, hash)
}

func (e *Watcher) lastBlock(logger *zap.Logger, hash string, receiver_id string) ([]byte, uint64, error) {
	t, err := e.getTxStatus(logger, hash, receiver_id)

	if err != nil {
		return nil, 0, err
	}

	last_block := uint64(0)

	outcomes := gjson.ParseBytes(t).Get("result.receipts_outcome")

	if !outcomes.Exists() {
		return nil, 0, err
	}

	for _, o := range outcomes.Array() {
		outcome := o.Get("outcome")
		if !outcome.Exists() {
			continue
		}

		executor_id := outcome.Get("executor_id")
		if !executor_id.Exists() {
			continue
		}

		if executor_id.String() == e.wormholeContract {
			l := outcome.Get("logs")
			if !l.Exists() {
				continue
			}
			for _, log := range l.Array() {
				event := log.String()
				if !strings.HasPrefix(event, "EVENT_JSON:") {
					continue
				}
				logger.Info("event", zap.String("event", event[11:]))

				event_json := gjson.ParseBytes([]byte(event[11:]))

				standard := event_json.Get("standard")
				if !standard.Exists() || standard.String() != "wormhole" {
					continue
				}
				event_type := event_json.Get("event")
				if !event_type.Exists() || event_type.String() != "publish" {
					continue
				}

				block := event_json.Get("block")
				if !block.Exists() {
					continue
				}

				b := block.Uint()

				if b > last_block {
					last_block = b
				}
			}
		}
	}

	return t, last_block, nil
}

func (e *Watcher) inspectBody(logger *zap.Logger, block uint64, body gjson.Result) error {
	logger.Info("inspectBody", zap.Uint64("block", block))

	result := body.Get("result.chunks.#.chunk_hash")
	if !result.Exists() {
		return nil
	}

	for _, name := range result.Array() {
		chunk, err := e.getChunk(name.String())
		if err != nil {
			return err
		}

		txns := gjson.ParseBytes(chunk).Get("result.transactions")
		if !txns.Exists() {
			continue
		}
		for _, r := range txns.Array() {
			hash := r.Get("hash")
			receiver_id := r.Get("receiver_id")
			if !hash.Exists() || !receiver_id.Exists() {
				continue
			}

			t, round, err := e.lastBlock(logger, hash.String(), receiver_id.String())
			if err != nil {
				return err
			}
			if round != 0 {

				if round <= e.final_round {
					logger.Info("parseStatus direct", zap.Uint64("block.height", round), zap.Uint64("e.final_round", e.final_round))
					err := e.parseStatus(logger, t, hash.String())
					if err != nil {
						return err
					}
				} else {
					logger.Info("pushing pending",
						zap.Uint64("block.height", round),
						zap.Uint64("e.final_round", e.final_round),
					)
					key := pendingKey{
						hash: hash.String(),
					}

					e.pendingMu.Lock()
					e.pending[key] = &pendingMessage{
						height: round,
					}
					e.pendingMu.Unlock()
				}
			}
		}

	}
	return nil
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
		ContractAddress: e.wormholeContract,
	})

	logger := supervisor.Logger(ctx)
	errC := make(chan error)

	logger.Info("Near watcher connecting to RPC node ", zap.String("url", e.nearRPC))

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-e.obsvReqC:
				if vaa.ChainID(r.ChainId) != vaa.ChainIDNear {
					panic("invalid chain ID")
				}

				txHash := base58.Encode(r.TxHash)

				logger.Info("Received obsv request", zap.String("tx_hash", txHash))

				err := e.inspectStatus(logger, txHash, e.wormholeContract)
				if err != nil {
					logger.Error(fmt.Sprintf("near obsvReqC: %s", err.Error()))
				}
			}
		}
	}()

	go func() {
		if e.next_round == 0 {
			finalBody, err := e.getFinalBlock()
			if err != nil {
				logger.Error("StatusAfterBlock", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
				errC <- err
				return
			}
			e.next_round = gjson.ParseBytes(finalBody).Get("result.chunks.0.height_created").Uint()
		}

		timer := time.NewTicker(time.Second * 1)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				finalBody, err := e.getFinalBlock()
				if err != nil {
					logger.Error(fmt.Sprintf("nearClient.Status: %s", err.Error()))

					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
					errC <- err
					return
				}
				parsedFinalBody := gjson.ParseBytes(finalBody)
				lastBlock := parsedFinalBody.Get("result.chunks.0.height_created").Uint()
				e.final_round = lastBlock

				e.pendingMu.Lock()
				for key, bLock := range e.pending {
					if bLock.height <= e.final_round {
						logger.Info("finalBlock",
							zap.Uint64("block.height", bLock.height),
							zap.Uint64("e.final_round", e.final_round),
							zap.String("key.hash", key.hash),
						)

						err := e.inspectStatus(logger, key.hash, e.wormholeContract)
						delete(e.pending, key)

						if err != nil {
							logger.Error("inspectStatus", zap.Error(err))
						}
					}
				}
				e.pendingMu.Unlock()

				logger.Info("lastBlock", zap.Uint64("lastBlock", lastBlock), zap.Uint64("next_round", e.next_round))

				for ; e.next_round <= lastBlock; e.next_round = e.next_round + 1 {
					currentNearHeight.Set(float64(e.next_round))
					p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
						Height:          int64(e.next_round),
						ContractAddress: e.wormholeContract,
					})
					readiness.SetReady(common.ReadinessNearSyncing)

					if e.next_round == lastBlock {
						err := e.inspectBody(logger, e.next_round, parsedFinalBody)
						if err != nil {
							logger.Error(fmt.Sprintf("inspectBody: %s", err.Error()))

							p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
							errC <- err
							return

						}
					} else {
						b, err := e.getBlock(e.next_round)
						if err != nil {
							logger.Error(fmt.Sprintf("nearClient.Status: %s", err.Error()))

							p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
							errC <- err
							return

						}
						err = e.inspectBody(logger, e.next_round, gjson.ParseBytes(b))
						if err != nil {
							logger.Error(fmt.Sprintf("inspectBody: %s", err.Error()))

							p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
							errC <- err
							return

						}
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
