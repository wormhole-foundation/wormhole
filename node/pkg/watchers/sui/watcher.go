package sui

import (
	"context"
	"errors"
	"fmt"
	"io"
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

		msgChan       chan<- *common.MessagePublication
		obsvReqC      <-chan *gossipv1.ObservationRequest
		readinessSync readiness.Component

		subId                     int64
		latestProcessedCheckpoint int64
		maximumBatchSize          int
		descendingOrder           bool
		loopDelayInSecs           int
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
	return &Watcher{
		suiRPC:                    suiRPC,
		suiMoveEventType:          suiMoveEventType,
		unsafeDevMode:             unsafeDevMode,
		msgChan:                   messageEvents,
		obsvReqC:                  obsvReqC,
		readinessSync:             common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDSui),
		subId:                     0,
		latestProcessedCheckpoint: 0,
		maximumBatchSize:          10,
		descendingOrder:           true, // Retrieve newest events first
		loopDelayInSecs:           3,    // SUI produces a checkpoint every ~3 seconds
	}
}

func (e *Watcher) inspectBody(logger *zap.Logger, body SuiResult, isReobservation bool) error {
	if body.ID.TxDigest == nil {
		return errors.New("missing TxDigest field")
	}
	if body.Type == nil {
		return errors.New("missing Type field")
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
		return errors.New("transaction hash is not 32 bytes")
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
		TxHash:           txHashEthFormat,
		Timestamp:        time.Unix(ts, 0),
		Nonce:            uint32(*fields.Nonce),
		Sequence:         seq,
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   emitter,
		Payload:          fields.Payload,
		ConsistencyLevel: *fields.ConsistencyLevel,
		IsReobservation:  isReobservation,
	}

	suiMessagesConfirmed.Inc()

	logger.Info("message observed",
		zap.Stringer("txHash", observation.TxHash),
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

	// Get the latest checkpoint sequence number.  This will be the starting point for the watcher.
	latest, err := e.getLatestCheckpointSN(logger)
	if err != nil {
		return fmt.Errorf("failed to get latest checkpoint sequence number: %w", err)
	}
	e.latestProcessedCheckpoint = latest

	logger.Info("Starting watcher",
		zap.String("watcher_name", "sui"),
		zap.String("suiRPC", e.suiRPC),
		zap.String("suiMoveEventType", e.suiMoveEventType),
		zap.Bool("unsafeDevMode", e.unsafeDevMode),
	)

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
				fmt.Println("Getting events...", time.Now().Format(time.RFC3339))
				// This will return an array of events in the correct range and order
				event, err := e.getEvents()
				if err != nil {
					logger.Error(fmt.Sprintf("sui_data_pump Error: %s", err.Error()))
					continue
				}
				for _, datum := range event.Data {
					err = e.inspectBody(logger, datum, false)
					if err != nil {
						logger.Error(fmt.Sprintf("inspectBody Error: %s", err.Error()))
						continue
					}
					// Get the checkpoint for the event
					fmt.Println("Getting checkpoint for returned event")
					lph, err := e.getCheckpointForEvent(datum)
					if err != nil {
						fmt.Println("Error:", err)
						continue
					}
					if lph > e.latestProcessedCheckpoint {
						e.latestProcessedCheckpoint = lph
					}
				}
				time.Sleep(time.Duration(e.loopDelayInSecs) * time.Second)
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
				resp, err := http.Post(e.suiRPC, "application/json", strings.NewReader(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getLatestCheckpointSequenceNumber", "params": []}`)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
				if err != nil {
					logger.Error("sui_getLatestCheckpointSequenceNumber failed", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to post: %w", err)
				}
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error("sui_getLatestCheckpointSequenceNumber failed", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to read: %w", err)
				}
				resp.Body.Close()
				logger.Debug("Body before unmarshalling", zap.String("body", string(body)))

				var res SuiCheckpointSN
				err = json.Unmarshal(body, &res)
				if err != nil {
					logger.Error("unmarshal failed into uint64", zap.String("body", string(body)), zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to unmarshal body: %s, error: %w", string(body), err)
				}

				height, pErr := strconv.ParseInt(res.Result, 0, 64)
				if pErr != nil {
					logger.Error("Failed to ParseInt")
				} else {
					currentSuiHeight.Set(float64(height))
					logger.Debug("sui_getLatestCheckpointSequenceNumber", zap.String("result", res.Result))

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
				if vaa.ChainID(r.ChainId) != vaa.ChainIDSui {
					panic("invalid chain ID")
				}

				tx58 := base58.Encode(r.TxHash)

				buf := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getEvents", "params": ["%s"]}`, tx58)

				resp, err := http.Post(e.suiRPC, "application/json", strings.NewReader(buf)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
				if err != nil {
					logger.Error("getEvents API failed", zap.String("suiRPC", e.suiRPC), zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					continue
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error("unexpected truncated body when calling getEvents", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui__fetch_obvs_req failed to post: %w", err)

				}
				resp.Body.Close()

				logger.Debug("receive", zap.String("body", string(body)))

				// Do we have an error?
				var err_res SuiTxnQueryError
				err = json.Unmarshal(body, &err_res)
				if err != nil {
					logger.Error("Failed to unmarshal event error message", zap.String("Result", string(body)))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return err
				}

				if err_res.Error.Message != nil {
					logger.Error("Failed to get events for re-observation request, detected error", zap.String("Result", string(body)))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					// Don't need to kill the watcher on this error. So, just continue.
					continue
				}
				var res SuiTxnQuery
				err = json.Unmarshal(body, &res)
				if err != nil {
					logger.Error("failed to unmarshal event message", zap.String("body", string(body)), zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
					return fmt.Errorf("sui__fetch_obvs_req failed to unmarshal: %w", err)

				}

				for i, chunk := range res.Result {
					err := e.inspectBody(logger, chunk, true)
					if err != nil {
						logger.Info("skipping event data in result", zap.String("txhash", tx58), zap.Int("index", i), zap.Error(err))
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

func (w *Watcher) getEvents() (SuiEventResponseData, error) {
	// Only get events newer than the last processed height
	var retVal SuiEventResponseData
	var nextCursor struct {
		TxDigest string
		EventSeq string
	}
	firstTime := true
	done := false
	for !done {
		var reader io.Reader
		if firstTime {
			reader = strings.NewReader(
				fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "suix_queryEvents", "params": [{ "MoveEventType": "%s" }, null, %d, %t]}`,
					w.suiMoveEventType, w.maximumBatchSize, w.descendingOrder))
		} else {
			reader = strings.NewReader(
				fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "suix_queryEvents", "params": [{ "MoveEventType": "%s" }, { "txDigest": "%s", "eventSeq": "%s" }, %d, %t]}`,
					w.suiMoveEventType, nextCursor.TxDigest, nextCursor.EventSeq, w.maximumBatchSize, w.descendingOrder))
		}
		resp, err := http.Post(w.suiRPC, "application/json", reader) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
		if err != nil {
			return retVal, fmt.Errorf("suix_queryEvents failed to post: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return retVal, fmt.Errorf("suix_queryEvents failed to read: %w", err)
		}
		resp.Body.Close()

		var res SuiEventResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return retVal, fmt.Errorf("suix_queryEvents failed to unmarshal body: %s, error: %w", string(body), err)
		}
		fmt.Println("Number of events:", len(res.Result.Data))
		for _, datum := range res.Result.Data {
			// Check if the event is newer than the last processed height
			height, hErr := w.getCheckpointForEvent(datum)
			if hErr != nil {
				fmt.Println("Error getting checkpoint for event:", hErr)
				return retVal, hErr
			}
			fmt.Println("Comparing Height:", height, "LastProcessedHeight:", w.latestProcessedCheckpoint)
			if height <= w.latestProcessedCheckpoint {
				done = true
				fmt.Println("Done processing events.")
				break
			} else {
				fmt.Println("Adding event to retVal")
				retVal.Data = append(retVal.Data, datum)
			}
		}
		if (!done) && res.Result.HasNextPage {
			fmt.Println("Getting next page...")
			nextCursor.TxDigest = res.Result.NextCursor.TxDigest
			nextCursor.EventSeq = res.Result.NextCursor.EventSeq
		} else {
			done = true
		}
	}

	retVal.Data = reverseArray(retVal.Data)
	return retVal, nil
}

func (e *Watcher) getLatestCheckpointSN(logger *zap.Logger) (int64, error) {
	resp, err := http.Post(e.suiRPC, "application/json", strings.NewReader(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getLatestCheckpointSequenceNumber", "params": []}`)) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
	if err != nil {
		logger.Error("sui_getLatestCheckpointSequenceNumber failed", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		return 0, fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to post: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("sui_getLatestCheckpointSequenceNumber failed", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		return 0, fmt.Errorf("sui_getLatestCheckpointSequenceNumber failed to read: %w", err)
	}
	resp.Body.Close()
	logger.Debug("Body before unmarshalling", zap.String("body", string(body)))

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
	} else {
		currentSuiHeight.Set(float64(height))
		logger.Debug("sui_getLatestCheckpointSequenceNumber", zap.String("result", res.Result))

		p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSui, &gossipv1.Heartbeat_Network{
			Height:          height,
			ContractAddress: e.suiMoveEventType,
		})
	}
	return height, nil
}

func (e *Watcher) getCheckpointForEvent(event SuiResult) (int64, error) {
	retVal := int64(0)
	reader := strings.NewReader(fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_getTransactionBlock", "params": [ "%s" ]}`,
		*event.ID.TxDigest))
	resp, err := http.Post(e.suiRPC, "application/json", reader) //nolint:noctx // TODO FIXME we should propagate context with Deadline here.
	if err != nil {
		return retVal, fmt.Errorf("sui_getTransactionBlock failed to post: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return retVal, fmt.Errorf("sui_getTransactionBlock failed to read: %w", err)
	}
	resp.Body.Close()

	var res GetCheckpointResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return retVal, fmt.Errorf("sui_getTransactionBlock failed to unmarshal body: %s, error: %w", string(body), err)
	}
	retVal, err = strconv.ParseInt(res.Result.Checkpoint, 10, 64)
	if err != nil {
		return retVal, fmt.Errorf("sui_getTransactionBlock failed to ParseInt: %w", err)
	}

	return retVal, nil
}

// reverseArray reverses the elements of the given slice
func reverseArray[T any](arr []T) []T {
	left := 0
	right := len(arr) - 1

	for left < right {
		// Swap the elements
		arr[left], arr[right] = arr[right], arr[left]
		left++
		right--
	}
	return arr
}
