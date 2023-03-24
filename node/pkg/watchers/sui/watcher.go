package sui

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"encoding/base64"
	"encoding/json"

	"github.com/gorilla/websocket"

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
	// Watcher is responsible for looking over Sui blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		suiRPC     string
		suiWS      string
		suiAccount string
		suiPackage string

		unsafeDevMode bool

		msgChan       chan *common.MessagePublication
		obsvReqC      chan *gossipv1.ObservationRequest
		readinessSync readiness.Component

		subId      int64
		subscribed bool
	}

	SuiResult struct {
		Timestamp *int64  `json:"timestamp"`
		TxDigest  *string `json:"txDigest"`
		Event     struct {
			MoveEvent *struct {
				PackageID         *string `json:"packageId"`
				TransactionModule *string `json:"transactionModule"`
				Sender            *string `json:"sender"`
				Type              *string `json:"type"`
				Fields            *struct {
					ConsistencyLevel *uint8  `json:"consistency_level"`
					Nonce            *uint64 `json:"nonce"`
					Payload          *string `json:"payload"`
					Sender           *uint64 `json:"sender"`
					Sequence         *uint64 `json:"sequence"`
				} `json:"fields"`
				Bcs string `json:"bcs"`
			} `json:"moveEvent"`
		} `json:"event"`
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

	SuiTxnQuery struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  struct {
			Data       []SuiResult `json:"data"`
			NextCursor interface{} `json:"nextCursor"`
		} `json:"result"`
		ID int `json:"id"`
	}

	SuiCommitteeInfo struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  struct {
			Epoch         int             `json:"epoch"`
			CommitteeInfo [][]interface{} `json:"committee_info"`
		} `json:"result"`
		ID int `json:"id"`
	}
)

var (
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
	suiAccount string,
	suiPackage string,
	unsafeDevMode bool,
	messageEvents chan *common.MessagePublication,
	obsvReqC chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		suiRPC:        suiRPC,
		suiWS:         suiWS,
		suiAccount:    suiAccount,
		suiPackage:    suiPackage,
		unsafeDevMode: unsafeDevMode,
		msgChan:       messageEvents,
		obsvReqC:      obsvReqC,
		readinessSync: common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDSui),
		subId:         0,
		subscribed:    false,
	}
}

