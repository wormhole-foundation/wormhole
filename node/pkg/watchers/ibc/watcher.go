// ToDos:
// - Publish wormchain block height on all connected chains.
// - Use Tendermint to query for block height and send observation requests.

package ibc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	tmAbci "github.com/tendermint/tendermint/abci/types"
	tmHttp "github.com/tendermint/tendermint/rpc/client/http"
	tmCoreTypes "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"

	ethCommon "github.com/ethereum/go-ethereum/common"

	// This should go away!
	"go.uber.org/zap"
)

type (
	// ChannelConfig defines the list of chains to be monitored by IBC, along with their IBC channel.
	ChannelConfig []ChannelConfigEntry

	// ChannelConfigEntry defines the entry for an IBC channel. Note that the json of this is used to set the -ibcConfig
	// parameter in the json config, so be careful about changing this.
	ChannelConfigEntry struct {
		// These are specified as json in the config.
		ChainID   vaa.ChainID
		ChannelID string

		// These are filled in before the watcher is instantiated.
		MsgC     chan<- *common.MessagePublication   `json:"-"`
		ObsvReqC <-chan *gossipv1.ObservationRequest `json:"-"`
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
			Help: "Total number of cases where the wormhole chain ID does not match the IBC channel ID",
		}, []string{"ibc_channel_id"})
)

