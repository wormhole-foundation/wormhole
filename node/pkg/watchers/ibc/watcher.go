package ibc

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/cosmwasm"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	ethCommon "github.com/ethereum/go-ethereum/common"

	"go.uber.org/zap"
)

type (
	// ChainConfig is the list of chains to be monitored over IBC, along with their channel data.
	ChainConfig []ChainConfigEntry

	// ChainConfigEntry defines the entry for chain being monitored by IBC.
	ChainConfigEntry struct {
		ChainID  vaa.ChainID
		MsgC     chan<- *common.MessagePublication
		ObsvReqC <-chan *gossipv1.ObservationRequest
	}
)

var (
	// Chains defines the list of chains to be monitored by IBC. Add new chains here as necessary.
	Chains = []vaa.ChainID{}

	// Features is the feature string to be published in the gossip heartbeat messages. It will include all chains that are actually enabled on IBC.
	Features = ""

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
	currentSlotHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_ibc_current_height",
			Help: "Current slot height on an IBC connected chain (the block height on wormchain)",
		}, []string{"chain_name"})
	invalidChainIdMismatches = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_ibc_chain_id_mismatches",
			Help: "Total number of cases where the wormhole chain ID does not match the IBC connection ID",
		}, []string{"ibc_channel_id"})
)

type (
	// Watcher is responsible for monitoring the IBC contract on wormchain and publishing wormhole messages for all chains connected via IBC.
	Watcher struct {
		wsUrl           string
		lcdUrl          string
		contractAddress string
		logger          *zap.Logger

		// chainConfig is the list of chains to be monitored, along with their channel data.
		chainConfig ChainConfig

		// channelMap provides access by IBC connection ID.
		connectionMap map[string]*connectionEntry

		// chainMap provides access by chain ID.
		chainMap map[vaa.ChainID]*connectionEntry

		// connectionIdMap provides a mapping from IBC channel ID to IBC connection ID.
		connectionIdMap map[string]string

		// connectionIdLock protects connectionIdMap.
		connectionIdLock sync.Mutex
	}

	// connectionEntry defines the chain that is associated with an IBC connection.
	connectionEntry struct {
		connectionID string
		chainID      vaa.ChainID
		chainName    string
		readiness    readiness.Component
		msgC         chan<- *common.MessagePublication
		obsvReqC     <-chan *gossipv1.ObservationRequest
	}
)

// NewWatcher creates a new IBC contract watcher
func NewWatcher(
	wsUrl string,
	lcdUrl string,
	contractAddress string,
	chainConfig ChainConfig,
) *Watcher {
	return &Watcher{
		wsUrl:           wsUrl,
		lcdUrl:          lcdUrl,
		contractAddress: contractAddress,
		chainConfig:     chainConfig,
		connectionIdMap: make(map[string]string),
		connectionMap:   make(map[string]*connectionEntry),
		chainMap:        make(map[vaa.ChainID]*connectionEntry),
	}
}

// clientRequest is used to subscribe for events from the contract.
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

// ibcReceivePublishEvent represents the log message received from the IBC receiver contract.
type ibcReceivePublishEvent struct {
	ChannelID string
	Msg       *common.MessagePublication
}

