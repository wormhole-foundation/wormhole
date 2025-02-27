package sui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/mr-tron/base58"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Sui blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		suiRPC           string
		suiMoveEventType string

		unsafeDevMode bool

		msgChan                   chan<- *common.MessagePublication
		obsvReqC                  <-chan *gossipv1.ObservationRequest
		readinessSync             readiness.Component
		latestProcessedCheckpoint int64
		maximumBatchSize          int
		descendingOrder           bool
		loopDelay                 time.Duration
		queryEventsCmd            string
		postTimeout               time.Duration
	}

	SuiEventResponse struct {
		Jsonrpc string               `json:"jsonrpc"`
		Result  SuiEventResponseData `json:"result"`
		ID      int                  `json:"id"`
	}
	SuiEventResponseData struct {
		Data       []SuiResult `json:"data"`
		NextCursor struct {
			TxDigest string `json:"txDigest"`
			EventSeq string `json:"eventSeq"`
		} `json:"nextCursor"`
		HasNextPage bool `json:"hasNextPage"`
	}

	SuiResultInfo struct {
		result     SuiResult
		checkpoint int64
	}

	FieldsData struct {
		ConsistencyLevel *uint8  `json:"consistency_level"`
		Nonce            *uint64 `json:"nonce"`
		Payload          []byte  `json:"payload"`
		Sender           *string `json:"sender"`
		Sequence         *string `json:"sequence"`
		Timestamp        *string `json:"timestamp"`
	}

	SuiResult struct {
		ID struct {
			TxDigest *string `json:"txDigest"`
			EventSeq *string `json:"eventSeq"`
		} `json:"id"`
		PackageID         *string          `json:"packageId"`
		TransactionModule *string          `json:"transactionModule"`
		Sender            *string          `json:"sender"`
		Type              *string          `json:"type"`
		Bcs               *string          `json:"bcs"`
		Timestamp         *string          `json:"timestampMs"`
		Fields            *json.RawMessage `json:"parsedJson"`
	}

	SuiEventError struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}

	SuiEventMsg struct {
		Jsonrpc string         `json:"jsonrpc"`
		Method  *string        `json:"method"`
		ID      *int64         `json:"id"`
		Error   *SuiEventError `json:"error"`
		Params  *struct {
			Subscription int64      `json:"subscription"`
			Result       *SuiResult `json:"result"`
		} `json:"params"`
	}

	SuiTxnQueryError struct {
		Jsonrpc string `json:"jsonrpc"`
		Error   struct {
			Code    int     `json:"code"`
			Message *string `json:"message"`
		} `json:"error"`
		ID int `json:"id"`
	}

	SuiTxnQuery struct {
		Jsonrpc string      `json:"jsonrpc"`
		Result  []SuiResult `json:"result"`
		ID      int         `json:"id"`
	}

	SuiCheckpointSN struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  string `json:"result"`
		ID      int    `json:"id"`
	}

	GetCheckpointResponse struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  struct {
			Digest      string `json:"digest"`
			TimestampMs string `json:"timestampMs"`
			Checkpoint  string `json:"checkpoint"`
		} `json:"result"`
		ID int `json:"id"`
	}

	RequestPayload struct {
		JSONRPC string     `json:"jsonrpc"`
		ID      int        `json:"id"`
		Method  string     `json:"method"`
		Params  [][]string `json:"params"`
	}

	TxBlockResult struct {
		Digest      string `json:"digest"`
		TimestampMs string `json:"timestampMs"`
		Checkpoint  string `json:"checkpoint"`
	}

	MultipleBlockResult struct {
		Jsonrpc string          `json:"jsonrpc"`
		Result  []TxBlockResult `json:"result"`
	}
)

var (
	suiMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_sui_observations_confirmed_total",
			Help: "Total number of verified Sui observations found",
		})
	currentSuiHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_sui_current_height",
			Help: "Current Sui block height",
		})
)

