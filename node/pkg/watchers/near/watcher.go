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
	Job struct {
		hash_id string
	}

	// Watcher is responsible for looking over Near blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		nearRPC          string
		wormholeContract string

		msgChan  chan *common.MessagePublication
		obsvReqC chan *gossipv1.ObservationRequest

		jobsChan chan Job

		prev_final_round uint64
		final_round      uint64

		pending   map[string]*pendingMessage
		pendingMu sync.Mutex

		final   map[string]*finalTime
		finalMu sync.Mutex

		workerCount int
	}

	pendingMessage struct {
		height uint64
	}
	finalTime struct {
		time uint64
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
) *Watcher {
	return &Watcher{
		nearRPC:          nearRPC,
		wormholeContract: wormholeContract,
		msgChan:          lockEvents,
		obsvReqC:         obsvReqC,
		jobsChan:         make(chan Job, 100),
		prev_final_round: 0,
		final_round:      0,
		pending:          map[string]*pendingMessage{},
		final:            map[string]*finalTime{},
		workerCount:      10,
	}
}

func (e *Watcher) getBlock(logger *zap.Logger, block_id uint64) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": %d}}`, block_id)
	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		logger.Error(fmt.Sprintf("%s: %s", s, err.Error()))
		// TODO: We should look at the specifics of the error before we try twice
		resp, err = http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (e *Watcher) getBlockHash(logger *zap.Logger, block_id string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": "%s"}}`, block_id)
	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		// TODO: We should look at the specifics of the error
		// before we try twice... what errors are we seeing?
		logger.Error(fmt.Sprintf("%s: %s", s, err.Error()))
		resp, err = http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// getFinalBlock gets a finalized block from the NEAR RPC API using the parameter "finality": "final" (https://docs.near.org/api/rpc/block-chunk)
func (e *Watcher) getFinalBlock(logger *zap.Logger) ([]byte, error) {
	s := `{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"finality": "final"}}`
	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		// TODO: We should look at the specifics of the error
		// before we try twice... what errors are we seeing?
		logger.Error(fmt.Sprintf("%s: %s", s, err.Error()))
		resp, err = http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (e *Watcher) getChunk(logger *zap.Logger, chunk_id string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "chunk", "params": {"chunk_id": "%s"}}`, chunk_id)

	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		// TODO: We should look at the specifics of the error
		// before we try twice... what errors are we seeing?
		logger.Error(fmt.Sprintf("%s: %s", s, err.Error()))
		resp, err = http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// getTxStatusWithReceipts queries status of a transaction by hash, returning the final transaction result and details of all receipts.
// sender_account_id is used to determine which shard to query for the transaction
// See https://docs.near.org/api/rpc/transactions#transaction-status-with-receipts
func (e *Watcher) getTxStatusWithReceipts(logger *zap.Logger, tx_hash string, sender_account_id string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "EXPERIMENTAL_tx_status", "params": ["%s", "%s"]}`, tx_hash, sender_account_id)

	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		// TODO: We should look at the specifics of the error
		// before we try twice... what errors are we seeing?
		logger.Error(fmt.Sprintf("%s: %s", s, err.Error()))

		resp, err = http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// processTxReceipts processes the result of getTxStatusWithReceipts
// we go through all receipt outcomes (result.receipts_outcome) and look for log emissions from the Wormhole core contract.
func (e *Watcher) processTxReceipts(logger *zap.Logger, t []byte, hash string) (error, bool, uint64) {
	logger.Info("txReceipts", zap.String("hash", hash))

	outcomes := gjson.ParseBytes(t).Get("result.receipts_outcome")

	if !outcomes.Exists() {
		// no outcomes means nothing to look at
		return nil, false, 0
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

		// SECURITY CRITICAL: Check that the outcome relates to the Wormhole core contract on NEAR.
		// according to near source documentation, executor_id is the id of the account on which the execution happens:
		// for transaction this is signer_id
		// for receipt this is receiver_id, i.e. the account on which the receipt has been applied
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

				// SECURITY CRITICAL
				// tbjump: if someone would be able to make a log emission from the wormhole contract
				// with the prefix "EVENT_JSON:", they could forge messages.
				// Unfortunately, NEAR does not yet support structured event emission like Ethereum.
				if !strings.HasPrefix(event, "EVENT_JSON:") {
					continue
				}
				logger.Info("event", zap.String("event", event[11:]))

				event_json := gjson.ParseBytes([]byte(event[11:]))

				standard := event_json.Get("standard")
				if !standard.Exists() || standard.String() != "wormhole" {
					// TODO trigger security alert
					continue
				}
				event_type := event_json.Get("event")
				if !event_type.Exists() || event_type.String() != "publish" {
					// TODO trigger security alert
					continue
				}

				// tbjump: TODO defense in depth: We could additionally call receipt-by-id and get the predecessor that way
				// see https://docs.near.org/api/rpc/transactions#receipt-by-id
				em := event_json.Get("emitter")
				if !em.Exists() {
					continue
				}

				emitter, err := hex.DecodeString(em.String())
				if err != nil {
					return err, false, 0
				}

				var a vaa.Address
				copy(a[:], emitter)

				id, err := base58.Decode(hash)
				if err != nil {
					return err, false, 0
				}

				var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

				v := event_json.Get("data")
				if !v.Exists() {
					logger.Info("data")
					return nil, false, 0
				}

				pl, err := hex.DecodeString(v.String())
				if err != nil {
					return err, false, 0
				}

				block_hash_str := block_hash.String()

				e.finalMu.Lock()
				ts_ptr, ok := e.final[block_hash_str]
				e.finalMu.Unlock()

				if !ok {
					block := event_json.Get("block")
					if !block.Exists() {
						continue
					}

					b := block.Uint()

					// if a block is more recent
					// then 120 blocks and being
					// referenced from a txn in a
					// final block but we don't know
					// it as final, lets wait...
					//
					// We could also walk forward
					// one block and see if this
					// block is final...
					if b > (e.final_round - 120) {
						return nil, true, b
					}

					txBlock, err := e.getBlockHash(logger, block_hash_str)
					if err != nil {
						return err, false, 0
					}
					body := gjson.ParseBytes(txBlock)
					if !body.Exists() {
						return errors.New("block parse error"), false, 0
					}
					ts_nanosec := body.Get("result.header.timestamp")
					if !ts_nanosec.Exists() {
						return errors.New("block parse error, missing timestamp"), false, 0
					}

					e.finalMu.Lock()
					ts_ptr = &finalTime{time: ts_nanosec.Uint()}
					e.final[block_hash_str] = ts_ptr
					e.finalMu.Unlock()
				}
				ts := uint64(ts_ptr.time) / 1000000000

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

	return nil, false, 0
}

func (e *Watcher) inspectTransaction(logger *zap.Logger, hash string, sender_account_id string) (error, bool) {
	t, err := e.getTxStatusWithReceipts(logger, hash, sender_account_id)

	if err != nil {
		return err, false
	}

	err, gated, _ := e.processTxReceipts(logger, t, hash)
	return err, gated
}

func (e *Watcher) processChunk(logger *zap.Logger, chunk_hash string) error {
	chunk_bytes, err := e.getChunk(logger, chunk_hash)
	if err != nil {
		return err
	}

	txns := gjson.ParseBytes(chunk_bytes).Get("result.transactions")
	if !txns.Exists() {
		return nil
	}
	for _, r := range txns.Array() {
		hash := r.Get("hash")
		receiver_id := r.Get("receiver_id")
		if !hash.Exists() || !receiver_id.Exists() {
			continue
		}

		t, err := e.getTxStatusWithReceipts(logger, hash.String(), receiver_id.String())

		if err != nil {
			logger.Error("getTxStatusWithReceipts", zap.Error(err))
			continue
		}

		err, gated, round := e.processTxReceipts(logger, t, hash.String())

		if err != nil {
			logger.Error("processTxReceipts", zap.Error(err))
			continue
		}

		if gated {
			logger.Info("pushing pending",
				zap.Uint64("block.height", round),
				zap.Uint64("e.final_round", e.final_round),
			)
			key := hash.String()

			e.pendingMu.Lock()
			e.pending[key] = &pendingMessage{
				height: round,
			}
			e.pendingMu.Unlock()
		}
	}
	return nil
}

func (e *Watcher) processBlock(logger *zap.Logger, block_id uint64) (error, uint64) {
	// TODO: We need to do something smart here... if the block cannot be retrieved, we have no idea how to
	// walk back to the previous block...  and if that block is final.
	//
	// I am returning 0 which effectively terminates the process of walking backwards.  Hopefully another
	// guardian will pick up any blocks we missed?

	b, err := e.getBlock(logger, block_id)
	if err != nil {
		return err, 0
	}
	body := gjson.ParseBytes(b)
	if !body.Exists() {
		return errors.New("block parse error"), 0
	}

	res := body.Get("result.header.prev_height")
	if !res.Exists() {
		return errors.New("no prev_height"), 0
	}
	prev_height := res.Uint()

	hash := body.Get("result.header.hash")
	if !hash.Exists() {
		return errors.New("no hash"), 0
	}

	ts_nanosec := body.Get("result.header.timestamp")
	if !ts_nanosec.Exists() {
		return errors.New("block parse error, missing timestamp"), 0
	}

	// mark this block as final and save away the time
	e.finalMu.Lock()
	e.final[hash.String()] = &finalTime{time: ts_nanosec.Uint()}
	e.finalMu.Unlock()

	// get the hashes of all chunks in the block
	chunk_hashes := body.Get("result.chunks.#.chunk_hash")
	if !chunk_hashes.Exists() {
		// if there are no hashes, there's nothing to do. Return early.
		return nil, prev_height
	}

	for _, chunk_hash := range chunk_hashes.Array() {
		e.jobsChan <- Job{hash_id: chunk_hash.String()}
	}

	return nil, prev_height
}

func worker(ctx context.Context, logger *zap.Logger, e *Watcher, worker int) {
	for {
		select {
		case <-ctx.Done():
			return

		case job := <-e.jobsChan:
			err := e.processChunk(logger, job.hash_id)
			if err != nil {
				logger.Error(fmt.Sprintf("near.processChunk: %s", err.Error()))

				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
			}

		}
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
		ContractAddress: e.wormholeContract,
	})

	logger := supervisor.Logger(ctx)
	defer logger.Sync()

	logger.Info("Near watcher connecting to RPC node ", zap.String("url", e.nearRPC))

	// poll every second. At each poll we process all blocks between the last poll and the current poll
	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	for i := 0; i < e.workerCount; i++ {
		go worker(ctx, logger, e, i)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case r := <-e.obsvReqC:
			if vaa.ChainID(r.ChainId) != vaa.ChainIDNear {
				panic("invalid chain ID")
			}

			txHash := base58.Encode(r.TxHash)

			logger.Info("Received obsv request", zap.String("tx_hash", txHash))

			err, gated := e.inspectTransaction(logger, txHash, e.wormholeContract)
			if gated {
				logger.Error(fmt.Sprintf("Ignoring obsv request for a non-final transaction"))
			}
			if err != nil {
				logger.Error(fmt.Sprintf("near obsvReqC: %s", err.Error()))
			}

		case <-timer.C:
			// poll for new blocks
			finalBody, err := e.getFinalBlock(logger)
			if err != nil {
				logger.Error(fmt.Sprintf("getFinalBlock: %s", err.Error()))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
				continue
			}

			parsedFinalBody := gjson.ParseBytes(finalBody)
			block_res := parsedFinalBody.Get("result.header.height")
			if !block_res.Exists() {
				logger.Error("result.header.height not found")
				continue
			}

			block := block_res.Uint()

			if e.prev_final_round == 0 {
				e.prev_final_round = block
			}

			logger.Info("near", zap.Uint64("prev_final_round", e.prev_final_round), zap.Uint64("final_block", block))
			if block <= e.prev_final_round {
				continue
			}
			e.final_round = block

			// go through all pending messages which may now be finalized
			e.pendingMu.Lock()
			for key, bLock := range e.pending {
				if bLock.height <= e.final_round {
					logger.Info("finalBlock",
						zap.Uint64("block.height", bLock.height),
						zap.Uint64("e.final_round", e.final_round),
						zap.String("key.hash", key),
					)

					err, gated := e.inspectTransaction(logger, key, e.wormholeContract)
					if gated {
						logger.Error("A txn we previously thought should be final is STILL not final.. we will try again later")
						continue
					}

					delete(e.pending, key)

					if err != nil {
						logger.Error("inspectTransaction", zap.Error(err))
					}
				}
			}
			e.pendingMu.Unlock()

			currentNearHeight.Set(float64(block))
			p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
				Height:          int64(block),
				ContractAddress: e.wormholeContract,
			})
			readiness.SetReady(common.ReadinessNearSyncing)

			for block > e.prev_final_round {
				err, block = e.processBlock(logger, block)
				if err != nil {
					logger.Error("processBlock", zap.Error(err))
				}
			}
			e.prev_final_round = e.final_round
		}
	}
}