// Run is the runnable for monitoring the IBC contract on wormchain.
func (w *Watcher) Run(ctx context.Context) error {
	w.logger = supervisor.Logger(ctx)

	errC := make(chan error)
	defer close(errC)

	// Query the contract for the chain ID to IBC connection ID mapping.
	connectionIdMap, err := w.queryConnectionIdMap()
	if err != nil {
		return fmt.Errorf("failed to query for connection ID map: %w", err)
	}

	// Build our internal data structures based on the config passed in.
	if len(w.connectionMap) == 0 {
		features := ""
		for _, chainToMonitor := range w.chainConfig {
			_, exists := w.chainMap[chainToMonitor.ChainID]
			if exists {
				return fmt.Errorf("detected duplicate chainID: %v", chainToMonitor.ChainID)
			}

			connID, exists := connectionIdMap[chainToMonitor.ChainID]
			if !exists {
				return fmt.Errorf("there is no IBC connection ID defined for chainID %v", chainToMonitor.ChainID)
			}

			ce := &connectionEntry{
				connectionID: connID,
				chainID:      chainToMonitor.ChainID,
				chainName:    vaa.ChainID(chainToMonitor.ChainID).String(),
				readiness:    common.MustConvertChainIdToReadinessSyncing(chainToMonitor.ChainID),
				msgC:         chainToMonitor.MsgC,
				obsvReqC:     chainToMonitor.ObsvReqC,
			}

			w.logger.Info("will monitor chain over IBC", zap.String("chain", ce.chainName), zap.String("IBC connection", ce.connectionID))
			w.connectionMap[ce.connectionID] = ce
			w.chainMap[ce.chainID] = ce

			if features == "" {
				features = "ibc:"
			} else {
				features += "|"
			}
			features += ce.chainID.String()

			p2p.DefaultRegistry.SetNetworkStats(ce.chainID, &gossipv1.Heartbeat_Network{ContractAddress: w.contractAddress})
		}

		Features = features
	}

	w.logger.Info("creating watcher",
		zap.String("wsUrl", w.wsUrl),
		zap.String("lcdUrl", w.lcdUrl),
		zap.String("contract", w.contractAddress),
		zap.String("features", Features),
	)

	c, _, err := websocket.Dial(ctx, w.wsUrl, nil)
	if err != nil {
		connectionErrors.WithLabelValues("websocket_dial_error").Inc()
		return fmt.Errorf("failed to establish tendermint websocket connection: %w", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	c.SetReadLimit(cosmwasm.ReadLimitSize)

	// Subscribe to smart contract transactions.
	params := [...]string{fmt.Sprintf("tm.event='Tx' AND wasm._contract_address='%s'", w.contractAddress)}
	command := &clientRequest{
		JSONRPC: "2.0",
		Method:  "subscribe",
		Params:  params,
		ID:      1,
	}
	err = wsjson.Write(ctx, c, command)
	if err != nil {
		connectionErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Wait for the success response.
	_, subResp, err := c.Read(ctx)
	if err != nil {
		connectionErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("failed to receive response to subscribe request: %w", err)
	}
	if strings.Contains(string(subResp), "error") {
		connectionErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("failed to subscribe to events, response: %s", string(subResp))
	}

	// Start a routine to listen for messages from the contract.
	common.RunWithScissors(ctx, errC, "ibc_data_pump", func(ctx context.Context) error {
		return w.handleEvents(ctx, c, errC)
	})

	// Start a routine to periodically query the wormchain block height.
	common.RunWithScissors(ctx, errC, "ibc_block_height", func(ctx context.Context) error {
		return w.handleQueryBlockHeight(ctx, c, errC)
	})

	// Start a routine for each chain to listen for observation requests.
	for _, ce := range w.chainMap {
		common.RunWithScissors(ctx, errC, "ibc_objs_req", func(ctx context.Context) error {
			return w.handleObservationRequests(ctx, errC, ce)
		})
	}

	// Signal to the supervisor that this runnable has finished initialization.
	supervisor.Signal(ctx, supervisor.SignalHealthy)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// handleEvents reads messages from the IBC receiver contract and processes them.
func (w *Watcher) handleEvents(ctx context.Context, c *websocket.Conn, errC chan error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errC:
			return fmt.Errorf("handleEvents died: %w", err)
		default:
			_, message, err := c.Read(ctx)
			if err != nil {
				w.logger.Error("failed to read socket", zap.Error(err))
				connectionErrors.WithLabelValues("channel_read_error").Inc()
				return fmt.Errorf("failed to read socket: %w", err)
			}

			// Received a message from the blockchain.
			json := string(message)

			txHashRaw := gjson.Get(json, "result.events.tx\\.hash.0")
			if !txHashRaw.Exists() {
				w.logger.Warn("message does not have tx hash", zap.String("payload", json))
				continue
			}
			txHash, err := vaa.StringToHash(txHashRaw.String())
			if err != nil {
				w.logger.Warn("failed to parse txHash", zap.String("txHash", txHashRaw.String()), zap.Error(err))
				continue
			}

			events := gjson.Get(json, "result.data.value.TxResult.result.events")
			if !events.Exists() {
				w.logger.Warn("message has no events", zap.String("payload", json))
				continue
			}

			for _, event := range events.Array() {
				if !event.IsObject() {
					w.logger.Warn("event is invalid", zap.Stringer("tx_hash", txHash), zap.String("event", event.String()))
					continue
				}
				eventType := gjson.Get(event.String(), "type").String()
				if eventType == "wasm" {
					evt, err := parseIbcReceivePublishEvent(w.logger, w.contractAddress, event, txHash)
					if err != nil {
						w.logger.Error("failed to parse wasm event", zap.Error(err), zap.String("event", event.String()))
						continue
					}

					if evt != nil {
						w.processIbcReceivePublishEvent(txHash, evt, "new")
					}
				} else {
					w.logger.Debug("ignoring uninteresting event", zap.String("eventType", eventType))
				}
			}
		}
	}
}

// handleQueryBlockHeight gets the latest block height from wormchain each interval and updates the status on all the connected chains.
func (w *Watcher) handleQueryBlockHeight(ctx context.Context, c *websocket.Conn, errC chan error) error {
	const latestBlockURL = "blocks/latest"

	t := time.NewTicker(5 * time.Second)
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errC:
			return fmt.Errorf("handleQueryBlockHeight died: %w", err)
		case <-t.C:
			resp, err := client.Get(fmt.Sprintf("%s/%s", w.lcdUrl, latestBlockURL))
			if err != nil {
				return fmt.Errorf("failed to query latest block: %w", err)
			}
			blocksBody, err := io.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				return fmt.Errorf("failed to read latest block body: %w", err)
			}
			resp.Body.Close()

			blockJSON := string(blocksBody)
			latestBlockAsInt := gjson.Get(blockJSON, "block.header.height").Int()
			latestBlockAsFloat := float64(latestBlockAsInt)
			w.logger.Debug("current block height", zap.Int64("height", latestBlockAsInt))

			for _, ce := range w.chainMap {
				currentSlotHeight.WithLabelValues(ce.chainName).Set(latestBlockAsFloat)
				p2p.DefaultRegistry.SetNetworkStats(ce.chainID, &gossipv1.Heartbeat_Network{
					Height:          latestBlockAsInt,
					ContractAddress: w.contractAddress,
				})

				readiness.SetReady(ce.readiness)
			}

			readiness.SetReady(common.ReadinessIBCSyncing)
		}
	}
}

