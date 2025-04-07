package ibc

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
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

	// ChainConfigEntry defines the entry for a chain being monitored by IBC.
	ChainConfigEntry struct {
		ChainID  vaa.ChainID
		MsgC     chan<- *common.MessagePublication
		ObsvReqC <-chan *gossipv1.ObservationRequest
	}
)

// GetFeatures returns the IBC feature string that can be published in heartbeat messages.
func GetFeatures() string {
	featuresMutex.Lock()
	defer featuresMutex.Unlock()
	return features
}

// setFeatures is used internally to set the IBC feature flag.
func setFeatures(f string) {
	featuresMutex.Lock()
	defer featuresMutex.Unlock()
	features = f
}

var (
	// Chains defines the list of chains to be monitored by IBC. Add new chains here as necessary.
	Chains = []vaa.ChainID{
		vaa.ChainIDOsmosis,
		vaa.ChainIDSei,
		vaa.ChainIDCosmoshub,
		vaa.ChainIDEvmos,
		vaa.ChainIDKujira,
		vaa.ChainIDNeutron,
		vaa.ChainIDCelestia,
		vaa.ChainIDStargaze,
		vaa.ChainIDSeda,
		vaa.ChainIDDymension,
		vaa.ChainIDProvenance,
		vaa.ChainIDNoble,
	}

	// features is the feature string to be published in the gossip heartbeat messages. It will include all chains that are actually enabled on IBC.
	features = ""

	// featuresMutex is used to protect the features flag.
	featuresMutex sync.Mutex

	ibcErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ibc_errors_by_reason",
			Help: "Total number of errors on IBC",
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
		blockHeightUrl  string
		contractAddress string
		logger          *zap.Logger

		// chainMap defines the data associated with all connected / enabled chains.
		chainMap map[vaa.ChainID]*chainEntry

		// channelIdToChainIdMap provides a mapping from IBC channel ID to chain ID. Note that there can be multiple channels IDs for the same chain.
		channelIdToChainIdMap map[string]vaa.ChainID

		// channelIdToChainIdLock protects channelIdToChainIdMap.
		channelIdToChainIdLock sync.Mutex

		// baseFeatures is used to create the feature string. It is the list of chains enabled, without the wormhole version.
		baseFeatures string
	}

	// chainEntry defines the data associated with a chain.
	chainEntry struct {
		chainID   vaa.ChainID
		chainName string
		readiness readiness.Component
		msgC      chan<- *common.MessagePublication
		obsvReqC  <-chan *gossipv1.ObservationRequest
	}
)