// NewWatcher creates a new Sui appid watcher
func NewWatcher(
	suiRPC string,
	suiMoveEventType string,
	unsafeDevMode bool,
	messageEvents chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	maxBatchSize := 10
	descOrder := true
	return &Watcher{
		suiRPC:                    suiRPC,
		suiMoveEventType:          suiMoveEventType,
		unsafeDevMode:             unsafeDevMode,
		msgChan:                   messageEvents,
		obsvReqC:                  obsvReqC,
		readinessSync:             common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDSui),
		latestProcessedCheckpoint: 0,
		maximumBatchSize:          maxBatchSize,
		descendingOrder:           descOrder,   // Retrieve newest events first
		loopDelay:                 time.Second, // SUI produces a checkpoint every ~3 seconds
		queryEventsCmd: fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "suix_queryEvents", "params": [{ "MoveEventType": "%s" }, null, %d, %t]}`,
			suiMoveEventType, maxBatchSize, descOrder),
		postTimeout: time.Second * 5,
	}
}

func (e *Watcher) inspectBody(logger *zap.Logger, body SuiResult, isReobservation bool) error {
	if body.ID.TxDigest == nil {
		return errors.New("Missing TxDigest field")
	}
	if body.Type == nil {
		return errors.New("Missing Type field")
	}

	// There may be moveEvents caught without these params.
	// So, not necessarily an error.
	if body.Fields == nil {
		return nil
	}

	if e.suiMoveEventType != *body.Type {
		logger.Info("type mismatch", zap.String("e.suiMoveEventType", e.suiMoveEventType), zap.String("type", *body.Type))
		return nil
	}

	// Now that we know this is a wormhole event, we can unmarshal the specifics.
	var fields FieldsData
	err := json.Unmarshal(*body.Fields, &fields)
	if err != nil {
		logger.Error("failed to unmarshal FieldsData", zap.String("SuiResult.Fields", string(*body.Fields)), zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		return fmt.Errorf("inspectBody failed to unmarshal FieldsData: %w", err)
	}

	// Check if all required fields exist
	if (fields.ConsistencyLevel == nil) || (fields.Nonce == nil) || (fields.Payload == nil) || (fields.Sender == nil) || (fields.Sequence == nil) {
		logger.Info("Missing required fields in event.")
		return nil
	}

	emitter, err := vaa.StringToAddress(*fields.Sender)
	if err != nil {
		return err
	}

	txHashBytes, err := base58.Decode(*body.ID.TxDigest)
	if err != nil {
		return err
	}

	if len(txHashBytes) != 32 {
		logger.Error(
			"Transaction hash is not 32 bytes",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.String("txHash", *body.ID.TxDigest),
		)
		return errors.New("Transaction hash is not 32 bytes")
	}

	txHashEthFormat := eth_common.BytesToHash(txHashBytes)

	seq, err := strconv.ParseUint(*fields.Sequence, 10, 64)
	if err != nil {
		logger.Info("Sequence decode error", zap.String("Sequence", *fields.Sequence))
		return err
	}
	ts, err := strconv.ParseInt(*fields.Timestamp, 10, 64)
	if err != nil {
		logger.Info("Timestamp decode error", zap.String("Timestamp", *fields.Timestamp))
		return err
	}

	observation := &common.MessagePublication{
		TxID:             txHashEthFormat.Bytes(),
		Timestamp:        time.Unix(ts, 0),
		Nonce:            uint32(*fields.Nonce), // #nosec G115 -- Nonce is 32 bits on chain
		Sequence:         seq,
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   emitter,
		Payload:          fields.Payload,
		ConsistencyLevel: *fields.ConsistencyLevel,
		IsReobservation:  isReobservation,
	}

	suiMessagesConfirmed.Inc()
	if isReobservation {
		watchers.ReobservationsByChain.WithLabelValues("sui", "std").Inc()
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

	e.msgChan <- observation

	return nil
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSui, &gossipv1.Heartbeat_Network{
		ContractAddress: e.suiMoveEventType,
	})

	logger := supervisor.Logger(ctx)

	logger.Info("Starting watcher",
		zap.String("watcher_name", "sui"),
		zap.String("suiRPC", e.suiRPC),
		zap.String("suiMoveEventType", e.suiMoveEventType),
		zap.Bool("unsafeDevMode", e.unsafeDevMode),
	)

	// Get the latest checkpoint sequence number.  This will be the starting point for the watcher.
	latest, err := e.getLatestCheckpointSN(ctx, logger)
	if err != nil {
		return fmt.Errorf("failed to get latest checkpoint sequence number: %w", err)
	}
	e.latestProcessedCheckpoint = latest

	timer := time.NewTicker(time.Second * 5)
	defer timer.Stop()

	errC := make(chan error)
	pumpData := make(chan []byte)
	defer close(pumpData)

	supervisor.Signal(ctx, supervisor.SignalHealthy)
	readiness.SetReady(e.readinessSync)

	common.RunWithScissors(ctx, errC, "sui_data_pump", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_data_pump context done")
				return ctx.Err()

			default:
				dataWithEvents, err := e.getEvents(ctx)
				if err != nil {
					logger.Error("sui_data_pump Error", zap.Error(err))
					continue
				}
				// dataWithEvents is in descending order, so we need to process it in reverse order.
				if len(dataWithEvents) > 0 {
					for idx := len(dataWithEvents) - 1; idx >= 0; idx-- {
						event := dataWithEvents[idx]
						err = e.inspectBody(logger, event.result, false)
						if err != nil {
							logger.Error("inspectBody Error", zap.Error(err))
							continue
						}
						if event.checkpoint > e.latestProcessedCheckpoint {
							e.latestProcessedCheckpoint = event.checkpoint
						}
					}
				}
				time.Sleep(e.loopDelay)
			}
		}
	})

	common.RunWithScissors(ctx, errC, "sui_block_height", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_block_height context done")
				return ctx.Err()

			case <-timer.C:
				height, err := e.getLatestCheckpointSN(ctx, logger)
				if err != nil {
					logger.Error("Failed to get latest checkpoint sequence number", zap.Error(err))
				} else {
					currentSuiHeight.Set(float64(height))
					logger.Debug("sui_getLatestCheckpointSequenceNumber", zap.Int64("result", height))

					p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSui, &gossipv1.Heartbeat_Network{
						Height:          height,
						ContractAddress: e.suiMoveEventType,
					})
				}

				readiness.SetReady(e.readinessSync)
			}
		}
	})

	common.RunWithScissors(ctx, errC, "sui_fetch_obvs_req", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_fetch_obvs_req context done")
				return ctx.Err()
			case r := <-e.obsvReqC:
				// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
				// and only writes to the channel for this chain id.
				// If either of the below cases are true, something has gone wrong
				if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != vaa.ChainIDSui {
					panic("invalid chain ID")
				}

				tx58 := base58.Encode(r.TxHash)

				payload := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getEvents", "params": ["%s"]}`, tx58)

				body, err := e.createAndExecReq(ctx, payload)
				if err != nil {
					logger.Error("sui_fetch_obvs_req failed", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui_fetch_obvs_req failed to create and execute request: %w", err)
				}

				logger.Debug("receive", zap.String("body", string(body)))

				// Do we have an error?
				var err_res SuiTxnQueryError
				err = json.Unmarshal(body, &err_res)
				if err != nil {
					logger.Error("sui_fetch_obvs_req failed to unmarshal event error message", zap.String("Result", string(body)))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return err
				}

				if err_res.Error.Message != nil {
					logger.Error("sui_fetch_obvs_req failed to get events for re-observation request, detected error", zap.String("Result", string(body)))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					// Don't need to kill the watcher on this error. So, just continue.
					continue
				}
				var res SuiTxnQuery
				err = json.Unmarshal(body, &res)
				if err != nil {
					logger.Error("failed to unmarshal event message", zap.String("body", string(body)), zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui_fetch_obvs_req failed to unmarshal: %w", err)

				}

				for i, chunk := range res.Result {
					err := e.inspectBody(logger, chunk, true)
					if err != nil {
						logger.Info("sui_fetch_obvs_req skipping event data in result", zap.String("txhash", tx58), zap.Int("index", i), zap.Error(err))
					}
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

func (w *Watcher) getEvents(ctx context.Context) ([]SuiResultInfo, error) {
	// Only get events newer than the last processed height
	var retVal []SuiResultInfo
	var results []SuiResult
	var txs []string
	var nextCursor struct {
		TxDigest string
		EventSeq string
	}
	firstTime := true
	for {
		var payload string
		if firstTime {
			payload = w.queryEventsCmd
			firstTime = false
		} else {
			payload = fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "suix_queryEvents", "params": [{ "MoveEventType": "%s" }, { "txDigest": "%s", "eventSeq": "%s" }, %d, %t]}`,
				w.suiMoveEventType, nextCursor.TxDigest, nextCursor.EventSeq, w.maximumBatchSize, w.descendingOrder)
		}
		res, err := w.suiQueryEvents(ctx, payload)
		if err != nil {
			return retVal, err
		}
		for _, datum := range res.Result.Data {
			txs = append(txs, *datum.ID.TxDigest)
			results = append(results, datum)
		}
		if (len(res.Result.Data) == 0) || (len(txs) == 0) {
			// In devnet (tilt) the core contract may not have any events and we don't want to flood the logs.
			if w.unsafeDevMode {
				return retVal, nil
			}
			return retVal, errors.New("getEvents was unable to get any events")
		}
		// Get and check the checkpoint for the last event against the lastProcessedHeight to see if we are done.
		height, hErr := w.getCheckpointForDigest(ctx, txs[len(txs)-1])
		if hErr != nil {
			return retVal, hErr
		}
		if height <= w.latestProcessedCheckpoint || !res.Result.HasNextPage {
			break
		}
		nextCursor.TxDigest = res.Result.NextCursor.TxDigest
		nextCursor.EventSeq = res.Result.NextCursor.EventSeq
	}
	// At this point we have events but no checkpoints.
	// Also, we probably have more events than we need.
	// Need to do a bulk query to get all the checkpoints and then filter out the ones we don't need.
	mbRes, err := w.getMultipleBlocks(ctx, txs)
	if err != nil {
		return retVal, err
	}
	if (len(mbRes) == 0) || (len(mbRes) != len(txs)) {
		return retVal, errors.New("getEvents error getting multiple blocks")
	}
	for idx, block := range mbRes {
		cp, err := strconv.ParseInt(block.Checkpoint, 10, 64)
		if err != nil {
			return retVal, fmt.Errorf("getEvents failed to ParseInt: %w", err)
		}
		if cp > w.latestProcessedCheckpoint {
			// Double check the digest here
			if txs[idx] != block.Digest {
				return retVal, fmt.Errorf("getEvents digest mismatch: [%s] [%s]", txs[idx], block.Digest)
			}
			sri := SuiResultInfo{result: results[idx], checkpoint: cp}
			retVal = append(retVal, sri)
		} else {
			// We can break here because the blocks are in order.
			break
		}
	}

	return retVal, nil
}

func (w *Watcher) getMultipleBlocks(ctx context.Context, txs []string) ([]TxBlockResult, error) {
	retVal := []TxBlockResult{}
	payload := RequestPayload{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "sui_multiGetTransactionBlocks",
		Params:  [][]string{txs},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return retVal, fmt.Errorf("getMultipleBlocks failed to marshal payload: %w", err)
	}

	body, err := w.createAndExecReq(ctx, string(payloadBytes))
	if err != nil {
		return retVal, fmt.Errorf("getMultipleBlocks failed to create and execute request: %w", err)
	}

	var res MultipleBlockResult
	err = json.Unmarshal(body, &res)
	if err != nil {
		return retVal, fmt.Errorf("getMultipleBlocks failed to unmarshal body: %s, error: %w", string(body), err)
	}
	retVal = res.Result

	return retVal, nil
}

func (e *Watcher) getLatestCheckpointSN(ctx context.Context, logger *zap.Logger) (int64, error) {
	payload := `{"jsonrpc":"2.0", "id": 1, "method": "sui_getLatestCheckpointSequenceNumber", "params": []}`

	body, err := e.createAndExecReq(ctx, payload)
	if err != nil {
		logger.Error("sui_getLatestCheckpointSequenceNumber failed", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		return 0, fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to create and execute request: %w", err)
	}

	var res SuiCheckpointSN
	err = json.Unmarshal(body, &res)
	if err != nil {
		logger.Error("unmarshal failed into uint64", zap.String("body", string(body)), zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		return 0, fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to unmarshal body: %s, error: %w", string(body), err)
	}

	height, pErr := strconv.ParseInt(res.Result, 0, 64)
	if pErr != nil {
		logger.Error("Failed to ParseInt")
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		return 0, fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to ParseInt, error: %w", err)
	}
	return height, nil
}

func (e *Watcher) getCheckpointForDigest(ctx context.Context, tx string) (int64, error) {
	retVal := int64(0)
	payload := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getTransactionBlock", "params": [ "%s" ]}`, tx)

	body, err := e.createAndExecReq(ctx, payload)
	if err != nil {
		return retVal, fmt.Errorf("getCheckpointForDigest failed to create and execute request: %w", err)
	}

	var res GetCheckpointResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return retVal, fmt.Errorf("getCheckpointForDigest failed to unmarshal body: %s, error: %w", string(body), err)
	}
	retVal, err = strconv.ParseInt(res.Result.Checkpoint, 10, 64)
	if err != nil {
		return retVal, fmt.Errorf("getCheckpointForDigest failed to ParseInt: %w", err)
	}

	return retVal, nil
}

func (w *Watcher) suiQueryEvents(ctx context.Context, payload string) (SuiEventResponse, error) {
	retVal := SuiEventResponse{}

	body, err := w.createAndExecReq(ctx, payload)
	if err != nil {
		return retVal, fmt.Errorf("suix_queryEvents failed to create and execute request: %w", err)
	}

	err = json.Unmarshal(body, &retVal)
	if err != nil {
		return retVal, fmt.Errorf("suix_queryEvents failed to unmarshal body: %s, error: %w", string(body), err)
	}
	return retVal, nil
}

func (w *Watcher) createAndExecReq(ctx context.Context, payload string) ([]byte, error) {
	var retVal []byte
	timeoutCtx, cancel := context.WithTimeout(ctx, w.postTimeout)
	defer cancel()
	// Create a new request with the context
	req, err := http.NewRequestWithContext(timeoutCtx, "POST", w.suiRPC, strings.NewReader(payload))
	if err != nil {
		return retVal, fmt.Errorf("createAndExecReq failed to create request: %w, payload: %s", err, payload)
	}

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Send the request using DefaultClient
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return retVal, fmt.Errorf("createAndExecReq failed to post: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return retVal, fmt.Errorf("createAndExecReq failed to read: %w", err)
	}
	resp.Body.Close()
	return body, nil
}