// handleObservationRequests listens for observation requests for a single chain and processes them by reading the requested transaction
// from wormchain and publishing the associated message. This function is instantiated for each connected chain.
func (w *Watcher) handleObservationRequests(ctx context.Context, errC chan error, ce *connectionEntry) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errC:
			return fmt.Errorf("handleObservationRequests died: %w", err)
		case r := <-ce.obsvReqC:
			if vaa.ChainID(r.ChainId) != ce.chainID {
				panic("invalid chain ID")
			}

			reqTxHashStr := hex.EncodeToString(r.TxHash)
			w.logger.Info("received observation request", zap.String("chain", ce.chainName), zap.String("txHash", reqTxHashStr))

			client := &http.Client{
				Timeout: time.Second * 5,
			}

			// Query for tx by hash.
			resp, err := client.Get(fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", w.lcdUrl, reqTxHashStr))
			if err != nil {
				w.logger.Error("query tx response error", zap.String("chain", ce.chainName), zap.Error(err))
				continue
			}
			txBody, err := io.ReadAll(resp.Body)
			if err != nil {
				w.logger.Error("query tx response read error", zap.String("chain", ce.chainName), zap.Error(err))
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			txJSON := string(txBody)
			txHashRaw := gjson.Get(txJSON, "tx_response.txhash")
			if !txHashRaw.Exists() {
				w.logger.Error("tx does not have tx hash", zap.String("chain", ce.chainName), zap.String("payload", txJSON))
				continue
			}
			txHashStr := txHashRaw.String()
			txHash, err := vaa.StringToHash(txHashStr)
			if err != nil {
				w.logger.Error("tx does not have tx hash", zap.String("chain", ce.chainName), zap.String("payload", txJSON))
				continue
			}

			events := gjson.Get(txJSON, "tx_response.events")
			if !events.Exists() {
				w.logger.Error("tx has no events", zap.String("chain", ce.chainName), zap.String("payload", txJSON))
				continue
			}

			for _, event := range events.Array() {
				if !event.IsObject() {
					w.logger.Error("event is invalid", zap.String("chain", ce.chainName), zap.String("tx_hash", txHashStr), zap.String("event", event.String()))
					continue
				}
				eventType := gjson.Get(event.String(), "type")
				if eventType.String() == "wasm" {
					w.logger.Debug("found wasm event in reobservation", zap.String("chain", ce.chainName), zap.Stringer("txHash", txHash))
					evt, err := parseIbcReceivePublishEvent(w.logger, w.contractAddress, event, txHash)
					if err != nil {
						w.logger.Error("failed to parse wasm event", zap.String("chain", ce.chainName), zap.Error(err), zap.Any("event", event))
						continue
					}

					if evt != nil {
						w.processIbcReceivePublishEvent(txHash, evt, "reobservation")
					}
				} else {
					w.logger.Debug("ignoring uninteresting event in reobservation", zap.String("chain", ce.chainName), zap.Stringer("txHash", txHash), zap.String("eventType", eventType.String()))
				}
			}
		}
	}
}

