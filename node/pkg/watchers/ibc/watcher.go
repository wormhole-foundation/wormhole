package ibc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
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
	// ConnectionConfig defines the list of chains to be monitored by IBC, along with their IBC connection ID.
	ConnectionConfig []ConnectionConfigEntry

	// ConnectionConfigEntry defines the entry for an IBC connection. Note that the json of this is used to set the -ibcConfig
	// parameter in the json config, so be careful about changing this.
	ConnectionConfigEntry struct {
		// These are specified as json in the config.
		ChainID vaa.ChainID
		ConnID  string
	}

	// ChannelData defines the message channels associated with the corresponding entry in ConnectionConfig. It is in lock step with ConnectionConfig.
	ChannelData []ChannelDataEntry

	// ChannelDataEntry defines the message channels associated with the corresponding entry in ConnectionConfig.
	ChannelDataEntry struct {
		ChainID  vaa.ChainID
		MsgC     chan<- *common.MessagePublication
		ObsvReqC <-chan *gossipv1.ObservationRequest
	}
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

		// connectionConfig is the list of chains to be monitored, along with their IBC connection IDs.
		connectionConfig ConnectionConfig

		// channelData is kept in lock sync with connectionConfig and provides the internal message channels associated with each monitored chain.
		channelData ChannelData

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

// ParseConfig parses the --ibcConfig parameter into a vector of configured chains. It also returns the feature string to be published in heartbeats.
func ParseConfig(ibcConfig string) (ConnectionConfig, string, error) {
	connections := make([]ConnectionConfigEntry, 0)
	features := ""

	if ibcConfig == "" {
		// This is not an error if IBC is not enabled.
		return connections, features, nil
	}

	// The config string is json formatted like this: `[{"ChainID":18,"ConnID":"connection-0"},{"ChainID":19,"ConnID":"connection-1"}]`
	err := json.Unmarshal([]byte(ibcConfig), &connections)
	if err != nil {
		return connections, features, fmt.Errorf("failed to parse IBC config string: %s, error: %w", ibcConfig, err)
	}

	// Build the feature string.
	for _, ch := range connections {
		if features == "" {
			features = "ibc:"
		} else {
			features += ","
		}
		features += fmt.Sprintf("%s:%s", ch.ChainID.String(), ch.ConnID)
	}

	return connections, features, nil
}

// NewWatcher creates a new IBC contract watcher
func NewWatcher(
	wsUrl string,
	lcdUrl string,
	contractAddress string,
	ConnectionConfig ConnectionConfig,
	channelData ChannelData,
) *Watcher {
	return &Watcher{
		wsUrl:            wsUrl,
		lcdUrl:           lcdUrl,
		contractAddress:  contractAddress,
		connectionConfig: ConnectionConfig,
		channelData:      channelData,
		connectionIdMap:  make(map[string]string),
		connectionMap:    make(map[string]*connectionEntry),
		chainMap:         make(map[vaa.ChainID]*connectionEntry),
	}
}

// ConvertUrlToTendermint takes a URL and does the following conversions if necessary:
// - Converts "ws://" to "http:".
// - Strips "/websocket" off the end.
func ConvertUrlToTendermint(input string) (string, error) {
	input = strings.TrimPrefix(input, "ws://")
	input = strings.TrimPrefix(input, "http://")
	input = strings.TrimSuffix(input, "/websocket")
	return "http://" + input, nil
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
	ChannelID      string
	EmitterChain   vaa.ChainID
	EmitterAddress vaa.Address
	Nonce          uint32
	Sequence       uint64
	Timestamp      time.Time
	Payload        []byte
}

