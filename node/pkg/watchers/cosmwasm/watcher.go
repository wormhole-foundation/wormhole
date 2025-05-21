package cosmwasm

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// ReadLimitSize can be used to increase the read limit size on the listening connection. The default read limit size is not large enough,
// causing "failed to read: read limited at 32769 bytes" errors during testing. Increasing this limit effects an internal buffer that
// is used to as part of the zero alloc/copy design.
const ReadLimitSize = 524288

type (
	// Watcher is responsible for looking over a cosmwasm blockchain and reporting new transactions to the contract
	Watcher struct {
		urlWS    string
		urlLCD   string
		contract string

		msgC chan<- *common.MessagePublication

		// Incoming re-observation requests from the network. Pre-filtered to only
		// include requests for our chainID.
		obsvReqC <-chan *gossipv1.ObservationRequest

		// Readiness component
		readinessSync readiness.Component
		// VAA ChainID of the network we're connecting to.
		chainID vaa.ChainID
		// Key for contract address in the wasm logs
		contractAddressFilterKey string
		// Key for contract address in the wasm logs
		contractAddressLogKey string

		// URL to get the latest block info from
		latestBlockURL string

		// b64Encoded indicates if transactions are base 64 encoded.
		b64Encoded bool
	}
)

var (
	connectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_terra_connection_errors_total",
			Help: "Total number of connection errors on a cosmwasm chain",
		}, []string{"terra_network", "reason"})
	messagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_terra_messages_confirmed_total",
			Help: "Total number of verified messages found on a cosmwasm chain",
		}, []string{"terra_network"})
	currentSlotHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_terra_current_height",
			Help: "Current slot height on a cosmwasm chain (at default commitment level, not the level used for observations)",
		}, []string{"terra_network"})
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_terra_query_latency",
			Help: "Latency histogram for RPC calls on a cosmwasm chain",
		}, []string{"terra_network", "operation"})
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

