package near

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Near blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		nearRPC          string
		wormholeContract string

		msgChan  chan *common.MessagePublication
		obsvReqC chan *gossipv1.ObservationRequest

		next_round uint64
		debug      bool
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
		next_round:       0,
		debug:            true,
	}
}

func getBlock(e *Watcher, block uint64) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": %d}}`, block)
	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func getFinalBlock(e *Watcher) ([]byte, error) {
	s := `{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"finality": "final"}}`
	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func getChunk(e *Watcher, chunk string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "chunk", "params": {"chunk_id": "%s"}}`, chunk)

	resp, err := http.Post(e.nearRPC, "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func inspectBody(e *Watcher, logger *zap.Logger, block uint64, body gjson.Result) error {
	logger.Info("inspectBody", zap.Uint64("block", block))

	result := body.Get("result.chunks.#.chunk_hash")
	for _, name := range result.Array() {
		chunk, err := getChunk(e, name.String())
		if err != nil {
			return err
		}
		receipts := gjson.ParseBytes(chunk).Get("result.receipts")
		if !receipts.Exists() {
			logger.Info("NoReceipts")
			return nil
		}
		for _, r := range receipts.Array() {
			v := r.Get("predecessor_id")
			if !v.Exists() {
				logger.Info("redecessor_id")
				return nil

			}
			p := v.String()
			if strings.HasSuffix(p, e.wormholeContract) {
				a := r.Get("receipt.Action.actions.#.FunctionCall")
				if !a.Exists() {
					logger.Info("receipt.Action.actions.#.FunctionCall")
					return nil
				}
				for _, c := range a.Array() {
					v = c.Get("method_name")
					if !v.Exists() {
						logger.Info("method_name")
						return nil
					}
					if v.String() == "message_published" {
						v = c.Get("args")
						if !v.Exists() {
							logger.Info("args")
							return nil
						}
						args := v.String()
						rawDecodedText, err := base64.StdEncoding.DecodeString(args)
						if err != nil {
							return err
						}

						logs := gjson.ParseBytes(rawDecodedText)

						v = logs.Get("emitter")
						if !v.Exists() {
							logger.Info("emitter")
							return nil
						}

						em := v.String()

						logger.Info("text", zap.String("text", string(rawDecodedText)), zap.String("emitter", em))

						emitter, err := hex.DecodeString(em)
						if err != nil {
							return err
						}

						if len(emitter) != 32 {
							logger.Error("wtf")
						}

						var a vaa.Address
						copy(a[:], emitter)

						// Still never found the txid in the block...    damn it...  we shall use receipt_id until we know better?  We could
						// use the chunk_id...  hmmmmm
						v = r.Get("receipt_id")
						if !v.Exists() {
							logger.Info("receipt_id")
							return nil
						}

						id, err := base58.Decode(v.String())
						if err != nil {
							return err
						}

						if e.debug {
							logger.Error("emitter: " + hex.EncodeToString(a[:]) + " id: " + hex.EncodeToString(id))
						}

						var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

						v = logs.Get("data")
						if !v.Exists() {
							logger.Info("data")
							return nil
						}
						pl, err := hex.DecodeString(v.String())
						if err != nil {
							return err
						}

						v = body.Get("result.header.timestamp")
						if !v.Exists() {
							logger.Info("result.header.timestamp")
							return nil
						}
						ts := uint64(v.Uint()) / 1000000000
						observation := &common.MessagePublication{
							TxHash:           txHash,
							Timestamp:        time.Unix(int64(ts), 0),
							Nonce:            uint32(logs.Get("nonce").Uint()), // uint32
							Sequence:         logs.Get("seq").Uint(),
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
		}
	}
	return nil
}

func (e *Watcher) Run(ctx context.Context) error {
	// an odd thing to broadcast...
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
		ContractAddress: e.wormholeContract,
	})

	logger := supervisor.Logger(ctx)
	errC := make(chan error)

	logger.Info("Near watcher connecting to RPC node ", zap.String("url", e.nearRPC))

	go func() {
		timer := time.NewTicker(time.Second * 1)
		defer timer.Stop()

		if e.next_round == 0 {
			finalBody, err := getFinalBlock(e)
			if err != nil {
				logger.Error("StatusAfterBlock", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
				errC <- err
				return
			}
			e.next_round = gjson.ParseBytes(finalBody).Get("result.chunks.0.height_created").Uint()
		}

		for {
			select {
			case <-ctx.Done():
				return
			case r := <-e.obsvReqC:
				if vaa.ChainID(r.ChainId) != vaa.ChainIDNear {
					panic("invalid chain ID")
				}
				panic("Unimplemented")

				/*
					logger.Info("Received obsv request",
						zap.String("tx_hash", hex.EncodeToString(r.TxHash)),
						zap.String("base32_tx_hash", base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(r.TxHash)))

					result, err := indexerClient.SearchForTransactions().TXID(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(r.TxHash)).Do(context.Background())
					if err != nil {
						logger.Error("SearchForTransactions", zap.Error(err))
						p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
						errC <- err
						return
					}
					for i := 0; i < len(result.Transactions); i++ {
						var t = result.Transactions[i]
						r := t.ConfirmedRound

						block, err := nearClient.Block(r).Do(context.Background())
						if err != nil {
							logger.Error("SearchForTransactions", zap.Error(err))
							p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
							errC <- err
							return
						}

						for _, element := range block.Payset {
							lookAtTxn(e, element, block, logger)
						}
					}
				*/

			case <-timer.C:
				finalBody, err := getFinalBlock(e)
				if err != nil {
					logger.Error(fmt.Sprintf("nearClient.Status: %s", err.Error()))

					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
					errC <- err
					return
				} else {
					parsedFinalBody := gjson.ParseBytes(finalBody)
					lastBlock := parsedFinalBody.Get("result.chunks.0.height_created").Uint()

					logger.Info("lastBlock", zap.Uint64("lastBlock", lastBlock), zap.Uint64("next_round", e.next_round))

					if lastBlock < e.next_round {
						logger.Info("Went backwards... ")
						e.next_round = lastBlock
					}

					for ; e.next_round <= lastBlock; e.next_round = e.next_round + 1 {
						if e.next_round == lastBlock {
							err := inspectBody(e, logger, e.next_round, parsedFinalBody)
							if err != nil {
								logger.Error(fmt.Sprintf("inspectBody: %s", err.Error()))

								p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
								errC <- err
								return

							}
						} else {
							b, err := getBlock(e, e.next_round)
							if err != nil {
								logger.Error(fmt.Sprintf("nearClient.Status: %s", err.Error()))

								p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
								errC <- err
								return

							} else {
								err := inspectBody(e, logger, e.next_round, gjson.ParseBytes(b))
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

				currentNearHeight.Set(float64(e.next_round - 1))
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
					Height:          int64(e.next_round - 1),
					ContractAddress: e.wormholeContract,
				})
				readiness.SetReady(common.ReadinessNearSyncing)
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
