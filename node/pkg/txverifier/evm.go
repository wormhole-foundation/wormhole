package txverifier

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	geth "github.com/ethereum/go-ethereum/core/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"
)

const (
	// Seconds to wait before trying to reconnect to the core contract event subscription.
	RECONNECT_DELAY = 5 * time.Second
)

// TODO: More errors should be converted into sentinel errors instead of being created and returned in-line.
var (
	ErrBadWrappedAssetQuery     = errors.New("asset is wrapped but wrappedAsset() call returned zero address")
	ErrCouldNotGetNativeDetails = errors.New("could not parse native details for transfer")
	ErrInvalidReceiptArgument   = errors.New("invalid TransferReceipt argument")
	ErrInvalidUpsertArgument    = errors.New("nil argument passed to upsert")
	ErrNoMsgsFromTokenBridge    = errors.New("no messages published from Token Bridge")
	ErrNotTransfer              = errors.New("payload is not a token transfer")
	ErrParsedReceiptIsNil       = errors.New("nil receipt after parsing")
	ErrReceiptHasNoMsgPub       = errors.New("receipt has no LogMessagePublished events")
	ErrTxHashIsZeroAddr         = errors.New("txHash is the zero address")
)

// Custom error type used to signal that a core invariant of the token bridge has been violated.
type InvariantError struct {
	Msg string
}

func (i InvariantError) Error() string {
	return fmt.Sprintf("invariant violated: %s", i.Msg)
}

// TransferIsValid processes a token transfer receipt based on a LogMessagePublished event. It fetches
// the full transaction receipt associated with the txHash, and parses all
// events emitted in the transaction, tracking LogMessagePublished events as outbound
// transfers and token deposits into the token bridge as inbound transfers. It then
// verifies that the sum of the inbound transfers is at least as much as the sum of
// the outbound transfers.
//
// Return values:
//
//	true:  the token transfer's receipt is valid.
//	false and nil: the token transfer has violated an invariant and is unsafe.
//	false and err: the receipt could not be properly processed or is not a token transfer.
func (tv *TransferVerifier[ethClient, Connector]) TransferIsValid(
	ctx context.Context,
	txHash common.Hash,
	// If nil, this code will fetch the receipt using the TransferVerifier's connector.
	receipt *geth.Receipt,
) (bool, error) {

	// Prune old transaction information from the cache.
	tv.pruneCache()

	if Cmp(txHash, ZERO_ADDRESS) == 0 {
		tv.logger.Error("txHash is the zero address")
		return false, ErrTxHashIsZeroAddr
	}

	tv.logger.Debug("detected LogMessagePublished event",
		zap.String("txHash", txHash.String()))

	// Caching: record used/inspected tx hash.
	if eval, exists := tv.evaluations[txHash]; exists {
		tv.logger.Debug("skip: transaction hash already processed",
			zap.String("txHash", txHash.String()))
		return eval.Result, eval.Err
	}

	// Get the full transaction receipt for this txHash if it was not provided as an argument.
	if receipt == nil {
		tv.logger.Debug("receipt was not passed as an argument. fetching it using the connector")
		var txReceiptErr error
		receipt, txReceiptErr = tv.evmConnector.TransactionReceipt(ctx, txHash)

		if txReceiptErr != nil {
			tv.addToCache(txHash, &evaluation{receipt, false, txReceiptErr})
			return false, txReceiptErr
		}
	}

	eval := evaluation{Receipt: receipt}

	// Caching: record a new lastBlockNumber and add the receipt to the cache.
	if receipt.BlockNumber != nil {
		tv.lastBlockNumber = receipt.BlockNumber.Uint64()
	}

	// Parse raw transaction receipt into high-level struct containing transfer details.
	transferReceipt, parseErr := tv.parseReceipt(receipt)

	if parseErr != nil {
		eval.Err = parseErr
		tv.addToCache(txHash, &eval)
		return false, parseErr
	}

	// ParseReceipt should only return nil when there is also an error, so we don't expect to get here.
	if transferReceipt == nil {
		eval.Err = ErrParsedReceiptIsNil
		tv.addToCache(txHash, &eval)
		return false, ErrParsedReceiptIsNil
	}

	// Invalid receipt: no message publications
	if len(*transferReceipt.MessagePublications) == 0 {
		eval.Err = ErrNoMsgsFromTokenBridge
		tv.addToCache(txHash, &eval)
		return false, ErrNoMsgsFromTokenBridge
	}

	// Add wormhole-specific data to the receipt by making
	// RPC calls for data that is not included in the logs,
	// such as a token's native address and its decimals.
	updateErr := tv.updateReceiptDetails(transferReceipt)
	if updateErr != nil {
		eval.Err = updateErr
		tv.addToCache(txHash, &eval)
		return false, updateErr
	}

	// Ensure that the amount coming in is at least as much as the amount requested out.
	summary, processErr := tv.processReceipt(transferReceipt)
	if summary != nil {
		tv.logger.Debug("finished processing receipt", zap.String("txHash", txHash.String()), zap.String("summary", summary.String()))
	} else {
		tv.logger.Debug("finished processing receipt (but summary is nil)", zap.String("txHash", txHash.String()))
	}

	if processErr != nil {
		eval.Err = processErr
		tv.addToCache(txHash, &eval)

		var invError *InvariantError
		if !errors.Is(processErr, invError) {
			// The receipt couldn't be parsed properly.
			return false, processErr
		}

		// This represents an invariant violation in token transfers.
		tv.logger.Error("invariant violation", zap.Error(processErr), zap.String("receipt summary", summary.String()))
		return false, nil
	}

	// Update statistics
	if summary.logsProcessed == 0 {
		tv.logger.Warn("receipt logs empty for tx", zap.String("txHash", txHash.Hex()))
	}

	// Cache results
	eval.Err = nil
	eval.Result = true
	tv.addToCache(txHash, &eval)

	return true, nil
}

