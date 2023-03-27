// Questions:
// - Won't be any block height for the target chains. Is it worth doing block height for this IBC connection?
// - How do reobservations work?
// - The repair script uses public RPC endpoints, so I think that should still work?

package ibc

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	"github.com/tidwall/gjson"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"go.uber.org/zap"
)

type (
	// ChainToMonitor defines a chain to be monitored over IBC.
	ChainToMonitor struct {
		// ChainID is the wormhole chain ID.
		ChainID vaa.ChainID

		// IBCChannelID is the IBC channel this chain publishes on.
		IBCChannelID string

		// readinessComponent is used to publish readiness for this chain.
		ReadinessComponent readiness.Component

		// MsgC is the channel used to publish messages for this chain.
		MsgC chan<- *common.MessagePublication

		// ObsvReqC is the channel used to listen for observation requests for this chain.
		ObsvReqC <-chan *gossipv1.ObservationRequest
	}

	// ChainToMonitor is the set of chains to be monitored over IBC.
	ChainsToMonitor []ChainToMonitor
)

var (
	connectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ibc_connection_errors_total",
			Help: "Total number of connection errors on IBC connection",
		}, []string{"reason"})
	messagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_ibc_messages_confirmed_total",
			Help: "Total number of verified messages found on an IBC connected chain",
		}, []string{"chain_name"})
	invalidChainIdMismatches = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_ibc_chain_id_mismatches",
			Help: "Total number of cases where the wormhole chain ID does not match the IBC channel ID",
		}, []string{"ibc_channel_id"})
)

type (
	// Watcher is responsible for monitoring the IBC contract on wormchain and publishing wormhole messages for all chains connected via IBC.
	Watcher struct {
		wsUrl           string
		lcdUrl          string
		contractAddress string
		chainsToMonitor ChainsToMonitor
		logger          *zap.Logger
	}

	// ChannelConfigEntry defines the entry for an IBC channel in the node config file.
	ChannelConfigEntry struct {
		ChainID   vaa.ChainID
		ChannelID string
	}

	// channelEntry defines the chain that is associated with an IBC channel.
	channelEntry struct {
		ibcChannelID string
		chainID      vaa.ChainID
		chainName    string
		readiness    readiness.Component
		msgC         chan<- *common.MessagePublication
		obsvReqC     <-chan *gossipv1.ObservationRequest
	}

	clientRequest struct {
		JSONRPC string `json:"jsonrpc"`
		// A String containing the name of the method to be invoked.
		Method string `json:"method"`
		// Object to pass as request parameter to the method.
		Params [1]string `json:"params"`
		// The request id. This can be of any type. It is used to match the
		// response with the request that it is replying to.
		ID uint64 `json:"id"`
	}

	// ibcReceivePublishEvent is the event published by the IBC contract for a wormhole message.
	ibcReceivePublishEvent struct {
		ChannelID      string      `json:"channel_id"`
		EmitterChain   vaa.ChainID `json:"message.chain_id"`
		EmitterAddress vaa.Address `json:"message.sender"`
		Nonce          uint32      `json:"message.nonce"`
		Sequence       uint64      `json:"message.sequence"`
		Timestamp      uint64      `json:"message.block_time"`
		Payload        []byte      `json:"message.message"`
	}
)

// NewWatcher creates a new IBC contract watcher
func NewWatcher(
	wsUrl string,
	lcdUrl string,
	contractAddress string,
	chainsToMonitor ChainsToMonitor,
) *Watcher {
	return &Watcher{
		wsUrl:           wsUrl,
		lcdUrl:          lcdUrl,
		contractAddress: contractAddress,
		chainsToMonitor: chainsToMonitor,
	}
}

const (
	contractAddressFilterKey = "execute._contract_address"
	contractAddressLogKey    = "_contract_address"
)