type (
	// Watcher is responsible for monitoring the IBC contract on wormchain and publishing wormhole messages for all chains connected via IBC.
	Watcher struct {
		wsUrl           string
		lcdUrl          string
		contractAddress string
		channelConfig   ChannelConfig
		logger          *zap.Logger

		channelMap map[string]*channelEntry
		chainMap   map[vaa.ChainID]*channelEntry

		wsConn  *tmHttp.HTTP
		lcdConn *tmHttp.HTTP
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
)

// ParseConfig parses the --ibcConfig parameter into a vector of configured chains. It also returns the feature string to be published in heartbeats.
func ParseConfig(ibcConfig string) (ChannelConfig, string, error) {
	channels := make([]ChannelConfigEntry, 0)
	features := ""

	if ibcConfig == "" {
		// This is not an error if IBC is not enabled.
		return channels, features, nil
	}

	// The config string is json formatted like this: `[{"ChainID":18,"ChannelID":"channel-0"},{"ChainID":19,"ChannelID":"channel-1"}]`
	err := json.Unmarshal([]byte(ibcConfig), &channels)
	if err != nil {
		return channels, features, fmt.Errorf("failed to parse IBC config string: %s, error: %w", ibcConfig, err)
	}

	// Build the feature string.
	for _, ch := range channels {
		if features == "" {
			features = "ibc:"
		} else {
			features += ","
		}
		features += fmt.Sprintf("%s:%s", ch.ChainID.String(), ch.ChannelID)
	}

	return channels, features, nil
}

// NewWatcher creates a new IBC contract watcher
func NewWatcher(
	wsUrl string,
	lcdUrl string,
	contractAddress string,
	channelConfig ChannelConfig,
) *Watcher {
	return &Watcher{
		wsUrl:           wsUrl,
		lcdUrl:          lcdUrl,
		contractAddress: contractAddress,
		channelConfig:   channelConfig,

		channelMap: make(map[string]*channelEntry),
		chainMap:   make(map[vaa.ChainID]*channelEntry),
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

// Run is the runnable for monitoring the IBC contract on wormchain.
func (w *Watcher) Run(ctx context.Context) error {
	w.logger = supervisor.Logger(ctx)

	errC := make(chan error)
	defer close(errC)

	for _, chainToMonitor := range w.channelConfig {
		ce := &channelEntry{
			ibcChannelID: chainToMonitor.ChannelID,
			chainID:      chainToMonitor.ChainID,
			chainName:    vaa.ChainID(chainToMonitor.ChainID).String(),
			readiness:    common.MustConvertChainIdToReadinessSyncing(chainToMonitor.ChainID),
			msgC:         chainToMonitor.MsgC,
			obsvReqC:     chainToMonitor.ObsvReqC,
		}

		_, exists := w.channelMap[ce.ibcChannelID]
		if exists {
			return fmt.Errorf("detected duplicate ibc channel: %v", ce.ibcChannelID)
		}

		_, exists = w.chainMap[ce.chainID]
		if exists {
			return fmt.Errorf("detected duplicate chainID: %v", ce.chainID)
		}

		w.logger.Info("ibc: will monitor chain over IBC", zap.String("chain", ce.chainName), zap.String("IBC channel", ce.ibcChannelID))
		w.channelMap[ce.ibcChannelID] = ce
		w.chainMap[ce.chainID] = ce

		p2p.DefaultRegistry.SetNetworkStats(ce.chainID, &gossipv1.Heartbeat_Network{ContractAddress: w.contractAddress})
	}

	wsUrl, err := ConvertUrlToTendermint(w.wsUrl)
	if err != nil {
		return fmt.Errorf("failed to parse websocket url: %s, error: %w", w.wsUrl, err)
	}

	lcdUrl, err := ConvertUrlToTendermint(w.lcdUrl)
	if err != nil {
		return fmt.Errorf("failed to parse lcd url: %s, error: %w", w.lcdUrl, err)
	}

	w.logger.Info("ibc: creating watcher",
		zap.String("wsUrl", wsUrl), zap.String("origWsUrl", w.wsUrl),
		zap.String("contract", w.contractAddress),
		zap.String("lcdUrl", lcdUrl), zap.String("origLcdUrl", w.lcdUrl),
		zap.String("contract", w.contractAddress),
	)
	w.wsConn, err = tmHttp.New(wsUrl, "/websocket")
	if err != nil {
		connectionErrors.WithLabelValues("websocket_dial_error").Inc()
		return fmt.Errorf("failed to establish tendermint websocket connection: %w", err)
	}

	if err := w.wsConn.Start(); err != nil {
		connectionErrors.WithLabelValues("websocket_start_error").Inc()
		return fmt.Errorf("failed to start tendermint websocket connection: %w", err)
	}
	defer func() {
		if err := w.wsConn.Stop(); err != nil {
			connectionErrors.WithLabelValues("websocket_stop_error").Inc()
			w.logger.Error("ibc: failed to stop tendermint websocket connection", zap.Error(err))
		}
	}()

	w.lcdConn, err = tmHttp.New(lcdUrl, "/http")
	if err != nil {
		connectionErrors.WithLabelValues("lcd_dial_error").Inc()
		return fmt.Errorf("failed to establish tendermint lcd connection: %w", err)
	}
	defer func() {
		if err := w.lcdConn.Stop(); err != nil {
			connectionErrors.WithLabelValues("lcd_stop_error").Inc()
			w.logger.Error("ibc: failed to stop tendermint lcd connection", zap.Error(err))
		}
	}()

	query := fmt.Sprintf("wasm._contract_address='%s'", w.contractAddress)
	w.logger.Info("ibc: subscribing to events", zap.String("query", query))
	events, err := w.wsConn.Subscribe(
		ctx,
		"guardiand",
		query,
		64, // channel capacity
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to accountant events: %w", err)
	}
	defer func() {
		if err := w.wsConn.UnsubscribeAll(ctx, "guardiand"); err != nil {
			w.logger.Error("ibc: failed to unsubscribe from events", zap.Error(err))
		}
	}()

	// Start a single routine to listen for messages from the contract and periodically query the wormchain block height.
	common.RunWithScissors(ctx, errC, "ibc_data_pump", func(ctx context.Context) error {
		return w.handleEvents(ctx, events, errC)
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

// handleEvents handles events from the tendermint client library.
func (w *Watcher) handleEvents(ctx context.Context, evts <-chan tmCoreTypes.ResultEvent, errC chan error) error {
	blockHeightTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-evts:
			tx, ok := e.Data.(tmTypes.EventDataTx)
			if !ok {
				w.logger.Error("ibc: unknown data from event subscription", zap.Stringer("e.Data", reflect.TypeOf(e.Data)), zap.Any("event", e))
				connectionErrors.WithLabelValues("read_error").Inc()
				continue
			}

			// var junk tmTypes.Tx
			// junk = tx.TxResult.Tx
			// txHash := junk.Hash()
			// txHash := tmTypes.Tx(tx.TxResult.Tx).Hash()
			// h := ethCommon.BytesToHash(txHash)
			txHash := ethCommon.BytesToHash(tmTypes.Tx(tx.TxResult.Tx).Hash())
			w.logger.Info("BigBOINK", zap.Stringer("txHash", txHash), zap.Any("tx", tx))

			for _, event := range tx.Result.Events {
				w.logger.Info("BOINK", zap.String("type", event.Type), zap.Any("event", event))
				if event.Type == "wasm" {
					evt, err := parseEvent[ibcReceivePublishEvent](w.logger, w.contractAddress, "receive_publish", event)
					if err != nil {
						w.logger.Error("ibc: failed to parse wasm event", zap.Error(err), zap.Stringer("e.Data", reflect.TypeOf(e.Data)), zap.Any("event", event))
						continue
					}

					w.processEvent(txHash, evt)
				} else {
					w.logger.Debug("ibc: ignoring uninteresting event", zap.String("eventType", event.Type))
				}
			}
		case <-blockHeightTicker.C:
			if err := w.queryBlockHeight(ctx); err != nil {
				connectionErrors.WithLabelValues("blockHeight_error").Inc()
				errC <- err
				return nil
			}
		}
	}
}

// queryBlockHeight gets the latest block height from wormchain and updates the status on all the updated chains.
func (w *Watcher) queryBlockHeight(ctx context.Context) error {
	abciInfo, err := w.wsConn.ABCIInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to query block height: %w", err)
	}

	latestBlockAsInt := abciInfo.Response.LastBlockHeight

	/* This fails with: "unknown query path: unknown request","info":"","index":"0","key":null,"value":null,"proofOps":null,"height":"0","codespace":"sdk"
	params := []byte{}
	path := "/cosmos/base/tendermint/v1beta1/blocks/latest"
	// path := "L2Nvc21vcy9iYXNlL3RlbmRlcm1pbnQvdjFiZXRhMS9ibG9ja3MvbGF0ZXN0"
	queryResp, err := w.wsConn.ABCIQuery(ctx, path, params)
	if err != nil {
		w.logger.Error("ibc: query latest block response error", zap.String("path", path), zap.Error(err))
	} else {
		w.logger.Info("ibc: queried block height", zap.String("path", path), zap.Any("queryResp", queryResp))
	}
	/* This fails with: ibc: query latest block response error	{"error": "error unmarshalling: invalid character '<' looking for beginning of value"}
	params := []byte{}
	resp, err := w.lcdConn.ABCIQuery(ctx, "blocks/latest", params)
	if err != nil {
		w.logger.Error("ibc: query latest block response error", zap.Error(err))
		return nil
	}
	w.logger.Info("ibc: query block height", zap.Any("resp", resp))
	*/

	/*********************************** This works.
	latestBlockURL := "blocks/latest"

	// ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := client.Get(fmt.Sprintf("%s/%s", w.lcdUrl, latestBlockURL))
	if err != nil {
		logger.Error("ibc: query latest block response error", zap.Error(err))
		return nil
	}
	blocksBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("ibc: query latest block response read error", zap.Error(err))
		resp.Body.Close()
		return nil
	}
	resp.Body.Close()

	blockJSON := string(blocksBody)
	latestBlockAsInt := gjson.Get(blockJSON, "block.header.height").Int()
	*/

	latestBlockAsFloat := float64(latestBlockAsInt)
	w.logger.Info("ibc: current block height", zap.Int64("height", latestBlockAsInt))

	for _, ce := range w.chainMap {
		currentSlotHeight.WithLabelValues(ce.chainName).Set(latestBlockAsFloat)
		p2p.DefaultRegistry.SetNetworkStats(ce.chainID, &gossipv1.Heartbeat_Network{
			Height:          latestBlockAsInt,
			ContractAddress: w.contractAddress,
		})

		readiness.SetReady(ce.readiness)
	}

	return nil
}

// handleObservationRequests listens for observation requests for a single chain and processes them by reading the requested transaction
// from wormchain and publishing the associated message.
func (w *Watcher) handleObservationRequests(ctx context.Context, errC chan error, ce *channelEntry) error {
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

			// Query for tx by hash
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
				w.logger.Info("BOINK", zap.String("type", eventType.String()), zap.Any("event", event))
				if eventType.String() != "wasm" {
					w.logger.Debug("ibc: found wasm event in reobservation", zap.String("chain", ce.chainName), zap.Stringer("txHash", txHash))
					// evt, err := parseEvent[ibcReceivePublishEvent](w.logger, w.contractAddress, "receive_publish", event)
					// if err != nil {
					// 	w.logger.Error("ibc: failed to parse wasm event", zap.String("chain", ce.chainName), zap.Error(err), zap.Any("event", event))
					// 	continue
					// }

					// w.processEvent(txHash, evt)
				} else {
					w.logger.Debug("ibc: ignoring uninteresting event in reobservation", zap.String("chain", ce.chainName), zap.Stringer("txHash", txHash), zap.String("eventType", eventType.String()))
				}
			}
		}
	}
}

