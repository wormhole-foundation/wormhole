// Questions:
// - Won't be any block height for the target chains. Is it worth doing block height for this IBC connection?
// - How do reobservations work?
// - The repair script uses public RPC endpoints, so I think that should still work?

package ibc

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	eth_common "github.com/ethereum/go-ethereum/common"

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

	ibcMessage struct {
		ibcChannelID string
		msg          *common.MessagePublication
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

					msgs := EventsToMessagePublications(w.contractAddress, txHash, events.Array(), logger, w.chainID, contractAddressLogKey)
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

				// Received a message from the blockchain
				json := string(message)

				wormchainTxHashRaw := gjson.Get(json, "result.events.tx\\.hash.0")
				if !wormchainTxHashRaw.Exists() {
					w.logger.Warn("message does not have tx hash", zap.String("payload", json))
					continue
				}
				wormchainTxHash := wormchainTxHashRaw.String()

				events := gjson.Get(json, "result.data.value.TxResult.result.events")
				if !events.Exists() {
					w.logger.Warn("message has no events", zap.String("payload", json))
					continue
				}

				msgs := w.eventsToMessagePublications(w.contractAddress, wormchainTxHash, events.Array(), w.logger, contractAddressLogKey)
				for _, msg := range msgs {
					ce, exists := channelMap[msg.ibcChannelID]
					if !exists {
						w.logger.Info("ignoring an event from an unexpected IBC channel", zap.String("ibcChannel", msg.ibcChannelID))
						continue
					}
					if msg.msg.EmitterChain != ce.chainID {
						w.logger.Error("chain id mismatch in IBC message",
							zap.String("ibcChannelID", msg.ibcChannelID),
							zap.String("wormchainTxHash", wormchainTxHash),
							zap.Uint16("expectedChainID", uint16(ce.chainID)),
							zap.Uint16("actualChainID", uint16(msg.msg.EmitterChain)),
							zap.String("txHash", msg.msg.TxHash.String()),
							zap.String("msgId", msg.msg.MessageIDString()),
						)
						invalidChainIdMismatches.WithLabelValues(msg.ibcChannelID).Inc()
						continue
					}

					w.logger.Info("new message detected on IBC",
						zap.String("ibcChannelID", ce.ibcChannelID),
						zap.String("wormchainTxHash", wormchainTxHash),
						zap.String("chainName", ce.chainName),
						zap.String("txHash", msg.msg.TxHash.String()),
						zap.String("msgId", msg.msg.MessageIDString()),
					)

					ce.msgC <- msg.msg
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

func (w *Watcher) eventsToMessagePublications(contract string, wormholeTxHash string, events []gjson.Result, logger *zap.Logger, contractAddressKey string) []*ibcMessage {
	msgs := make([]*ibcMessage, 0, len(events))
	for _, event := range events {
		if !event.IsObject() {
			logger.Warn("event is invalid", zap.String("wormholeTxHash", wormholeTxHash), zap.String("event", event.String()))
			continue
		}
		eventType := gjson.Get(event.String(), "type")
		if eventType.String() != "wasm" {
			continue
		}

		attributes := gjson.Get(event.String(), "attributes")
		if !attributes.Exists() {
			logger.Warn("message event has no attributes", zap.String("wormholeTxHash", wormholeTxHash), zap.String("event", event.String()))
			continue
		}
		mappedAttributes := map[string]string{}
		for _, attribute := range attributes.Array() {
			if !attribute.IsObject() {
				logger.Warn("event attribute is invalid", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attribute", attribute.String()))
				continue
			}
			keyBase := gjson.Get(attribute.String(), "key")
			if !keyBase.Exists() {
				logger.Warn("event attribute does not have key", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attribute", attribute.String()))
				continue
			}
			valueBase := gjson.Get(attribute.String(), "value")
			if !valueBase.Exists() {
				logger.Warn("event attribute does not have value", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attribute", attribute.String()))
				continue
			}

			key, err := base64.StdEncoding.DecodeString(keyBase.String())
			if err != nil {
				logger.Warn("event key attribute is invalid", zap.String("wormholeTxHash", wormholeTxHash), zap.String("key", keyBase.String()))
				continue
			}
			value, err := base64.StdEncoding.DecodeString(valueBase.String())
			if err != nil {
				logger.Warn("event value attribute is invalid", zap.String("wormholeTxHash", wormholeTxHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			if _, ok := mappedAttributes[string(key)]; ok {
				logger.Debug("duplicate key in events", zap.String("wormholeTxHash", wormholeTxHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			mappedAttributes[string(key)] = string(value)
		}

		contractAddress, ok := mappedAttributes[contractAddressKey]
		if !ok {
			logger.Warn("wasm event without contract address field set", zap.String("event", event.String()))
			continue
		}
		// This is not a wormhole message
		if contractAddress != contract {
			continue
		}
		ibcChannelID, ok := mappedAttributes["message.channel_id"]
		if !ok {
			logger.Error("wormhole event does not have a channel_id field", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attributes", attributes.String()))
			continue
		}
		payload, ok := mappedAttributes["message.message"]
		if !ok {
			logger.Error("wormhole event does not have a message field", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attributes", attributes.String()))
			continue
		}
		sender, ok := mappedAttributes["message.sender"]
		if !ok {
			logger.Error("wormhole event does not have a sender field", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attributes", attributes.String()))
			continue
		}
		chainId, ok := mappedAttributes["message.chain_id"]
		if !ok {
			logger.Error("wormhole event does not have a chain_id field", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attributes", attributes.String()))
			continue
		}
		txHash, ok := mappedAttributes["message.tx_hash"]
		if !ok {
			logger.Error("wormhole event does not have a chain_id field", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attributes", attributes.String()))
			continue
		}
		nonce, ok := mappedAttributes["message.nonce"]
		if !ok {
			logger.Error("wormhole event does not have a nonce field", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attributes", attributes.String()))
			continue
		}
		sequence, ok := mappedAttributes["message.sequence"]
		if !ok {
			logger.Error("wormhole event does not have a sequence field", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attributes", attributes.String()))
			continue
		}
		blockTime, ok := mappedAttributes["message.block_time"]
		if !ok {
			logger.Error("wormhole event does not have a block_time field", zap.String("wormholeTxHash", wormholeTxHash), zap.String("attributes", attributes.String()))
			continue
		}

		senderAddress, err := StringToAddress(sender)
		if err != nil {
			logger.Error("cannot decode emitter hex", zap.String("wormholeTxHash", wormholeTxHash), zap.String("value", sender))
			continue
		}
		chainIdInt, err := strconv.ParseUint(chainId, 10, 16)
		if err != nil {
			logger.Error("chainID cannot be parsed as int", zap.String("wormholeTxHash", wormholeTxHash), zap.String("value", blockTime))
			continue
		}
		chainID := vaa.ChainID(chainIdInt)
		txHashValue, err := StringToHash(txHash)
		if err != nil {
			logger.Error("cannot decode tx hash hex", zap.String("wormholeTxHash", wormholeTxHash), zap.String("value", wormholeTxHash))
			continue
		}
		payloadValue, err := hex.DecodeString(payload)
		if err != nil {
			logger.Error("cannot decode payload", zap.String("wormholeTxHash", wormholeTxHash), zap.String("value", payload))
			continue
		}

		blockTimeInt, err := strconv.ParseInt(blockTime, 10, 64)
		if err != nil {
			logger.Error("blocktime cannot be parsed as int", zap.String("wormholeTxHash", wormholeTxHash), zap.String("value", blockTime))
			continue
		}
		nonceInt, err := strconv.ParseUint(nonce, 10, 32)
		if err != nil {
			logger.Error("nonce cannot be parsed as int", zap.String("wormholeTxHash", wormholeTxHash), zap.String("value", blockTime))
			continue
		}
		sequenceInt, err := strconv.ParseUint(sequence, 10, 64)
		if err != nil {
			logger.Error("sequence cannot be parsed as int", zap.String("wormholeTxHash", wormholeTxHash), zap.String("value", blockTime))
			continue
		}
		msgs = append(msgs, &ibcMessage{
			ibcChannelID: ibcChannelID,
			msg: &common.MessagePublication{
				TxHash:           txHashValue,
				Timestamp:        time.Unix(blockTimeInt, 0),
				Nonce:            uint32(nonceInt),
				Sequence:         sequenceInt,
				EmitterChain:     chainID,
				EmitterAddress:   senderAddress,
				Payload:          payloadValue,
				ConsistencyLevel: 0, // Instant finality
			},
		})
	}
	return msgs
}

// StringToAddress convert string into address
func StringToAddress(value string) (vaa.Address, error) {
	var address vaa.Address
	res, err := hex.DecodeString(value)
	if err != nil {
		return address, err
	}
	copy(address[:], res)
	return address, nil
}

// StringToHash convert string into transaction hash
func StringToHash(value string) (eth_common.Hash, error) {
	var hash eth_common.Hash
	res, err := hex.DecodeString(value)
	if err != nil {
		return hash, err
	}
	copy(hash[:], res)
	return hash, nil
}
