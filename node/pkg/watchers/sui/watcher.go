package sui

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"encoding/base64"

	"golang.org/x/net/websocket"

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
	// Watcher is responsible for looking over Sui blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		suiRPC     string
		suiWS      string
		suiAccount string
		suiPackage string

		unsafeDevMode bool

		msgChan  chan *common.MessagePublication
		obsvReqC chan *gossipv1.ObservationRequest
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
	}
}

func (e *Watcher) inspectBody(logger *zap.Logger, body gjson.Result) error {
	txDigest := body.Get("txDigest")
	timestamp := body.Get("timestamp")
	packageId := body.Get("event.moveEvent.packageId") // defense in depth: check this
	account := body.Get("event.moveEvent.sender")      // defense in depth: check this
	consistency_level := body.Get("event.moveEvent.fields.consistency_level")
	nonce := body.Get("event.moveEvent.fields.nonce")
	payload := body.Get("event.moveEvent.fields.payload")
	sender := body.Get("event.moveEvent.fields.sender")
	sequence := body.Get("event.moveEvent.fields.sequence")

	if !txDigest.Exists() || !timestamp.Exists() || !packageId.Exists() || !account.Exists() || !consistency_level.Exists() || !nonce.Exists() || !payload.Exists() || !sender.Exists() || !sequence.Exists() {
		return errors.New("block parse error")
	}

	emitter := make([]byte, 8)
	binary.BigEndian.PutUint64(emitter, sender.Uint())

	var a vaa.Address
	copy(a[24:], emitter)

	id, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(txDigest.String())
	if err != nil {
		return err
	}

	var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

	pl, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(payload.String())
	if err != nil {
		return err
	}

	observation := &common.MessagePublication{
		TxHash:           txHash,
		Timestamp:        time.Unix(int64(timestamp.Uint()), 0),
		Nonce:            uint32(nonce.Uint()), // uint32
		Sequence:         sequence.Uint(),
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   a,
		Payload:          pl,
		ConsistencyLevel: uint8(consistency_level.Uint()),
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

	logger.Info("Sui watcher connecting to WS node ", zap.String("url", e.suiWS))

	ws, err := websocket.Dial(e.suiWS, "", "http://guardian")
	if err != nil {
		logger.Error(fmt.Sprintf("e.suiWS: %s", err.Error()))
		return err
	}

	var s string
	if e.unsafeDevMode {
		// There is no way to have a fixed packet id on
		// deployment.  This means that in devnet, everytime
		// we publish the contracts we will get a new package
		// id.  The solution is to just subscribe to the whole
		// deployer account instead of to a specific package
		// in that account...
		s = fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_subscribeEvent", "params": [{"SenderAddress": "%s"}]}`, e.suiAccount)
	} else {
		s = fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_subscribeEvent", "params": [{"SenderAddress": "%s", "Package": "%s"}]}`, e.suiAccount, e.suiPackage)
	}

	logger.Info("Subscribing using", zap.String("filter", s))

	if _, err := ws.Write([]byte(s)); err != nil {
		logger.Error(fmt.Sprintf("write: %s", err.Error()))
		return err
	}

	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-e.obsvReqC:
			if vaa.ChainID(r.ChainId) != vaa.ChainIDSui {
				panic("invalid chain ID")
			}

			logger.Info("Received obsv request")

		case <-timer.C:
			if e.unsafeDevMode {
				logger.Info("tick unsafe")
			} else {
				logger.Info("tick safe")
			}
			for {
				var msg = make([]byte, 10000)
				var n int
				ws.SetReadDeadline(time.Now().Local().Add(100_000_000))
				if n, err = ws.Read(msg); err != nil {
					break
				} else {
					parsedMsg := gjson.ParseBytes(msg)
					if !parsedMsg.Exists() {
						logger.Error("error", zap.String("body", string(msg[:n])), zap.Uint64("len", uint64(n)))
						continue
					}
					logger.Info("receive", zap.String("body", string(msg[:n])), zap.Uint64("len", uint64(n)))
					result := parsedMsg.Get("params.result")
					if !result.Exists() {
						// Other messages come through on the channel.. we can ignore them safely
						continue
					}
					err := e.inspectBody(logger, result)
					if err != nil {
						logger.Error(fmt.Sprintf("inspectBody: %s", err.Error()))
					}
				}
			}
			readiness.SetReady(common.ReadinessSuiSyncing)
		}
	}
}
