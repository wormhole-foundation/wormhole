package sui

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"nhooyr.io/websocket"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/mr-tron/base58"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Sui blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		suiRPC           string
		suiWS            string
		suiMoveEventType string

		unsafeDevMode bool

		msgChan       chan *common.MessagePublication
		obsvReqC      chan *gossipv1.ObservationRequest
		readinessSync readiness.Component

		subId int64
	}

	FieldsData struct {
		ConsistencyLevel *uint8  `json:"consistency_level"`
		Nonce            *uint64 `json:"nonce"`
		Payload          []byte  `json:"payload"`
		Sender           *string `json:"sender"`
		Sequence         *string `json:"sequence"`
		Timestamp        *string `json:"timestamp"`
	}

	SuiResult struct {
		ID struct {
			TxDigest *string `json:"txDigest"`
			EventSeq *string `json:"eventSeq"`
		} `json:"id"`
		PackageID         *string     `json:"packageId"`
		TransactionModule *string     `json:"transactionModule"`
		Sender            *string     `json:"sender"`
		Type              *string     `json:"type"`
		Bcs               *string     `json:"bcs"`
		Timestamp         *string     `json:"timestampMs"`
		Fields            *FieldsData `json:"parsedJson"`
	}

	SuiEventError struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}

	SuiEventMsg struct {
		Jsonrpc string         `json:"jsonrpc"`
		Method  *string        `json:"method"`
		ID      *int64         `json:"id"`
		Error   *SuiEventError `json:"error"`
		Params  *struct {
			Subscription int64      `json:"subscription"`
			Result       *SuiResult `json:"result"`
		} `json:"params"`
	}

	SuiTxnQueryError struct {
		Jsonrpc string `json:"jsonrpc"`
		Error   struct {
			Code    int     `json:"code"`
			Message *string `json:"message"`
		} `json:"error"`
		ID int `json:"id"`
	}

	SuiTxnQuery struct {
		Jsonrpc string      `json:"jsonrpc"`
		Result  []SuiResult `json:"result"`
		ID      int         `json:"id"`
	}
	// 	{
	//   "jsonrpc": "2.0",
	//   "result": [
	//     {
	//       "id": {
	//         "txDigest": "6Yff8smmPZMandj6Psjy6wgZv5Deii78o1Sbghh5sHPA",
	//         "eventSeq": "0"
	//       },
	//       "packageId": "0x8b04a73ab3cb1e36bee5a86fdbfa481e97d3cc7ce8b594edea1400103ff0134d",
	//       "transactionModule": "sender",
	//       "sender": "0xed867315e3f7c83ae82e6d5858b6a6cc57c291fd84f7509646ebc8162169cf96",
	//       "type": "0x7483d0db53a140eed72bd6cb133daa59c539844f4c053924b9e3f0d2d7ba146d::publish_message::WormholeMessage",
	//       "parsedJson": {
	//         "consistency_level": 0,
	//         "nonce": 0,
	//         "payload": [104, 101, 108, 108, 111],
	//         "sender": "0x71c2aa2c549bb7381e88fbeca7eeb791be0afd455c8af9184613ce5db5ddba47",
	//         "sequence": "0",
	//         "timestamp": "1681411389"
	//       },
	//       "bcs": "5ZuknLT3Xsicr2D8zyk828thPByMBfR1cPJyEHF67k16AcEotDWhrpCDCTbk6BBbpSSs3bUk3msfADzrs"
	//     }
	//   ],
	//   "id": 1
	// }

	SuiCheckpointSN struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  string `json:"result"`
		ID      int    `json:"id"`
	}
)

var (
	suiConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_sui_connection_errors_total",
			Help: "Total number of SUI connection errors",
		}, []string{"reason"})
	suiMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_sui_observations_confirmed_total",
			Help: "Total number of verified Sui observations found",
		})
	currentSuiHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_sui_current_height",
			Help: "Current Sui block height",
		})
)

// NewWatcher creates a new Sui appid watcher
func NewWatcher(
	suiRPC string,
	suiWS string,
	suiMoveEventType string,
	unsafeDevMode bool,
	messageEvents chan *common.MessagePublication,
	obsvReqC chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		suiRPC:           suiRPC,
		suiWS:            suiWS,
		suiMoveEventType: suiMoveEventType,
		unsafeDevMode:    unsafeDevMode,
		msgChan:          messageEvents,
		obsvReqC:         obsvReqC,
		readinessSync:    common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDSui),
		subId:            0,
	}
}