// Run is the runnable for monitoring the IBC contract on wormchain.
func (w *Watcher) Run(ctx context.Context) error {
	w.logger = supervisor.Logger(ctx)
	errC := make(chan error)

	channelMap := make(map[string]*channelEntry)
	chainMap := make(map[vaa.ChainID]*channelEntry)

	for _, chainToMonitor := range w.chainsToMonitor {
		ce := &channelEntry{
			ibcChannelID: chainToMonitor.IBCChannelID,
			chainID:      chainToMonitor.ChainID,
			chainName:    vaa.ChainID(chainToMonitor.ChainID).String(),
			readiness:    chainToMonitor.ReadinessComponent,
			msgC:         chainToMonitor.MsgC,
			obsvReqC:     chainToMonitor.ObsvReqC,
		}

		_, exists := channelMap[ce.ibcChannelID]
		if exists {
			return fmt.Errorf("detected duplicate ibc channel: %v", ce.ibcChannelID)
		}

		_, exists = chainMap[ce.chainID]
		if exists {
			return fmt.Errorf("detected duplicate chainID: %v", ce.chainID)
		}

		w.logger.Info("Will monitor chain over IBC", zap.String("chain", ce.chainName), zap.String("IBC channel", ce.ibcChannelID))
		channelMap[ce.ibcChannelID] = ce
		chainMap[ce.chainID] = ce

		p2p.DefaultRegistry.SetNetworkStats(ce.chainID, &gossipv1.Heartbeat_Network{ContractAddress: w.contractAddress})
	}

	w.logger.Info("connecting to websocket", zap.String("url", w.wsUrl))

	c, _, err := websocket.Dial(ctx, w.wsUrl, nil)
	if err != nil {
		// p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		connectionErrors.WithLabelValues("websocket_dial_error").Inc()
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	// During testing, I got a message larger then the default
	// 32768.  Increasing this limit effects an internal buffer that is used
	// to as part of the zero alloc/copy design.
	c.SetReadLimit(524288)

	// Subscribe to smart contract transactions
	params := [...]string{fmt.Sprintf("tm.event='Tx' AND %s='%s'", contractAddressFilterKey, w.contractAddress)}
	command := &clientRequest{
		JSONRPC: "2.0",
		Method:  "subscribe",
		Params:  params,
		ID:      1,
	}
	err = wsjson.Write(ctx, c, command)
	if err != nil {
		// p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		connectionErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("websocket subscription failed: %w", err)
	}

	// Wait for the success response
	_, _, err = c.Read(ctx)
	if err != nil {
		// p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		connectionErrors.WithLabelValues("event_subscription_error").Inc()
		return fmt.Errorf("event subscription failed: %w", err)
	}
	w.logger.Info("subscribed to new transaction events")

	for _, ce := range chainMap {
		readiness.SetReady(ce.readiness)
	}

	/*
		common.RunWithScissors(ctx, errC, "ibc_objs_req", func(ctx context.Context) error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case r := <-w.obsvReqC:
					if vaa.ChainID(r.ChainId) != w.chainID {
						panic("invalid chain ID")
					}

					tx := hex.EncodeToString(r.TxHash)

					w.logger.Info("received observation request", zap.String("wormholeTxHash", tx))

					client := &http.Client{
						Timeout: time.Second * 5,
					}

					// Query for tx by hash
					resp, err := client.Get(fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", w.lcdUrl, tx))
					if err != nil {
						w.logger.Error("query tx response error", zap.Error(err))
						continue
					}
					txBody, err := io.ReadAll(resp.Body)
					if err != nil {
						w.logger.Error("query tx response read error", zap.Error(err))
						resp.Body.Close()
						continue
					}
					resp.Body.Close()

					txJSON := string(txBody)

					txHashRaw := gjson.Get(txJSON, "tx_response.txhash")
					if !txHashRaw.Exists() {
						w.logger.Error("tx does not have tx hash", zap.String("payload", txJSON))
						continue
					}
					txHash := txHashRaw.String()

					events := gjson.Get(txJSON, "tx_response.events")
					if !events.Exists() {
						w.logger.Error("tx has no events", zap.String("payload", txJSON))
						continue
					}

					msgs := parseEvents(w.contractAddress, txHash, events.Array(), logger, w.chainID, contractAddressLogKey)
					for _, msg := range msgs {
						w.msgC <- msg
						messagesConfirmed.WithLabelValues(networkName).Inc()
					}
				}
			}
		})
	*/

	common.RunWithScissors(ctx, errC, "ibc_data_pump", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				_, message, err := c.Read(ctx)
				if err != nil {
					// p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
					connectionErrors.WithLabelValues("channel_read_error").Inc()
					w.logger.Error("error reading channel", zap.Error(err))
					errC <- err
					return nil
				}

				json := string(message)

				txHashRaw := gjson.Get(json, "result.events.tx\\.hash.0")
				if !txHashRaw.Exists() {
					w.logger.Warn("message does not have tx hash", zap.String("payload", json))
					continue
				}
				txHashStr := txHashRaw.String()

				txHash, err := vaa.StringToHash(txHashStr)
				if err != nil {
					// p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
					connectionErrors.WithLabelValues("parse_error").Inc()
					w.logger.Error("failed to parse txHash", zap.String("txHash", txHashStr), zap.Error(err))
					errC <- err
					return nil
				}

				eventsJson := gjson.Get(json, "result.data.value.TxResult.result.events")
				if !eventsJson.Exists() {
					w.logger.Warn("message has no events", zap.String("payload", json))
					continue
				}

				events, err := w.parseEvents(txHashStr, eventsJson.Array())
				if err != nil {
					// p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
					connectionErrors.WithLabelValues("channel_parse_error").Inc()
					w.logger.Error("failed to parse events", zap.String("txHash", txHashStr), zap.Error(err))
					errC <- err
					return nil
				}

				for _, evt := range events {
					ce, exists := channelMap[evt.ChannelID]
					if !exists {
						w.logger.Info("ignoring an event from an unexpected IBC channel", zap.String("ibcChannel", evt.ChannelID))
						continue
					}

					if evt.EmitterChain != ce.chainID {
						w.logger.Error("chain id mismatch in IBC message",
							zap.String("ibcChannelID", evt.ChannelID),
							zap.String("txHash", txHashStr),
							zap.Uint16("expectedChainID", uint16(ce.chainID)),
							zap.Uint16("actualChainID", uint16(evt.EmitterChain)),
							zap.String("msgId", evt.msgId()),
						)
						invalidChainIdMismatches.WithLabelValues(evt.ChannelID).Inc()
						continue
					}

					w.logger.Info("new message detected on IBC",
						zap.String("ibcChannelID", ce.ibcChannelID),
						zap.String("chainName", ce.chainName),
						zap.String("msgId", evt.msgId()),
						zap.String("txHash", txHashStr),
						zap.Uint64("timeStamp", evt.Timestamp),
					)

					msg := &common.MessagePublication{
						TxHash:           txHash,
						Timestamp:        time.Unix(int64(evt.Timestamp), 0),
						Nonce:            evt.Nonce,
						Sequence:         evt.Sequence,
						EmitterChain:     evt.EmitterChain,
						EmitterAddress:   evt.EmitterAddress,
						Payload:          evt.Payload,
						ConsistencyLevel: 0,
					}

					ce.msgC <- msg
					messagesConfirmed.WithLabelValues(ce.chainName).Inc()
				}
			}
		}
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// parseEvents parses incoming events and returns any IBC receive_publish events.
func (w *Watcher) parseEvents(txHash string, events []gjson.Result) ([]*ibcReceivePublishEvent, error) {
	msgs := make([]*ibcReceivePublishEvent, 0, len(events))
	for _, event := range events {
		msg, err := parseWasmEvent[ibcReceivePublishEvent](w.logger, w.contractAddress, "receive_publish", event)
		if err != nil {
			return msgs, err
		}

		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// parseWasmEvent parses a wasm event. If it is from the desired contract and for the desired action, it returns an event. Otherwise, it returns nil.
func parseWasmEvent[T any](logger *zap.Logger, desiredContract string, desiredAction string, event gjson.Result) (*T, error) {
	attrBytes, err := parseWasmAttributes(logger, desiredContract, desiredAction, event)
	if err != nil {
		return nil, fmt.Errorf("failed to parse attributes: %w", err)
	}

	if attrBytes == nil {
		return nil, nil
	}

	evt := new(T)
	if err := json.Unmarshal(attrBytes, evt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event attributes: %w", err)
	}

	return evt, nil
}

// parseWasmAttributes parses the attributes in a wasm event. If the contract and action match the desired values (or the desired values are not set)
// the attributes are loaded into a byte array of the marshaled json. If the event is not of the desired type, it returns nil.
func parseWasmAttributes(logger *zap.Logger, desiredContract string, desiredAction string, event gjson.Result) ([]byte, error) {
	if !event.IsObject() {
		return nil, fmt.Errorf("event is invalid: %s", event.String())
	}

	eventType := gjson.Get(event.String(), "type")
	if eventType.String() != "wasm" {
		return nil, nil
	}

	attributes := gjson.Get(event.String(), "attributes")
	if !attributes.Exists() {
		return nil, fmt.Errorf("message event has no attributes: %s", event.String())
	}

	contractAddressSeen := false
	actionSeen := false
	attrs := make(map[string]json.RawMessage)
	for _, attribute := range attributes.Array() {
		if !attribute.IsObject() {
			return nil, fmt.Errorf("event attribute is invalid: %s", attribute.String())
		}

		keyBase := gjson.Get(attribute.String(), "key")
		if !keyBase.Exists() {
			return nil, fmt.Errorf("event attribute does not have key: %s", attribute.String())
		}

		valueBase := gjson.Get(attribute.String(), "value")
		if !valueBase.Exists() {
			return nil, fmt.Errorf("event attribute does not have value: %s", attribute.String())
		}

		keyBytes, err := base64.StdEncoding.DecodeString(keyBase.String())
		if err != nil {
			return nil, fmt.Errorf("event attribute key is not valid base64: %s", attribute.String())
		}
		key := string(keyBytes)

		value, err := base64.StdEncoding.DecodeString(valueBase.String())
		if err != nil {
			return nil, fmt.Errorf("event attribute value is not valid base64: %s", attribute.String())
		}

		if key == "_contract_address" {
			contractAddressSeen = true
			if desiredContract != "" && string(value) != desiredContract {
				logger.Debug("ibc: ignoring event from an unexpected contract", zap.String("contract", string(value)), zap.String("desiredContract", desiredContract))
				return nil, nil
			}
		} else if key == "action" {
			actionSeen = true
			if desiredAction != "" && string(value) != desiredAction {
				logger.Debug("ibc: ignoring event with an unexpected action", zap.String("key", key), zap.String("value", string(value)), zap.String("desiredAction", desiredAction))
				return nil, nil
			}
		} else {
			if _, ok := attrs[key]; ok {
				logger.Debug("ibc: duplicate key in events", zap.String("key", key), zap.String("value", string(value)))
				continue
			}

			logger.Debug("ibc: event attribute", zap.String("key", key), zap.String("value", string(value)), zap.String("desiredAction", desiredAction))
			attrs[string(key)] = value
		}
	}

	if !contractAddressSeen && desiredContract != "" {
		logger.Debug("ibc: contract address not specified, which does not match the desired value")
		return nil, nil
	}

	if !actionSeen && desiredAction != "" {
		logger.Debug("ibc: action not specified, which does not match the desired value")
		return nil, nil
	}

	attrBytes, err := json.Marshal(attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event attributes: %w", err)
	}

	return attrBytes, nil
}

func (e ibcReceivePublishEvent) msgId() string {
	return fmt.Sprintf("%v/%v/%v", e.EmitterChain, hex.EncodeToString(e.EmitterAddress[:]), e.Sequence)
}
