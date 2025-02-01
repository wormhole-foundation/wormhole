package solana

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

	// CCQ_MAX_BLOCK_READ_ATTEMPTS the total number of times we will try to read the block when it returns "Block not available".
	CCQ_MAX_BLOCK_READ_ATTEMPTS = 3

	// CCQ_BLOCK_RETRY_DELAY is how long we sleep between attempts to read the block time.
	CCQ_BLOCK_RETRY_DELAY = 250 * time.Millisecond
)

// ccqStart starts up CCQ query processing.
func (w *SolanaWatcher) ccqStart(ctx context.Context) {
	query.StartWorkers(ctx, w.ccqLogger, w.errC, w, w.queryReqC, w.ccqConfig, w.chainID.String())
}

// ccqSendQueryResponse sends a response back to the query handler.
func (w *SolanaWatcher) ccqSendQueryResponse(queryResponse *query.PerChainQueryResponseInternal) {
	select {
	case w.queryResponseC <- queryResponse:
		w.ccqLogger.Debug("published query response to handler")
	default:
		w.ccqLogger.Error("failed to published query response error to handler")
	}
}

// ccqSendErrorResponse creates an error query response and sends it back to the query handler. It sets the response field to nil.
func (w *SolanaWatcher) ccqSendErrorResponse(req *query.PerChainQueryInternal, status query.QueryStatus) {
	queryResponse := query.CreatePerChainQueryResponseInternal(req.RequestID, req.RequestIdx, req.Request.ChainId, status, nil)
	w.ccqSendQueryResponse(queryResponse)
}

// QueryHandler is the top-level query handler. It breaks out the requests based on the type and calls the appropriate handler.
func (w *SolanaWatcher) QueryHandler(ctx context.Context, queryRequest *query.PerChainQueryInternal) {
	// This can't happen unless there is a programming error - the caller
	// is expected to send us only requests for our chainID.
	if queryRequest.Request.ChainId != w.chainID {
		panic("ccqevm: invalid chain ID")
	}

	start := time.Now()

	giveUpTime := start.Add(query.RetryInterval).Add(-CCQ_RETRY_SLOP)
	switch req := queryRequest.Request.Query.(type) {
	case *query.SolanaAccountQueryRequest:
		w.ccqHandleSolanaAccountQueryRequest(ctx, queryRequest, req, giveUpTime)
	case *query.SolanaPdaQueryRequest:
		w.ccqHandleSolanaPdaQueryRequest(ctx, queryRequest, req, giveUpTime)
	default:
		w.ccqLogger.Warn("received unsupported request type",
			zap.Uint8("payload", uint8(queryRequest.Request.Query.Type())),
		)
		w.ccqSendErrorResponse(queryRequest, query.QueryFatalError)
	}

	query.TotalWatcherTime.WithLabelValues(w.chainID.String()).Observe(float64(time.Since(start).Milliseconds()))
}

// ccqCustomPublisher is an interface used by ccqBaseHandleSolanaAccountQueryRequest to specify how to publish the response from a query.
type ccqCustomPublisher interface {
	// publish should take a sol_account query response and publish it as the appropriate response type.
	publish(*query.PerChainQueryResponseInternal, *query.SolanaAccountQueryResponse)
}