func (e *Watcher) inspectBody(logger *zap.Logger, body SuiResult) error {
	if (body.Timestamp == nil) || (body.TxDigest == nil) {
		return errors.New("Missing event fields")
	}
	if body.Event.MoveEvent == nil {
		return nil
	}
	moveEvent := *body.Event.MoveEvent
	if (moveEvent.PackageID == nil) || (moveEvent.Sender == nil) {
		return errors.New("Missing event fields")
	}

	if moveEvent.Fields == nil {
		return nil
	}
	fields := *moveEvent.Fields
	if (fields.ConsistencyLevel == nil) || (fields.Nonce == nil) || (fields.Payload == nil) || (fields.Sender == nil) || (fields.Sequence == nil) {
		return nil
	}

	if e.suiAccount != *moveEvent.Sender {
		logger.Info("account mismatch", zap.String("e.suiAccount", e.suiAccount), zap.String("account", *moveEvent.Sender))
		return errors.New("account mismatch")
	}

	if !e.unsafeDevMode && e.suiPackage != *moveEvent.PackageID {
		logger.Info("package mismatch", zap.String("e.suiPackage", e.suiPackage), zap.String("package", *moveEvent.PackageID))
		return errors.New("package mismatch")
	}

	emitter := make([]byte, 8)
	binary.BigEndian.PutUint64(emitter, *fields.Sender)

	var a vaa.Address
	copy(a[24:], emitter)

	id, err := base64.StdEncoding.DecodeString(*body.TxDigest)
	if err != nil {
		return err
	}

	var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

	pl, err := base64.StdEncoding.DecodeString(*fields.Payload)
	if err != nil {
		return err

	}

	observation := &common.MessagePublication{
		TxHash: txHash,
		// We do NOT have a useful source of timestamp
		// information.  Every node has its own concept of a
		// timestamp and nothing is persisted into the
		// blockchain to make re-observation possible.  Later
		// we could explore putting the epoch or block height
		// here but even those are currently not available.
		//
		// Timestamp:        time.Unix(int64(timestamp.Uint()/1000), 0),
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(*fields.Nonce), // uint32
		Sequence:         *fields.Sequence,
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   a,
		Payload:          pl,
		ConsistencyLevel: uint8(*fields.ConsistencyLevel),
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
		ContractAddress: e.suiAccount,
	})

	logger := supervisor.Logger(ctx)

	u := url.URL{Scheme: "ws", Host: e.suiWS}

	logger.Info("Sui watcher connecting to WS node ", zap.String("url", u.String()))

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logger.Error(fmt.Sprintf("e.suiWS: %s", err.Error()))
		return err
	}

	var s string

	nBig, _ := rand.Int(rand.Reader, big.NewInt(27))
	e.subId = nBig.Int64()

	if e.unsafeDevMode {
		// There is no way to have a fixed package id on
		// deployment.  This means that in devnet, everytime
		// we publish the contracts we will get a new package
		// id.  The solution is to just subscribe to the whole
		// deployer account instead of to a specific package
		// in that account...
		s = fmt.Sprintf(`{"jsonrpc":"2.0", "id": %d, "method": "sui_subscribeEvent", "params": [{"SenderAddress": "%s"}]}`, e.subId, e.suiAccount)
	} else {
		s = fmt.Sprintf(`{"jsonrpc":"2.0", "id": %d, "method": "sui_subscribeEvent", "params": [{"SenderAddress": "%s", "Package": "%s"}]}`, e.subId, e.suiAccount, e.suiPackage)
	}

	logger.Info("Subscribing using", zap.String("filter", s))

	if err := ws.WriteMessage(websocket.TextMessage, []byte(s)); err != nil {
		logger.Error(fmt.Sprintf("write: %s", err.Error()))
		return err
	}

	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	errC := make(chan error)
	defer close(errC)
	pumpData := make(chan []byte)
	defer close(pumpData)

	go func() {
		for {
			if _, msg, err := ws.ReadMessage(); err != nil {
				logger.Error(fmt.Sprintf("ReadMessage: '%s'", err.Error()))
				if strings.HasSuffix(err.Error(), "EOF") {
					errC <- err
					return
				}
			} else {
				pumpData <- msg
			}
		}
	}()

	for {
		select {
		case err := <-errC:
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			logger.Error("Pump died")
			return err
		case <-ctx.Done():
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return ctx.Err()
		case r := <-e.obsvReqC:
			if vaa.ChainID(r.ChainId) != vaa.ChainIDSui {
				panic("invalid chain ID")
			}

			id := base64.StdEncoding.EncodeToString(r.TxHash)

			logger.Info("obsv request", zap.String("TxHash", string(id)))

			buf := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getEvents", "params": [{"Transaction": "%s"}, null, 10, true]}`, id)

			resp, err := http.Post(e.suiRPC, "application/json", strings.NewReader(buf))
			if err != nil {
				logger.Error("getEvents API failed", zap.String("suiRPC", e.suiRPC), zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error("unexpected truncated body when calling getEvents", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
				continue

			}
			resp.Body.Close()

			logger.Info("receive", zap.String("body", string(body)))

			var res SuiTxnQuery
			err = json.Unmarshal(body, &res)
			if err != nil {
				logger.Error("failed to unmarshal event message", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
				continue

			}

			for _, chunk := range res.Result.Data {
				err := e.inspectBody(logger, chunk)
				if err != nil {
					logger.Error("unspected error while parsing chunk data in event", zap.Error(err))
				}
			}
		case msg := <-pumpData:
			logger.Info("receive", zap.String("body", string(msg)))

			var res SuiEventMsg
			err = json.Unmarshal(msg, &res)
			if err != nil {
				logger.Error("Failed to unmarshal SuiEventMsg", zap.String("body", string(msg)), zap.Error(err))
				continue
			}
			if res.Error != nil {
				_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return errors.New((*res.Error).Message)
			}
			if res.ID != nil {
				if *res.ID == e.subId {
					logger.Debug("Subscribed set to true")
					e.subscribed = true
				}
				continue
			}

			if res.Params != nil && (*res.Params).Result != nil {
				err := e.inspectBody(logger, *(*res.Params).Result)
				if err != nil {
					logger.Error(fmt.Sprintf("inspectBody: %s", err.Error()))
				}
				continue
			}

		case <-timer.C:
			resp, err := http.Post(e.suiRPC, "application/json", strings.NewReader(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getCommitteeInfo", "params": []}`))
			if err != nil {
				logger.Error("sui_getCommitteeInfo failed", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
				break

			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error("sui_getCommitteeInfo failed", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
				break
			}
			resp.Body.Close()

			var res SuiCommitteeInfo
			err = json.Unmarshal(body, &res)
			if err != nil {
				logger.Error("unmarshal failed into SuiCommitteeInfo", zap.String("body", string(body)), zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
				continue

			}

			// Epoch is currently not ticking in 0.16.0.  They also
			// might release another API that gives a
			// proper block height as we traditionally
			// understand it...
			currentSuiHeight.Set(float64(res.Result.Epoch))
			p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSui, &gossipv1.Heartbeat_Network{
				Height:          int64(res.Result.Epoch),
				ContractAddress: e.suiAccount,
			})

			if e.subscribed {
				readiness.SetReady(e.readinessSync)
			}
		}
	}
}