// NewWatcher creates a new IBC contract watcher
func NewWatcher(
	wsUrl string,
	lcdUrl string,
	blockHeightUrl string,
	contractAddress string,
	chainConfig ChainConfig,
) *Watcher {
	feats := ""
	chainMap := make(map[vaa.ChainID]*chainEntry)
	for _, chainToMonitor := range chainConfig {
		_, exists := chainMap[chainToMonitor.ChainID]
		if exists {
			panic(fmt.Sprintf("detected duplicate chainID: %v", chainToMonitor.ChainID))
		}

		ce := &chainEntry{
			chainID:   chainToMonitor.ChainID,
			chainName: chainToMonitor.ChainID.String(),
			readiness: common.MustConvertChainIdToReadinessSyncing(chainToMonitor.ChainID),
			msgC:      chainToMonitor.MsgC,
			obsvReqC:  chainToMonitor.ObsvReqC,
		}

		chainMap[ce.chainID] = ce

		if feats == "" {
			feats = "ibc:"
		} else {
			feats += "|"
		}
		feats += ce.chainID.String()
	}

	setFeatures(feats)

	return &Watcher{
		wsUrl:                 wsUrl,
		lcdUrl:                lcdUrl,
		blockHeightUrl:        blockHeightUrl,
		contractAddress:       contractAddress,
		chainMap:              chainMap,
		channelIdToChainIdMap: make(map[string]vaa.ChainID),
		baseFeatures:          feats,
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

// abciInfoResults is used to parse the abci_info query response to get the wormhole version and block height.
type abciInfoResults struct {
	Result struct {
		Response struct {
			Version         string `json:"version"`
			LastBlockHeight string `json:"last_block_height"`
		}
	}
}

// Run is the runnable for monitoring the IBC contract on wormchain.
func (w *Watcher) Run(ctx context.Context) error {
	w.logger = supervisor.Logger(ctx)

	w.logger.Info("Starting watcher",
		zap.String("watcher_name", "ibc"),
		zap.String("wsUrl", w.wsUrl),
		zap.String("lcdUrl", w.lcdUrl),
		zap.String("blockHeightUrl", w.blockHeightUrl),
		zap.String("contractAddress", w.contractAddress),
	)

	errC := make(chan error)

	blockHeightUrl := w.blockHeightUrl
	if blockHeightUrl == "" {
		blockHeightUrl = convertWsUrlToHttpUrl(w.wsUrl)
	}
	blockHeightUrl = blockHeightUrl + "/abci_info"

	w.logger.Info("creating watcher",
		zap.String("wsUrl", w.wsUrl),
		zap.String("lcdUrl", w.lcdUrl),
		zap.String("blockHeightUrl", blockHeightUrl),
		zap.String("contract", w.contractAddress),
		zap.String("features", GetFeatures()),
	)

	for _, ce := range w.chainMap {
		w.logger.Info("will monitor chain over IBC", zap.String("chain", ce.chainName))
		p2p.DefaultRegistry.SetNetworkStats(ce.chainID, &gossipv1.Heartbeat_Network{ContractAddress: w.contractAddress})
	}

	// nolint:bodyclose // Misses the close below
	c, _, err := websocket.Dial(ctx, w.wsUrl, nil)
	if err != nil {
		ibcErrors.WithLabelValues("websocket_dial_error").Inc()
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
		ibcErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Wait for the success response.
	_, subResp, err := c.Read(ctx)
	if err != nil {
		ibcErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("failed to receive response to subscribe request: %w", err)
	}
	if strings.Contains(string(subResp), "error") {
		ibcErrors.WithLabelValues("websocket_subscription_error").Inc()
		return fmt.Errorf("failed to subscribe to events, response: %s", string(subResp))
	}

	// Start a routine to listen for messages from the contract.
	common.RunWithScissors(ctx, errC, "ibc_data_pump", func(ctx context.Context) error {
		return w.handleEvents(ctx, c)
	})

	// Start a routine to periodically query the wormchain block height.
	common.RunWithScissors(ctx, errC, "ibc_block_height", func(ctx context.Context) error {
		return w.handleQueryBlockHeight(ctx, blockHeightUrl)
	})

	// Start a routine for each chain to listen for observation requests.
	for _, ce := range w.chainMap {
		ce := ce // Avoid data race.
		common.RunWithScissors(ctx, errC, "ibc_objs_req", func(ctx context.Context) error {
			return w.handleObservationRequests(ctx, ce)
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
func (w *Watcher) handleEvents(ctx context.Context, c *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			_, message, err := c.Read(ctx)
			if err != nil {
				w.logger.Error("failed to read socket", zap.Error(err))
				ibcErrors.WithLabelValues("channel_read_error").Inc()
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
						if err := w.processIbcReceivePublishEvent(evt, "new"); err != nil {
							return fmt.Errorf("failed to process new IBC event: %w", err)
						}
					}
				} else {
					w.logger.Debug("ignoring uninteresting event", zap.String("eventType", eventType))
				}
			}
		}
	}
}

// convertWsUrlToHttpUrl takes a string like "ws://wormchain:26657/websocket" and converts it to "http://wormchain:26657". This is
// used to query for the abci_info. That query doesn't work on the LCD. We have to do it on the websocket port, using an http URL.
func convertWsUrlToHttpUrl(url string) string {
	url = strings.TrimPrefix(url, "ws://")
	url = strings.TrimPrefix(url, "wss://")
	url = strings.TrimSuffix(url, "/websocket")
	return "http://" + url
}

// handleQueryBlockHeight gets the latest block height from wormchain each interval and updates the status on all the connected chains.
func (w *Watcher) handleQueryBlockHeight(ctx context.Context, queryUrl string) error {
	t := time.NewTicker(5 * time.Second)
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			resp, err := client.Get(queryUrl) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
			if err != nil {
				return fmt.Errorf("failed to query latest block: %w", err)
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return fmt.Errorf("failed to read latest block body: %w", err)
			}

			var abciInfo abciInfoResults
			err = json.Unmarshal(body, &abciInfo)
			if err != nil {
				w.logger.Error("failed to unmarshal abci info", zap.String("json", string(body)), zap.Error(err))
				continue
			}

			blockHeight, err := strconv.ParseInt(abciInfo.Result.Response.LastBlockHeight, 10, 64)
			if err != nil {
				w.logger.Error("failed to parse block height", zap.String("blockHeight", abciInfo.Result.Response.LastBlockHeight), zap.Error(err))
				continue
			}

			w.logger.Debug("current block height", zap.Int64("height", blockHeight), zap.String("wormchainVersion", abciInfo.Result.Response.Version))

			for _, ce := range w.chainMap {
				currentSlotHeight.WithLabelValues(ce.chainName).Set(float64(blockHeight))
				p2p.DefaultRegistry.SetNetworkStats(ce.chainID, &gossipv1.Heartbeat_Network{
					Height:          blockHeight,
					ContractAddress: w.contractAddress,
				})

				readiness.SetReady(ce.readiness)
			}

			readiness.SetReady(common.ReadinessIBCSyncing)
			setFeatures(w.baseFeatures + ":" + abciInfo.Result.Response.Version)
		}
	}
}

// handleObservationRequests listens for observation requests for a single chain and processes them by reading the requested transaction
// from wormchain and publishing the associated message. This function is instantiated for each connected chain.
func (w *Watcher) handleObservationRequests(ctx context.Context, ce *chainEntry) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case r := <-ce.obsvReqC:
			// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
			// and only writes to the channel for this chain id.
			// If either of the below cases are true, something has gone wrong
			if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != ce.chainID {
				panic("invalid chain ID")
			}

			reqTxHashStr := hex.EncodeToString(r.TxHash)
			w.logger.Info("received observation request", zap.String("chain", ce.chainName), zap.String("txHash", reqTxHashStr))

			client := &http.Client{
				Timeout: time.Second * 5,
			}

			// Query for tx by hash.
			resp, err := client.Get(fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", w.lcdUrl, reqTxHashStr)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
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
						evt.Msg.IsReobservation = true
						if err := w.processIbcReceivePublishEvent(evt, "reobservation"); err != nil {
							return fmt.Errorf("failed to process reobserved IBC event: %w", err)
						}
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
	evt.Msg.TxID = txHash.Bytes()

	evt.ChannelID, err = attributes.GetAsString("channel_id")
	if err != nil {
		return evt, err
	}

	unumber, err := attributes.GetAsUint("message.chain_id", 16)
	if err != nil {
		return evt, err
	}
	evt.Msg.EmitterChain = vaa.ChainID(unumber) // #nosec G115 -- Already range checked by `GetAsUint`

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
	evt.Msg.Nonce = uint32(unumber) // #nosec G115 -- This conversion is safe based on GetAsUint usage above

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
func (w *Watcher) processIbcReceivePublishEvent(evt *ibcReceivePublishEvent, observationType string) error {
	mappedChainID, err := w.getChainIdFromChannelID(evt.ChannelID)
	if err != nil {
		w.logger.Error("query for IBC channel ID failed",
			zap.String("IbcChannelID", evt.ChannelID),
			zap.String("TxID", evt.Msg.TxIDString()),
			zap.Stringer("EmitterChain", evt.Msg.EmitterChain),
			zap.Stringer("EmitterAddress", evt.Msg.EmitterAddress),
			zap.Uint64("Sequence", evt.Msg.Sequence),
			zap.Uint32("Nonce", evt.Msg.Nonce),
			zap.Stringer("Timestamp", evt.Msg.Timestamp),
			zap.Uint8("ConsistencyLevel", evt.Msg.ConsistencyLevel),
			zap.Error(err),
		)
		ibcErrors.WithLabelValues("query_error").Inc()
		return fmt.Errorf("failed to query IBC channel ID mapping: %w", err)
	}

	if mappedChainID == vaa.ChainIDUnset {
		// This can happen if the channel ID to chain ID mapping in the contract hasn't been updated yet (pending governance).
		// Therefore we don't want to return an error here. Restarting won't help.
		w.logger.Error(fmt.Sprintf("received %s message from unknown IBC channel, dropping observation", observationType),
			zap.String("IbcChannelID", evt.ChannelID),
			zap.String("TxID", evt.Msg.TxIDString()),
			zap.Stringer("EmitterChain", evt.Msg.EmitterChain),
			zap.Stringer("EmitterAddress", evt.Msg.EmitterAddress),
			zap.Uint64("Sequence", evt.Msg.Sequence),
			zap.Uint32("Nonce", evt.Msg.Nonce),
			zap.Stringer("Timestamp", evt.Msg.Timestamp),
			zap.Uint8("ConsistencyLevel", evt.Msg.ConsistencyLevel),
		)
		ibcErrors.WithLabelValues("unexpected_ibc_channel_error").Inc()
		return nil
	}

	ce, exists := w.chainMap[mappedChainID]
	if !exists {
		// This is not an error because some guardians may choose to run the full node and not listen to this chain over IBC.
		w.logger.Debug(fmt.Sprintf("received %s message from an unconfigured chain, dropping observation", observationType),
			zap.String("IbcChannelID", evt.ChannelID),
			zap.Stringer("ChainID", mappedChainID),
			zap.String("TxID", evt.Msg.TxIDString()),
			zap.Stringer("EmitterChain", evt.Msg.EmitterChain),
			zap.Stringer("EmitterAddress", evt.Msg.EmitterAddress),
			zap.Uint64("Sequence", evt.Msg.Sequence),
			zap.Uint32("Nonce", evt.Msg.Nonce),
			zap.Stringer("Timestamp", evt.Msg.Timestamp),
			zap.Uint8("ConsistencyLevel", evt.Msg.ConsistencyLevel),
		)
		return nil
	}

	if evt.Msg.EmitterChain != ce.chainID {
		w.logger.Error(fmt.Sprintf("chain id mismatch in %s message", observationType),
			zap.String("IbcChannelID", evt.ChannelID),
			zap.Uint16("MappedChainID", uint16(mappedChainID)),
			zap.Uint16("ExpectedChainID", uint16(ce.chainID)),
			zap.String("TxID", evt.Msg.TxIDString()),
			zap.Stringer("EmitterChain", evt.Msg.EmitterChain),
			zap.Stringer("EmitterAddress", evt.Msg.EmitterAddress),
			zap.Uint64("Sequence", evt.Msg.Sequence),
			zap.Uint32("Nonce", evt.Msg.Nonce),
			zap.Stringer("Timestamp", evt.Msg.Timestamp),
			zap.Uint8("ConsistencyLevel", evt.Msg.ConsistencyLevel),
		)
		invalidChainIdMismatches.WithLabelValues(evt.ChannelID).Inc()
		return nil // Don't return an error here because we don't want an external source to be able to kill the watcher.
	}

	w.logger.Info(fmt.Sprintf("%s message detected", observationType),
		zap.String("IbcChannelID", evt.ChannelID),
		zap.String("ChainName", ce.chainName),
		zap.String("TxID", evt.Msg.TxIDString()),
		zap.Stringer("EmitterChain", evt.Msg.EmitterChain),
		zap.Stringer("EmitterAddress", evt.Msg.EmitterAddress),
		zap.Uint64("Sequence", evt.Msg.Sequence),
		zap.Uint32("Nonce", evt.Msg.Nonce),
		zap.Stringer("Timestamp", evt.Msg.Timestamp),
		zap.Uint8("ConsistencyLevel", evt.Msg.ConsistencyLevel),
	)

	ce.msgC <- evt.Msg
	messagesConfirmed.WithLabelValues(ce.chainName).Inc()
	if evt.Msg.IsReobservation {
		watchers.ReobservationsByChain.WithLabelValues(evt.Msg.EmitterChain.String(), "std").Inc()
	}
	return nil
}

// getChainIdFromChannelID returns the chain ID associated with the specified IBC channel. It uses a cache to avoid constantly querying
// wormchain. This works because once an IBC channel is closed its ID will never be reused. This also means that there could be multiple
// IBC channels for the same chain ID.
// See the IBC spec for details: https://github.com/cosmos/ibc/tree/main/spec/core/ics-004-channel-and-packet-semantics#closing-handshake
func (w *Watcher) getChainIdFromChannelID(channelID string) (vaa.ChainID, error) {
	w.channelIdToChainIdLock.Lock()
	defer w.channelIdToChainIdLock.Unlock()
	chainID, exists := w.channelIdToChainIdMap[channelID]
	if exists {
		return chainID, nil
	}

	// We continue to hold the lock here because we don't want two routines (event handler and reobservation) both querying at the same time.
	channelIdToChainIdMap, err := w.queryChannelIdToChainIdMapping()
	if err != nil {
		w.logger.Error("failed to query channelID to chainID mapping", zap.Error(err))
		return vaa.ChainIDUnset, err
	}

	w.channelIdToChainIdMap = channelIdToChainIdMap

	chainID, exists = w.channelIdToChainIdMap[channelID]
	if exists {
		return chainID, nil
	}

	return vaa.ChainIDUnset, nil
}

/*
This query:
'{"all_channel_chains": {}}' is `eyJhbGxfY2hhbm5lbF9jaGFpbnMiOiB7fX0=`

which becomes:
http://localhost:1319/cosmwasm/wasm/v1/contract/wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj/smart/eyJhbGxfY2hhbm5lbF9jaGFpbnMiOiB7fX0%3D

Returns something like this:
{
  "data": {
    "channels_chains": [
      [
        "Y2hhbm5lbC0w",
        18
      ]
    ]
  }
}

*/

type ibcAllChannelChainsQueryResults struct {
	Data struct {
		ChannelChains [][]interface{} `json:"channels_chains"`
	}
}

var allChannelChainsQuery = url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(`{"all_channel_chains": {}}`)))