func (e *Watcher) inspectBody(logger *zap.Logger, body SuiResult) error {
	if body.ID.TxDigest == nil {
		return errors.New("Missing TxDigest field")
	}
	if body.Type == nil {
		return errors.New("Missing Type field")
	}

	// There may be moveEvents caught without these params.
	// So, not necessarily an error.
	if body.Fields == nil {
		return nil
	}

	if e.suiMoveEventType != *body.Type {
		logger.Info("type mismatch", zap.String("e.suiMoveEventType", e.suiMoveEventType), zap.String("type", *body.Type))
		return errors.New("type mismatch")
	}

	fields := *body.Fields
	if (fields.ConsistencyLevel == nil) || (fields.Nonce == nil) || (fields.Payload == nil) || (fields.Sender == nil) || (fields.Sequence == nil) {
		logger.Info("Missing required fields in event.")
		return nil
	}

	emitter, err := vaa.StringToAddress(*fields.Sender)
	if err != nil {
		return err
	}

	txHashBytes, err := base58.Decode(*body.ID.TxDigest)
	if err != nil {
		return err
	}

	if len(txHashBytes) != 32 {
		logger.Error(
			"Transaction hash is not 32 bytes",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.String("txHash", *body.ID.TxDigest),
		)
		return errors.New("Transaction hash is not 32 bytes")
	}

	txHashEthFormat := eth_common.BytesToHash(txHashBytes)

	seq, err := strconv.ParseUint(*fields.Sequence, 10, 64)
	if err != nil {
		logger.Info("Sequence decode error", zap.String("Sequence", *fields.Sequence))
		return err
	}
	ts, err := strconv.ParseInt(*fields.Timestamp, 10, 64)
	if err != nil {
		logger.Info("Timestamp decode error", zap.String("Timestamp", *fields.Timestamp))
		return err
	}

	observation := &common.MessagePublication{
		TxHash:           txHashEthFormat,
		Timestamp:        time.Unix(ts, 0),
		Nonce:            uint32(*fields.Nonce),
		Sequence:         seq,
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   emitter,
		Payload:          fields.Payload,
		ConsistencyLevel: *fields.ConsistencyLevel,
	}

	suiMessagesConfirmed.Inc()

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

	e.msgChan <- observation

	return nil
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSui, &gossipv1.Heartbeat_Network{
		ContractAddress: e.suiMoveEventType,
	})

	logger := supervisor.Logger(ctx)

	u := url.URL{Scheme: "ws", Host: e.suiWS}

	logger.Info("Sui watcher connecting to WS node ", zap.String("url", u.String()))
	logger.Debug("SUI watcher:", zap.String("suiRPC", e.suiRPC), zap.String("suiWS", e.suiWS), zap.String("suiMoveEventType", e.suiMoveEventType))

	ws, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		suiConnectionErrors.WithLabelValues("websocket_dial_error").Inc()
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")

	nBig, _ := rand.Int(rand.Reader, big.NewInt(27))
	e.subId = nBig.Int64()

	subscription := fmt.Sprintf(`{"jsonrpc":"2.0", "id": %d, "method": "suix_subscribeEvent", "params": [{"MoveEventType": "%s"}]}`, e.subId, e.suiMoveEventType)

	logger.Debug("Subscribing using", zap.String("json:", subscription))

	err = ws.Write(ctx, websocket.MessageText, []byte(subscription))
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		suiConnectionErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("websocket subscription failed: %w", err)
	}
	// Wait for the success response
	mType, p, err := ws.Read(ctx)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		suiConnectionErrors.WithLabelValues("event_subscription_error").Inc()
		return fmt.Errorf("event subscription failed: %w", err)
	}
	var subRes map[string]any
	err = json.Unmarshal(p, &subRes)
	if err != nil {
		return fmt.Errorf("failed to Unmarshal the subscription result: %w", err)
	}
	logger.Debug("Unmarshalled json", zap.Any("subRes", subRes))
	actualResult := subRes["result"]
	logger.Debug("actualResult", zap.Any("res", actualResult))
	if actualResult == nil {
		return fmt.Errorf("Failed to request filter in subscription request")
	}
	logger.Debug("subscribed to new transaction events", zap.Int("messageType", int(mType)), zap.String("bytes", string(p)))

	timer := time.NewTicker(time.Second * 5)
	defer timer.Stop()

	errC := make(chan error)
	pumpData := make(chan []byte)
	defer close(pumpData)

	supervisor.Signal(ctx, supervisor.SignalHealthy)
	readiness.SetReady(e.readinessSync)

	common.RunWithScissors(ctx, errC, "sui_data_pump", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_data_pump context done")
				return ctx.Err()

			default:
				_, msg, err := ws.Read(ctx)
				if err != nil {
					logger.Error(fmt.Sprintf("ReadMessage: '%s'", err.Error()))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					suiConnectionErrors.WithLabelValues("channel_read_error").Inc()
					return err
				}

				var res SuiEventMsg
				err = json.Unmarshal(msg, &res)
				if err != nil {
					logger.Error("Failed to unmarshal SuiEventMsg", zap.String("body", string(msg)), zap.Error(err))
					return fmt.Errorf("Failed to unmarshal SuiEventMsg, body: %s, error: %w", string(msg), err)
				}
				if res.Error != nil {
					return fmt.Errorf("Bad SuiEventMsg, body: %s, error: %w", string(msg), err)
				}
				logger.Debug("SUI result message", zap.String("message", string(msg)), zap.Any("event", res))
				if res.ID != nil {
					logger.Error("Found an unexpected res.ID")
					continue
				}

				if res.Params != nil && (*res.Params).Result != nil {
					err := e.inspectBody(logger, *(*res.Params).Result)
					if err != nil {
						logger.Error(fmt.Sprintf("inspectBody: %s", err.Error()))
					}
					continue
				}
			}
		}
	})

	common.RunWithScissors(ctx, errC, "sui_block_height", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_block_height context done")
				return ctx.Err()

			case <-timer.C:
				resp, err := http.Post(e.suiRPC, "application/json", strings.NewReader(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getLatestCheckpointSequenceNumber", "params": []}`)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
				if err != nil {
					logger.Error("sui_getLatestCheckpointSequenceNumber failed", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to post: %w", err)
				}
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error("sui_getLatestCheckpointSequenceNumber failed", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to read: %w", err)
				}
				resp.Body.Close()
				logger.Debug("Body before unmarshalling", zap.String("body", string(body)))

				var res SuiCheckpointSN
				err = json.Unmarshal(body, &res)
				if err != nil {
					logger.Error("unmarshal failed into uint64", zap.String("body", string(body)), zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to unmarshal body: %s, error: %w", string(body), err)
				}

				height, pErr := strconv.ParseInt(res.Result, 0, 64)
				if pErr != nil {
					logger.Error("Failed to ParseInt")
				} else {
					currentSuiHeight.Set(float64(height))
					logger.Debug("sui_getLatestCheckpointSequenceNumber", zap.String("result", res.Result))

					p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSui, &gossipv1.Heartbeat_Network{
						Height:          height,
						ContractAddress: e.suiMoveEventType,
					})
				}

				readiness.SetReady(e.readinessSync)
			}
		}
	})

	common.RunWithScissors(ctx, errC, "sui_fetch_obvs_req", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_fetch_obvs_req context done")
				return ctx.Err()
			case r := <-e.obsvReqC:
				if vaa.ChainID(r.ChainId) != vaa.ChainIDSui {
					panic("invalid chain ID")
				}

				tx58 := base58.Encode(r.TxHash)

				buf := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getEvents", "params": ["%s"]}`, tx58)

				resp, err := http.Post(e.suiRPC, "application/json", strings.NewReader(buf)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
				if err != nil {
					logger.Error("getEvents API failed", zap.String("suiRPC", e.suiRPC), zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					continue
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error("unexpected truncated body when calling getEvents", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui__fetch_obvs_req failed to post: %w", err)

				}
				resp.Body.Close()

				logger.Debug("receive", zap.String("body", string(body)))

				// Do we have an error?
				var err_res SuiTxnQueryError
				err = json.Unmarshal(body, &err_res)
				if err != nil {
					logger.Error("Failed to unmarshal event error message", zap.String("Result", string(body)))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return err
				}

				if err_res.Error.Message != nil {
					logger.Error("Failed to get events for re-observation request, detected error", zap.String("Result", string(body)))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					// Don't need to kill the watcher on this error. So, just continue.
					continue
				}
				var res SuiTxnQuery
				err = json.Unmarshal(body, &res)
				if err != nil {
					logger.Error("failed to unmarshal event message", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui__fetch_obvs_req failed to unmarshal: %w", err)

				}

				for i, chunk := range res.Result {
					err := e.inspectBody(logger, chunk)
					if err != nil {
						logger.Info("skipping event data in result", zap.String("txhash", tx58), zap.Int("index", i), zap.Error(err))
					}
				}
			}
		}
	})

	select {
	case <-ctx.Done():
		_ = ws.Close(websocket.StatusNormalClosure, "")
		return ctx.Err()
	case err := <-errC:
		_ = ws.Close(websocket.StatusInternalError, err.Error())
		return err
	}
}