// parseIbcReceivePublishEvent parses a wasm event into an object. Since the watcher only subscribes to events from a single contract, this function returns an error
// if the contract does not match the desired one. However, since the contract publishes multiple action types, this function returns nil rather than an error
// if the event is not for the desired action.
func parseIbcReceivePublishEvent(logger *zap.Logger, desiredContract string, event gjson.Result, txHash ethCommon.Hash) (*ibcReceivePublishEvent, error) {
	var attributes WasmAttributes
	err := attributes.Parse(logger, event)
	if err != nil {
		logger.Error("failed to parse event attributes", zap.Error(err), zap.String("event", event.String()))
		return nil, fmt.Errorf("failed to parse attributes: %w", err)
	}

	str, err := attributes.GetAsString("_contract_address")
	if err != nil {
		return nil, err
	}
	if str != desiredContract {
		return nil, fmt.Errorf("received an event from an unexpected contract: %s", str)
	}

	str, err = attributes.GetAsString("action")
	if err != nil || str != "receive_publish" {
		return nil, nil
	}

	evt := new(ibcReceivePublishEvent)
	evt.Msg = new(common.MessagePublication)
	evt.Msg.TxHash = txHash

	evt.ChannelID, err = attributes.GetAsString("channel_id")
	if err != nil {
		return evt, err
	}

	unumber, err := attributes.GetAsUint("message.chain_id", 16)
	if err != nil {
		return evt, err
	}
	evt.Msg.EmitterChain = vaa.ChainID(unumber)

	str, err = attributes.GetAsString("message.sender")
	if err != nil {
		return evt, err
	}
	evt.Msg.EmitterAddress, err = vaa.StringToAddress(str)
	if err != nil {
		return evt, fmt.Errorf("failed to parse message.sender attribute %s: %w", str, err)
	}

	unumber, err = attributes.GetAsUint("message.nonce", 32)
	if err != nil {
		return evt, err
	}
	evt.Msg.Nonce = uint32(unumber)

	unumber, err = attributes.GetAsUint("message.sequence", 64)
	if err != nil {
		return evt, err
	}
	evt.Msg.Sequence = unumber

	snumber, err := attributes.GetAsInt("message.block_time", 64)
	if err != nil {
		return evt, err
	}
	evt.Msg.Timestamp = time.Unix(snumber, 0)

	str, err = attributes.GetAsString("message.message")
	if err != nil {
		return evt, err
	}
	evt.Msg.Payload, err = hex.DecodeString(str)
	if err != nil {
		return evt, fmt.Errorf("failed to parse message.message attribute %s: %w", str, err)
	}

	return evt, nil
}

