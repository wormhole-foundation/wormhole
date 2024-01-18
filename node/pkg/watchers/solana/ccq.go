package solana

import (
	"context"
	"encoding/hex"
	"time"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
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
		w.ccqHandleSolanaAccountQueryRequest(ctx, queryRequest, req)
	default:
		w.ccqLogger.Warn("received unsupported request type",
			zap.Uint8("payload", uint8(queryRequest.Request.Query.Type())),
		)
		w.ccqSendQueryResponse(queryRequest, query.QueryFatalError, nil)
	}

	query.TotalWatcherTime.WithLabelValues(w.chainID.String()).Observe(float64(time.Since(start).Milliseconds()))
}

// ccqHandleSolanaAccountQueryRequest is the query handler for a sol_account request.
func (w *SolanaWatcher) ccqHandleSolanaAccountQueryRequest(ctx context.Context, queryRequest *query.PerChainQueryInternal, req *query.SolanaAccountQueryRequest) {
	requestId := "sol_account:" + queryRequest.ID()
	w.ccqLogger.Info("received a sol_account query", zap.String("requestId", requestId))

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
	info, err := w.rpcClient.GetMultipleAccountsWithOpts(rCtx, accounts, &params)
	if err != nil {
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
		zap.Any("blockHeight", block.BlockHeight),
	)

	w.ccqSendQueryResponse(queryRequest, query.QuerySuccess, resp)
}

/*
func (s *SolanaWatcher) testQuery(ctx context.Context, logger *zap.Logger) {
	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	acc, err := solana.PublicKeyFromBase58("Jito4APyf642JPZPx3hGc6WWJ8zPKtRbRs4P815Awbb")
	if err != nil {
		logger.Error("SOLTEST: failed to parse account", zap.Error(err))
		return
	}

	slot := uint64(239676280)
	logger.Info("SOLTEST: doing read", zap.Any("commitment", s.commitment), zap.Any("account", acc), zap.Uint64("slot", slot))
	info, err := s.rpcClient.GetAccountInfoWithOpts(rCtx, acc, &rpc.GetAccountInfoOpts{
		Encoding:       solana.EncodingBase64,
		Commitment:     s.commitment,
		MinContextSlot: &slot,
	})

	if err != nil {
		logger.Error("SOLTEST: read failed", zap.Error(err))
		return
	}

	logger.Info("SOLTEST: read succeeded", zap.Uint64("slot", info.Context.Slot), zap.Any("data", info.Value.Data))
}
*/