// Run is the runnable for monitoring the IBC contract on wormchain.
func (w *Watcher) Run(ctx context.Context) error {
	w.logger = supervisor.Logger(ctx)

	errC := make(chan error)
	defer close(errC)

	// Build our internal data structures based on the config passed in.
	if len(w.connectionMap) == 0 {
		for idx, chainToMonitor := range w.connectionConfig {
			if w.channelData[idx].ChainID != chainToMonitor.ChainID {
				panic("channelData is not in sync with chainToMonitor") // This would be a program bug!
			}

			_, exists := w.connectionMap[chainToMonitor.ConnID]
			if exists {
				return fmt.Errorf("detected duplicate IBC connection: %v", chainToMonitor.ConnID)
			}

			_, exists = w.chainMap[chainToMonitor.ChainID]
			if exists {
				return fmt.Errorf("detected duplicate chainID: %v", chainToMonitor.ChainID)
			}

			ce := &connectionEntry{
				connectionID: chainToMonitor.ConnID,
				chainID:      chainToMonitor.ChainID,
				chainName:    vaa.ChainID(chainToMonitor.ChainID).String(),
				readiness:    common.MustConvertChainIdToReadinessSyncing(chainToMonitor.ChainID),
				msgC:         w.channelData[idx].MsgC,
				obsvReqC:     w.channelData[idx].ObsvReqC,
			}

			w.logger.Info("will monitor chain over IBC", zap.String("chain", ce.chainName), zap.String("IBC connection", ce.connectionID))
			w.connectionMap[ce.connectionID] = ce
			w.chainMap[ce.chainID] = ce

			p2p.DefaultRegistry.SetNetworkStats(ce.chainID, &gossipv1.Heartbeat_Network{ContractAddress: w.contractAddress})
		}
	}

	w.logger.Info("creating watcher", zap.String("wsUrl", w.wsUrl), zap.String("lcdUrl", w.lcdUrl), zap.String("contract", w.contractAddress))

	c, _, err := websocket.Dial(ctx, w.wsUrl, nil)
	if err != nil {
		connectionErrors.WithLabelValues("websocket_dial_error").Inc()
		return fmt.Errorf("failed to establish tendermint websocket connection: %w", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	// This was copied from the cosmwasm watcher. It sure it's still necessary. . .
	c.SetReadLimit(524288)

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
		default:
			_, message, err := c.Read(ctx)
			if err != nil {
				w.logger.Error("failed to read socket", zap.Error(err))
				connectionErrors.WithLabelValues("channel_read_error").Inc()
				errC <- fmt.Errorf("failed to read socket: %w", err)
				return nil
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
					evt, err := parseIbcReceivePublishEvent(w.logger, w.contractAddress, event)
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
					w.logger.Warn("event is invalid", zap.String("chain", ce.chainName), zap.String("tx_hash", txHashStr), zap.String("event", event.String()))
					continue
				}
				eventType := gjson.Get(event.String(), "type")
				if eventType.String() == "wasm" {
					w.logger.Debug("found wasm event in reobservation", zap.String("chain", ce.chainName), zap.Stringer("txHash", txHash))
					evt, err := parseIbcReceivePublishEvent(w.logger, w.contractAddress, event)
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

// parseIbcReceivePublishEvent parses a wasm event. If it is from the desired contract and for the desired action, it returns an event. Otherwise, it returns nil.
func parseIbcReceivePublishEvent(logger *zap.Logger, desiredContract string, event gjson.Result) (*ibcReceivePublishEvent, error) {
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

	evt.ChannelID, err = attributes.GetAsString("channel_id")
	if err != nil {
		return evt, err
	}

	unumber, err := attributes.GetAsUint64("message.chain_id", 16)
	if err != nil {
		return evt, err
	}
	evt.EmitterChain = vaa.ChainID(unumber)

	str, err = attributes.GetAsString("message.sender")
	if err != nil {
		return evt, err
	}
	evt.EmitterAddress, err = vaa.StringToAddress(str)
	if err != nil {
		return evt, fmt.Errorf("failed to parse message.sender attribute %s: %w", str, err)
	}

	unumber, err = attributes.GetAsUint64("message.nonce", 32)
	if err != nil {
		return evt, err
	}
	evt.Nonce = uint32(unumber)

	unumber, err = attributes.GetAsUint64("message.sequence", 64)
	if err != nil {
		return evt, err
	}
	evt.Sequence = unumber

	snumber, err := attributes.GetAsInt64("message.block_time", 64)
	if err != nil {
		return evt, err
	}
	evt.Timestamp = time.Unix(snumber, 0)

	str, err = attributes.GetAsString("message.message")
	if err != nil {
		return evt, err
	}
	evt.Payload, err = hex.DecodeString(str)
	if err != nil {
		return evt, fmt.Errorf("failed to parse message.message attribute %s: %w", str, err)
	}

	return evt, nil
}

// processIbcReceivePublishEvent takes an IBC event, maps it to a message publication and publishes it.
func (w *Watcher) processIbcReceivePublishEvent(txHash ethCommon.Hash, evt *ibcReceivePublishEvent, observationType string) {
	connectionID, err := w.getConnectionID(evt.ChannelID)
	if err != nil {
		w.logger.Info("failed to query IBC connectionID for channel", zap.String("ibcChannel", evt.ChannelID), zap.Error(err))
		connectionErrors.WithLabelValues("unexpected_ibc_channel_error").Inc()
		return
	}

	ce, exists := w.connectionMap[connectionID]
	if !exists {
		w.logger.Info("ignoring an event from an unexpected IBC connection", zap.String("ibcConnection", connectionID))
		connectionErrors.WithLabelValues("unexpected_ibc_channel_error").Inc()
		return
	}

	if evt.EmitterChain != ce.chainID {
		w.logger.Error(fmt.Sprintf("chain id mismatch in %s message", observationType),
			zap.String("ibcConnectionID", connectionID),
			zap.String("ibcChannelID", evt.ChannelID),
			zap.Uint16("expectedChainID", uint16(ce.chainID)),
			zap.Uint16("actualChainID", uint16(evt.EmitterChain)),
			zap.Stringer("txHash", txHash),
			zap.String("msgId", evt.msgId()),
		)
		invalidChainIdMismatches.WithLabelValues(evt.ChannelID).Inc()
		return
	}

	msg := &common.MessagePublication{
		TxHash:           txHash,
		Timestamp:        evt.Timestamp,
		Nonce:            evt.Nonce,
		Sequence:         evt.Sequence,
		EmitterChain:     evt.EmitterChain,
		EmitterAddress:   evt.EmitterAddress,
		Payload:          evt.Payload,
		ConsistencyLevel: 0,
	}

	w.logger.Info(fmt.Sprintf("%s message detected", observationType),
		zap.String("ChannelID", evt.ChannelID),
		zap.String("ConnectionID", ce.connectionID),
		zap.String("ChainName", ce.chainName),
		zap.Stringer("TxHash", msg.TxHash),
		zap.Stringer("EmitterChain", msg.EmitterChain),
		zap.Stringer("EmitterAddress", msg.EmitterAddress),
		zap.Uint64("Sequence", msg.Sequence),
		zap.Uint32("Nonce", msg.Nonce),
		zap.Stringer("Timestamp", msg.Timestamp),
		zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
	)

	ce.msgC <- msg
	messagesConfirmed.WithLabelValues(ce.chainName).Inc()
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

// ibcChannelQueryResults is used to parse the result from the IBC connection ID query.
type ibcChannelQueryResults struct {
	Channel struct {
		State          string
		ConnectionHops []string `json:"connection_hops"`
		Version        string
	}
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

// getConnectionID queries the contract on wormchain to map a channel ID to a connection ID.
func (w *Watcher) queryConnectionID(channelID string) (string, error) {
	// TODO: cache the channelID -> connectionID mapping. This should be safe because once a channel is closed, that number can never be reused.
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

// msgId generates a message ID string for the specified IBC receive publish event.
func (e ibcReceivePublishEvent) msgId() string {
	return fmt.Sprintf("%v/%v/%v", uint16(e.EmitterChain), hex.EncodeToString(e.EmitterAddress[:]), e.Sequence)
}