func (tv *TransferVerifier[ethClient, Connector]) pruneCache() {
	// Prune the cache of processed receipts
	numPrunedEvals := int(0)

	// Iterate over recorded transaction hashes, and clear receipts older than `pruneDelta` blocks
	for hash, eval := range tv.evaluations {
		if eval.Receipt != nil {
			if eval.Receipt.BlockNumber.Uint64() < tv.lastBlockNumber-tv.pruneHeightDelta {
				numPrunedEvals++
				delete(tv.evaluations, hash)
			}
		} else {
			// NOTE: Kind of sloppy to delete these right away, but we shouldn't be caching
			// many nil receipts anyway.
			numPrunedEvals++
			delete(tv.evaluations, hash)
		}
	}

	if numPrunedEvals > 0 {
		tv.logger.Debug("pruned cached receipt evaluations",
			zap.Int("numPruned", numPrunedEvals),
			zap.Int("cacheSize", len(tv.evaluations)),
		)
	}
	// TODO: The other caches should be pruned here too.
}

// updateReceiptDetails performs additional processing on the raw data that has been parsed. This
// consists of checking whether assets are wrapped. If so, unwrap the
// assets and fetch information about the origin chain, origin address,
// and token decimals.
//
// All of this information is required to determine
// whether the amounts deposited into the token bridge match the amount
// that was requested out.
//
// This is done separately from parsing step so
// that RPC calls are done independently of parsing code, which
// facilitates testing.
//
// Updates the receipt argument directly.
func (tv *TransferVerifier[ethClient, Connector]) updateReceiptDetails(
	receipt *TransferReceipt,
) error {
	if receipt == nil {
		return ErrInvalidReceiptArgument
	}

	invalidErr := receipt.Validate()
	if invalidErr != nil {
		return errors.Join(
			ErrInvalidReceiptArgument,
			invalidErr,
		)
	}

	tv.logger.Debug(
		"updating details for receipt",
		zap.String("receiptRaw", receipt.String()),
	)

	if len(*receipt.Deposits) > 0 {
		tv.logger.Debug("populating details for deposits")
	} else {
		tv.logger.Debug("no deposits in receipt")
	}
	for i, deposit := range *receipt.Deposits {
		tv.logger.Debug("processing deposit", zap.Int("index", i))

		// Update Amount
		decimals, decimalErr := tv.getDecimals(deposit.TokenAddress)
		if decimalErr != nil {
			return decimalErr
		}
		normalize := normalize(deposit.Amount, decimals)
		deposit.Amount = normalize
	}

	// Populate the origin chain ID and address fields for ERC20 transfers so that they can be compared against the destination asset in the
	// LogMessagePublished payload.
	if len(*receipt.Transfers) > 0 {
		tv.logger.Debug("populating details for ERC20 Transfers")
	} else {
		tv.logger.Debug("no deposits in receipt")
	}
	for i, transfer := range *receipt.Transfers {
		tv.logger.Debug("processing transfer", zap.Int("index", i))
		isWrapped, isWrappedErr := tv.isWrappedAsset(transfer.TokenAddress)
		if isWrappedErr != nil {
			return errors.Join(errors.New("isWrapped: Can't get isWrappedAsset result for token. Can't continue to process this receipt"), isWrappedErr)
		}

		var (
			queryAddr   = transfer.TokenAddress
			originChain vaa.ChainID
			originAddr  vaa.Address
		)
		// Handle and return early for native assets.
		if !isWrapped {
			tv.logger.Debug("token is native (not wrapped)")
			originAddr = VAAAddrFrom(transfer.TokenAddress)
			originChain = tv.chainIds.wormholeChainId
		} else {
			queryAddr := transfer.TokenAddress

			// Update ChainID
			var fetchErr error
			originChain, fetchErr = tv.chainId(queryAddr)
			if fetchErr != nil {
				return errors.Join(errors.New("chainId: Can't get chainId for token. Can't continue to process this receipt"), fetchErr)
			}
			if originChain == vaa.ChainIDUnset {
				return errors.Join(errors.New("chainId: chainId for token is unset after query. Can't continue to process this receipt"))
			}

			// Find the origin address by querying the wrapped asset address.
			nativeAddr, err := tv.nativeContract(queryAddr)
			if err != nil {
				return errors.Join(errors.New("chainId: chainId for token is unset after query. Can't continue to process this receipt"))
			}

			tv.logger.Debug("got origin address from nativeContract() call",
				zap.String("originAddress", nativeAddr.String()),
			)
			originAddr = nativeAddr
		}

		// In both the wrapped and unwrapped case, the decimals are required at this point
		// so that the amount can be compared against the normalized amount in the LogMessagePublished event.
		decimals, decimalErr := tv.getDecimals(queryAddr)
		if decimalErr != nil {
			return decimalErr
		}
		normalized := normalize(transfer.Amount, decimals)

		// This shouldn't happen and likely indicates a programming error.
		if Cmp(originAddr, ZERO_ADDRESS) == 0 {
			tv.logger.Error("origin addr is zero")
			return ErrCouldNotGetNativeDetails
		}

		if originChain == vaa.ChainIDUnset {
			tv.logger.Error("origin chain is zero")
			return ErrCouldNotGetNativeDetails
		}

		transfer.Amount = normalized
		transfer.OriginAddr = originAddr
		transfer.TokenChain = originChain

		tv.logger.Debug("updated info for token transfer",
			zap.String("tokenAddr", transfer.TokenAddress.String()),
			zap.Uint16("new chainID", uint16(originChain)),
			zap.String("chain name", originChain.String()),
			zap.String("normalizedAmount", normalized.String()))
	}

	// No processing required for LogMessagePublished. We are comparing against its origin address
	// and Amount field (which is always normalized).

	// Populate Amount and OriginAddress based on raw byte info.
	// The unwrapped address and the denormalized amount are necessary for checking
	// that the amount matches.
	// tv.logger.Debug("populating native data for LogMessagePublished assets")
	// for i, message := range *receipt.MessagePublications {
	// 	tv.logger.Debug("processing message publication", zap.Int("index", i))
	// 	newDetails, logFetchErr := tv.fetchLogMessageDetails(message.TransferDetails)
	// 	if logFetchErr != nil {
	// 		return errors.Join(errors.New("error when populating wormhole details. cannot verify receipt"), logFetchErr)
	// 	}
	// 	if newDetails == nil {
	// 		errMsg := "fetchLogMessageDetails returned nil but did not return error. cannot continue"
	// 		tv.logger.Error(errMsg, zap.String("message", message.String()))
	// 		return errors.New(errMsg)
	// 	}
	// 	message.TransferDetails = newDetails
	// }

	tv.logger.Debug(
		"finished updating receipt details",
		zap.String("receipt", receipt.String()),
	)

	return nil
}

