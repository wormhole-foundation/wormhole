package solana

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

const (
	// CCQ_RETRY_SLOP gets subtracted from the query retry interval to determine how long we can continue fast retries.
	// We don't want the fast retry time to be too close to the query retry interval.
	CCQ_RETRY_SLOP = 250 * time.Millisecond

	// CCQ_ESTIMATED_SLOT_TIME is the estimated Solana slot time used for estimating how long until the MinContextSlot will be reached.
	CCQ_ESTIMATED_SLOT_TIME = 400 * time.Millisecond

	// CCQ_FAST_RETRY_INTERVAL is how long we sleep between fast retry attempts.
	CCQ_FAST_RETRY_INTERVAL = 200 * time.Millisecond
)

// ccqSendQueryResponse sends a response back to the query handler. In the case of an error, the response parameter may be nil.
func (w *SolanaWatcher) ccqSendQueryResponse(req *query.PerChainQueryInternal, status query.QueryStatus, response query.ChainSpecificResponse) {
	queryResponse := query.CreatePerChainQueryResponseInternal(req.RequestID, req.RequestIdx, req.Request.ChainId, status, response)
	select {
	case w.queryResponseC <- queryResponse:
		w.ccqLogger.Debug("published query response to handler")
	default:
		w.ccqLogger.Error("failed to published query response error to handler")
	}
}

