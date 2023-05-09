package wormchain

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over wormchain blockchain and reporting new transactions to the core bridge
	Watcher struct {
		urlWS  string
		urlLCD string

		msgC chan<- *common.MessagePublication

		// Incoming re-observation requests from the network. Pre-filtered to only
		// include requests for our chainID.
		obsvReqC <-chan *gossipv1.ObservationRequest

		readinessSync readiness.Component
	}
)

var (
	wormchainConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_wormchain_connection_errors_total",
			Help: "Total number of Wormchain connection errors",
		}, []string{"reason"})
	wormchainMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_wormchain_messages_confirmed_total",
			Help: "Total number of verified wormchain messages found",
		})
	currentWormchainHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_wormchain_current_height",
			Help: "Current wormchain slot height (at default commitment level, not the level used for observations)",
		})
)

type clientRequest struct {
	JSONRPC string `json:"jsonrpc"`
	// A String containing the name of the method to be invoked.
	Method string `json:"method"`
	// Object to pass as request parameter to the method.
	Params [1]string `json:"params"`
	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	ID uint64 `json:"id"`
}

// NewWatcher creates a new Wormchain event watcher
func NewWatcher(
	urlWS string,
	urlLCD string,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest) *Watcher {
	return &Watcher{
		urlWS:         urlWS,
		urlLCD:        urlLCD,
		msgC:          msgC,
		obsvReqC:      obsvReqC,
		readinessSync: common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDWormchain),
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDWormchain, &gossipv1.Heartbeat_Network{})

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	logger.Info("connecting to websocket", zap.String("url", e.urlWS))

	c, _, err := websocket.DefaultDialer.DialContext(ctx, e.urlWS, nil)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDWormchain, 1)
		wormchainConnectionErrors.WithLabelValues("websocket_dial_error").Inc()
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer c.Close()

	// Subscribe transactions which cause EventPostedMessage
	params := [...]string{"tm.event='Tx' AND wormhole_foundation.wormchain.wormhole.EventPostedMessage.sequence EXISTS"}
	// alternately, "tm.event='Tx' AND wormhole_foundation.wormchain.wormhole.EventPostedMessage.sequence >= 0"
	command := &clientRequest{
		JSONRPC: "2.0",
		Method:  "subscribe",
		Params:  params,
		ID:      1,
	}
	err = c.WriteJSON(command)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDWormchain, 1)
		wormchainConnectionErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("websocket subscription failed: %w", err)
	}

	// Wait for the success response
	_, _, err = c.ReadMessage()
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDWormchain, 1)
		wormchainConnectionErrors.WithLabelValues("event_subscription_error").Inc()
		return fmt.Errorf("event subscription failed: %w", err)
	}
	logger.Info("subscribed to new transaction events")

	readiness.SetReady(e.readinessSync)

	go func() {
		t := time.NewTicker(5 * time.Second)
		client := &http.Client{
			Timeout: time.Second * 5,
		}

		for {
			<-t.C

			// Query and report height and set currentWormchainHeight
			resp, err := client.Get(fmt.Sprintf("%s/blocks/latest", e.urlLCD)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
			if err != nil {
				logger.Error("query latest block response error", zap.Error(err))
				continue
			}
			blocksBody, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error("query latest block response read error", zap.Error(err))
				errC <- err
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			blockJSON := string(blocksBody)
			latestBlock := gjson.Get(blockJSON, "block.header.height")
			logger.Debug("current Wormchain height", zap.Int64("block", latestBlock.Int()))
			currentWormchainHeight.Set(float64(latestBlock.Int()))
			p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDWormchain, &gossipv1.Heartbeat_Network{
				Height: latestBlock.Int(),
			})

			readiness.SetReady(e.readinessSync)
		}
	}()

	//TODO verify that this needs no changes
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-e.obsvReqC:
				if vaa.ChainID(r.ChainId) != vaa.ChainIDWormchain {
					panic("invalid chain ID")
				}

				tx := hex.EncodeToString(r.TxHash)

				logger.Info("received observation request for wormchain",
					zap.String("tx_hash", tx))

				client := &http.Client{
					Timeout: time.Second * 5,
				}

				// Query for tx by hash
				resp, err := client.Get(fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", e.urlLCD, tx)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
				if err != nil {
					logger.Error("query tx response error", zap.Error(err))
					continue
				}
				txBody, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error("query tx response read error", zap.Error(err))
					resp.Body.Close()
					continue
				}
				resp.Body.Close()

				txJSON := string(txBody)

				txHashRaw := gjson.Get(txJSON, "tx_response.txhash")
				if !txHashRaw.Exists() {
					logger.Error("wormchain tx does not have tx hash", zap.String("payload", txJSON))
					continue
				}
				txHash := txHashRaw.String()

				events := gjson.Get(txJSON, "tx_response.events")
				if !events.Exists() {
					logger.Error("wormchain tx has no events", zap.String("payload", txJSON))
					continue
				}

				msgs := EventsToMessagePublications(txHash, events.Array(), logger)
				for _, msg := range msgs {
					e.msgC <- msg
					wormchainMessagesConfirmed.Inc()
				}
			}
		}
	}()

	go func() {
		defer close(errC)

		for {
			_, message, err := c.ReadMessage()

			if err != nil {
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDWormchain, 1)
				wormchainConnectionErrors.WithLabelValues("channel_read_error").Inc()
				logger.Error("error reading channel", zap.Error(err))
				errC <- err
				return
			}

			// Received a message from the blockchain
			json := string(message)
			txHashRaw := gjson.Get(json, "result.events.tx\\.hash.0")
			if !txHashRaw.Exists() {
				logger.Warn("wormchain message does not have tx hash", zap.String("payload", json))
				continue
			}
			txHash := txHashRaw.String()

			events := gjson.Get(json, "result.data.value.TxResult.result.events")
			if !events.Exists() {
				logger.Warn("wormchain message has no events", zap.String("payload", json))
				continue
			}

			msgs := EventsToMessagePublications(txHash, events.Array(), logger)
			for _, msg := range msgs {
				e.msgC <- msg
				wormchainMessagesConfirmed.Inc()
			}
		}
	}()

	select {
	case <-ctx.Done():
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			logger.Error("error on closing socket ", zap.Error(err))
		}
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// TODO adjust this function as needed for wormchain
func EventsToMessagePublications(txHash string, events []gjson.Result, logger *zap.Logger) []*common.MessagePublication {
	msgs := make([]*common.MessagePublication, 0, len(events))
	for _, event := range events {
		if !event.IsObject() {
			logger.Warn("wormchain event is invalid", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		eventType := gjson.Get(event.String(), "type")
		if eventType.String() != "wormhole_foundation.wormchain.wormhole.EventPostedMessage" {
			continue
		}

		attributes := gjson.Get(event.String(), "attributes")
		if !attributes.Exists() {
			logger.Warn("wormchain message event has no attributes", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		mappedAttributes := map[string]string{}
		for _, attribute := range attributes.Array() {
			if !attribute.IsObject() {
				logger.Warn("wormchain event attribute is invalid", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			keyBase := gjson.Get(attribute.String(), "key")
			if !keyBase.Exists() {
				logger.Warn("wormchain event attribute does not have key", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			valueBase := gjson.Get(attribute.String(), "value")
			if !valueBase.Exists() {
				logger.Warn("wormchain event attribute does not have value", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			key, err := base64.StdEncoding.DecodeString(keyBase.String())
			if err != nil {
				logger.Warn("wormchain event key attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()))
				continue
			}
			value, err := base64.StdEncoding.DecodeString(valueBase.String())
			if err != nil {
				logger.Warn("wormchain event value attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			if _, ok := mappedAttributes[string(key)]; ok {
				logger.Debug("duplicate key in events", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			mappedAttributes[string(key)] = string(value)
		}

		payload, ok := mappedAttributes["payload"]
		if !ok {
			logger.Error("wormhole event does not have a payload field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		emitter, ok := mappedAttributes["emitter"]
		if !ok {
			logger.Error("wormhole event does not have a emitter field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		// currently not logged
		// chainId, ok := mappedAttributes["message.chain_id"]
		// if !ok {
		// 	logger.Error("wormhole event does not have a chain_id field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
		// 	continue
		// }

		nonce, ok := mappedAttributes["nonce"]
		if !ok {
			logger.Error("wormhole event does not have a nonce field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		sequence, ok := mappedAttributes["sequence"]
		if !ok {
			logger.Error("wormhole event does not have a sequence field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		//TODO This is not currently logged. Change this to read off the logs once it is added.
		blockTime, ok := "0", true //mappedAttributes["blockTime"]
		if !ok {
			logger.Error("wormhole event does not have a blockTime field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		logger.Info("new message detected on wormchain",
			zap.String("chainId", vaa.ChainIDWormchain.String()),
			zap.String("txHash", txHash),
			zap.String("emitter", emitter),
			zap.String("nonce", nonce),
			zap.String("sequence", sequence),
			zap.String("blockTime", blockTime),
		)

		emitterAddress, err := StringToAddress(emitter)
		if err != nil {
			logger.Error("cannot decode emitter hex", zap.String("tx_hash", txHash), zap.String("value", emitter))
			continue
		}
		txHashValue, err := StringToHash(txHash)
		if err != nil {
			logger.Error("cannot decode tx hash hex", zap.String("tx_hash", txHash), zap.String("value", txHash))
			continue
		}
		payloadValue, err := secondDecode(payload)
		if err != nil {
			logger.Error("cannot decode payload", zap.String("tx_hash", txHash), zap.String("value", payload))
			continue
		}

		blockTimeInt, err := strconv.ParseInt(blockTime, 10, 64)
		if err != nil {
			logger.Error("blocktime cannot be parsed as int", zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		nonceInt, err := strconv.ParseUint(nonce, 10, 32)
		if err != nil {
			logger.Error("nonce cannot be parsed as int", zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		sequenceInt, err := stringToUint(sequence)
		if err != nil {
			logger.Error("sequence cannot be parsed as int", zap.String("tx_hash", txHash), zap.String("value", sequence))
			continue
		}
		messagePublication := &common.MessagePublication{
			TxHash:           txHashValue,
			Timestamp:        time.Unix(blockTimeInt, 0), //TODO read off emitted message
			Nonce:            uint32(nonceInt),
			Sequence:         sequenceInt,
			EmitterChain:     vaa.ChainIDWormchain,
			EmitterAddress:   emitterAddress,
			Payload:          payloadValue,
			ConsistencyLevel: 0, // Instant finality
		}
		msgs = append(msgs, messagePublication)
	}
	return msgs
}

// TODO this encoding comes out of the logs oddly, and probably requires a change on the chain
// StringToAddress convert string into address
func StringToAddress(value string) (vaa.Address, error) {
	var address vaa.Address
	res, err := secondDecode(value)
	if err != nil {
		return address, err
	}
	copy(address[:], res)
	return address, nil
}

func stringToUint(value string) (uint64, error) {
	value = strings.TrimSuffix(value, "\"")
	value = strings.TrimPrefix(value, "\"")
	res, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return res, nil
}

func secondDecode(value string) ([]byte, error) {
	//These string are double base64 encoded, and there is a pair of quotes which get included between the first and second encoding
	value = strings.TrimSuffix(value, "\"")
	value = strings.TrimPrefix(value, "\"")
	res, err := base64.StdEncoding.DecodeString(value) //hex.DecodeString(value)
	fmt.Println("address after second decode " + string(res))
	if err != nil {
		return nil, err
	}

	return res, nil
}

// StringToHash convert string into transaction hash
func StringToHash(value string) (eth_common.Hash, error) {
	var hash eth_common.Hash
	//TODO base64? Is this correct? Double check against the logs
	res, err := hex.DecodeString(value)
	if err != nil {
		return hash, err
	}
	copy(hash[:], res)
	return hash, nil
}