// ParseReceipt converts a go-ethereum receipt struct into a TransferReceipt.
// It makes use of the ethConnector to parse information from the logs within
// the receipt. This function is mainly helpful to isolate the parsing code
// from the verification logic, which makes the latter easier to test without
// needing an active RPC connection.

// This function parses only events with topics needed for Transfer
// Verification. Any other events will be discarded.
// This function is not responsible for checking that the values for the
// various fields are relevant, only that they are well-formed.
//
// This function must return nil TransferReceipt when an error occurs.
func (tv *TransferVerifier[evmClient, connector]) parseReceipt(
	receipt *geth.Receipt,
) (*TransferReceipt, error) {

	// Sanity checks.
	if receipt == nil {
		return nil, ErrInvalidReceiptArgument
	}

	tv.logger.Debug(
		"begin processing receipt",
		zap.String("txHash", receipt.TxHash.String()),
	)

	if receipt.Status != 1 {
		return nil, errors.Join(
			ErrInvalidReceiptArgument,
			errors.New("non-success transaction status"),
		)
	}
	if len(receipt.Logs) == 0 {
		return nil, errors.Join(
			ErrInvalidReceiptArgument,
			errors.New("no logs in receipt"),
		)
	}

	var (
		deposits            []*NativeDeposit
		transfers           []*ERC20Transfer
		messagePublications []*LogMessagePublished
		receiptErr          error
	)
	for _, log := range receipt.Logs {
		// Bounds check.
		if len(log.Topics) == 0 {
			tv.logger.Debug(
				"skipping log: no indexed topics",
				zap.String("txHash", log.TxHash.String()),
			)
			continue
		}

		switch log.Topics[0] {
		case common.HexToHash(EVENTHASH_WETH_DEPOSIT):
			deposit, depositErr := DepositFromLog(log, tv.chainIds.wormholeChainId)

			if depositErr != nil {
				tv.logger.Error("error when parsing Deposit from log",
					zap.Error(depositErr),
					zap.String("txHash", log.TxHash.String()),
				)
				receiptErr = errors.Join(receiptErr, depositErr)
				continue
			}

			tv.logger.Debug("adding deposit", zap.String("deposit", deposit.String()))
			deposits = append(deposits, deposit)

		// Process ERC20 Transfers.
		case common.HexToHash(EVENTHASH_ERC20_TRANSFER):

			transfer, transferErr := ERC20TransferFromLog(log, tv.chainIds.wormholeChainId)

			if transferErr != nil {
				tv.logger.Error("error when parsing ERC20 Transfer from log",
					zap.Error(transferErr),
					zap.String("txHash", log.TxHash.String()),
				)
				receiptErr = errors.Join(receiptErr, transferErr)
				continue
			}

			// Log when the zero address is used in non-obvious ways.
			if transfer.From == ZERO_ADDRESS {
				tv.logger.Info("transfer's From field is the zero address. This is likely a mint operation",
					zap.String("txHash", log.TxHash.String()),
				)
			}
			if transfer.To == ZERO_ADDRESS {
				tv.logger.Info("transfer's To field is the zero address. This is likely a burn operation",
					zap.String("txHash", log.TxHash.String()),
				)
			}

			tv.logger.Debug("adding transfer", zap.String("transfer", transfer.String()))
			transfers = append(transfers, transfer)

		// Process LogMessagePublished events.
		case common.HexToHash(EVENTHASH_WORMHOLE_LOG_MESSAGE_PUBLISHED):
			if len(log.Data) == 0 {
				receiptErr = errors.Join(receiptErr, errors.New("receipt data has length 0"))
				continue
			}

			logMessagePublished, parseLogErr := tv.evmConnector.ParseLogMessagePublished(*log)
			if parseLogErr != nil {
				tv.logger.Error("failed to parse LogMessagePublished event")
				receiptErr = errors.Join(receiptErr, parseLogErr)
				continue
			}

			// This check is required. Payload parsing will fail if performed on a message emitted from another contract or sent
			// by a contract other than the token bridge
			if log.Address != tv.Addresses.CoreBridgeAddr {
				tv.logger.Debug("skip: LogMessagePublished not emitted from the core bridge",
					zap.String("emitter", log.Address.String()))
				continue
			}

			// If there is no payload, then there's no point in further processing.
			// This should never happen.
			if len(logMessagePublished.Payload) == 0 {
				emptyErr := errors.New("a LogMessagePayload event from the token bridge was received with a zero-sized payload")
				tv.logger.Error(
					"issue parsing receipt",
					zap.Error(emptyErr),
					zap.String("txhash", log.TxHash.String()))
				receiptErr = errors.Join(receiptErr, emptyErr)
				continue
			}

			// Only token transfers are relevant (not attestations or any other payload type).
			if !vaa.IsTransfer(logMessagePublished.Payload) {
				tv.logger.Info("skip: LogMessagePublished is not a token transfer",
					zap.String("txHash", log.TxHash.String()),
					zap.String("payloadByte", fmt.Sprintf("%x", logMessagePublished.Payload[0])),
				)
				continue
			}

			// Bounds check.
			if len(log.Topics) < 2 {
				tv.logger.Warn("skip: LogMessagePublished has less than 2 topics",
					zap.String("txhash", log.TxHash.String()),
				)
				receiptErr = errors.Join(receiptErr, errors.New("not enough topics"))
				continue
			}

			if log.Topics[1] != tv.Addresses.TokenBridgeAddr.Hash() {
				tv.logger.Debug("skip: LogMessagePublished with sender not equal to the token bridge",
					zap.String("sender", log.Topics[1].String()),
					zap.String("tokenBridgeAddr", tv.Addresses.TokenBridgeAddr.Hex()),
				)
				continue
			}

			// Validation is complete. Now, parse the raw bytes of the payload into a TransferDetails instance.
			transferDetails, parsePayloadErr := parseLogMessagePublishedPayload(logMessagePublished.Payload)
			if parsePayloadErr != nil {
				receiptErr = errors.Join(receiptErr, parsePayloadErr)
				continue
			}

			if transferDetails == nil {
				tv.logger.Debug("skip: parsed successfully but no relevant transfer found",
					zap.String("txhash", log.TxHash.String()))
				continue
			}

			// If everything went well, append the message publication
			messagePublications = append(messagePublications, &LogMessagePublished{
				EventEmitter:    log.Address,
				MsgSender:       logMessagePublished.Sender,
				TransferDetails: transferDetails,
			})

		}
	}

	if len(messagePublications) == 0 {
		receiptErr = errors.Join(receiptErr, ErrNoMsgsFromTokenBridge)
	}

	if receiptErr != nil {
		return nil, receiptErr
	}

	return &TransferReceipt{
			Deposits:            &deposits,
			Transfers:           &transfers,
			MessagePublications: &messagePublications},
		nil
}