// processEvent takes an IBC event, maps it to a message publication and publishes it.
func (w *Watcher) processEvent(txHash ethCommon.Hash, evt *ibcReceivePublishEvent) {
	ce, exists := w.channelMap[evt.ChannelID]
	if !exists {
		w.logger.Info("ignoring an event from an unexpected IBC channel", zap.String("ibcChannel", evt.ChannelID))
		connectionErrors.WithLabelValues("unexpected_ibc_channel_error").Inc()
		return
	}

	if evt.EmitterChain != ce.chainID {
		w.logger.Error("chain id mismatch in IBC message",
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

	w.logger.Info("ibc: new message detected",
		zap.String("ChannelID", ce.ibcChannelID),
		zap.String("ChainName", ce.chainName),
		zap.Stringer("TxHash", msg.TxHash),
		zap.Stringer("EmitterChain", msg.EmitterChain),
		zap.Stringer("EmitterAddress", msg.EmitterAddress),
		zap.Uint64("Sequence", msg.Sequence),
		zap.Uint32("Sequence", msg.Nonce),
		zap.Stringer("Timestamp", msg.Timestamp),
		zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
	)

	ce.msgC <- msg
	messagesConfirmed.WithLabelValues(ce.chainName).Inc()
}

// parseEvent parses a wasm event. If it is from the desired contract and for the desired action, it returns an event. Otherwise, it returns nil.
func parseEvent[T any](logger *zap.Logger, desiredContract string, desiredAction string, event tmAbci.Event) (*T, error) {
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
func parseWasmAttributes(logger *zap.Logger, desiredContract string, desiredAction string, event tmAbci.Event) ([]byte, error) {
	contractAddressSeen := false
	actionSeen := false
	attrs := make(map[string]string)
	for _, attr := range event.Attributes {
		key := string(attr.Key)
		value := string(attr.Value)
		if key == "_contract_address" {
			contractAddressSeen = true
			if desiredContract != "" && value != desiredContract {
				logger.Debug("ibc: ignoring event from an unexpected contract", zap.String("contract", value), zap.String("desiredContract", desiredContract))
				return nil, nil
			}
		} else if key == "action" {
			actionSeen = true
			if desiredAction != "" && value != desiredAction {
				logger.Debug("ibc: ignoring event with an unexpected action", zap.String("key", key), zap.String("value", value), zap.String("desiredAction", desiredAction))
				return nil, nil
			}
		} else {
			if _, ok := attrs[key]; ok {
				logger.Debug("ibc: duplicate key in events", zap.String("key", key), zap.String("value", value))
				continue
			}

			logger.Debug("ibc: event attribute", zap.String("key", key), zap.String("value", value), zap.String("desiredAction", desiredAction))
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

// The IBC receiver contract represents all the fields as strings so define our own unmarshal method to convert the attributes into the real data types.
func (evt *ibcReceivePublishEvent) UnmarshalJSON(data []byte) error {
	raw := &struct {
		ChannelID      string `json:"channel_id"`
		EmitterChain   string `json:"message.chain_id"`
		EmitterAddress string `json:"message.sender"`
		Nonce          string `json:"message.nonce"`
		Sequence       string `json:"message.sequence"`
		Timestamp      string `json:"message.block_time"`
		Payload        string `json:"message.message"`
	}{}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	evt.ChannelID = raw.ChannelID

	val, err := strconv.ParseUint(raw.EmitterChain, 10, 16)
	if err != nil {
		return fmt.Errorf("failed to unmarshal EmitterChain: %s, error: %w", raw.EmitterChain, err)
	}
	evt.EmitterChain = vaa.ChainID(val)

	evt.EmitterAddress, err = vaa.StringToAddress(raw.EmitterAddress)
	if err != nil {
		return fmt.Errorf("failed to unmarshal EmitterAddress: %s, error: %w", raw.EmitterAddress, err)
	}

	val, err = strconv.ParseUint(raw.Nonce, 10, 32)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Nonce: %s, error: %w", raw.Nonce, err)
	}
	evt.Nonce = uint32(val)

	val, err = strconv.ParseUint(raw.Sequence, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Sequence: %s, error: %w", raw.Sequence, err)
	}
	evt.Sequence = uint64(val)

	val, err = strconv.ParseUint(raw.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Timestamp: %s, error: %w", raw.Timestamp, err)
	}
	evt.Timestamp = time.Unix(int64(val), 0)

	evt.Payload, err = hex.DecodeString(raw.Payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Payload: %s, error: %w", raw.Payload, err)
	}

	return nil
}

// msgId generates a message ID string for the specified IBC receive publish event.
func (e ibcReceivePublishEvent) msgId() string {
	return fmt.Sprintf("%v/%v/%v", uint16(e.EmitterChain), hex.EncodeToString(e.EmitterAddress[:]), e.Sequence)
}