// processIbcReceivePublishEvent takes an IBC event, maps it to a message publication and publishes it.
func (w *Watcher) processIbcReceivePublishEvent(txHash ethCommon.Hash, evt *ibcReceivePublishEvent, observationType string) {
	connectionID, err := w.getConnectionID(evt.ChannelID)
	if err != nil {
		w.logger.Error("failed to query IBC connectionID for channel", zap.String("ibcChannel", evt.ChannelID), zap.Error(err))
		connectionErrors.WithLabelValues("unexpected_ibc_channel_error").Inc()
		return
	}

	ce, exists := w.connectionMap[connectionID]
	if !exists {
		w.logger.Error("ignoring an event from an unexpected IBC connection", zap.String("ibcConnection", connectionID))
		connectionErrors.WithLabelValues("unexpected_ibc_channel_error").Inc()
		return
	}

	if evt.Msg.EmitterChain != ce.chainID {
		w.logger.Error(fmt.Sprintf("chain id mismatch in %s message", observationType),
			zap.String("ibcConnectionID", connectionID),
			zap.String("ibcChannelID", evt.ChannelID),
			zap.Uint16("expectedChainID", uint16(ce.chainID)),
			zap.Uint16("actualChainID", uint16(evt.Msg.EmitterChain)),
			zap.Stringer("txHash", txHash),
			zap.String("msgId", evt.Msg.MessageIDString()),
		)
		invalidChainIdMismatches.WithLabelValues(evt.ChannelID).Inc()
		return
	}

	w.logger.Info(fmt.Sprintf("%s message detected", observationType),
		zap.String("ChannelID", evt.ChannelID),
		zap.String("ConnectionID", ce.connectionID),
		zap.String("ChainName", ce.chainName),
		zap.Stringer("TxHash", evt.Msg.TxHash),
		zap.Stringer("EmitterChain", evt.Msg.EmitterChain),
		zap.Stringer("EmitterAddress", evt.Msg.EmitterAddress),
		zap.Uint64("Sequence", evt.Msg.Sequence),
		zap.Uint32("Nonce", evt.Msg.Nonce),
		zap.Stringer("Timestamp", evt.Msg.Timestamp),
		zap.Uint8("ConsistencyLevel", evt.Msg.ConsistencyLevel),
	)

	ce.msgC <- evt.Msg
	messagesConfirmed.WithLabelValues(ce.chainName).Inc()
}

/*
This query:
  `{"all_chain_connections":{}}` is `eyJhbGxfY2hhaW5fY29ubmVjdGlvbnMiOnt9fQ==`
  which becomes:
  http://localhost:1319/cosmwasm/wasm/v1/contract/wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj/smart/eyJhbGxfY2hhaW5fY29ubmVjdGlvbnMiOnt9fQ%3D%3D

Returns something like this:
{
	"data": {
		"chain_connections": [
			[
				"Y29ubmVjdGlvbi0w",
				18
			],
			[
				"Y29ubmVjdGlvbi00Mg==",
				22
			]
		]
	}
}

*/

type ibcAllChainConnectionsQueryResults struct {
	Data struct {
		ChainConnections [][]interface{} `json:"chain_connections"`
	}
}

var connectionIdMapQuery = url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(`{"all_chain_connections":{}}`)))

