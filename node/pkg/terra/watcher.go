package terra

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Terra blockchain and reporting new transactions to the contract
	Watcher struct {
		urlWS    string
		urlLCD   string
		contract string

		msgChan chan *common.MessagePublication
		setChan chan *common.GuardianSet
	}
)

var (
	terraConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_terra_connection_errors_total",
			Help: "Total number of Terra connection errors",
		}, []string{"reason"})
	terraMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_terra_messages_confirmed_total",
			Help: "Total number of verified terra messages found",
		})
	currentTerraHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_terra_current_height",
			Help: "Current terra slot height (at default commitment level, not the level used for observations)",
		})
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_terra_query_latency",
			Help: "Latency histogram for terra RPC calls",
		}, []string{"operation"})
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

// NewWatcher creates a new Terra contract watcher
func NewWatcher(urlWS string, urlLCD string, contract string, lockEvents chan *common.MessagePublication, setEvents chan *common.GuardianSet) *Watcher {
	return &Watcher{urlWS: urlWS, urlLCD: urlLCD, contract: contract, msgChan: lockEvents, setChan: setEvents}
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDTerra, &gossipv1.Heartbeat_Network{
		ContractAddress: e.contract,
	})

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	logger.Info("connecting to websocket", zap.String("url", e.urlWS))

	c, _, err := websocket.DefaultDialer.DialContext(ctx, e.urlWS, nil)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTerra, 1)
		terraConnectionErrors.WithLabelValues("websocket_dial_error").Inc()
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer c.Close()

	// Subscribe to smart contract transactions
	params := [...]string{fmt.Sprintf("tm.event='Tx' AND execute_contract.contract_address='%s'", e.contract)}
	command := &clientRequest{
		JSONRPC: "2.0",
		Method:  "subscribe",
		Params:  params,
		ID:      1,
	}
	err = c.WriteJSON(command)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTerra, 1)
		terraConnectionErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("websocket subscription failed: %w", err)
	}

	// Wait for the success response
	_, _, err = c.ReadMessage()
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTerra, 1)
		terraConnectionErrors.WithLabelValues("event_subscription_error").Inc()
		return fmt.Errorf("event subscription failed: %w", err)
	}
	logger.Info("subscribed to new transaction events")

	readiness.SetReady(common.ReadinessTerraSyncing)

	go func() {
		t := time.NewTicker(5 * time.Second)
		client := &http.Client{
			Timeout: time.Second * 5,
		}

		for {
			<-t.C

			// Query and report height and set currentTerraHeight
			resp, err := client.Get(fmt.Sprintf("%s/blocks/latest", e.urlLCD))
			if err != nil {
				logger.Error("query latest block response error", zap.Error(err))
				continue
			}
			blocksBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logger.Error("query guardian set error", zap.Error(err))
				errC <- err
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			blockJSON := string(blocksBody)
			latestBlock := gjson.Get(blockJSON, "block.header.height")
			logger.Info("current Terra height", zap.Int64("block", latestBlock.Int()))
			currentTerraHeight.Set(float64(latestBlock.Int()))
			p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDTerra, &gossipv1.Heartbeat_Network{
				Height:          latestBlock.Int(),
				ContractAddress: e.contract,
			})
		}
	}()

	go func() {
		defer close(errC)

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTerra, 1)
				terraConnectionErrors.WithLabelValues("channel_read_error").Inc()
				logger.Error("error reading channel", zap.Error(err))
				errC <- err
				return
			}

			// Received a message from the blockchain
			json := string(message)

			txHashRaw := gjson.Get(json, "result.events.tx\\.hash.0")
			if !txHashRaw.Exists() {
				logger.Warn("terra message does not have tx hash", zap.String("payload", json))
				continue
			}
			txHash := txHashRaw.String()

			events := gjson.Get(json, "result.data.value.TxResult.result.events")
			if !events.Exists() {
				logger.Warn("terra message has no events", zap.String("payload", json))
				continue
			}

			for _, event := range events.Array() {
				if !event.IsObject() {
					logger.Warn("terra event is invalid", zap.String("tx_hash", txHash), zap.String("event", event.String()))
					continue
				}
				eventType := gjson.Get(event.String(), "type")
				if eventType.String() != "wasm" {
					continue
				}

				attributes := gjson.Get(event.String(), "attributes")
				if !attributes.Exists() {
					logger.Warn("terra message event has no attributes", zap.String("payload", json), zap.String("event", event.String()))
					continue
				}
				mappedAttributes := map[string]string{}
				for _, attribute := range attributes.Array() {
					if !attribute.IsObject() {
						logger.Warn("terra event attribute is invalid", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
						continue
					}
					keyBase := gjson.Get(attribute.String(), "key")
					if !keyBase.Exists(){
						logger.Warn("terra event attribute does not have key", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
						continue
					}
					valueBase := gjson.Get(attribute.String(), "value")
					if !valueBase.Exists(){
						logger.Warn("terra event attribute does not have value", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
						continue
					}

					key, err := base64.StdEncoding.DecodeString(keyBase.String())
					if err != nil {
						logger.Warn("terra event key attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()))
						continue
					}
					value, err := base64.StdEncoding.DecodeString(valueBase.String())
					if err != nil {
						logger.Warn("terra event value attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
						continue
					}

					if _, ok := mappedAttributes[string(key)]; ok {
						logger.Debug("duplicate key in events", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
						continue
					}

					mappedAttributes[string(key)] = string(value)
				}

				contractAddress, ok := mappedAttributes["contract_address"]
				if !ok {
					logger.Warn("terra wasm event without contract address field set", zap.String("event", event.String()))
					continue
				}
				// This is not a wormhole message
				if contractAddress != e.contract {
					continue
				}

				payload, ok := mappedAttributes["message.message"]
				if !ok {
					logger.Error("wormhole event does not have a message field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
					continue
				}
				sender, ok := mappedAttributes["message.sender"]
				if !ok {
					logger.Error("wormhole event does not have a sender field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
					continue
				}
				chainId, ok := mappedAttributes["message.chain_id"]
				if !ok {
					logger.Error("wormhole event does not have a chain_id field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
					continue
				}
				nonce, ok := mappedAttributes["message.nonce"]
				if !ok {
					logger.Error("wormhole event does not have a nonce field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
					continue
				}
				sequence, ok := mappedAttributes["message.sequence"]
				if !ok {
					logger.Error("wormhole event does not have a sequence field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
					continue
				}
				blockTime, ok := mappedAttributes["message.block_time"]
				if !ok {
					logger.Error("wormhole event does not have a block_time field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
					continue
				}

				logger.Info("new message detected on terra",
					zap.String("chainId", chainId),
					zap.String("txHash", txHash),
					zap.String("sender", sender),
					zap.String("nonce", nonce),
					zap.String("sequence", sequence),
					zap.String("blockTime", blockTime),
				)

				senderAddress, err := StringToAddress(sender)
				if err != nil {
					logger.Error("cannot decode emitter hex", zap.String("tx_hash", txHash), zap.String("value", sender))
					continue
				}
				txHashValue, err := StringToHash(txHash)
				if err != nil {
					logger.Error("cannot decode tx hash hex", zap.String("tx_hash", txHash), zap.String("value", txHash))
					continue
				}
				payloadValue, err := hex.DecodeString(payload)
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
				sequenceInt, err := strconv.ParseUint(sequence, 10, 64)
				if err != nil {
					logger.Error("sequence cannot be parsed as int", zap.String("tx_hash", txHash), zap.String("value", blockTime))
					continue
				}
				messagePublication := &common.MessagePublication{
					TxHash:           txHashValue,
					Timestamp:        time.Unix(blockTimeInt, 0),
					Nonce:            uint32(nonceInt),
					Sequence:         sequenceInt,
					EmitterChain:     vaa.ChainIDTerra,
					EmitterAddress:   senderAddress,
					Payload:          payloadValue,
					ConsistencyLevel: 0, // Instant finality
				}
				e.msgChan <- messagePublication
				terraMessagesConfirmed.Inc()
			}

			client := &http.Client{
				Timeout: time.Second * 15,
			}

			// Query and report guardian set status
			requestURL := fmt.Sprintf("%s/wasm/contracts/%s/store?query_msg={\"guardian_set_info\":{}}", e.urlLCD, e.contract)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
			if err != nil {
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTerra, 1)
				terraConnectionErrors.WithLabelValues("guardian_set_req_error").Inc()
				logger.Error("query guardian set request error", zap.Error(err))
				errC <- err
				return
			}

			msm := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTerra, 1)
				logger.Error("query guardian set response error", zap.Error(err))
				errC <- err
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			queryLatency.WithLabelValues("guardian_set_info").Observe(time.Since(msm).Seconds())
			if err != nil {
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTerra, 1)
				logger.Error("query guardian set error", zap.Error(err))
				errC <- err
				resp.Body.Close()
				return
			}

			json = string(body)
			guardianSetIndex := gjson.Get(json, "result.guardian_set_index")
			addresses := gjson.Get(json, "result.addresses.#.bytes")

			logger.Debug("current guardian set on Terra",
				zap.Any("guardianSetIndex", guardianSetIndex),
				zap.Any("addresses", addresses))

			resp.Body.Close()

			// We do not send guardian changes to the processor - ETH guardians are the source of truth.
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
