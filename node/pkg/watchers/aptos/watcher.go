package aptos

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Aptos blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		chainID   vaa.ChainID
		networkID string

		aptosRPC          string
		aptosAccount      string
		aptosHandle       string
		aptosIndexerRPC   string
		aptosIndexerToken string
		useIndexer        bool

		msgC          chan<- *common.MessagePublication
		obsvReqC      <-chan *gossipv1.ObservationRequest
		readinessSync readiness.Component
	}
)

var (
	aptosMessagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_aptos_observations_confirmed_total",
			Help: "Total number of verified observations found for the chain",
		}, []string{"chain_name"})

	currentAptosHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_aptos_current_height",
			Help: "Current block height for the chain",
		}, []string{"chain_name"})
)

// NewWatcher creates a new Aptos appid watcher
func NewWatcher(
	chainID vaa.ChainID,
	networkID watchers.NetworkID,
	aptosRPC string,
	aptosAccount string,
	aptosHandle string,
	aptosIndexerRPC string,
	aptosIndexerToken string,
	useIndexer bool,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		chainID:           chainID,
		networkID:         string(networkID),
		aptosRPC:          aptosRPC,
		aptosAccount:      aptosAccount,
		aptosHandle:       aptosHandle,
		aptosIndexerRPC:   aptosIndexerRPC,
		aptosIndexerToken: aptosIndexerToken,
		useIndexer:        useIndexer,
		msgC:              msgC,
		obsvReqC:          obsvReqC,
		readinessSync:     common.MustConvertChainIdToReadinessSyncing(chainID),
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: e.aptosAccount,
	})

	logger := supervisor.Logger(ctx)

	if e.useIndexer {
		logger.Info("Starting Aptos watcher in indexer mode",
			zap.String("watcher_name", e.networkID),
			zap.String("rpc", e.aptosRPC),
			zap.String("account", e.aptosAccount),
			zap.String("handle", e.aptosHandle),
			zap.String("indexerRpc", e.aptosIndexerRPC),
		)
	} else {
		logger.Info("Starting Aptos watcher in legacy mode",
			zap.String("watcher_name", e.networkID),
			zap.String("rpc", e.aptosRPC),
			zap.String("account", e.aptosAccount),
			zap.String("handle", e.aptosHandle),
		)
	}

	// SECURITY: the API guarantees that we only get the events from the right
	// contract

	// Important: There are 3 different sequence/version numbers in play:
	// 1. Aptos Version: Transaction version number (like block number in EVM)
	// 2. Aptos Sequence Number: Event-specific counter for WormholeMessage events
	// 3. Wormhole Sequence: Protocol sequence inside the event data (goes into VAA)
	//

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	if e.useIndexer {
		return e.runIndexerMode(ctx, logger)
	} else {
		return e.runLegacyMode(ctx, logger)
	}
}