// ccqBaseHandleSolanaAccountQueryRequest is the base Solana Account query handler. It does the actual account queries, and if necessary does fast retries
// until the minimum context slot is reached. It does not publish the response, but instead invokes the query specific publisher that is passed in.
func (w *SolanaWatcher) ccqBaseHandleSolanaAccountQueryRequest(
	ctx context.Context,
	queryRequest *query.PerChainQueryInternal,
	req *query.SolanaAccountQueryRequest,
	giveUpTime time.Time,
	tag string,
	requestId string,
	isRetry bool,
	publisher ccqCustomPublisher,
	numFastRetries int,
) {
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
		if w.ccqCheckForMinSlotContext(ctx, queryRequest, req, requestId, err, giveUpTime, !isRetry, tag, publisher, numFastRetries) {
			// Return without posting a response because a go routine was created to handle it.
			return
		}
		w.ccqLogger.Info(fmt.Sprintf("read failed for %s query request", tag),
			zap.String("requestId", requestId),
			zap.Any("accounts", accounts),
			zap.Any("params", params),
			zap.Error(err),
		)

		w.ccqSendErrorResponse(queryRequest, query.QueryRetryNeeded)
		return
	}

	// Read the block for this slot to get the block time.
	var block *rpc.GetBlockResult
	var numBlockReadAttempts int
	for {
		maxSupportedTransactionVersion := uint64(0)
		block, err = w.rpcClient.GetBlockWithOpts(rCtx, info.Context.Slot, &rpc.GetBlockOpts{
			Encoding:                       solana.EncodingBase64,
			Commitment:                     params.Commitment,
			TransactionDetails:             rpc.TransactionDetailsNone,
			MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
		})
		if err == nil {
			break
		}

		if !ccqIsBlockNotAvailable(err) {
			w.ccqLogger.Error(fmt.Sprintf("failed to read block time for %s query request", tag),
				zap.String("requestId", requestId),
				zap.Uint64("slotNumber", info.Context.Slot),
				zap.Error(err),
			)

			w.ccqSendErrorResponse(queryRequest, query.QueryRetryNeeded)
			return
		}

		numBlockReadAttempts += 1
		if numBlockReadAttempts >= CCQ_MAX_BLOCK_READ_ATTEMPTS {
			w.ccqLogger.Error(fmt.Sprintf("repeatedly failed to read block time for %s query request, giving up", tag),
				zap.String("requestId", requestId),
				zap.Uint64("slotNumber", info.Context.Slot),
				zap.Error(err),
			)

			w.ccqSendErrorResponse(queryRequest, query.QueryRetryNeeded)
			return
		}

		time.Sleep(CCQ_BLOCK_RETRY_DELAY)
	}

	if info == nil {
		w.ccqLogger.Error(fmt.Sprintf("read for %s query request returned nil info", tag), zap.String("requestId", requestId))
		w.ccqSendErrorResponse(queryRequest, query.QueryFatalError)
		return
	}

	if info.Value == nil {
		w.ccqLogger.Error(fmt.Sprintf("read for %s query request returned nil value", tag), zap.String("requestId", requestId))
		w.ccqSendErrorResponse(queryRequest, query.QueryFatalError)
		return
	}

	if len(info.Value) != len(req.Accounts) {
		w.ccqLogger.Error(fmt.Sprintf("read for %s query request returned unexpected number of results", tag),
			zap.String("requestId", requestId),
			zap.Int("numAccounts", len(req.Accounts)),
			zap.Int("numValues", len(info.Value)),
		)

		w.ccqSendErrorResponse(queryRequest, query.QueryFatalError)
		return
	}

	// Extract the results.
	results := make([]query.SolanaAccountResult, 0, len(req.Accounts))
	for idx, val := range info.Value {
		if val == nil { // This can happen for an invalid account.
			w.ccqLogger.Error(fmt.Sprintf("read of account for %s query request failed, val is nil", tag), zap.String("requestId", requestId), zap.Any("account", req.Accounts[idx]))
			w.ccqSendErrorResponse(queryRequest, query.QueryFatalError)
			return
		}
		if val.Data == nil {
			w.ccqLogger.Error(fmt.Sprintf("read of account for %s query request failed, data is nil", tag), zap.String("requestId", requestId), zap.Any("account", req.Accounts[idx]))
			w.ccqSendErrorResponse(queryRequest, query.QueryFatalError)
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

	w.ccqLogger.Info(fmt.Sprintf("account read for %s query succeeded", tag),
		zap.String("requestId", requestId),
		zap.Uint64("slotNumber", info.Context.Slot),
		zap.Uint64("blockTime", uint64(*block.BlockTime)), // #nosec G115 -- This conversion is safe indefinitely
		zap.String("blockHash", hex.EncodeToString(block.Blockhash[:])),
		zap.Uint64("blockHeight", *block.BlockHeight),
		zap.Int("numFastRetries", numFastRetries),
	)

	// Publish the response using the custom publisher.
	publisher.publish(query.CreatePerChainQueryResponseInternal(queryRequest.RequestID, queryRequest.RequestIdx, queryRequest.Request.ChainId, query.QuerySuccess, resp), resp)
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
	tag string,
	publisher ccqCustomPublisher,
	numFastRetries int,
) bool {
	if req.MinContextSlot == 0 {
		return false
	}

	if time.Now().After(giveUpTime) {
		w.ccqLogger.Info("giving up on fast retry", zap.String("requestId", requestId))
		return false
	}

	isMinContext, currentSlotFromError := ccqIsMinContextSlotError(err)
	if !isMinContext {
		return false
	}

	var currentSlot uint64
	if currentSlotFromError != 0 {
		currentSlot = currentSlotFromError
	} else {
		currentSlot = w.GetLatestFinalizedBlockNumber()
	}

	// Estimate how far in the future the requested slot is, using our estimated slot time.
	futureSlotEstimate := time.Duration(req.MinContextSlot-currentSlot) * CCQ_ESTIMATED_SLOT_TIME // #nosec G115 -- This conversion is safe indefinitely

	// If the requested slot is definitively more than the retry interval, use the regular retry mechanism.
	if futureSlotEstimate > query.RetryInterval*2 {
		w.ccqLogger.Info("minimum context slot is too far in the future, requesting slow retry",
			zap.String("requestId", requestId),
			zap.Uint64("currentSlot", currentSlot),
			zap.Uint64("currentSlotFromError", currentSlotFromError),
			zap.Uint64("minContextSlot", req.MinContextSlot),
			zap.Stringer("futureSlotEstimate", futureSlotEstimate),
		)
		return false
	}

	// Kick off the retry after a short delay.
	go w.ccqSleepAndRetryAccountQuery(ctx, queryRequest, req, requestId, currentSlot, currentSlotFromError, giveUpTime, log, tag, publisher, numFastRetries)
	return true
}

// ccqSleepAndRetryAccountQuery does a short sleep and then initiates a retry.
func (w *SolanaWatcher) ccqSleepAndRetryAccountQuery(
	ctx context.Context,
	queryRequest *query.PerChainQueryInternal,
	req *query.SolanaAccountQueryRequest,
	requestId string,
	currentSlot uint64,
	currentSlotFromError uint64,
	giveUpTime time.Time,
	log bool,
	tag string,
	publisher ccqCustomPublisher,
	numFastRetries int,
) {
	if log {
		w.ccqLogger.Info("minimum context slot has not been reached, will retry shortly",
			zap.String("requestId", requestId),
			zap.Uint64("currentSlot", currentSlot),
			zap.Uint64("currentSlotFromError", currentSlotFromError),
			zap.Uint64("minContextSlot", req.MinContextSlot),
			zap.Stringer("retryInterval", CCQ_FAST_RETRY_INTERVAL),
		)
	}

	time.Sleep(CCQ_FAST_RETRY_INTERVAL)

	if log {
		w.ccqLogger.Info("initiating fast retry", zap.String("requestId", requestId))
	}

	w.ccqBaseHandleSolanaAccountQueryRequest(ctx, queryRequest, req, giveUpTime, tag, requestId, true, publisher, numFastRetries+1)
}

// ccqIsMinContextSlotError parses an error to see if it is "Minimum context slot has not been reached". If it is, it returns the slot number
func ccqIsMinContextSlotError(err error) (bool, uint64) {
	/*
		  A MinContextSlot error looks like this (and contains the context slot):
		  "(*jsonrpc.RPCError)(0xc00b3881b0)({\n Code: (int) -32016,\n Message: (string) (len=41) \"Minimum context slot has not been reached\",\n Data: (map[string]interface {}) (len=1) {\n  (string) (len=11) \"contextSlot\": (json.Number) (len=4) \"3630\"\n }\n})\n"

			Except some endpoints return something like this instead:
			"(*jsonrpc.RPCError)(0xc03c0bcd20)({\n Code: (int) -32016,\n Message: (string) (len=41) \"Minimum context slot has not been reached\",\n Data: (interface {}) <nil>\n})\n"
	*/
	var rpcErr *jsonrpc.RPCError
	if !errors.As(err, &rpcErr) {
		return false, 0 // Some other kind of error.
	}

	if rpcErr.Code != -32016 { // Minimum context slot has not been reached
		return false, 0 // Some other kind of RPC error.
	}

	// We know it is a MinContextSlot error. If it contains the current slot number, extract and return that.
	// Since some Solana endpoints do not return that, we can't treat it as an error if it is missing.
	m, ok := rpcErr.Data.(map[string]interface{})
	if !ok {
		return true, 0
	}

	contextSlot, ok := m["contextSlot"]
	if !ok {
		return true, 0
	}

	currentSlotAsJson, ok := contextSlot.(json.Number)
	if !ok {
		return true, 0
	}

	currentSlot, typeErr := strconv.ParseUint(currentSlotAsJson.String(), 10, 64)
	if typeErr != nil {
		return true, 0
	}

	return true, currentSlot
}

// ccqHandleSolanaAccountQueryRequest is the query handler for a sol_account request.
func (w *SolanaWatcher) ccqHandleSolanaAccountQueryRequest(ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.SolanaAccountQueryRequest, giveUpTime time.Time) {
	requestId := "sol_account" + ":" + queryRequest.ID()
	w.ccqLogger.Info("received a sol_account query",
		zap.Uint64("minContextSlot", req.MinContextSlot),
		zap.Uint64("dataSliceOffset", req.DataSliceOffset),
		zap.Uint64("dataSliceLength", req.DataSliceLength),
		zap.Int("numAccounts", len(req.Accounts)),
		zap.String("requestId", requestId),
	)

	publisher := ccqSolanaAccountPublisher{w}
	w.ccqBaseHandleSolanaAccountQueryRequest(ctx, queryRequest, req, giveUpTime, "sol_account", requestId, false, publisher, 0)
}

// ccqSolanaAccountPublisher is the publisher for the sol_account query. All it has to do is forward the response passed in to the watcher, as is.
type ccqSolanaAccountPublisher struct {
	w *SolanaWatcher
}

func (impl ccqSolanaAccountPublisher) publish(resp *query.PerChainQueryResponseInternal, _ *query.SolanaAccountQueryResponse) {
	impl.w.ccqSendQueryResponse(resp)
}

// ccqHandleSolanaPdaQueryRequest is the query handler for a sol_pda request.
func (w *SolanaWatcher) ccqHandleSolanaPdaQueryRequest(ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.SolanaPdaQueryRequest, giveUpTime time.Time) {
	requestId := "sol_pda:" + queryRequest.ID()
	w.ccqLogger.Info("received a sol_pda query",
		zap.Uint64("minContextSlot", req.MinContextSlot),
		zap.Uint64("dataSliceOffset", req.DataSliceOffset),
		zap.Uint64("dataSliceLength", req.DataSliceLength),
		zap.Int("numPdas", len(req.PDAs)),
		zap.String("requestId", requestId),
	)

	// Derive the list of accounts from the PDAs and save those along with the bumps.
	accounts := make([][query.SolanaPublicKeyLength]byte, 0, len(req.PDAs))
	bumps := make([]uint8, 0, len(req.PDAs))
	for _, pda := range req.PDAs {
		account, bump, err := solana.FindProgramAddress(pda.Seeds, pda.ProgramAddress)
		if err != nil {
			w.ccqLogger.Error("failed to derive account from pda for sol_pda query",
				zap.String("requestId", requestId),
				zap.String("programAddress", hex.EncodeToString(pda.ProgramAddress[:])),
				zap.Any("seeds", pda.Seeds),
				zap.Error(err),
			)

			w.ccqSendErrorResponse(queryRequest, query.QueryFatalError)
			return
		}

		accounts = append(accounts, account)
		bumps = append(bumps, bump)
	}

	// Build a standard sol_account query using the derived accounts.
	acctReq := &query.SolanaAccountQueryRequest{
		Commitment:      req.Commitment,
		MinContextSlot:  req.MinContextSlot,
		DataSliceOffset: req.DataSliceOffset,
		DataSliceLength: req.DataSliceLength,
		Accounts:        accounts,
	}

	publisher := ccqPdaPublisher{
		w:            w,
		queryRequest: queryRequest,
		requestId:    requestId,
		accounts:     accounts,
		bumps:        bumps,
	}

	// Execute the standard sol_account query passing in the publisher to publish a sol_pda response.
	w.ccqBaseHandleSolanaAccountQueryRequest(ctx, queryRequest, acctReq, giveUpTime, "sol_pda", requestId, false, publisher, 0)
}

// ccqPdaPublisher is a custom publisher that publishes a sol_pda response.
type ccqPdaPublisher struct {
	w            *SolanaWatcher
	queryRequest *query.PerChainQueryInternal
	requestId    string
	accounts     [][query.SolanaPublicKeyLength]byte
	bumps        []uint8
}

func (pub ccqPdaPublisher) publish(pcrResp *query.PerChainQueryResponseInternal, acctResp *query.SolanaAccountQueryResponse) {
	if pcrResp == nil {
		pub.w.ccqLogger.Error("sol_pda query failed, pcrResp is nil", zap.String("requestId", pub.requestId))
		pub.w.ccqSendErrorResponse(pub.queryRequest, query.QueryFatalError)
		return
	}

	if pcrResp.Status != query.QuerySuccess {
		// publish() should only get called in success cases.
		pub.w.ccqLogger.Error("received an unexpected query response for sol_pda query", zap.String("requestId", pub.requestId), zap.Any("pcrResp", pcrResp))
		pub.w.ccqSendErrorResponse(pub.queryRequest, query.QueryFatalError)
		return
	}

	if acctResp == nil {
		pub.w.ccqLogger.Error("sol_pda query failed, acctResp is nil", zap.String("requestId", pub.requestId))
		pub.w.ccqSendErrorResponse(pub.queryRequest, query.QueryFatalError)
		return
	}

	if len(acctResp.Results) != len(pub.accounts) {
		pub.w.ccqLogger.Error("sol_pda query failed, unexpected number of results", zap.String("requestId", pub.requestId), zap.Int("numResults", len(acctResp.Results)), zap.Int("expectedResults", len(pub.accounts)))
		pub.w.ccqSendErrorResponse(pub.queryRequest, query.QueryFatalError)
		return
	}

	// Build the PDA response from the base response.
	results := make([]query.SolanaPdaResult, 0, len(pub.accounts))
	for idx, acctResult := range acctResp.Results {
		results = append(results, query.SolanaPdaResult{
			Account:    pub.accounts[idx],
			Bump:       pub.bumps[idx],
			Lamports:   acctResult.Lamports,
			RentEpoch:  acctResult.RentEpoch,
			Executable: acctResult.Executable,
			Owner:      acctResult.Owner,
			Data:       acctResult.Data,
		})
	}

	resp := &query.SolanaPdaQueryResponse{
		SlotNumber: acctResp.SlotNumber,
		BlockTime:  acctResp.BlockTime,
		BlockHash:  acctResp.BlockHash,
		Results:    results,
	}

	// Finally, publish the result.
	pub.w.ccqSendQueryResponse(query.CreatePerChainQueryResponseInternal(pub.queryRequest.RequestID, pub.queryRequest.RequestIdx, pub.queryRequest.Request.ChainId, query.QuerySuccess, resp))
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

// ccqIsBlockNotAvailable parses an error to see if it is a "Block not available for slot" error.
func ccqIsBlockNotAvailable(err error) bool {
	/*
		  A "Block not available for slot" error looks like this:
			"(*jsonrpc.RPCError)(0xc0208a0270)({\n Code: (int) -32004,\n Message: (string) (len=38) \"Block not available for slot 282135928\",\n Data: (interface {}) <nil>\n})\n"

			A "Minimum context slot has not been reached" error looks like this:
			(*jsonrpc.RPCError)(0xc21e4f8ea0)({\n Code: (int) -32016,\n Message: (string) (len=41) \"Minimum context slot has not been reached\",\n Data: (map[string]interface {}) (len=1) {\n  (string) (len=11) \"contextSlot\": (json.Number) (len=9) \"303955907\"\n }\n})\n"
	*/
	var rpcErr *jsonrpc.RPCError
	if !errors.As(err, &rpcErr) {
		return false // Some other kind of error.
	}

	if rpcErr.Code != -32004 && // Block not available for slot
		rpcErr.Code != -32016 { // Minimum context slot has not been reached
		return false // Some other kind of RPC error.
	}

	return strings.Contains(rpcErr.Message, "Block not available for slot")
}