// processReceipt verifies that a receipt for a LogMessagedPublished event does
// not verify a fundamental invariant of Wormhole token transfers: when the
// core bridge reports a transfer has occurred, there must be a corresponding
// transfer in the token bridge. This is determined by iterating through the
// logs of the receipt and ensuring that the sum transferred into the token
// bridge does not exceed the sum emitted by the core bridge.
//
// This function makes use of [InvariantError] to report invalid token transfers. Callers
// can match on this error type to determine whether the transfer was safe or not.
//
// Returns a summary of the events that were processed, or nil when a parsing error occurs.
func (tv *TransferVerifier[evmClient, connector]) processReceipt(
	receipt *TransferReceipt,
) (*ReceiptSummary, error) {

	// Sanity checks.
	if receipt == nil {
		return nil, ErrInvalidReceiptArgument
	}

	invalidErr := receipt.Validate()
	if invalidErr != nil {
		return nil, errors.Join(
			ErrInvalidReceiptArgument,
			invalidErr,
		)
	}

	tv.logger.Debug("beginning to process receipt",
		zap.String("receipt", receipt.String()),
	)

	summary := NewReceiptSummary()

	if len(*receipt.MessagePublications) == 0 {
		return nil, ErrNoMsgsFromTokenBridge
	}

	if len(*receipt.Deposits) == 0 && len(*receipt.Transfers) == 0 {
		// This should result in an invariant error below but this is helpful context.
		tv.logger.Warn("receipt is a token transfer but has no deposits or transfers",
			zap.String("receipt", receipt.String()),
		)
	}

	// Process NativeDeposits
	for _, deposit := range *receipt.Deposits {

		validateErr := validate[*NativeDeposit](deposit)
		if validateErr != nil {
			return nil, validateErr
		}

		key, relevant := relevant[*NativeDeposit](deposit, tv.Addresses)
		if !relevant {
			tv.logger.Debug("skip: irrelevant deposit",
				zap.String("emitter", deposit.Emitter().String()),
				zap.String("deposit", deposit.String()),
			)
			continue
		}
		if key == "" {
			return nil, errors.New("couldn't get key for deposit")
		}

		upsertErr := upsert(&summary.in, key, deposit.TransferAmount())
		if upsertErr != nil {
			return nil, upsertErr
		}

		tv.logger.Debug("a deposit into the token bridge was recorded",
			zap.String("tokenAddress", deposit.TokenAddress.String()),
			zap.String("amount", deposit.Amount.String()))
	}

	// Process ERC20Transfers
	for _, transfer := range *receipt.Transfers {
		validateErr := validate[*ERC20Transfer](transfer)
		if validateErr != nil {
			return nil, validateErr
		}

		key, relevant := relevant[*ERC20Transfer](transfer, tv.Addresses)
		if !relevant {
			tv.logger.Debug("skip: transfer's destination is not the token bridge",
				zap.String("emitter", transfer.Emitter().String()),
				zap.String("erc20Transfer", transfer.String()))
			continue
		}
		if key == "" {
			return nil, errors.New("couldn't get key for transfer")
		}

		upsertErr := upsert(&summary.in, key, transfer.TransferAmount())
		if upsertErr != nil {
			return nil, upsertErr
		}

		tv.logger.Debug("identified ERC20 transfer to the token bridge",
			zap.String("tokenAddress", transfer.TokenAddress.String()),
			zap.String("amount", transfer.Amount.String()))
	}

	// Process LogMessagePublished events.
	for _, message := range *receipt.MessagePublications {
		td := message.TransferDetails

		validateErr := validate[*LogMessagePublished](message)
		if validateErr != nil {
			return nil, validateErr
		}

		key, relevant := relevant[*LogMessagePublished](message, tv.Addresses)
		if !relevant {
			tv.logger.Debug("skip: irrelevant LogMessagePublished event")
			continue
		}

		upsertErr := upsert(&summary.out, key, message.TransferAmount())
		if upsertErr != nil {
			return nil, upsertErr
		}

		tv.logger.Debug("successfully parsed a LogMessagePublished event payload",
			zap.String("tokenAddress", td.OriginAddress.String()),
			zap.String("tokenChain", td.TokenChain.String()),
			zap.String("amount", td.Amount.String()))

		summary.logsProcessed++
	}

	tv.logger.Debug("done building receipt summary", zap.String("summary", summary.String()))

	var err error
	for key, amountOut := range summary.out {
		if amountIn, exists := summary.in[key]; !exists {
			err = &InvariantError{Msg: "transfer-out request for tokens that were never deposited"}
		} else {
			if amountOut.Cmp(amountIn) == 1 {
				err = &InvariantError{Msg: "requested amount out is larger than amount in"}
			}

			// Normally the amounts should be equal. This case indicates
			// an unusual transfer or else a bug in the program.
			if amountOut.Cmp(amountIn) == -1 {
				tv.logger.Info("requested amount in is larger than amount out.",
					zap.String("key", key),
					zap.String("amountIn", amountIn.String()),
					zap.String("amountOut", amountOut.String()),
				)
			}

			tv.logger.Debug("bridge request processed",
				zap.String("key", key),
				zap.String("amountOut", amountOut.String()),
				zap.String("amountIn", amountIn.String()))
		}
	}

	return summary, err
}