func (e *Watcher) runLegacyMode(ctx context.Context, logger *zap.Logger) error {
	// Legacy REST API implementation from original code
	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	// nextSequence tracks the Aptos Sequence Number (not version, not wormhole seq)
	var nextSequence uint64 = 0
	var eventsEndpoint = fmt.Sprintf(`%s/v1/accounts/%s/events/%s/event`, e.aptosRPC, e.aptosAccount, e.aptosHandle)
	var aptosHealth = fmt.Sprintf(`%s/v1`, e.aptosRPC)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-e.obsvReqC:
			// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
			// and only writes to the channel for this chain id.
			// If either of the below cases are true, something has gone wrong
			if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != e.chainID {
				panic("invalid chain ID")
			}

			// uint64 will read the *first* 8 bytes, but the sequence is stored in the *last* 8.
			nativeSeq := binary.BigEndian.Uint64(r.TxHash[24:])

			logger.Info("Received obsv request", zap.Uint64("tx_hash", nativeSeq))

			s := fmt.Sprintf(`%s?start=%d&limit=1`, eventsEndpoint, nativeSeq)

			body, err := e.retrievePayload(s)
			if err != nil {
				logger.Error("retrievePayload", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			if !gjson.Valid(string(body)) {
				logger.Error("InvalidJson: " + string(body))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				break

			}

			outcomes := gjson.ParseBytes(body)

			for _, chunk := range outcomes.Array() {
				newSeq := chunk.Get("sequence_number")
				if !newSeq.Exists() {
					break
				}

				if newSeq.Uint() != nativeSeq {
					logger.Error("newSeq != nativeSeq")
					break

				}

				data := chunk.Get("data")
				if !data.Exists() {
					break
				}
				e.observeData(logger, data, nativeSeq, true)
			}

		case <-timer.C:
			s := ""

			if nextSequence == 0 {
				// if nextSequence is 0, we look up the most recent event
				s = fmt.Sprintf(`%s?limit=1`, eventsEndpoint)
			} else {
				// otherwise just look up events starting at nextSequence.
				// this will potentially return multiple events (whatever
				// the default limit is per page), so we'll handle all of them.
				s = fmt.Sprintf(`%s?start=%d`, eventsEndpoint, nextSequence)
			}

			eventsJson, err := e.retrievePayload(s)
			if err != nil {
				logger.Error("retrievePayload", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			// data doesn't exist yet. skip, and try again later
			// this happens when the sequence id we're looking up hasn't
			// been used yet.
			if string(eventsJson) == "" {
				continue
			}

			if !gjson.Valid(string(eventsJson)) {
				logger.Error("InvalidJson: " + string(eventsJson))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue

			}

			events := gjson.ParseBytes(eventsJson)

			// the endpoint returns an array of events, ordered by sequence
			// id (ASC)

			// Check if the array is empty. It will be empty most of the time until there are new events.
			if !events.Exists() || !events.IsArray() || len(events.Array()) == 0 {
				logger.Debug("No new events found")
			} else {
				for _, event := range events.Array() {
					eventSequence := event.Get("sequence_number")
					if !eventSequence.Exists() {
						continue
					}

					eventSeq := eventSequence.Uint()
					if nextSequence == 0 && eventSeq != 0 {
						// Avoid publishing an old observation on startup. This does not block the first message on a new chain (when eventSeq would be zero).
						nextSequence = eventSeq + 1
						continue
					}

					// this is interesting in the last iteration, whereby we
					// find the next sequence that comes after the array
					nextSequence = eventSeq + 1

					data := event.Get("data")
					if !data.Exists() {
						continue
					}
					e.observeData(logger, data, eventSeq, false)
				}
			}

			health, err := e.retrievePayload(aptosHealth)
			if err != nil {
				logger.Error("health", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			if !gjson.Valid(string(health)) {
				logger.Error("Invalid JSON in health response: " + string(health))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue

			}

			// TODO: Make this log more useful for humans
			logger.Debug(string(health) + string(eventsJson))

			pHealth := gjson.ParseBytes(health)

			blockHeight := pHealth.Get("block_height")

			if blockHeight.Uint() > math.MaxInt64 {
				logger.Error("Block height not a valid uint64: ", zap.Uint64("blockHeight", blockHeight.Uint()))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue
			}

			if blockHeight.Exists() {
				currentAptosHeight.WithLabelValues(e.networkID).Set(float64(blockHeight.Uint()))
				p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
					Height:          int64(blockHeight.Uint()), // #nosec G115 -- This is validated above
					ContractAddress: e.aptosAccount,
				})

				readiness.SetReady(e.readinessSync)
			}
		}
	}
}

func (e *Watcher) runIndexerMode(ctx context.Context, logger *zap.Logger) error {
	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	// nextSequence tracks the Aptos Sequence Number (not version, not wormhole seq)
	var nextSequence uint64 = 0

	// Add authorization header for indexer API calls, if token is present
	headers := make(map[string]string)
	if e.aptosIndexerToken != "" {
		headers["Authorization"] = "Bearer " + e.aptosIndexerToken
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-e.obsvReqC:
			// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
			// and only writes to the channel for this chain id.
			// If either of the below cases are true, something has gone wrong
			if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != e.chainID {
				panic("invalid chain ID")
			}

			// uint64 will read the *first* 8 bytes, but the sequence is stored in the *last* 8.
			nativeSeq := binary.BigEndian.Uint64(r.TxHash[24:])

			logger.Info("Received obsv request", zap.Uint64("tx_hash", nativeSeq))

			query := fmt.Sprintf(`query GetVersionBySequence { msg(where: {sequence_num: {_eq: %d}}) { version } }`, nativeSeq)

			body, err := e.queryIndexer(query, headers)
			if err != nil {
				logger.Error("retrievePayload", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			if !gjson.Valid(string(body)) {
				logger.Error("InvalidJson: " + string(body))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			// Parse the GraphQL response
			response := gjson.ParseBytes(body)

			// Extract the msg array from data.msg
			messages := response.Get("data.msg")
			// NOTE: Without this check, the code below panics if there are no results
			if !messages.Exists() || len(messages.Array()) == 0 {
				logger.Error("No data.msg field in indexer response or empty array")
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			// Get the version from the first (and should be only) result
			// Check if there is more than one result?
			if len(messages.Array()) > 1 {
				logger.Warn("More than one result found for sequence in indexer response", zap.Uint64("sequence", nativeSeq), zap.Int("count", len(messages.Array())))
			}
			// Get the version from the first result
			version := messages.Array()[0].Get("version")
			if !version.Exists() {
				logger.Error("No version field in indexer response")
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			versionNum := version.Uint()

			// Process the transaction and extract WormholeMessage events
			// isReobservation=true
			if err := e.processTransactionVersion(logger, versionNum, true); err != nil {
				logger.Error("Failed to process transaction for reobservation",
					zap.Uint64("version", versionNum),
					zap.Uint64("sequence", nativeSeq),
					zap.Error(err))
				continue
			}

		case <-timer.C:
			query := ""

			if nextSequence == 0 {
				// if nextSequence is 0, we look up the most recent event
				// Get both version (for fetching tx) and sequence_num (for observeData)
				query = "query GetLastEvent { msg(order_by: {sequence_num: desc}, limit: 1) { version sequence_num } }"
			} else {
				// otherwise just look up events starting at nextSequence.
				// this will potentially return multiple events (whatever
				// the default limit is per page), so we'll handle all of them.
				query = fmt.Sprintf(`query GetNextEvents { msg(where: {sequence_num: {_gt: %d}}, order_by: {sequence_num: asc}) { version sequence_num } }`, nextSequence)
			}

			eventsJson, err := e.queryIndexer(query, headers)
			if err != nil {
				logger.Error("queryIndexer", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			// data doesn't exist yet. skip, and try again later
			// this happens when the sequence id we're looking up hasn't
			// been used yet.
			if string(eventsJson) == "" {
				continue
			}

			if !gjson.Valid(string(eventsJson)) {
				logger.Error("InvalidJson: " + string(eventsJson))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			// Parse the GraphQL response
			response := gjson.ParseBytes(eventsJson)

			// Extract the msg array from data.msg
			messages := response.Get("data.msg")
			if !messages.Exists() {
				logger.Error("No data.msg field in indexer response")
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}
			// Check if the array is empty
			if len(messages.Array()) == 0 {
				logger.Debug("No new events found in indexer response")
			} else {
				// Walk through the array of events
				for _, msg := range messages.Array() {
					version := msg.Get("version")
					sequenceNum := msg.Get("sequence_num")
					if !version.Exists() || !sequenceNum.Exists() {
						continue
					}

					versionNum := version.Uint()
					aptosSeqNum := sequenceNum.Uint()
					logger.Debug("Found event from indexer",
						zap.Uint64("version", versionNum),
						zap.Uint64("sequence_num", aptosSeqNum))

					if nextSequence == 0 && aptosSeqNum != 0 {
						// Avoid publishing an old observation on startup. This does not block the first message on a new chain (when eventSeq would be zero).
						nextSequence = aptosSeqNum + 1
						continue
					}

					// Process the transaction and extract WormholeMessage events
					// isReobservation=false
					if err := e.processTransactionVersion(logger, versionNum, false); err != nil {
						logger.Error("Failed to process transaction",
							zap.Uint64("version", versionNum),
							zap.Uint64("sequence", aptosSeqNum),
							zap.Error(err))
						continue
					}

					// Update nextSequence to track progress using the Aptos sequence_num
					if aptosSeqNum > nextSequence {
						nextSequence = aptosSeqNum
					}
				}
			}

			// Health check endpoint
			aptosHealth := fmt.Sprintf(`%s/v1`, e.aptosRPC)
			health, err := e.retrievePayload(aptosHealth)
			if err != nil {
				logger.Error("health", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			if !gjson.Valid(string(health)) {
				logger.Error("Invalid JSON in health response: " + string(health))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue

			}

			// TODO: Make this log more useful for humans
			logger.Debug(string(health) + string(eventsJson))

			pHealth := gjson.ParseBytes(health)

			blockHeight := pHealth.Get("block_height")

			if blockHeight.Uint() > math.MaxInt64 {
				logger.Error("Block height not a valid uint64: ", zap.Uint64("blockHeight", blockHeight.Uint()))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue
			}

			if blockHeight.Exists() {
				currentAptosHeight.WithLabelValues(e.networkID).Set(float64(blockHeight.Uint()))
				p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
					Height:          int64(blockHeight.Uint()), // #nosec G115 -- This is validated above
					ContractAddress: e.aptosAccount,
				})

				readiness.SetReady(e.readinessSync)
			}
		}
	}
}

//nolint:noctx // TODO: this function should use a context. (Also this line flags nolintlint unless placed here.)
func (e *Watcher) retrievePayload(s string) ([]byte, error) {
	//nolint:gosec // the URL is hard-coded to the Aptos RPC endpoint.
	res, err := http.Get(s)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := common.SafeRead(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

//nolint:noctx // TODO: this function should use a context.
func (e *Watcher) queryIndexer(query string, headers map[string]string) ([]byte, error) {
	// Create GraphQL request body
	requestBody := map[string]interface{}{
		"query": query,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", e.aptosIndexerRPC, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	// Set content type for GraphQL
	req.Header.Set("Content-Type", "application/json")

	// Add additional headers if provided
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// The indexer doesn't return errors in the HTTP status code.
	// If there is no "data" field in the response, treat it as an error.
	// The "data" field may, also, be empty, which may or may not be an error,
	// depending on the query.
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := common.SafeRead(res.Body)
	if err != nil {
		return nil, err
	}

	// Check for GraphQL errors in the response
	// The indexer may return errors with a 200 status code
	if gjson.Valid(string(body)) {
		response := gjson.ParseBytes(body)
		errors := response.Get("errors")
		if errors.Exists() && errors.IsArray() && len(errors.Array()) > 0 {
			// Get the first error message
			firstError := errors.Array()[0]
			message := firstError.Get("message")
			if message.Exists() {
				return nil, fmt.Errorf("indexer error: %s", message.String())
			}
			return nil, fmt.Errorf("indexer error: %s", errors.String())
		}
	}

	return body, nil
}

// processTransactionVersion fetches a transaction by version and extracts WormholeMessage events
func (e *Watcher) processTransactionVersion(
	logger *zap.Logger,
	versionNum uint64,
	isReobservation bool,
) error {
	// Fetch transaction details for this version
	var txEndpoint = fmt.Sprintf(`%s/v1/transactions/by_version/%d`, e.aptosRPC, versionNum)
	txData, err := e.retrievePayload(txEndpoint)
	if err != nil {
		logger.Error("retrievePayload", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		return err
	}

	if !gjson.Valid(string(txData)) {
		logger.Error("Invalid JSON in transaction response", zap.String("data", string(txData)))
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		return fmt.Errorf("invalid JSON in transaction response")
	}

	// Parse the transaction data
	txResult := gjson.ParseBytes(txData)

	// Look for WormholeMessage in the events array
	whEventType := fmt.Sprintf("%s::state::WormholeMessage", e.aptosAccount)
	events := txResult.Get("events")
	if !events.Exists() || !events.IsArray() || len(events.Array()) == 0 {
		logger.Warn("No events found in transaction", zap.Uint64("version", versionNum))
	} else {
		for _, event := range events.Array() {
			eventType := event.Get("type")
			if eventType.String() == whEventType {
				// Get the Aptos sequence number from this specific event
				eventSeq := event.Get("sequence_number")
				if !eventSeq.Exists() {
					logger.Error("WormholeMessage event missing sequence_number field",
						zap.Uint64("version", versionNum))
					continue
				}

				// Extract the event data
				data := event.Get("data")
				if data.Exists() {
					// The event data has the fields we need for observeData
					e.observeData(logger, data, eventSeq.Uint(), isReobservation)
				} else {
					logger.Error("WormholeMessage event missing data field", zap.Uint64("version", versionNum))
				}
			}
		}
	}
	return nil
}

func (e *Watcher) observeData(logger *zap.Logger, data gjson.Result, nativeSeq uint64, isReobservation bool) {
	em := data.Get("sender")
	if !em.Exists() {
		logger.Error("sender field missing")
		return
	}

	emitter := make([]byte, 8)
	binary.BigEndian.PutUint64(emitter, em.Uint())

	var a vaa.Address
	copy(a[24:], emitter)

	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, nativeSeq)

	var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

	v := data.Get("payload")
	if !v.Exists() {
		logger.Error("payload field missing")
		return
	}

	pl, err := hex.DecodeString(v.String()[2:])
	if err != nil {
		logger.Error("payload decode")
		return
	}

	ts := data.Get("timestamp")
	if !ts.Exists() {
		logger.Error("timestamp field missing")
		return
	}

	nonce := data.Get("nonce")
	if !nonce.Exists() {
		logger.Error("nonce field missing")
		return
	}

	sequence := data.Get("sequence")
	if !sequence.Exists() {
		logger.Error("sequence field missing")
		return
	}

	consistencyLevel := data.Get("consistency_level")
	if !consistencyLevel.Exists() {
		logger.Error("consistencyLevel field missing")
		return
	}

	if nonce.Uint() > math.MaxUint32 {
		logger.Error("nonce is larger than expected MaxUint32")
		return
	}

	if consistencyLevel.Uint() > math.MaxUint8 {
		logger.Error("consistency level is larger than expected MaxUint8")
		return
	}

	observation := &common.MessagePublication{
		TxID:             txHash.Bytes(),
		Timestamp:        time.Unix(int64(ts.Uint()), 0), // #nosec G115 -- This conversion is safe indefinitely
		Nonce:            uint32(nonce.Uint()),           // #nosec G115 -- This is validated above
		Sequence:         sequence.Uint(),
		EmitterChain:     e.chainID,
		EmitterAddress:   a,
		Payload:          pl,
		ConsistencyLevel: uint8(consistencyLevel.Uint()), // #nosec G115 -- This is validated above
		IsReobservation:  isReobservation,
	}

	aptosMessagesConfirmed.WithLabelValues(e.networkID).Inc()
	if isReobservation {
		watchers.ReobservationsByChain.WithLabelValues(e.chainID.String(), "std").Inc()
	}

	logger.Info("message observed",
		zap.String("txHash", observation.TxIDString()),
		zap.Time("timestamp", observation.Timestamp),
		zap.Uint32("nonce", observation.Nonce),
		zap.Uint64("sequence", observation.Sequence),
		zap.Stringer("emitter_chain", observation.EmitterChain),
		zap.Stringer("emitter_address", observation.EmitterAddress),
		zap.Binary("payload", observation.Payload),
		zap.Uint8("consistencyLevel", observation.ConsistencyLevel),
	)

	e.msgC <- observation //nolint:channelcheck // The channel to the processor is buffered and shared across chains, if it backs up we should stop processing new observations
}