// NewWatcher creates a new cosmwasm contract watcher
func NewWatcher(
	urlWS string,
	urlLCD string,
	contract string,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	chainID vaa.ChainID,
	env common.Environment,
) *Watcher {

	// CosmWasm 1.0.0
	contractAddressFilterKey := "execute._contract_address"
	contractAddressLogKey := "_contract_address"

	// Do not add a leading slash
	latestBlockURL := "cosmos/base/tendermint/v1beta1/blocks/latest"

	// Injective does not base64 encode parameters (as of release v1.11.2).
	// Terra does not base64 encode parameters (as of v3.0.1 software upgrade)
	// Terra2 no longer base64 encodes parameters.
	b64Encoded := env == common.UnsafeDevNet || (chainID != vaa.ChainIDInjective && chainID != vaa.ChainIDTerra2 && chainID != vaa.ChainIDTerra)

	return &Watcher{
		urlWS:                    urlWS,
		urlLCD:                   urlLCD,
		contract:                 contract,
		msgC:                     msgC,
		obsvReqC:                 obsvReqC,
		readinessSync:            common.MustConvertChainIdToReadinessSyncing(chainID),
		chainID:                  chainID,
		contractAddressFilterKey: contractAddressFilterKey,
		contractAddressLogKey:    contractAddressLogKey,
		latestBlockURL:           latestBlockURL,
		b64Encoded:               b64Encoded,
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	networkName := e.chainID.String()

	p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: e.contract,
	})

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	logger.Info("Starting watcher",
		zap.String("watcher_name", "cosmwasm"),
		zap.String("urlWS", e.urlWS),
		zap.String("urlLCD", e.urlLCD),
		zap.String("contract", e.contract),
		zap.String("chainID", e.chainID.String()),
	)

	logger.Info("connecting to websocket", zap.String("network", networkName), zap.String("url", e.urlWS))

	//nolint:bodyclose // The close is down below. The linter misses it.
	c, _, err := websocket.Dial(ctx, e.urlWS, nil)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		connectionErrors.WithLabelValues(networkName, "websocket_dial_error").Inc()
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	c.SetReadLimit(ReadLimitSize)

	// Subscribe to smart contract transactions
	params := [...]string{fmt.Sprintf("tm.event='Tx' AND %s='%s'", e.contractAddressFilterKey, e.contract)}
	command := &clientRequest{
		JSONRPC: "2.0",
		Method:  "subscribe",
		Params:  params,
		ID:      1,
	}
	err = wsjson.Write(ctx, c, command)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		connectionErrors.WithLabelValues(networkName, "websocket_subscription_error").Inc()
		return fmt.Errorf("websocket subscription failed: %w", err)
	}

	// Wait for the success response
	_, _, err = c.Read(ctx)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		connectionErrors.WithLabelValues(networkName, "event_subscription_error").Inc()
		return fmt.Errorf("event subscription failed: %w", err)
	}
	logger.Info("subscribed to new transaction events", zap.String("network", networkName))

	readiness.SetReady(e.readinessSync)

	common.RunWithScissors(ctx, errC, "cosmwasm_block_height", func(ctx context.Context) error {
		t := time.NewTicker(5 * time.Second)
		client := &http.Client{
			Timeout: time.Second * 5,
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				msm := time.Now()
				// Query and report height and set currentSlotHeight
				resp, err := client.Get(fmt.Sprintf("%s/%s", e.urlLCD, e.latestBlockURL)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
				if err != nil {
					logger.Error("query latest block response error", zap.String("network", networkName), zap.Error(err))
					continue
				}
				blocksBody, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error("query latest block response read error", zap.String("network", networkName), zap.Error(err))
					errC <- err
					resp.Body.Close()
					continue
				}
				resp.Body.Close()

				// Update the prom metrics with how long the http request took to the rpc
				queryLatency.WithLabelValues(networkName, "block_latest").Observe(time.Since(msm).Seconds())

				blockJSON := string(blocksBody)
				latestBlock := gjson.Get(blockJSON, "block.header.height")
				logger.Debug("current height", zap.String("network", networkName), zap.Int64("block", latestBlock.Int()))
				currentSlotHeight.WithLabelValues(networkName).Set(float64(latestBlock.Int()))
				p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
					Height:          latestBlock.Int(),
					ContractAddress: e.contract,
				})

				readiness.SetReady(e.readinessSync)
			}
		}
	})

	common.RunWithScissors(ctx, errC, "cosmwasm_objs_req", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case r := <-e.obsvReqC:
				// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
				// and only writes to the channel for this chain id.
				// If either of the below cases are true, something has gone wrong
				if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != e.chainID {
					panic("invalid chain ID")
				}

				tx := hex.EncodeToString(r.TxHash)

				logger.Info("received observation request", zap.String("network", networkName), zap.String("tx_hash", tx))

				client := &http.Client{
					Timeout: time.Second * 5,
				}

				// Query for tx by hash
				resp, err := client.Get(fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", e.urlLCD, tx)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
				if err != nil {
					logger.Error("query tx response error", zap.String("network", networkName), zap.Error(err))
					continue
				}
				txBody, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error("query tx response read error", zap.String("network", networkName), zap.Error(err))
					resp.Body.Close()
					continue
				}
				resp.Body.Close()

				txJSON := string(txBody)

				txHashRaw := gjson.Get(txJSON, "tx_response.txhash")
				if !txHashRaw.Exists() {
					logger.Error("tx does not have tx hash", zap.String("network", networkName), zap.String("payload", txJSON))
					continue
				}
				txHash := txHashRaw.String()

				events := gjson.Get(txJSON, "tx_response.events")
				if !events.Exists() {
					logger.Error("tx has no events", zap.String("network", networkName), zap.String("payload", txJSON))
					continue
				}

				contractAddressLogKey := e.contractAddressLogKey
				if e.chainID == vaa.ChainIDTerra {
					// Terra Classic upgraded WASM versions starting at block 13215800. If this transaction is from before that, we need to use the old contract address format.
					blockHeightStr := gjson.Get(txJSON, "tx_response.height")
					if !blockHeightStr.Exists() {
						logger.Error("failed to look up block height on old reobserved tx", zap.String("network", networkName), zap.String("txHash", txHash), zap.String("payload", txJSON))
						continue
					}
					blockHeight := blockHeightStr.Int()
					if blockHeight < 13215800 {
						logger.Info("doing look up of old tx", zap.String("network", networkName), zap.String("txHash", txHash), zap.Int64("blockHeight", blockHeight))
						contractAddressLogKey = "contract_address"
					}
				}

				msgs := EventsToMessagePublications(e.contract, txHash, events.Array(), logger, e.chainID, contractAddressLogKey, e.b64Encoded)
				for _, msg := range msgs {
					msg.IsReobservation = true
					e.msgC <- msg
					messagesConfirmed.WithLabelValues(networkName).Inc()
					watchers.ReobservationsByChain.WithLabelValues(networkName, "std").Inc()
				}
			}
		}
	})

	common.RunWithScissors(ctx, errC, "cosmwasm_data_pump", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				_, message, err := c.Read(ctx)
				if err != nil {
					p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
					connectionErrors.WithLabelValues(networkName, "channel_read_error").Inc()
					logger.Error("error reading channel", zap.String("network", networkName), zap.Error(err))
					errC <- err
					return nil
				}

				// Received a message from the blockchain
				json := string(message)

				txHashRaw := gjson.Get(json, "result.events.tx\\.hash.0")
				if !txHashRaw.Exists() {
					logger.Warn("message does not have tx hash", zap.String("network", networkName), zap.String("payload", json))
					continue
				}
				txHash := txHashRaw.String()

				events := gjson.Get(json, "result.data.value.TxResult.result.events")
				if !events.Exists() {
					logger.Warn("message has no events", zap.String("network", networkName), zap.String("payload", json))
					continue
				}

				msgs := EventsToMessagePublications(e.contract, txHash, events.Array(), logger, e.chainID, e.contractAddressLogKey, e.b64Encoded)
				for _, msg := range msgs {
					e.msgC <- msg
					messagesConfirmed.WithLabelValues(networkName).Inc()
				}

				// We do not send guardian changes to the processor - ETH guardians are the source of truth.
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

func EventsToMessagePublications(contract string, txHash string, events []gjson.Result, logger *zap.Logger, chainID vaa.ChainID, contractAddressKey string, b64Encoded bool) []*common.MessagePublication {
	networkName := chainID.String()
	msgs := make([]*common.MessagePublication, 0, len(events))
	for _, event := range events {
		if !event.IsObject() {
			logger.Warn("event is invalid", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		eventType := gjson.Get(event.String(), "type")
		if eventType.String() == "recv_packet" && chainID != vaa.ChainIDWormchain {
			logger.Warn("processing ibc-related events is disabled", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("event", event.String()))
			return []*common.MessagePublication{}
		}

		if eventType.String() != "wasm" {
			continue
		}

		attributes := gjson.Get(event.String(), "attributes")
		if !attributes.Exists() {
			logger.Warn("message event has no attributes", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		mappedAttributes := map[string]string{}
		for _, attribute := range attributes.Array() {
			if !attribute.IsObject() {
				logger.Warn("event attribute is invalid", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			keyBase := gjson.Get(attribute.String(), "key")
			if !keyBase.Exists() {
				logger.Warn("event attribute does not have key", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			valueBase := gjson.Get(attribute.String(), "value")
			if !valueBase.Exists() {
				logger.Warn("event attribute does not have value", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			var key, value []byte
			if b64Encoded {
				var err error
				key, err = base64.StdEncoding.DecodeString(keyBase.String())
				if err != nil {
					logger.Warn("event key attribute is invalid", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("key", keyBase.String()))
					continue
				}
				value, err = base64.StdEncoding.DecodeString(valueBase.String())
				if err != nil {
					logger.Warn("event value attribute is invalid", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
					continue
				}
			} else {
				key = []byte(keyBase.String())
				value = []byte(valueBase.String())
			}

			if _, ok := mappedAttributes[string(key)]; ok {
				logger.Debug("duplicate key in events", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			logger.Debug("msg attribute",
				zap.String("network", networkName),
				zap.String("tx_hash", txHash), zap.String("key", string(key)), zap.String("value", string(value)))

			mappedAttributes[string(key)] = string(value)
		}

		contractAddress, ok := mappedAttributes[contractAddressKey]
		if !ok {
			logger.Warn("wasm event without contract address field set", zap.String("network", networkName), zap.String("event", event.String()))
			continue
		}
		// This is not a wormhole message
		if contractAddress != contract {
			continue
		}

		payload, ok := mappedAttributes["message.message"]
		if !ok {
			logger.Error("wormhole event does not have a message field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		sender, ok := mappedAttributes["message.sender"]
		if !ok {
			logger.Error("wormhole event does not have a sender field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		chainId, ok := mappedAttributes["message.chain_id"]
		if !ok {
			logger.Error("wormhole event does not have a chain_id field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		nonce, ok := mappedAttributes["message.nonce"]
		if !ok {
			logger.Error("wormhole event does not have a nonce field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		sequence, ok := mappedAttributes["message.sequence"]
		if !ok {
			logger.Error("wormhole event does not have a sequence field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		blockTime, ok := mappedAttributes["message.block_time"]
		if !ok {
			logger.Error("wormhole event does not have a block_time field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		logger.Info("new message detected on cosmwasm",
			zap.String("network", networkName),
			zap.String("chainId", chainId),
			zap.String("txHash", txHash),
			zap.String("sender", sender),
			zap.String("nonce", nonce),
			zap.String("sequence", sequence),
			zap.String("blockTime", blockTime),
		)

		senderAddress, err := StringToAddress(sender)
		if err != nil {
			logger.Error("cannot decode emitter hex", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", sender))
			continue
		}
		txHashValue, err := StringToHash(txHash)
		if err != nil {
			logger.Error("cannot decode tx hash hex", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", txHash))
			continue
		}
		payloadValue, err := hex.DecodeString(payload)
		if err != nil {
			logger.Error("cannot decode payload", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", payload))
			continue
		}

		blockTimeInt, err := strconv.ParseInt(blockTime, 10, 64)
		if err != nil {
			logger.Error("blocktime cannot be parsed as int", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		nonceInt, err := strconv.ParseUint(nonce, 10, 32)
		if err != nil {
			logger.Error("nonce cannot be parsed as int", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		sequenceInt, err := strconv.ParseUint(sequence, 10, 64)
		if err != nil {
			logger.Error("sequence cannot be parsed as int", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		messagePublication := &common.MessagePublication{
			TxID:             txHashValue.Bytes(),
			Timestamp:        time.Unix(blockTimeInt, 0),
			Nonce:            uint32(nonceInt),
			Sequence:         sequenceInt,
			EmitterChain:     chainID,
			EmitterAddress:   senderAddress,
			Payload:          payloadValue,
			ConsistencyLevel: 0, // Instant finality
		}
		msgs = append(msgs, messagePublication)
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