// queryChannelIdToChainIdMapping queries the contract for the set of IBC channels and their correspond chain IDs.
func (w *Watcher) queryChannelIdToChainIdMapping() (map[string]vaa.ChainID, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	query := fmt.Sprintf(`%s/cosmwasm/wasm/v1/contract/%s/smart/%s`, w.lcdUrl, w.contractAddress, allChannelChainsQuery)
	resp, err := client.Get(query) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	resp.Body.Close()

	var result ibcAllChannelChainsQueryResults
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %s, error: %w", string(body), err)
	}

	if len(result.Data.ChannelChains) == 0 {
		return nil, fmt.Errorf("query did not return any data")
	}

	w.logger.Info("queried IBC channel ID mapping", zap.Int("numEntriesReturned", len(result.Data.ChannelChains)))

	ret := make(map[string]vaa.ChainID)
	for idx, entry := range result.Data.ChannelChains {
		if len(entry) != 2 {
			return nil, fmt.Errorf("channel map entry %d contains %d items when it should contain exactly two, json: %s", idx, len(entry), string(body))
		}

		channelIdBytes, err := base64.StdEncoding.DecodeString(entry[0].(string))
		if err != nil {
			return nil, fmt.Errorf("channel ID for entry %d is invalid base64: %s, err: %s", idx, entry[0], err)
		}

		channelID := string(channelIdBytes)
		chainIdFloat, ok := entry[1].(float64)
		if !ok {
			return nil, fmt.Errorf("error converting channelId to float64")
		}
		chainID := vaa.ChainID(chainIdFloat)
		ret[channelID] = chainID
		w.logger.Info("IBC channel ID mapping", zap.String("channelID", channelID), zap.Uint16("chainID", uint16(chainID)))
	}

	return ret, nil
}
