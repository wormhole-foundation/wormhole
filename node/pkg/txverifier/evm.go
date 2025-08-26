package txverifier

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
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

var (
	ErrCachedReceiptHasNoDataForMessage = errors.New("cached receipt has no data for this message")
	ErrCouldNotGetNativeDetails         = errors.New("could not parse native details for transfer")
	ErrDepositFromWrongAddress          = errors.New("parsed Deposit event has wrong address")
	ErrInvalidReceiptArgument           = errors.New("invalid TransferReceipt argument")
	ErrInvalidUpsertArgument            = errors.New("nil argument passed to upsert")
	ErrNoMsgsFromTokenBridge            = errors.New("no messages published from Token Bridge")
	ErrNotTransfer                      = errors.New("payload is not a token transfer")
	ErrParsedReceiptIsNil               = errors.New("nil receipt after parsing")
	ErrTxHashIsZeroAddr                 = errors.New("txHash is the zero address")
)

// TransferIsValid processes a Message Publication representing a token transfer.
// It fetches the full transaction receipt associated with the txHash and parses all
// events emitted in the transaction, tracking LogMessagePublished events as outbound
// transfers and token deposits/transfers into the token bridge as inbound transfers. It then
// verifies that the sum of the inbound transfers is at least as much as the sum of
// the outbound transfers.
//
// Return values:
//
//	true, nil: the token transfer in the Message Publication is valid.
//	false and nil: the Message Publication has violated an invariant.
//	false and err: the receipt could not be properly processed or is not a token transfer.
//	true and err: Should never happen.
func (tv *TransferVerifier[ethClient, Connector]) TransferIsValid(
	ctx context.Context,
	// The message ID is a combination of the emitter chain, emitter address, and sequence.
	// It is identical to the VAA's "message ID" that will eventually be created based on the
	// MessagePublication. See the Go SDK for more information.
	//
	// If empty, the code will return true if all of the messages are valid, and false otherwise.
	// This should only be done for testing.
	msgIDStr string,
	txHash common.Hash,
	// If nil, this code will fetch the receipt using the TransferVerifier's connector.
	receipt *geth.Receipt,
) (bool, error) {

	tv.pruneCache()

	if Cmp(txHash, ZERO_ADDRESS) == 0 {
		tv.logger.Error(ErrTxHashIsZeroAddr.Error())
		return false, ErrTxHashIsZeroAddr
	}

	tv.logger.Debug("detected LogMessagePublished event",
		zap.String("txHash", txHash.String()))

	// Remains empty if the caller wants to check all messages in a receipt and the receipt itself.
	checkWholeReceipt := msgIDStr == ""

	// Get the specific message ID if provided.
	var msgID msgID
	if !checkWholeReceipt {
		// Validate the validMsgID parameter if provided.
		validMsgID, err := tv.ParseMsgID(msgIDStr)
		if err != nil {
			return false, err
		}

		msgID = validMsgID
	}

	// Check to see if the receipt is cached.
	if evaluation, exists := tv.evaluations[txHash]; exists {

		// Check if the message has already been verified, and if so, return the cached result.
		if !checkWholeReceipt && !evaluation.msgInCache(msgID) {
			// Should never happen, probably indicates a bug in the code.
			tv.logger.Warn(
				ErrCachedReceiptHasNoDataForMessage.Error(),
				zap.String("msgID", msgID.String()),
			)
			return false, ErrCachedReceiptHasNoDataForMessage
		}

		// Return true if the entire receipt is safe.
		if evaluation.ReceiptSummary.isSafe() {
			return true, nil
		}

		//  Return true if the message is safe.
		return evaluation.ReceiptSummary.isMsgSafe(msgID), nil
	}

	// Get the full transaction receipt for this txHash if it was not provided as an argument.
	if receipt == nil {
		tv.logger.Debug("receipt was not passed as an argument. fetching it using the connector")
		var txReceiptErr error
		receipt, txReceiptErr = tv.evmConnector.TransactionReceipt(ctx, txHash)

		if txReceiptErr != nil {
			return false, txReceiptErr
		}
	}

	// Caching: if the new receipt is newer than the last one, update the last block number.
	// This is important to prevent the Transfer Verifier from recording a block number that is older
	// than the one being processed, as this would result in the verifier not being able to
	// correctly prune the cache.
	if receipt.BlockNumber != nil && receipt.BlockNumber.Uint64() > tv.lastBlockNumber {
		tv.lastBlockNumber = receipt.BlockNumber.Uint64()
	}

	// Parse raw transaction receipt into high-level struct containing transfer details.
	transferReceipt, parseErr := tv.parseReceipt(receipt)

	if parseErr != nil {
		return false, parseErr
	}

	// ParseReceipt should only return nil when there is also an error, so we don't expect to get here.
	if transferReceipt == nil {
		return false, ErrParsedReceiptIsNil
	}

	// Invalid receipt: no message publications
	if len(*transferReceipt.MessagePublications) == 0 {
		return false, ErrNoMsgsFromTokenBridge
	}

	// Add wormhole-specific data to the receipt by making
	// RPC calls for data that is not included in the logs,
	// such as a token's native address and its decimals.
	updateErr := tv.updateReceiptDetails(transferReceipt)
	if updateErr != nil {
		return false, updateErr
	}

	// Evaluate the receipt. This returns a summary of the receipt and a list of invariant errors,
	// if any. If the receipt is valid, the summary will be non-nil and the invariant errors will be nil.
	// If a receipt is valid, then all of its messages are valid and the receipt is safe.
	// If a receipt is invalid, then some of its messages are invalid and the receipt is not safe.
	summary, validateReceiptErr := tv.validateReceipt(transferReceipt)
	tv.logger.Debug("finished processing receipt", zap.String("txHash", txHash.String()), zap.String("summary", summary.String()))

	if summary == nil {
		tv.logger.Warn("receipt summary is nil", zap.String("txHash", txHash.String()))
		return false, validateReceiptErr
	}

	// Check for parsing errors, which are bugs in the code.
	if validateReceiptErr != nil {
		// If we have a parsing error, we can't make any claims about the validity of the message.
		// To be safe, fail closed by returning false.
		// Results are not cached, as the parsing error may be transient.
		return false, validateReceiptErr
	}

	if !summary.isSafe() {
		// This represents an invariant violation, either in the receipt as a whole or in a single message.
		tv.logger.Error("receipt is not solvent", zap.String("txHash", txHash.String()), zap.Int("invalidMsgCount", summary.invalidMessageCount()), zap.String("insolventAssets", strings.Join(summary.insolventAssets(), ",")))
	}

	// Cache receipt/message validations results (except for parsing errors, as they may be transient).
	cacheEvaluation := NewReceiptEvaluation(summary, receipt.BlockNumber.Uint64())
	tv.addToCache(txHash, &cacheEvaluation)

	if checkWholeReceipt {
		// Should only be done for testing.
		return summary.isSafe(), nil
	}

	// Get the result of the message's validation.
	msgSafe := summary.isMsgSafe(msgID)
	tv.logger.Debug("msgIsValid results", zap.String("msg", msgID.String()), zap.Bool("msgSafe", msgSafe))

	return msgSafe, nil
}