// queryConnectionIdMap queries the contract for the set of chain IDs available on IBC and their corresponding IBC connections.
func (w *Watcher) queryConnectionIdMap() (map[vaa.ChainID]string, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	query := fmt.Sprintf(`%s/cosmwasm/wasm/v1/contract/%s/smart/%s`, w.lcdUrl, w.contractAddress, connectionIdMapQuery)
	connResp, err := client.Get(query)
	if err != nil {
		w.logger.Error("channel map query failed", zap.String("query", query), zap.Error(err))
		return nil, fmt.Errorf("query failed: %w", err)
	}
	connBody, err := io.ReadAll(connResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	connResp.Body.Close()

	var result ibcAllChainConnectionsQueryResults
	err = json.Unmarshal(connBody, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %s, error: %w", string(connBody), err)
	}

	if len(result.Data.ChainConnections) == 0 {
		return nil, fmt.Errorf("query did not return any data")
	}

	connIdMap := make(map[vaa.ChainID]string)
	for idx, connData := range result.Data.ChainConnections {
		if len(connData) != 2 {
			return nil, fmt.Errorf("connection map entry %d contains %d items when it should contain exactly two, json: %s", idx, len(connData), string(connBody))
		}

		connectionIdBytes, err := base64.StdEncoding.DecodeString(connData[0].(string))
		if err != nil {
			return nil, fmt.Errorf("connection ID for entry %d is invalid base64: %s, err: %s", idx, connData[0], err)
		}

		connectionID := string(connectionIdBytes)
		chainID := vaa.ChainID(connData[1].(float64))
		connIdMap[chainID] = connectionID
		w.logger.Debug("IBC connection ID mapping", zap.String("connectionID", connectionID), zap.Uint16("chainID", uint16(chainID)))
	}

	return connIdMap, nil
}

// getConnectionID returns the IBC connection ID associated with the given IBC channel. It uses a cache to avoid constantly
// querying worm chain. This works because once an IBC channel is closed its ID will never be reused.
func (w *Watcher) getConnectionID(channelID string) (string, error) {
	w.connectionIdLock.Lock()
	defer w.connectionIdLock.Unlock()
	connectionID, exists := w.connectionIdMap[channelID]
	if exists {
		return connectionID, nil
	}

	connectionID, err := w.queryConnectionID(channelID)
	if err != nil {
		return connectionID, err
	}

	w.connectionIdMap[channelID] = connectionID
	return connectionID, nil
}

/*
This query:
  http://localhost:1319/ibc/core/channel/v1/channels/channel-0/ports/wasm.wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj

Returns something like this:
{
  "channel": {
    "state": "STATE_OPEN",
    "ordering": "ORDER_UNORDERED",
    "counterparty": {
      "port_id": "wasm.terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
      "channel_id": "channel-0"
    },
    "connection_hops": [
      "connection-0"
    ],
    "version": "ibc-wormhole-v1"
  },
  "proof": null,
  "proof_height": {
    "revision_number": "0",
    "revision_height": "358"
  }
}
*/

// ibcChannelQueryResults is used to parse the result from the IBC connection ID query.
type ibcChannelQueryResults struct {
	Channel struct {
		State          string
		ConnectionHops []string `json:"connection_hops"`
		Version        string
	}
}

// getConnectionID queries the contract on wormchain to map a channel ID to a connection ID.
func (w *Watcher) queryConnectionID(channelID string) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	query := fmt.Sprintf("%s/ibc/core/channel/v1/channels/%s/ports/wasm.%s", w.lcdUrl, channelID, w.contractAddress)
	connResp, err := client.Get(query)
	if err != nil {
		w.logger.Error("channel query failed", zap.String("query", query), zap.Error(err))
		return "", fmt.Errorf("query failed: %w", err)
	}
	connBody, err := io.ReadAll(connResp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	connResp.Body.Close()

	var result ibcChannelQueryResults
	err = json.Unmarshal(connBody, &result)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %s, error: %w", string(connBody), err)
	}

	if len(result.Channel.ConnectionHops) != 1 {
		return "", fmt.Errorf("response contained %d hops when it should return exactly one, json: %s", len(result.Channel.ConnectionHops), string(connBody))
	}

	w.logger.Info("queried connection ID", zap.String("channelID", channelID), zap.String("connectionID", result.Channel.ConnectionHops[0]))
	return result.Channel.ConnectionHops[0], nil
}