// parseLogMessagePublishedPayload parses the details of a transfer from a
// LogMessagePublished event's Payload field.
func parseLogMessagePublishedPayload(
	// Corresponds to LogMessagePublished.Payload as returned by the ABI parsing operation in the ethConnector.
	data []byte,
) (*TransferDetails, error) {
	if !vaa.IsTransfer(data) {
		return nil, ErrNotTransfer
	}

	// Note: vaa.DecodeTransferPayloadHdr performs validation on data, e.g. length checks.
	hdr, err := vaa.DecodeTransferPayloadHdr(data)
	if err != nil {
		return nil, errors.Join(errors.New("could not parse LogMessagePublished payload"), err)
	}
	return &TransferDetails{
		PayloadType:   VAAPayloadType(hdr.Type),
		TokenChain:    vaa.ChainID(hdr.OriginChain),
		TargetAddress: hdr.TargetAddress,
		Amount:        hdr.Amount,
		OriginAddress: hdr.OriginAddress,
	}, nil
}

func (tv *TransferVerifier[ethClient, Connector]) addToCache(
	txHash common.Hash,
	evaluation *evaluation,
) {
	if _, exists := tv.evaluations[txHash]; !exists {
		tv.evaluations[txHash] = evaluation
	}
}

// upsert inserts a new key and value into a map or update the value if the key already exists.
func upsert(
	dict *map[string]*big.Int,
	key string,
	amount *big.Int,
) error {
	if dict == nil || amount == nil {
		return ErrInvalidUpsertArgument
	}
	d := *dict
	if _, exists := d[key]; !exists {
		d[key] = new(big.Int).Set(amount)
	} else {
		d[key] = new(big.Int).Add(d[key], amount)
	}
	return nil
}