// ccqHandleQuery is the top-level query handler. It breaks out the requests based on the type and calls the appropriate handler.
func (w *SolanaWatcher) ccqHandleQuery(ctx context.Context, queryRequest *query.PerChainQueryInternal) {

	// This can't happen unless there is a programming error - the caller
	// is expected to send us only requests for our chainID.
	if queryRequest.Request.ChainId != w.chainID {
		panic("ccqevm: invalid chain ID")
	}

	start := time.Now()

	switch req := queryRequest.Request.Query.(type) {
	case *query.SolanaAccountQueryRequest:
		giveUpTime := start.Add(query.RetryInterval).Add(-CCQ_RETRY_SLOP)
		w.ccqHandleSolanaAccountQueryRequest(ctx, queryRequest, req, giveUpTime, false)
	default:
		w.ccqLogger.Warn("received unsupported request type",
			zap.Uint8("payload", uint8(queryRequest.Request.Query.Type())),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
	}

	query.TotalWatcherTime.WithLabelValues(w.chainID.String()).Observe(float64(time.Since(start).Milliseconds()))
}

// ccqHandleSolanaAccountQueryRequest is the query handler for a sol_account request.
func (w *SolanaWatcher) ccqHandleSolanaAccountQueryRequest(ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.SolanaAccountQueryRequest, giveUpTime time.Time, isRetry bool) {
	requestId := "sol_account:" + queryRequest.ID()
	if !isRetry {
		w.ccqLogger.Info("received a sol_account query",
			zap.Uint64("minContextSlot", req.MinContextSlot),
			zap.Uint64("dataSliceOffset", req.DataSliceOffset),
			zap.Uint64("dataSliceLength", req.DataSliceLength),
			zap.Int("numAccounts", len(req.Accounts)),
			zap.String("requestId", requestId),
		)
	}

	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	// Convert the accounts from byte arrays to public keys.
	accounts := make([]solana.PublicKey, 0, len(req.Accounts))
	for _, acct := range req.Accounts {
		accounts = append(accounts, acct)
	}

	// Create the parameters needed for the account read and add any optional parameters.
	params := rpc.GetMultipleAccountsOpts{
		Encoding:   solana.EncodingBase64,
		Commitment: rpc.CommitmentType(req.Commitment),
	}

	if req.MinContextSlot != 0 {
		params.MinContextSlot = &req.MinContextSlot
	}

	if req.DataSliceLength != 0 {
		params.DataSlice = &rpc.DataSlice{
			Offset: &req.DataSliceOffset,
			Length: &req.DataSliceLength,
		}
	}

	// Read the accounts.
	info, err := w.getMultipleAccountsWithOpts(rCtx, accounts, &params)
	if err != nil {
		if w.ccqCheckForMinSlotContext(ctx, queryRequest, req, requestId, err, giveUpTime, !isRetry) {
			// Return without posting a response because a go routine was created to handle it.
			return
		}
		w.ccqLogger.Error("read failed for sol_account query request",
			zap.String("requestId", requestId),
			zap.Any("accounts", accounts),
			zap.Any("params", params),
			zap.Error(err),
		)

		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	// Read the block for this slot to get the block time.
	maxSupportedTransactionVersion := uint64(0)
	block, err := w.rpcClient.GetBlockWithOpts(rCtx, info.Context.Slot, &rpc.GetBlockOpts{
		Encoding:                       solana.EncodingBase64,
		Commitment:                     params.Commitment,
		TransactionDetails:             rpc.TransactionDetailsNone,
		MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
	})
	if err != nil {
		w.ccqLogger.Error("failed to read block time for sol_account query request",
			zap.String("requestId", requestId),
			zap.Uint64("slotNumber", info.Context.Slot),
			zap.Error(err),
		)

		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	if info == nil {
		w.ccqLogger.Error("read for sol_account query request returned nil info", zap.String("requestId", requestId))
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	if info.Value == nil {
		w.ccqLogger.Error("read for sol_account query request returned nil value", zap.String("requestId", requestId))
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	if len(info.Value) != len(req.Accounts) {
		w.ccqLogger.Error("read for sol_account query request returned unexpected number of results",
			zap.String("requestId", requestId),
			zap.Int("numAccounts", len(req.Accounts)),
			zap.Int("numValues", len(info.Value)),
		)

		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	// Extract the results.
	results := make([]query.SolanaAccountResult, 0, len(req.Accounts))
	for idx, val := range info.Value {
		if val == nil { // This can happen for an invalid account.
			w.ccqLogger.Error("read of account for sol_account query request failed, val is nil", zap.String("requestId", requestId), zap.Any("account", req.Accounts[idx]))
			w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
			return
		}
		if val.Data == nil {
			w.ccqLogger.Error("read of account for sol_account query request failed, data is nil", zap.String("requestId", requestId), zap.Any("account", req.Accounts[idx]))
			w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
			return
		}
		results = append(results, query.SolanaAccountResult{
			Lamports:   val.Lamports,
			RentEpoch:  val.RentEpoch,
			Executable: val.Executable,
			Owner:      val.Owner,
			Data:       val.Data.GetBinary(),
		})
	}

	// Finally, build the response and publish it.
	resp := &query.SolanaAccountQueryResponse{
		SlotNumber: info.Context.Slot,
		BlockTime:  time.Unix(int64(*block.BlockTime), 0),
		BlockHash:  block.Blockhash,
		Results:    results,
	}

	w.ccqLogger.Info("account read for sol_account_query succeeded",
		zap.String("requestId", requestId),
		zap.Uint64("slotNumber", info.Context.Slot),
		zap.Uint64("blockTime", uint64(*block.BlockTime)),
		zap.String("blockHash", hex.EncodeToString(block.Blockhash[:])),
		zap.Uint64("blockHeight", *block.BlockHeight),
	)

	w.ccqSendQueryResponse(queryRequest, query.QuerySuccess, resp)
}

// ccqCheckForMinSlotContext checks to see if the returned error was due to the min context slot not being reached.
// If so, and the estimated time in the future is not too great, it kicks off a go routine to sleep and do a retry.
// In that case, it returns true, telling the caller that it is handling the request so it should not post a response.
// Note that the go routine only does a single retry, but may result in another go routine being initiated to do another, and so on.
func (w *SolanaWatcher) ccqCheckForMinSlotContext(
	ctx context.Context,
	queryRequest *query.PerChainQueryInternal,
	req *query.SolanaAccountQueryRequest,
	requestId string,
	err error,
	giveUpTime time.Time,
	log bool,
) bool {
	if req.MinContextSlot == 0 {
		return false
	}

	if time.Now().After(giveUpTime) {
		w.ccqLogger.Info("giving up on fast retry", zap.String("requestId", requestId))
		return false
	}

	isMinContext, currentSlot, err := ccqIsMinContextSlotError(err)
	if err != nil {
		w.ccqLogger.Error("failed to parse for min context slot error", zap.Error(err))
		return false
	}

	if !isMinContext {
		return false
	}

	// Estimate how far in the future the requested slot is, using our estimated slot time.
	futureSlotEstimate := time.Duration(req.MinContextSlot-currentSlot) * CCQ_ESTIMATED_SLOT_TIME

	// If the requested slot is more than ten seconds in the future, use the regular retry mechanism.
	if futureSlotEstimate > query.RetryInterval {
		w.ccqLogger.Info("minimum context slot is too far in the future, requesting slow retry",
			zap.String("requestId", requestId),
			zap.Uint64("currentSlot", currentSlot),
			zap.Uint64("minContextSlot", req.MinContextSlot),
			zap.Stringer("futureSlotEstimate", futureSlotEstimate),
		)
		return false
	}

	// Kick off the retry after a short delay.
	go w.ccqSleepAndRetryAccountQuery(ctx, queryRequest, req, requestId, currentSlot, giveUpTime, log)
	return true
}

// ccqSleepAndRetryAccountQuery does a short sleep and then initiates a retry.
func (w *SolanaWatcher) ccqSleepAndRetryAccountQuery(ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.SolanaAccountQueryRequest, requestId string, currentSlot uint64, giveUpTime time.Time, log bool) {
	if log {
		w.ccqLogger.Info("minimum context slot has not been reached, will retry shortly",
			zap.String("requestId", requestId),
			zap.Uint64("currentSlot", currentSlot),
			zap.Uint64("minContextSlot", req.MinContextSlot),
			zap.Stringer("retryInterval", CCQ_FAST_RETRY_INTERVAL),
		)
	}

	time.Sleep(CCQ_FAST_RETRY_INTERVAL)

	if log {
		w.ccqLogger.Info("initiating fast retry", zap.String("requestId", requestId))
	}

	w.ccqHandleSolanaAccountQueryRequest(ctx, queryRequest, req, giveUpTime, true)
}

// ccqIsMinContextSlotError parses an error to see if it is "Minimum context slot has not been reached". If it is, it returns the slot number
func ccqIsMinContextSlotError(err error) (bool, uint64, error) {
	/*
	  A MinContextSlot error looks like this (and contains the context slot):
	  "(*jsonrpc.RPCError)(0xc00b3881b0)({\n Code: (int) -32016,\n Message: (string) (len=41) \"Minimum context slot has not been reached\",\n Data: (map[string]interface {}) (len=1) {\n  (string) (len=11) \"contextSlot\": (json.Number) (len=4) \"3630\"\n }\n})\n"
	*/
	var rpcErr *jsonrpc.RPCError
	if !errors.As(err, &rpcErr) {
		return false, 0, nil // Some other kind of error. That's okay.
	}

	if rpcErr.Code != -32016 { // Minimum context slot has not been reached
		return false, 0, nil // Some other kind of RPC error. That's okay.
	}

	// From here on down, any error is bad because the MinContextSlot error is not in the expected format.
	m, ok := rpcErr.Data.(map[string]interface{})
	if !ok {
		return false, 0, fmt.Errorf("failed to extract data from min context slot error")
	}

	contextSlot, ok := m["contextSlot"]
	if !ok {
		return false, 0, fmt.Errorf(`min context slot error does not contain "contextSlot"`)
	}

	currentSlotAsJson, ok := contextSlot.(json.Number)
	if !ok {
		return false, 0, fmt.Errorf(`min context slot error "contextSlot" is not json.Number`)
	}

	currentSlot, typeErr := strconv.ParseUint(currentSlotAsJson.String(), 10, 64)
	if typeErr != nil {
		return false, 0, fmt.Errorf(`min context slot error "contextSlot" is not uint64: %w`, err)
	}

	return true, currentSlot, nil
}

type M map[string]interface{}

// getMultipleAccountsWithOpts is a work-around for the fact that the library call doesn't honor MinContextSlot.
// Opened the following issue against the library: https://github.com/gagliardetto/solana-go/issues/170
func (w *SolanaWatcher) getMultipleAccountsWithOpts(
	ctx context.Context,
	accounts []solana.PublicKey,
	opts *rpc.GetMultipleAccountsOpts,
) (out *rpc.GetMultipleAccountsResult, err error) {
	params := []interface{}{accounts}

	if opts != nil {
		obj := M{}
		if opts.Encoding != "" {
			obj["encoding"] = opts.Encoding
		}
		if opts.Commitment != "" {
			obj["commitment"] = opts.Commitment
		}
		if opts.DataSlice != nil {
			obj["dataSlice"] = M{
				"offset": opts.DataSlice.Offset,
				"length": opts.DataSlice.Length,
			}
			if opts.Encoding == solana.EncodingJSONParsed {
				return nil, errors.New("cannot use dataSlice with EncodingJSONParsed")
			}
		}
		if opts.MinContextSlot != nil {
			obj["minContextSlot"] = *opts.MinContextSlot
		}
		if len(obj) > 0 {
			params = append(params, obj)
		}
	}

	err = w.rpcClient.RPCCallForInto(ctx, &out, "getMultipleAccounts", params)
	if err != nil {
		return nil, err
	}
	if out.Value == nil {
		return nil, rpc.ErrNotFound
	}
	return
}