// pruneCache removes cached evaluations and RPC calls.
// TODO: This functionality should be replaced by an LRU cache library.
func (tv *TransferVerifier[ethClient, Connector]) pruneCache() {
	var numPrunedEvals int

	// Iterate over recorded transaction hashes, and clear receipts older than `pruneDelta` blocks
	for txHash, eval := range tv.evaluations {
		if eval != nil {
			if eval.blockNumber < tv.lastBlockNumber-tv.pruneHeightDelta {
				numPrunedEvals++
				delete(tv.evaluations, txHash)
			}
		} else {
			// NOTE: Kind of sloppy to delete these right away, but we shouldn't be caching
			// many nil receipts anyway.
			numPrunedEvals++
			delete(tv.evaluations, txHash)
		}
	}

	if numPrunedEvals > 0 {
		tv.logger.Info("pruned receipt evaluations cache",
			zap.Int("numPruned", numPrunedEvals),
			zap.Int("cacheSize", len(tv.evaluations)),
		)
	}

	// For the other caches, just prune them if they're too big.
	if len(tv.chainIdCache) >= CacheMaxSize {
		numDeleted, err := deleteEntries(&tv.chainIdCache)
		if err != nil {
			tv.logger.Warn("pruneCache: chainId() cache:", zap.Error(err))

		} else {
			tv.logger.Info("pruned cached calls to chainId()",
				zap.Int("numDeleted", numDeleted),
				zap.Int("cacheSize", len(tv.chainIdCache)),
			)
		}
	}
	if len(tv.decimalsCache) >= CacheMaxSize {
		numDeleted, err := deleteEntries(&tv.decimalsCache)
		if err != nil {
			tv.logger.Warn("pruneCache: decimals() cache:", zap.Error(err))

		} else {
			tv.logger.Info("pruned cached calls to decimals()",
				zap.Int("numDeleted", numDeleted),
				zap.Int("cacheSize", len(tv.decimalsCache)),
			)
		}
	}
	if len(tv.isWrappedCache) >= CacheMaxSize {
		numDeleted, err := deleteEntries(&tv.isWrappedCache)
		if err != nil {
			tv.logger.Warn("pruneCache: isWrapped() cache:", zap.Error(err))

		} else {
			tv.logger.Info("pruned cached calls to isWrapped()",
				zap.Int("numDeleted", numDeleted),
				zap.Int("cacheSize", len(tv.isWrappedCache)),
			)
		}
	}
	if len(tv.nativeContractCache) >= CacheMaxSize {
		numDeleted, err := deleteEntries(&tv.nativeContractCache)
		if err != nil {
			tv.logger.Warn("pruneCache: nativeContract() cache:", zap.Error(err))
		}
		tv.logger.Info("pruned cached calls to nativeContract()",
			zap.Int("numDeleted", numDeleted),
			zap.Int("cacheSize", len(tv.nativeContractCache)),
		)
	}
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

	invalidErr := receipt.SanityCheck()
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
				return err
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

			// Ensure that the deposit event is from the correct contract.
			if log.Address != tv.Addresses.WrappedNativeAddr {
				tv.logger.Debug(ErrDepositFromWrongAddress.Error(),
					zap.String("txHash", log.TxHash.String()),
				)
				continue
			}

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

				// Just skip transfers that aren't from ERC20 contracts.
				if errors.Is(transferErr, ErrTransferIsNotERC20) {
					tv.logger.Debug(
						fmt.Sprintf("skip: %s", ErrTransferIsNotERC20.Error()),
						zap.String("txHash", log.TxHash.String()),
					)
					continue
				}

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

			// Make sure the core bridge is the emitter of the event.
			// This check is required. Payload parsing will fail if performed on a message emitted from another contract or sent
			// by a contract other than the token bridge
			if log.Address != tv.Addresses.CoreBridgeAddr {
				tv.logger.Debug("skip: LogMessagePublished not emitted from the core bridge",
					zap.String("emitter", log.Address.String()))
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

			// Make sure the token bridge is the sender.
			if log.Topics[1] != tv.Addresses.TokenBridgeAddr.Hash() {
				tv.logger.Debug("skip: LogMessagePublished with sender not equal to the token bridge",
					zap.String("sender", log.Topics[1].String()),
					zap.String("tokenBridgeAddr", tv.Addresses.TokenBridgeAddr.Hex()),
				)
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

			tv.logger.Debug("parsed TransferDetails from LogMessagePublished", zap.String("transferDetails", transferDetails.String()))

			// If everything went well, append the message publication
			messagePublications = append(messagePublications, &LogMessagePublished{
				EventEmitter:    log.Address,
				MsgSender:       logMessagePublished.Sender,
				Sequence:        logMessagePublished.Sequence,
				TransferDetails: transferDetails,
			})

		}
	}

	// Return an error when there are no valid messages from the token bridge
	// after filtering out irrelevant events.
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

// validateReceipt verifies that a receipt for a LogMessagedPublished event does
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
func (tv *TransferVerifier[evmClient, connector]) validateReceipt(
	receipt *TransferReceipt,
) (*ReceiptSummary, error) {

	// Sanity checks.
	if receipt == nil {
		return nil, ErrInvalidReceiptArgument
	}

	invalidErr := receipt.SanityCheck()
	if invalidErr != nil {
		return nil, errors.Join(
			ErrInvalidReceiptArgument,
			invalidErr,
		)
	}

	tv.logger.Debug("beginning to validate receipt",
		zap.String("receipt", receipt.String()),
	)

	if len(*receipt.Deposits) == 0 && len(*receipt.Transfers) == 0 {
		// This should result in an invariant error below but this is helpful context.
		tv.logger.Warn("receipt is a token transfer but has no deposits or transfers",
			zap.String("receipt", receipt.String()),
		)
	}

	receiptSummary := NewReceiptSummary()

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

		upsertErr := upsert(&receiptSummary.in, key, deposit.TransferAmount())
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

		upsertErr := upsert(&receiptSummary.in, key, transfer.TransferAmount())
		if upsertErr != nil {
			return nil, upsertErr
		}

		tv.logger.Debug("identified ERC20 transfer to the token bridge",
			zap.String("tokenAddress", transfer.TokenAddress.String()),
			zap.String("amount", transfer.Amount.String()))
	}

	// Process LogMessagePublished events.
	for _, message := range *receipt.MessagePublications {

		// It's invalid for the core bridge to send a message to itself.
		// This check is already done in validate() but doing it here
		// allows us to use TransferVerifier's logger.
		if Cmp(message.Sender(), message.Emitter()) == 0 {
			tv.logger.Error("msg.sender cannot be equal to emitter",
				zap.String("message", message.String()),
			)
		}

		validateErr := validate[*LogMessagePublished](message)
		if validateErr != nil {
			return nil, validateErr
		}

		key, relevant := relevant[*LogMessagePublished](message, tv.Addresses)
		if !relevant {
			tv.logger.Debug("skip: irrelevant LogMessagePublished event")
			continue
		}
		msgID, err := tv.MsgID(message)
		if err != nil {
			return nil, err
		}

		// Record that tokens were requested out of the bridge for this message.
		if _, exists := receiptSummary.out[msgID]; !exists {
			receiptSummary.out[msgID] = transferOut{key, message.TransferAmount()}
			// Ensure that each message ID is populated in the map.
			receiptSummary.msgPubResult[msgID] = true
		} else {
			return nil, errors.New("duplicate message ID in receipt's LogMessagePublished events")
		}
	}

	// Sanity check: the receipt summary should have the same number of messages as the receipt.
	if len(receiptSummary.msgPubResult) == 0 || len(*receipt.MessagePublications) != len(receiptSummary.msgPubResult) {
		return nil, ErrInvalidReceiptArgument
	}

	tv.logger.Debug("done building receipt summary", zap.String("summary", receiptSummary.String()))

	// This is the core evaluation algorithm that determines whether the receipt is valid, and whether the
	// individual messages in the receipt are valid.
	//
	// There is one fundamental invariant that must be satisfied for the receipt to be valid:
	// the sum of the outbound transfers must be at least as much as the sum of the inbound transfers.
	insolventAssets := receiptSummary.insolventAssets()
	if len(insolventAssets) > 0 {
		tv.logger.Warn("receipt has insolvent assets", zap.Strings("insolventAssets", insolventAssets))
	}
	tv.logger.Debug("receipt summary after solvency check", zap.String("summary", receiptSummary.String()))

	// At this stage, each msgPubResult is true. If insolvent assets are detected, then the msgPubResult
	// will be set to false.
	if len(*receipt.MessagePublications) == 1 {
		// Only one message publication exists in the receipt. Check for solvency.
		// There is only one message publication in the receipt, but we don't have the msgID to do a lookup.
		for msgID := range receiptSummary.out {
			receiptSummary.msgPubResult[msgID] = len(insolventAssets) == 0
		}
	} else {
		// Multi-message case.
		for msgID, transferOut := range receiptSummary.out {
			if _, exists := receiptSummary.in[transferOut.tokenID]; !exists {
				// This is never OK, even in the edge cases documented above.
				receiptSummary.msgPubResult[msgID] = false
			}
		}
	}

	// Post-processing: if the receipt has insolvent assets, then mark the messages that involve those assets as invalid.
	// It's possible that the above processing marks individual messages as valid, but the receipt as a whole is not solvent.
	// To avoid processing these messages individually in the future, we mark them as invalid here.
	if len(insolventAssets) > 0 {
		for _, msgPub := range *receipt.MessagePublications {
			msgID, err := tv.MsgID(msgPub)

			// Parsing error.
			if err != nil {
				return nil, err
			}

			// Explicitly use int representation of the chain ID to avoid it being converted to its chain name
			// via the String() method.
			tokenID := fmt.Sprintf("%s-%d", msgPub.OriginAddress(), uint16(msgPub.OriginChain()))
			tv.logger.Debug("insolvent assets", zap.String("assets", strings.Join(insolventAssets, ",")))
			tv.logger.Debug("tokenID", zap.String("tokenID", tokenID))
			if slices.Contains(insolventAssets, tokenID) {
				tv.logger.Info("marking message as invalid because it contains an insolvent asset", zap.String("msgID", msgID.String()), zap.String("tokenID", tokenID))
				receiptSummary.msgPubResult[msgID] = false
			}
		}
	}

	tv.logger.Debug("evaluated receipt safety:",
		zap.Int("invalidMessageCount", receiptSummary.invalidMessageCount()),
		zap.Bool("receiptIsSolvent?", len(insolventAssets) == 0),
		zap.Bool("receipt isSafe?", receiptSummary.isSafe()),
	)

	return receiptSummary, nil
}

