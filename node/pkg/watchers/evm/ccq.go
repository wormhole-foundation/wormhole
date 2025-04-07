package evm

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"

	"github.com/ethereum/go-ethereum/rpc"

	eth_common "github.com/ethereum/go-ethereum/common"
	eth_hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/query"
)

// ccqStart starts up CCQ query processing.
func (w *Watcher) ccqStart(ctx context.Context, errC chan error) {
	if w.ccqTimestampCache != nil && w.ccqBackfillCache {
		w.ccqBackfillStart(ctx, errC)
	}

	query.StartWorkers(ctx, w.ccqLogger, errC, w, w.queryReqC, w.ccqConfig, w.chainID.String())
}

// ccqSendQueryResponse sends a response back to the query handler. In the case of an error, the response parameter may be nil.
func (w *Watcher) ccqSendQueryResponse(req *query.PerChainQueryInternal, status query.QueryStatus, response query.ChainSpecificResponse) {
	queryResponse := query.CreatePerChainQueryResponseInternal(req.RequestID, req.RequestIdx, req.Request.ChainId, status, response)
	select {
	case w.queryResponseC <- queryResponse:
		w.ccqLogger.Debug("published query response to handler")
	default:
		w.ccqLogger.Error("failed to published query response to handler")
	}
}

// QueryHandler is the top-level query handler. It breaks out the requests based on the type and calls the appropriate handler.
func (w *Watcher) QueryHandler(ctx context.Context, queryRequest *query.PerChainQueryInternal) {

	// This can't happen unless there is a programming error - the caller
	// is expected to send us only requests for our chainID.
	if queryRequest.Request.ChainId != w.chainID {
		panic("ccqevm: invalid chain ID")
	}

	start := time.Now()

	switch req := queryRequest.Request.Query.(type) {
	case *query.EthCallQueryRequest:
		w.ccqHandleEthCallQueryRequest(ctx, queryRequest, req)
	case *query.EthCallByTimestampQueryRequest:
		w.ccqHandleEthCallByTimestampQueryRequest(ctx, queryRequest, req)
	case *query.EthCallWithFinalityQueryRequest:
		w.ccqHandleEthCallWithFinalityQueryRequest(ctx, queryRequest, req)
	default:
		w.ccqLogger.Warn("received unsupported request type",
			zap.Uint8("payload", uint8(queryRequest.Request.Query.Type())),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
	}

	query.TotalWatcherTime.WithLabelValues(w.chainID.String()).Observe(float64(time.Since(start).Milliseconds()))
}

// EvmCallData contains the details of a single call in the batch.
type EvmCallData struct {
	To         eth_common.Address
	Data       string
	CallResult *eth_hexutil.Bytes

	// These are lowercase so they don't get marshaled for logging purposes. JSON doesn't print anything meaningful for them anyway.
	callErr            error
	callTransactionArg map[string]interface{}
}

func (ecd EvmCallData) String() string {
	bytes, err := json.Marshal(ecd)
	if err != nil {
		bytes = []byte("invalid json")
	}

	return string(bytes)
}

// ccqHandleEthCallQueryRequest is the query handler for an eth_call request.
func (w *Watcher) ccqHandleEthCallQueryRequest(ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.EthCallQueryRequest) {
	requestId := "eth_call:" + queryRequest.ID()
	block := req.BlockId
	w.ccqLogger.Info("received eth_call query request",
		zap.String("requestId", requestId),
		zap.String("block", block),
		zap.Int("numRequests", len(req.CallData)),
	)

	// Create the block query args.
	blockMethod, callBlockArg, err := ccqCreateBlockRequest(block)
	if err != nil {
		w.ccqLogger.Info("invalid block id in eth_call query request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	// Create the batch of requested calls for the specified block.
	batch, evmCallData := ccqBuildBatchFromCallData(req, callBlockArg)

	// Add the block query to the batch.
	var blockResult connectors.BlockMarshaller
	var blockError error
	batch = append(batch, rpc.BatchElem{
		Method: blockMethod,
		Args: []interface{}{
			block,
			false, // no full transaction details
		},
		Result: &blockResult,
		Error:  blockError,
	})

	// Query the RPC.
	start := time.Now()
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = w.ethConn.RawBatchCallContext(timeout, batch)
	if err != nil {
		w.ccqLogger.Info("failed to process eth_call query request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	// Verify that the block read was successful.
	if err := w.ccqVerifyBlockResult(blockError, blockResult); err != nil {
		w.ccqLogger.Debug("failed to verify block for eth_call query",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	// Verify all the call results and build the batch of results.
	results, err := w.ccqVerifyAndExtractQueryResults(requestId, evmCallData)
	if err != nil {
		w.ccqLogger.Info("failed to process eth_call query call request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	w.ccqLogger.Info("query complete for eth_call",
		zap.String("requestId", requestId),
		zap.String("block", block),
		zap.String("blockNumber", blockResult.Number.String()),
		zap.String("blockHash", blockResult.Hash.Hex()),
		zap.String("blockTime", blockResult.Time.String()),
		zap.Int64("duration", time.Since(start).Milliseconds()),
	)

	// Finally, build the response and publish it.
	resp := query.EthCallQueryResponse{
		BlockNumber: blockResult.Number.ToInt().Uint64(),
		Hash:        blockResult.Hash,
		Time:        time.Unix(int64(blockResult.Time), 0), // #nosec G115 -- This conversion is safe indefinitely
		Results:     results,
	}

	w.ccqSendQueryResponse(queryRequest, query.QuerySuccess, &resp)
}

// ccqHandleEthCallByTimestampQueryRequest is the query handler for an eth_call_by_timestamp request.
func (w *Watcher) ccqHandleEthCallByTimestampQueryRequest(ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.EthCallByTimestampQueryRequest) {
	requestId := "eth_call_by_timestamp:" + queryRequest.ID()
	block := req.TargetBlockIdHint
	nextBlock := req.FollowingBlockIdHint
	w.ccqLogger.Info("received eth_call_by_timestamp query request",
		zap.String("requestId", requestId),
		zap.Uint64("timestamp", req.TargetTimestamp),
		zap.String("block", block),
		zap.String("nextBlock", nextBlock),
		zap.Int("numRequests", len(req.CallData)),
	)

	// Verify that the two block hints are consistent, either both set, or both unset.
	if (block == "") != (nextBlock == "") {
		w.ccqLogger.Info("invalid block id hints in eth_call_by_timestamp query request, both should be either set or unset",
			zap.String("requestId", requestId),
			zap.Uint64("timestamp", req.TargetTimestamp),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	// Look up the blocks based on the timestamp if necessary.
	if block == "" {
		if w.ccqTimestampCache == nil {
			w.ccqLogger.Info("error in block id hints in eth_call_by_timestamp query request, they are unset and chain does not support timestamp caching")
			w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
			return
		}

		// Look the timestamp up in the cache. Note that the cache uses native EVM time, which is seconds, but CCQ uses microseconds, so we have to convert.
		timestampForCache := req.TargetTimestamp / 1000000
		blockNum, nextBlockNum, found := w.ccqTimestampCache.LookUp(timestampForCache)
		if !found {
			status := query.QueryRetryNeeded
			firstBlockNum, firstBlockTime, lastBlockNum, lastBlockTime := w.ccqTimestampCache.GetRange()
			if nextBlockNum == 0 {
				w.ccqLogger.Info("block look up failed in eth_call_by_timestamp query request, timestamp beyond the end of the cache, will wait and retry",
					zap.String("requestId", requestId),
					zap.Uint64("timestamp", req.TargetTimestamp),
					zap.String("block", block),
					zap.String("nextBlock", nextBlock),
					zap.Uint64("blockNum", blockNum),
					zap.Uint64("nextBlockNum", nextBlockNum),
					zap.Uint64("timestampForCache", timestampForCache),
					zap.Uint64("firstBlockNum", firstBlockNum),
					zap.Uint64("firstBlockTime", firstBlockTime),
					zap.Uint64("lastBlockNum", lastBlockNum),
					zap.Uint64("lastBlockTime", lastBlockTime),
				)
			} else if blockNum == 0 {
				w.ccqLogger.Info("block look up failed in eth_call_by_timestamp query request, timestamp too old, failing request",
					zap.String("requestId", requestId),
					zap.Uint64("timestamp", req.TargetTimestamp),
					zap.String("block", block),
					zap.String("nextBlock", nextBlock),
					zap.Uint64("blockNum", blockNum),
					zap.Uint64("nextBlockNum", nextBlockNum),
					zap.Uint64("timestampForCache", timestampForCache),
					zap.Uint64("firstBlockNum", firstBlockNum),
					zap.Uint64("firstBlockTime", firstBlockTime),
					zap.Uint64("lastBlockNum", lastBlockNum),
					zap.Uint64("lastBlockTime", lastBlockTime),
				)
				status = query.QueryFatalError
			} else if w.ccqBackfillCache {
				w.ccqLogger.Info("block look up failed in eth_call_by_timestamp query request, timestamp is in a gap in the cache, will request a backfill and retry",
					zap.String("requestId", requestId),
					zap.Uint64("timestamp", req.TargetTimestamp),
					zap.String("block", block),
					zap.String("nextBlock", nextBlock),
					zap.Uint64("blockNum", blockNum),
					zap.Uint64("nextBlockNum", nextBlockNum),
					zap.Uint64("timestampForCache", timestampForCache),
					zap.Uint64("firstBlockNum", firstBlockNum),
					zap.Uint64("firstBlockTime", firstBlockTime),
					zap.Uint64("lastBlockNum", lastBlockNum),
					zap.Uint64("lastBlockTime", lastBlockTime),
				)
				w.ccqRequestBackfill(timestampForCache)
			} else {
				w.ccqLogger.Info("block look up failed in eth_call_by_timestamp query request, timestamp is in a gap in the cache, failing request",
					zap.String("requestId", requestId),
					zap.Uint64("timestamp", req.TargetTimestamp),
					zap.String("block", block),
					zap.String("nextBlock", nextBlock),
					zap.Uint64("blockNum", blockNum),
					zap.Uint64("nextBlockNum", nextBlockNum),
					zap.Uint64("timestampForCache", timestampForCache),
					zap.Uint64("firstBlockNum", firstBlockNum),
					zap.Uint64("firstBlockTime", firstBlockTime),
					zap.Uint64("lastBlockNum", lastBlockNum),
					zap.Uint64("lastBlockTime", lastBlockTime),
				)
				status = query.QueryFatalError
			}
			w.ccqSendQueryResponse(queryRequest, status, nil)
			return
		}

		block = fmt.Sprintf("0x%x", blockNum)
		nextBlock = fmt.Sprintf("0x%x", nextBlockNum)

		w.ccqLogger.Info("cache look up in eth_call_by_timestamp query request mapped timestamp to blocks",
			zap.String("requestId", requestId),
			zap.Uint64("timestamp", req.TargetTimestamp),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Uint64("blockNum", blockNum),
			zap.Uint64("nextBlockNum", nextBlockNum),
		)
	}

	// Create the query args for both blocks.
	blockMethod, callBlockArg, err := ccqCreateBlockRequest(block)
	if err != nil {
		w.ccqLogger.Info("invalid target block id hint in eth_call_by_timestamp query request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	nextBlockMethod, _, err := ccqCreateBlockRequest(nextBlock)
	if err != nil {
		w.ccqLogger.Info("invalid following block id hint in eth_call_by_timestamp query request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	// Create the batch of requested calls for the specified block.
	batch, evmCallData := ccqBuildBatchFromCallData(req, callBlockArg)

	// Add the block query to the batch.
	var blockResult connectors.BlockMarshaller
	var blockError error
	batch = append(batch, rpc.BatchElem{
		Method: blockMethod,
		Args: []interface{}{
			block,
			false, // no full transaction details
		},
		Result: &blockResult,
		Error:  blockError,
	})

	// Add the next block query to the batch.
	var nextBlockResult connectors.BlockMarshaller
	var nextBlockError error
	batch = append(batch, rpc.BatchElem{
		Method: nextBlockMethod,
		Args: []interface{}{
			nextBlock,
			false, // no full transaction details
		},
		Result: &nextBlockResult,
		Error:  nextBlockError,
	})

	// Query the RPC.
	start := time.Now()
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = w.ethConn.RawBatchCallContext(timeout, batch)
	if err != nil {
		w.ccqLogger.Info("failed to process eth_call_by_timestamp query request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	// Verify the target block read was successful.
	if err := w.ccqVerifyBlockResult(blockError, blockResult); err != nil {
		w.ccqLogger.Debug("failed to verify target block for eth_call_by_timestamp query",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	// Verify the following block read was successful.
	if err := w.ccqVerifyBlockResult(nextBlockError, nextBlockResult); err != nil {
		w.ccqLogger.Debug("failed to verify next block for eth_call_by_timestamp query",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	/*
		target_block.timestamp <= target_time < following_block.timestamp
		and
		following_block_num - 1 == target_block_num
	*/

	targetBlockNum := blockResult.Number.ToInt().Uint64()
	followingBlockNum := nextBlockResult.Number.ToInt().Uint64()

	// The req.TargetTimestamp is in microseconds but EVM returns seconds. Convert to microseconds.
	targetTimestamp := uint64(blockResult.Time * 1000000)
	followingTimestamp := uint64(nextBlockResult.Time * 1000000)

	if targetBlockNum+1 != followingBlockNum {
		w.ccqLogger.Info("eth_call_by_timestamp query blocks are not adjacent",
			zap.String("requestId", requestId),
			zap.Uint64("desiredTimestamp", req.TargetTimestamp),
			zap.Uint64("targetTimestamp", targetTimestamp),
			zap.Uint64("followingTimestamp", followingTimestamp),
			zap.String("targetBlockNumber", blockResult.Number.String()),
			zap.String("followingBlockNumber", nextBlockResult.Number.String()),
			zap.String("targetBlockHash", blockResult.Hash.Hex()),
			zap.String("followingBlockHash", nextBlockResult.Hash.Hex()),
			zap.String("targetBlockTime", blockResult.Time.String()),
			zap.String("followingBlockTime", nextBlockResult.Time.String()),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	if req.TargetTimestamp < targetTimestamp || req.TargetTimestamp >= followingTimestamp {
		w.ccqLogger.Info("eth_call_by_timestamp desired timestamp falls outside of block range",
			zap.String("requestId", requestId),
			zap.Uint64("desiredTimestamp", req.TargetTimestamp),
			zap.Uint64("targetTimestamp", targetTimestamp),
			zap.Uint64("followingTimestamp", followingTimestamp),
			zap.String("targetBlockNumber", blockResult.Number.String()),
			zap.String("followingBlockNumber", nextBlockResult.Number.String()),
			zap.String("targetBlockHash", blockResult.Hash.Hex()),
			zap.String("followingBlockHash", nextBlockResult.Hash.Hex()),
			zap.String("targetBlockTime", blockResult.Time.String()),
			zap.String("followingBlockTime", nextBlockResult.Time.String()),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	// Verify all the call results and build the batch of results.
	results, err := w.ccqVerifyAndExtractQueryResults(requestId, evmCallData)
	if err != nil {
		w.ccqLogger.Info("failed to process eth_call_by_timestamp query call request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	w.ccqLogger.Info("query complete for eth_call_by_timestamp",
		zap.String("requestId", requestId),
		zap.Uint64("desiredTimestamp", req.TargetTimestamp),
		zap.Uint64("targetTimestamp", targetTimestamp),
		zap.Uint64("followingTimestamp", followingTimestamp),
		zap.String("targetBlockNumber", blockResult.Number.String()),
		zap.String("followingBlockNumber", nextBlockResult.Number.String()),
		zap.String("targetBlockHash", blockResult.Hash.Hex()),
		zap.String("followingBlockHash", nextBlockResult.Hash.Hex()),
		zap.String("targetBlockTime", blockResult.Time.String()),
		zap.String("followingBlockTime", nextBlockResult.Time.String()),
		zap.Int64("duration", time.Since(start).Milliseconds()),
	)

	// Finally, build the response and publish it.
	resp := query.EthCallByTimestampQueryResponse{
		TargetBlockNumber:    targetBlockNum,
		TargetBlockHash:      blockResult.Hash,
		TargetBlockTime:      time.Unix(int64(blockResult.Time), 0), // #nosec G115 -- This conversion is safe indefinitely
		FollowingBlockNumber: followingBlockNum,
		FollowingBlockHash:   nextBlockResult.Hash,
		FollowingBlockTime:   time.Unix(int64(nextBlockResult.Time), 0), // #nosec G115 -- This conversion is safe indefinitely
		Results:              results,
	}

	w.ccqSendQueryResponse(queryRequest, query.QuerySuccess, &resp)
}

// ccqHandleEthCallWithFinalityQueryRequest is the query handler for an eth_call_with_finality request.
func (w *Watcher) ccqHandleEthCallWithFinalityQueryRequest(ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.EthCallWithFinalityQueryRequest) {
	requestId := "eth_call:" + queryRequest.ID()
	block := req.BlockId
	w.ccqLogger.Info("received eth_call_with_finality query request",
		zap.String("requestId", requestId),
		zap.String("block", block),
		zap.String("finality", req.Finality),
		zap.Int("numRequests", len(req.CallData)),
	)

	// Validate the requested finality.
	safeMode := req.Finality == "safe"
	if req.Finality != "finalized" && !safeMode {
		w.ccqLogger.Info("invalid finality in eth_call_with_finality query request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.String("finality", req.Finality),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	// Create the block query args.
	blockMethod, callBlockArg, err := ccqCreateBlockRequest(block)
	if err != nil {
		w.ccqLogger.Info("invalid block id in eth_call_with_finality query request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
		return
	}

	// Create the batch of requested calls for the specified block.
	batch, evmCallData := ccqBuildBatchFromCallData(req, callBlockArg)

	// Add the block query to the batch.
	var blockResult connectors.BlockMarshaller
	var blockError error
	batch = append(batch, rpc.BatchElem{
		Method: blockMethod,
		Args: []interface{}{
			block,
			false, // no full transaction details
		},
		Result: &blockResult,
		Error:  blockError,
	})

	// Query the RPC.
	start := time.Now()
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = w.ethConn.RawBatchCallContext(timeout, batch)
	if err != nil {
		w.ccqLogger.Info("failed to process eth_call_with_finality query request",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	// Verify that the block read was successful.
	if err := w.ccqVerifyBlockResult(blockError, blockResult); err != nil {
		w.ccqLogger.Debug("failed to verify block for eth_call_with_finality query",
			zap.String("requestId", requestId),
			zap.String("block", block),
			zap.Any("batch", batch),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	// Get the latest block number based on the requested finality.
	var latestBlockNum uint64
	if safeMode {
		latestBlockNum = w.getLatestSafeBlockNumber()
	} else {
		latestBlockNum = w.GetLatestFinalizedBlockNumber()
	}

	// Make sure the block has reached requested finality.
	blockNumber := blockResult.Number.ToInt().Uint64()
	if blockNumber > latestBlockNum {
		w.ccqLogger.Info("requested block for eth_call_with_finality has not yet reached the requested finality",
			zap.String("requestId", requestId),
			zap.String("finality", req.Finality),
			zap.Uint64("requestedBlockNumber", blockNumber),
			zap.Uint64("latestBlockNumber", latestBlockNum),
			zap.String("blockHash", blockResult.Hash.Hex()),
			zap.String("blockTime", blockResult.Time.String()),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	// Verify all the call results and build the batch of results.
	results, err := w.ccqVerifyAndExtractQueryResults(requestId, evmCallData)
	if err != nil {
		w.ccqLogger.Info("failed to process eth_call_with_finality query call request",
			zap.String("requestId", requestId),
			zap.String("finality", req.Finality),
			zap.Uint64("requestedBlockNumber", blockNumber),
			zap.Uint64("latestBlockNumber", latestBlockNum),
			zap.String("blockHash", blockResult.Hash.Hex()),
			zap.String("blockTime", blockResult.Time.String()),
			zap.Error(err),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryRetryNeeded, nil)
		return
	}

	w.ccqLogger.Info("query complete for eth_call_with_finality",
		zap.String("requestId", requestId),
		zap.String("finality", req.Finality),
		zap.Uint64("requestedBlockNumber", blockNumber),
		zap.Uint64("latestBlockNumber", latestBlockNum),
		zap.String("blockHash", blockResult.Hash.Hex()),
		zap.String("blockTime", blockResult.Time.String()),
		zap.Int64("duration", time.Since(start).Milliseconds()),
	)

	// Finally, build the response and publish it.
	resp := query.EthCallWithFinalityQueryResponse{
		BlockNumber: blockNumber,
		Hash:        blockResult.Hash,
		Time:        time.Unix(int64(blockResult.Time), 0), // #nosec G115 -- This conversion is safe indefinitely
		Results:     results,
	}

	w.ccqSendQueryResponse(queryRequest, query.QuerySuccess, &resp)
}

// ccqCreateBlockRequest creates a block query. It parses the block string, allowing for both a block number or a block hash. Note that for now, strings like "latest", "finalized" or "safe"
// are not supported, and the block must be a hex string starting with 0x. The determination of whether it is a block number or a block hash is based on the overall length of the string,
// since a hash is 32 bytes (64 hex digits).
func ccqCreateBlockRequest(block string) (string, interface{}, error) {
	// like https://github.com/ethereum/go-ethereum/blob/master/ethclient/ethclient.go#L610

	var blockMethod string
	var callBlockArg interface{}

	if block == "" {
		return blockMethod, callBlockArg, fmt.Errorf("block id is required")
	}

	if !strings.HasPrefix(block, "0x") {
		return blockMethod, callBlockArg, fmt.Errorf("block id must start with 0x")
	}
	blk := strings.Trim(block, "0x")

	// Devnet can give us block IDs like this: "0x365".
	if len(blk)%2 != 0 {
		blk = "0" + blk
	}

	// Make sure it is valid hex.
	if _, err := hex.DecodeString(blk); err != nil {
		return blockMethod, callBlockArg, fmt.Errorf("block id is not valid hex")
	}

	if len(blk) == 64 {
		blockMethod = "eth_getBlockByHash"
		// looks like a hash which requires the object parameter
		// https://eips.ethereum.org/EIPS/eip-1898
		// https://docs.alchemy.com/reference/eth-call
		hash := eth_common.HexToHash(block)
		callBlockArg = rpc.BlockNumberOrHash{
			BlockHash:        &hash,
			RequireCanonical: true,
		}
	} else {
		blockMethod = "eth_getBlockByNumber"
		callBlockArg = block
	}

	return blockMethod, callBlockArg, nil
}

type EthCallDataIntf interface {
	CallDataList() []*query.EthCallData
}

// ccqBuildBatchFromCallData builds two slices. The first is the batch submitted to the RPC call. It contains one entry for each query plus one to query the block.
// The second is the data associated with each request (but not the block request). The index into both is the index into the request call data.
func ccqBuildBatchFromCallData(req EthCallDataIntf, callBlockArg interface{}) ([]rpc.BatchElem, []EvmCallData) {
	batch := []rpc.BatchElem{}
	evmCallData := []EvmCallData{}
	// Add each requested query to the batch.
	for _, callData := range req.CallDataList() {
		// like https://github.com/ethereum/go-ethereum/blob/master/ethclient/ethclient.go#L610
		to := eth_common.BytesToAddress(callData.To)
		data := eth_hexutil.Encode(callData.Data)
		ecd := EvmCallData{
			To:   to,
			Data: data,
			callTransactionArg: map[string]interface{}{
				"to":   to,
				"data": data,
			},
			CallResult: &eth_hexutil.Bytes{},
		}
		evmCallData = append(evmCallData, ecd)

		batch = append(batch, rpc.BatchElem{
			Method: "eth_call",
			Args: []interface{}{
				ecd.callTransactionArg,
				callBlockArg,
			},
			Result: ecd.CallResult,
			Error:  ecd.callErr,
		})
	}

	return batch, evmCallData
}

// ccqVerifyBlockResult does basic verification on the results of the block query.
func (w *Watcher) ccqVerifyBlockResult(blockError error, blockResult connectors.BlockMarshaller) error { //nolint:unparam
	if blockError != nil {
		return blockError
	}

	if blockResult.Number == nil {
		return fmt.Errorf("block result is nil")
	}

	if blockResult.Number.ToInt().Cmp(w.ccqMaxBlockNumber) > 0 {
		return fmt.Errorf("block number is too large")
	}

	return nil
}

// ccqVerifyAndExtractQueryResults verifies the array of call results and returns a vector of those results to be published.
func (w *Watcher) ccqVerifyAndExtractQueryResults(requestId string, evmCallData []EvmCallData) ([][]byte, error) {
	var err error
	results := [][]byte{}
	for idx, evmCD := range evmCallData {
		if evmCD.callErr != nil {
			return nil, fmt.Errorf("call %d failed: %w", idx, evmCD.callErr)
		}

		// Nil or Empty results are not valid eth_call will return empty when the state doesn't exist for a block
		if len(*evmCD.CallResult) == 0 {
			return nil, fmt.Errorf("call %d failed: result is empty", idx)
		}

		w.ccqLogger.Info("query call data result",
			zap.String("requestId", requestId),
			zap.Int("idx", idx),
			zap.Stringer("callData", evmCD),
		)

		results = append(results, *evmCD.CallResult)
	}

	return results, err
}

// ccqAddLatestBlock adds the latest block to the timestamp cache. The cache handles rollbacks.
func (w *Watcher) ccqAddLatestBlock(ev *connectors.NewBlock) {
	if w.ccqTimestampCache != nil {
		w.ccqTimestampCache.AddLatest(w.ccqLogger, ev.Time, ev.Number.Uint64())
	}
}
