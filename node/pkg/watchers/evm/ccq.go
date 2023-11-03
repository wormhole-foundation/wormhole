package evm

import (
	"context"
	"encoding/hex"
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

// ccqSendQueryResponseForError sends an error response back to the query handler.
func (w *Watcher) ccqSendQueryResponseForError(logger *zap.Logger, req *query.PerChainQueryInternal, status query.QueryStatus) {
	queryResponse := query.CreatePerChainQueryResponseInternal(req.RequestID, req.RequestIdx, req.Request.ChainId, status, nil)
	select {
	case w.queryResponseC <- queryResponse:
		logger.Debug("published query response error to handler", zap.String("component", "ccqevm"))
	default:
		logger.Error("failed to published query response error to handler", zap.String("component", "ccqevm"))
	}
}

func (w *Watcher) ccqHandleQuery(logger *zap.Logger, ctx context.Context, queryRequest *query.PerChainQueryInternal) {

	// This can't happen unless there is a programming error - the caller
	// is expected to send us only requests for our chainID.
	if queryRequest.Request.ChainId != w.chainID {
		panic("ccqevm: invalid chain ID")
	}

	switch req := queryRequest.Request.Query.(type) {
	case *query.EthCallQueryRequest:
		w.ccqHandleEthCallQueryRequest(logger, ctx, queryRequest, req)
	case *query.EthCallByTimestampQueryRequest:
		w.ccqHandleEthCallByTimestampQueryRequest(logger, ctx, queryRequest, req)
	case *query.EthCallWithFinalityQueryRequest:
		w.ccqHandleEthCallWithFinalityQueryRequest(logger, ctx, queryRequest, req)
	default:
		logger.Warn("received unsupported request type",
			zap.Uint8("payload", uint8(queryRequest.Request.Query.Type())),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryFatalError)
	}
}

// EvmCallData contains the details of a single query in the batch.
type EvmCallData struct {
	to                 eth_common.Address
	data               string
	callTransactionArg map[string]interface{}
	callResult         *eth_hexutil.Bytes
	callErr            error
}

func (w *Watcher) ccqHandleEthCallQueryRequest(logger *zap.Logger, ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.EthCallQueryRequest) {
	block := req.BlockId
	logger.Info("received eth_call query request",
		zap.String("block", block),
		zap.Int("numRequests", len(req.CallData)),
	)

	blockMethod, callBlockArg, err := ccqCreateBlockRequest(block)
	if err != nil {
		logger.Error("invalid block id in eth_call query request",
			zap.Error(err),
			zap.String("block", block),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryFatalError)
		return
	}

	// We build two slices. The first is the batch submitted to the RPC call. It contains one entry for each query plus one to query the block.
	// The second is the data associated with each request (but not the block request). The index into both is the index into the request call data.
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
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = w.ethConn.RawBatchCallContext(timeout, batch)

	if err != nil {
		logger.Error("failed to process eth_call query request",
			zap.Error(err),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if blockError != nil {
		logger.Error("failed to process eth_call query block request",
			zap.Error(blockError),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if blockResult.Number == nil {
		logger.Error("invalid eth_call query block result",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if blockResult.Number.ToInt().Cmp(w.ccqMaxBlockNumber) > 0 {
		logger.Error("block number too large for eth_call",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	resp := query.EthCallQueryResponse{
		BlockNumber: blockResult.Number.ToInt().Uint64(),
		Hash:        blockResult.Hash,
		Time:        time.Unix(int64(blockResult.Time), 0),
		Results:     [][]byte{},
	}

	errFound := false
	for idx := range req.CallData {
		if evmCallData[idx].callErr != nil {
			logger.Error("failed to process eth_call query call request",
				zap.Error(evmCallData[idx].callErr),
				zap.String("block", block),
				zap.Int("errorIdx", idx),
				zap.Any("batch", batch),
			)
			w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
			errFound = true
			break
		}

		// Nil or Empty results are not valid
		// eth_call will return empty when the state doesn't exist for a block
		if len(*evmCallData[idx].callResult) == 0 {
			logger.Error("invalid call result for eth_call",
				zap.String("eth_network", w.networkName),
				zap.String("block", block),
				zap.Int("errorIdx", idx),
				zap.Any("batch", batch),
			)
			w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
			errFound = true
			break
		}

		logger.Info("query result for eth_call",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.String("blockNumber", blockResult.Number.String()),
			zap.String("blockHash", blockResult.Hash.Hex()),
			zap.String("blockTime", blockResult.Time.String()),
			zap.Int("idx", idx),
			zap.String("to", evmCallData[idx].to.Hex()),
			zap.Any("data", evmCallData[idx].data),
			zap.String("result", evmCallData[idx].callResult.String()),
		)

		resp.Results = append(resp.Results, *evmCallData[idx].callResult)
	}

	if !errFound {
		queryResponse := query.CreatePerChainQueryResponseInternal(queryRequest.RequestID, queryRequest.RequestIdx, queryRequest.Request.ChainId, query.QuerySuccess, &resp)
		select {
		case w.queryResponseC <- queryResponse:
			logger.Debug("published query response error to handler", zap.String("component", "ccqevm"))
		default:
			logger.Error("failed to published query response error to handler", zap.String("component", "ccqevm"))
		}
	}
}

func (w *Watcher) ccqHandleEthCallByTimestampQueryRequest(logger *zap.Logger, ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.EthCallByTimestampQueryRequest) {
	block := req.TargetBlockIdHint
	nextBlock := req.FollowingBlockIdHint
	logger.Info("received eth_call_by_timestamp query request",
		zap.Uint64("timestamp", req.TargetTimestamp),
		zap.String("block", block),
		zap.String("nextBlock", nextBlock),
		zap.Int("numRequests", len(req.CallData)),
	)

	blockMethod, callBlockArg, err := ccqCreateBlockRequest(block)
	if err != nil {
		logger.Error("invalid target block id hint in eth_call_by_timestamp query request",
			zap.Error(err),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryFatalError)
		return
	}

	nextBlockMethod, _, err := ccqCreateBlockRequest(nextBlock)
	if err != nil {
		logger.Error("invalid following block id hint in eth_call_by_timestamp query request",
			zap.Error(err),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryFatalError)
		return
	}

	// We build two slices. The first is the batch submitted to the RPC call. It contains one entry for each query plus one to query the block and one for the next block.
	// The second is the data associated with each request (but not the block requests). The index into both is the index into the request call data.
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
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = w.ethConn.RawBatchCallContext(timeout, batch)

	if err != nil {
		logger.Error("failed to process eth_call_by_timestamp query request",
			zap.Error(err),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	// Checks on the target block.
	if blockError != nil {
		logger.Error("failed to process eth_call_by_timestamp query target block request",
			zap.Error(blockError),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if blockResult.Number == nil {
		logger.Error("invalid eth_call_by_timestamp query target block result",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if blockResult.Number.ToInt().Cmp(w.ccqMaxBlockNumber) > 0 {
		logger.Error("target block number too large for eth_call_by_timestamp",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	// Checks on the following block.
	if nextBlockError != nil {
		logger.Error("failed to process eth_call_by_timestamp query following block request",
			zap.Error(nextBlockError),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if nextBlockResult.Number == nil {
		logger.Error("invalid eth_call_by_timestamp query following block result",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if nextBlockResult.Number.ToInt().Cmp(w.ccqMaxBlockNumber) > 0 {
		logger.Error("following block number too large for eth_call_by_timestamp",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.String("nextBlock", nextBlock),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
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
		logger.Error(" eth_call_by_timestamp query blocks are not adjacent",
			zap.String("eth_network", w.networkName),
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
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryFatalError)
		return
	}

	if req.TargetTimestamp < targetTimestamp || req.TargetTimestamp >= followingTimestamp {
		logger.Error(" eth_call_by_timestamp desired timestamp falls outside of block range",
			zap.String("eth_network", w.networkName),
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
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryFatalError)
		return
	}

	resp := query.EthCallByTimestampQueryResponse{
		TargetBlockNumber:    targetBlockNum,
		TargetBlockHash:      blockResult.Hash,
		TargetBlockTime:      time.Unix(int64(blockResult.Time), 0),
		FollowingBlockNumber: followingBlockNum,
		FollowingBlockHash:   nextBlockResult.Hash,
		FollowingBlockTime:   time.Unix(int64(nextBlockResult.Time), 0),
		Results:              [][]byte{},
	}

	errFound := false
	for idx := range req.CallData {
		if evmCallData[idx].callErr != nil {
			logger.Error("failed to process eth_call_by_timestamp query call request",
				zap.Error(evmCallData[idx].callErr),
				zap.String("block", block),
				zap.String("nextBlock", nextBlock),
				zap.Int("errorIdx", idx),
				zap.Any("batch", batch),
			)
			w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
			errFound = true
			break
		}

		// Nil or Empty results are not valid
		// eth_call will return empty when the state doesn't exist for a block
		if len(*evmCallData[idx].callResult) == 0 {
			logger.Error("invalid call result for eth_call_by_timestamp",
				zap.String("eth_network", w.networkName),
				zap.String("block", block),
				zap.String("nextBlock", nextBlock),
				zap.Int("errorIdx", idx),
				zap.Any("batch", batch),
			)
			w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
			errFound = true
			break
		}

		logger.Info(" eth_call_by_timestamp query result",
			zap.String("eth_network", w.networkName),
			zap.Uint64("desiredTimestamp", req.TargetTimestamp),
			zap.Uint64("targetTimestamp", targetTimestamp),
			zap.Uint64("followingTimestamp", followingTimestamp),
			zap.String("targetBlockNumber", blockResult.Number.String()),
			zap.String("followingBlockNumber", nextBlockResult.Number.String()),
			zap.String("targetBlockHash", blockResult.Hash.Hex()),
			zap.String("followingBlockHash", nextBlockResult.Hash.Hex()),
			zap.String("targetBlockTime", blockResult.Time.String()),
			zap.String("followingBlockTime", nextBlockResult.Time.String()),
			zap.Int("idx", idx),
			zap.String("to", evmCallData[idx].to.Hex()),
			zap.Any("data", evmCallData[idx].data),
			zap.String("result", evmCallData[idx].callResult.String()),
		)

		resp.Results = append(resp.Results, *evmCallData[idx].callResult)
	}

	if !errFound {
		queryResponse := query.CreatePerChainQueryResponseInternal(queryRequest.RequestID, queryRequest.RequestIdx, queryRequest.Request.ChainId, query.QuerySuccess, &resp)
		select {
		case w.queryResponseC <- queryResponse:
			logger.Debug("published query response error to handler", zap.String("component", "ccqevm"))
		default:
			logger.Error("failed to published query response error to handler", zap.String("component", "ccqevm"))
		}
	}
}

func (w *Watcher) ccqHandleEthCallWithFinalityQueryRequest(logger *zap.Logger, ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.EthCallWithFinalityQueryRequest) {
	block := req.BlockId
	logger.Info("received eth_call_with_finality query request",
		zap.String("block", block),
		zap.String("finality", req.Finality),
		zap.Int("numRequests", len(req.CallData)),
	)

	safeMode := req.Finality == "safe"
	if req.Finality != "finalized" && !safeMode {
		logger.Error("invalid finality in eth_call_with_finality query request", zap.String("block", block), zap.String("finality", req.Finality), zap.String("block", block))
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryFatalError)
		return
	}

	blockMethod, callBlockArg, err := ccqCreateBlockRequest(block)
	if err != nil {
		logger.Error("invalid block id in eth_call_with_finality query request",
			zap.Error(err),
			zap.String("block", block),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryFatalError)
		return
	}

	// We build two slices. The first is the batch submitted to the RPC call. It contains one entry for each query plus one to query the block.
	// The second is the data associated with each request (but not the block request). The index into both is the index into the request call data.
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
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = w.ethConn.RawBatchCallContext(timeout, batch)

	if err != nil {
		logger.Error("failed to process eth_call_with_finality query request",
			zap.Error(err),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if blockError != nil {
		logger.Error("failed to process eth_call_with_finality query block request",
			zap.Error(blockError),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if blockResult.Number == nil {
		logger.Error("invalid eth_call_with_finality query block result",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	if blockResult.Number.ToInt().Cmp(w.ccqMaxBlockNumber) > 0 {
		logger.Error("block number too large for eth_call_with_finality",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	blockNumber := blockResult.Number.ToInt().Uint64()
	var latestBlockNum uint64
	if safeMode {
		latestBlockNum = w.getLatestSafeBlockNumber()
	} else {
		latestBlockNum = w.GetLatestFinalizedBlockNumber()
	}

	if blockNumber > latestBlockNum {
		logger.Info("requested block for eth_call_with_finality has not yet reached the requested finality",
			zap.String("finality", req.Finality),
			zap.Uint64("requestedBlockNumber", blockNumber),
			zap.Uint64("latestBlockNumber", latestBlockNum),
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.Any("batch", batch),
		)
		w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
		return
	}

	resp := query.EthCallWithFinalityQueryResponse{
		BlockNumber: blockNumber,
		Hash:        blockResult.Hash,
		Time:        time.Unix(int64(blockResult.Time), 0),
		Results:     [][]byte{},
	}

	errFound := false
	for idx := range req.CallData {
		if evmCallData[idx].callErr != nil {
			logger.Error("failed to process eth_call_with_finality query call request",
				zap.Error(evmCallData[idx].callErr),
				zap.String("block", block),
				zap.Int("errorIdx", idx),
				zap.Any("batch", batch),
			)
			w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
			errFound = true
			break
		}

		// Nil or Empty results are not valid
		// eth_call_with_finality will return empty when the state doesn't exist for a block
		if len(*evmCallData[idx].callResult) == 0 {
			logger.Error("invalid call result for eth_call_with_finality",
				zap.String("eth_network", w.networkName),
				zap.String("block", block),
				zap.Int("errorIdx", idx),
				zap.Any("batch", batch),
			)
			w.ccqSendQueryResponseForError(logger, queryRequest, query.QueryRetryNeeded)
			errFound = true
			break
		}

		logger.Info("query result for eth_call_with_finality",
			zap.String("eth_network", w.networkName),
			zap.String("block", block),
			zap.String("finality", req.Finality),
			zap.Uint64("requestedBlockNumber", blockNumber),
			zap.Uint64("latestBlockNumber", latestBlockNum),
			zap.String("blockHash", blockResult.Hash.Hex()),
			zap.String("blockTime", blockResult.Time.String()),
			zap.Int("idx", idx),
			zap.String("to", evmCallData[idx].to.Hex()),
			zap.Any("data", evmCallData[idx].data),
			zap.String("result", evmCallData[idx].callResult.String()),
		)

		resp.Results = append(resp.Results, *evmCallData[idx].callResult)
	}

	if !errFound {
		queryResponse := query.CreatePerChainQueryResponseInternal(queryRequest.RequestID, queryRequest.RequestIdx, queryRequest.Request.ChainId, query.QuerySuccess, &resp)
		select {
		case w.queryResponseC <- queryResponse:
			logger.Debug("published query response error to handler", zap.String("component", "ccqevm"))
		default:
			logger.Error("failed to published query response error to handler", zap.String("component", "ccqevm"))
		}
	}
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

func ccqBuildBatchFromCallData(req EthCallDataIntf, callBlockArg interface{}) ([]rpc.BatchElem, []EvmCallData) {
	batch := []rpc.BatchElem{}
	evmCallData := []EvmCallData{}
	// Add each requested query to the batch.
	for _, callData := range req.CallDataList() {
		// like https://github.com/ethereum/go-ethereum/blob/master/ethclient/ethclient.go#L610
		to := eth_common.BytesToAddress(callData.To)
		data := eth_hexutil.Encode(callData.Data)
		ecd := EvmCallData{
			to:   to,
			data: data,
			callTransactionArg: map[string]interface{}{
				"to":   to,
				"data": data,
			},
			callResult: &eth_hexutil.Bytes{},
		}
		evmCallData = append(evmCallData, ecd)

		batch = append(batch, rpc.BatchElem{
			Method: "eth_call",
			Args: []interface{}{
				ecd.callTransactionArg,
				callBlockArg,
			},
			Result: ecd.callResult,
			Error:  ecd.callErr,
		})
	}

	return batch, evmCallData
}