// insolventAssets() reports whether the sum of the outbound transfers is at least as much as the sum of the inbound transfers.
// It is used to examine the solvency of the entire receipt, without considering individual messages.
func (receiptSummary *ReceiptSummary) insolventAssets() []string {
	if len(receiptSummary.out) == 0 {
		return []string{}
	}

	var insolventAssets []string

	// Track individual token balances separately from the summary's totals so that we can
	// safely subtract from the totals while iterating over the outbound transfers.

	// Important: the map is manually deep copied to avoid mutating the summary's `in` map.
	// the map hold pointers to big.Ints, so we need to make a copy of their values.
	tokenBalances := make(map[string]*big.Int)
	for tokenID, amount := range receiptSummary.in {
		tokenBalances[tokenID] = new(big.Int).Set(amount)
	}

	for _, transferOut := range receiptSummary.out {
		if _, exists := receiptSummary.in[transferOut.tokenID]; !exists {
			// Some outbound asset is not present in the inbound transfers.
			insolventAssets = append(insolventAssets, transferOut.tokenID)
		} else {
			// Subtract the transfer out amount from the total balance.
			tokenBalances[transferOut.tokenID].Sub(tokenBalances[transferOut.tokenID], transferOut.amount)
		}
	}

	// If any token balance is less than zero, then the entire receipt is insolvent.
	for tokenID, adjustedTotal := range tokenBalances {
		if adjustedTotal.Cmp(big.NewInt(0)) < 0 {
			insolventAssets = append(insolventAssets, tokenID)
		}
	}

	return insolventAssets
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
	evaluation *receiptEvaluation,
) {
	if _, exists := tv.evaluations[txHash]; !exists {
		tv.evaluations[txHash] = evaluation
	}
}
