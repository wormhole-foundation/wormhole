package xrpl

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	addresscodec "github.com/Peersyst/xrpl-go/address-codec"
	xrplcommon "github.com/Peersyst/xrpl-go/xrpl/queries/common"
	streamtypes "github.com/Peersyst/xrpl-go/xrpl/queries/subscription/types"
	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"
	"github.com/certusone/wormhole/node/pkg/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// nttMemoFormat is the hex-encoded MemoFormat for NTT transfers: "application/x-ntt-transfer"
// Per XRPL docs, MemoFormat conventionally contains the MIME type of the MemoData content.
const nttMemoFormat = "6170706C69636174696F6E2F782D6E74742D7472616E73666572"

// Prefixes for NTT payloads
var transceiverPrefix = [4]byte{0x99, 0x45, 0xFF, 0x10}
var nttPrefix = [4]byte{0x99, 0x4E, 0x54, 0x54}

// NTT constants
const (
	memoDataLength       = 72 // Length of memo data: prefix(4) + recipientNTTManager(32) + recipientAddress(32) + recipientChain(2) + fromDecimals(1) + toDecimals(1)
	tokenTypeXRP         = 0x00
	tokenTypeIssued      = 0x01
	tokenTypeMPT         = 0x02
	xrpDecimals          = 6
	maxNTTDecimals       = 8
	nttManagerPayloadLen = 143 // Fixed length of NTT manager payload
)

// tesSUCCESS is the XRPL transaction result code for successful transactions
const tesSUCCESS = "tesSUCCESS"

// rippleEpochOffset is the number of seconds between Unix epoch (1970-01-01) and
// Ripple epoch (2000-01-01). XRPL timestamps are seconds since the Ripple epoch.
const rippleEpochOffset = 946684800

// memoData contains cross-chain recipient information parsed from the transaction memo
type memoData struct {
	recipientNTTManager [32]byte
	recipientAddress    [32]byte
	recipientChain      uint16
	fromDecimals        uint8
	toDecimals          uint8
}

// tokenInfo contains information about the token being transferred
type tokenInfo struct {
	tokenType    uint8
	sourceToken  [32]byte
	amount       uint64
	fromDecimals uint8
}

// MPTAssetScaleFetcher is a function type for fetching MPT AssetScale from the ledger.
// This allows the parsing logic to be tested without network calls.
type MPTAssetScaleFetcher func(mptIssuanceID string) (uint8, error)

// Parser handles parsing of XRPL NTT transactions.
// It is stateless except for the MPT asset scale fetcher, which can be injected for testing.
type Parser struct {
	fetchMPTAssetScale MPTAssetScaleFetcher
}

// The stream and request return different transaction structs
// This unifies them for parsing
type GenericTx struct {
	Transaction           transaction.FlatTransaction
	Timestamp             time.Time
	Hash                  string
	LedgerIndex           xrplcommon.LedgerIndex
	MetaDeliveredAmount   any
	MetaTransactionIndex  uint64
	MetaTransactionResult string
}

// NewParser creates a new Parser with the given MPT asset scale fetcher.
func NewParser(fetchMPTAssetScale MPTAssetScaleFetcher) *Parser {
	return &Parser{
		fetchMPTAssetScale: fetchMPTAssetScale,
	}
}

// =============================================================================
// Transaction parsing entry points
// =============================================================================

// ParseTransactionStream converts an XRPL TransactionStream into a MessagePublication.
//
// SECURITY: This function does not verify that the transaction is included in a validated ledger.
// Callers MUST check tx.Validated before calling this function.
func (p *Parser) ParseTransactionStream(tx *streamtypes.TransactionStream) (*common.MessagePublication, error) {
	// Parse ledger close time
	timestamp, err := time.Parse(time.RFC3339, tx.CloseTimeISO)
	if err != nil {
		return nil, fmt.Errorf("failed to parse close time: %w", err)
	}

	return p.parseTransaction(GenericTx{
		Transaction:           tx.Transaction,
		Hash:                  string(tx.Hash),
		LedgerIndex:           tx.LedgerIndex,
		MetaDeliveredAmount:   tx.Meta.DeliveredAmount,
		MetaTransactionIndex:  tx.Meta.TransactionIndex,
		MetaTransactionResult: tx.Meta.TransactionResult,
		Timestamp:             timestamp,
	})
}

// ParseTxResponse converts a TxResponse (from reobservation) into a MessagePublication.
//
// SECURITY: This function does not verify that the transaction is included in a validated ledger.
// Callers MUST check tx.Validated before calling this function.
func (p *Parser) ParseTxResponse(tx *transactions.TxResponse) (*common.MessagePublication, error) {
	// Convert Ripple epoch timestamp to Unix timestamp
	if tx.Date > math.MaxInt64-rippleEpochOffset {
		return nil, fmt.Errorf("transaction date %d would overflow int64", tx.Date)
	}
	timestamp := time.Unix(int64(tx.Date)+rippleEpochOffset, 0)

	return p.parseTransaction(GenericTx{
		Transaction:           tx.TxJSON,
		Hash:                  tx.Hash.String(),
		LedgerIndex:           tx.LedgerIndex,
		MetaDeliveredAmount:   tx.Meta.DeliveredAmount,
		MetaTransactionIndex:  tx.Meta.TransactionIndex,
		MetaTransactionResult: tx.Meta.TransactionResult,
		Timestamp:             timestamp,
	})
}

// parseTransaction contains the shared logic for parsing both TransactionStream and TxResponse.
// Returns (nil, nil) if no NTT memo is found (not an NTT transaction).
//
// SECURITY: This function does not verify that the transaction is included in a validated ledger.
// Callers MUST check the Validated field before calling this function.
func (p *Parser) parseTransaction(
	tx GenericTx,
) (*common.MessagePublication, error) {
	// Parse memo data first - if no NTT memo, this isn't an NTT transaction
	memo, err := p.parseMemoData(tx.Transaction)
	if err != nil {
		return nil, err
	}
	if memo == nil {
		return nil, nil
	}

	// Validate transaction result is tesSUCCESS
	if err := p.validateTransactionResult(tx); err != nil {
		return nil, err
	}

	// Validate transaction type is Payment
	if err := p.validateTransactionType(tx.Transaction); err != nil {
		return nil, err
	}

	// Extract sender address
	sender, err := p.extractSender(tx.Transaction)
	if err != nil {
		return nil, err
	}

	// Extract destination address (the NTT manager on XRPL)
	destination, err := p.extractDestination(tx.Transaction)
	if err != nil {
		return nil, err
	}

	// Parse delivered amount to get token info
	// This also validates: non-zero amount, memo.fromDecimals matches token type
	tokenInfo, err := p.parseDeliveredAmount(tx.MetaDeliveredAmount, memo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse delivered amount: %w", err)
	}

	// Scale the amount: decimals = min(min(8, fromDecimals), toDecimals)
	scaledAmount, decimals := p.scaleAmount(tokenInfo.amount, memo.fromDecimals, memo.toDecimals)

	// Check that scaled amount is not zero (can happen due to integer division with small amounts)
	if scaledAmount == 0 {
		return nil, fmt.Errorf("scaled amount is zero (original amount too small for decimal conversion)")
	}

	// Extract transaction hash (32 bytes)
	txHash, err := hex.DecodeString(tx.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tx hash: %w", err)
	}

	// This should never happen based on the current rippled implementation
	// https://github.com/XRPLF/rippled/blob/677758b1cc9d8afc190582a75160425096708f54/include/xrpl/protocol/detail/sfields.macro#L77
	if tx.MetaTransactionIndex > math.MaxUint32 {
		return nil, fmt.Errorf("invalid transaction index: %d", tx.MetaTransactionIndex)
	}

	// Calculate sequence: (ledgerIndex << 32) | txIndex
	sequence := (uint64(tx.LedgerIndex.Uint32()) << 32) | tx.MetaTransactionIndex

	// Convert destination (payment recipient) to source NTT manager (32-byte left-padded)
	sourceNTTManager, err := p.addressToEmitter(destination)
	if err != nil {
		return nil, fmt.Errorf("failed to convert source NTT manager address: %w", err)
	}

	// Calculate emitter address: keccak256(source_ntt_manager + source_token)
	emitterAddress := p.calculateEmitterAddress(sourceNTTManager, tokenInfo.sourceToken)

	// Build the NTT payload
	payload := p.buildNTTPayload(
		sourceNTTManager,
		memo.recipientNTTManager,
		sequence,
		sender,
		decimals,
		scaledAmount,
		tokenInfo.sourceToken,
		memo.recipientAddress,
		memo.recipientChain,
	)

	return &common.MessagePublication{
		TxID:      txHash,
		Timestamp: tx.Timestamp,
		Nonce:     0, // NTT payloads do not include a nonce
		// See: https://github.com/wormhole-foundation/native-token-transfers/blob/fbe42df37ba19d3c05db8bb77b56c47fc0467c0e/evm/src/Transceiver/WormholeTransceiver/WormholeTransceiver.sol#L134
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDXRPL,
		EmitterAddress:   emitterAddress,
		Payload:          payload,
		ConsistencyLevel: 0, // XRPL validated ledgers are final
		IsReobservation:  false,
	}, nil
}

// =============================================================================
// Transaction validation helpers
// =============================================================================

// validateTransactionResult checks that the transaction result is tesSUCCESS.
// Returns nil if valid, or an error describing why the transaction should be skipped.
// This function is strict - if the result cannot be determined, it returns an error.
func (p *Parser) validateTransactionResult(tx GenericTx) error {
	if tx.MetaTransactionResult != tesSUCCESS {
		return fmt.Errorf("transaction result is %s, not %s", tx.MetaTransactionResult, tesSUCCESS)
	}

	return nil
}

// validateTransactionType checks that the transaction type is "Payment".
// Returns nil if valid, or an error if the transaction type doesn't match.
func (p *Parser) validateTransactionType(tx transaction.FlatTransaction) error {
	txTypeRaw, ok := tx["TransactionType"]
	if !ok {
		return fmt.Errorf("transaction has no TransactionType field")
	}

	txType, ok := txTypeRaw.(string)
	if !ok {
		return fmt.Errorf("transaction TransactionType field is not a string")
	}

	if txType != "Payment" {
		return fmt.Errorf("transaction type is %s, not Payment", txType)
	}

	return nil
}

// extractSender extracts the sender address from the transaction Account field
// and converts it to a 32-byte format.
func (p *Parser) extractSender(tx transaction.FlatTransaction) ([32]byte, error) {
	var sender [32]byte

	accountRaw, ok := tx["Account"]
	if !ok {
		return sender, fmt.Errorf("transaction has no Account field")
	}

	account, ok := accountRaw.(string)
	if !ok {
		return sender, fmt.Errorf("transaction Account field is not a string")
	}

	emitter, err := p.addressToEmitter(account)
	if err != nil {
		return sender, fmt.Errorf("failed to convert sender address: %w", err)
	}

	return emitter, nil
}

// extractDestination extracts the destination address from the transaction Destination field.
func (p *Parser) extractDestination(tx transaction.FlatTransaction) (string, error) {
	destRaw, ok := tx["Destination"]
	if !ok {
		return "", fmt.Errorf("transaction has no Destination field")
	}

	dest, ok := destRaw.(string)
	if !ok {
		return "", fmt.Errorf("transaction Destination field is not a string")
	}

	return dest, nil
}

// =============================================================================
// Memo parsing
// =============================================================================

// parseMemoData extracts and parses the 72-byte NTT memo data from transaction Memos.
// Returns the parsed memoData and any error.
// Returns (nil, nil) if no NTT memo is found (not an error, just not an NTT transaction).
//
// Memo format (72 bytes):
//   - [4]byte   prefix = 0x994E5454
//   - [32]byte  recipient_ntt_manager_address
//   - [32]byte  recipient_address
//   - uint16    recipient_chain
//   - uint8     from_decimals
//   - uint8     to_decimals
func (p *Parser) parseMemoData(tx transaction.FlatTransaction) (*memoData, error) {
	// FlatTransaction is map[string]interface{}
	// Memos is an array of objects with structure: [{"Memo": {"MemoType": "...", "MemoData": "..."}}]
	memosRaw, ok := tx["Memos"]
	if !ok {
		return nil, nil
	}

	memos, ok := memosRaw.([]any)
	if !ok {
		return nil, nil
	}

	for _, memoWrapperRaw := range memos {
		memoWrapper, ok := memoWrapperRaw.(map[string]any)
		if !ok {
			continue
		}

		memoRaw, ok := memoWrapper["Memo"]
		if !ok {
			continue
		}

		memo, ok := memoRaw.(map[string]any)
		if !ok {
			continue
		}

		// Check MemoFormat matches NTT transfer MIME type
		memoFormatStr, ok := memo["MemoFormat"].(string)
		if !ok || memoFormatStr != nttMemoFormat {
			continue
		}

		// Extract and decode MemoData (hex-encoded payload)
		memoDataStr, ok := memo["MemoData"].(string)
		if !ok {
			continue
		}

		data, err := hex.DecodeString(memoDataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode MemoData: %w", err)
		}

		// Verify length
		if len(data) != memoDataLength {
			return nil, fmt.Errorf("invalid memo data length: got %d, want %d", len(data), memoDataLength)
		}

		// Verify NTT prefix
		if data[0] != nttPrefix[0] || data[1] != nttPrefix[1] ||
			data[2] != nttPrefix[2] || data[3] != nttPrefix[3] {
			continue
		}

		// Parse the memo data
		result := &memoData{}
		copy(result.recipientNTTManager[:], data[4:36])
		copy(result.recipientAddress[:], data[36:68])
		result.recipientChain = binary.BigEndian.Uint16(data[68:70])
		result.fromDecimals = data[70]
		result.toDecimals = data[71]

		return result, nil
	}

	return nil, nil
}

// =============================================================================
// Address conversion
// =============================================================================

// addressToEmitter converts an XRPL address to a 32-byte VAA emitter address.
// XRPL addresses are base58-encoded (r-address format) and decode to 20-byte account IDs.
// The account ID is left-padded with 12 zero bytes to create the 32-byte emitter address.
func (p *Parser) addressToEmitter(address string) (vaa.Address, error) {
	// DecodeClassicAddressToAccountID returns the type prefix and 20-byte account ID
	_, accountID, err := addresscodec.DecodeClassicAddressToAccountID(address)
	if err != nil {
		return vaa.Address{}, fmt.Errorf("failed to decode XRPL address %s: %w", address, err)
	}

	// Account ID should be 20 bytes.
	// COVERAGE: This branch is unreachable in practice because the xrpl-go library's
	// DecodeClassicAddressToAccountID always returns exactly 20 bytes for valid addresses,
	// and returns an error for invalid addresses (which is caught above). This check exists
	// as defensive programming against potential library bugs or future changes.
	if len(accountID) != addresscodec.AccountAddressLength {
		return vaa.Address{}, fmt.Errorf("unexpected account ID length: got %d, want %d", len(accountID), addresscodec.AccountAddressLength)
	}

	// Left-pad with zeros to create 32-byte emitter address
	// vaa.Address is [32]byte, accountID is 20 bytes
	// Place accountID in the last 20 bytes (indices 12-31)
	var emitter vaa.Address
	copy(emitter[32-addresscodec.AccountAddressLength:], accountID)

	return emitter, nil
}

// =============================================================================
// Amount parsing
// =============================================================================

// parseDeliveredAmount parses the delivered amount and returns token info.
// It uses the xrpl-go types package to parse the currency amount.
// For Trust Line tokens, memo is required to get the fromDecimals for proper scaling.
//
// This function also validates:
// - The amount is non-zero (NTT requirement)
// - The memo's fromDecimals matches the expected value for the token type
func (p *Parser) parseDeliveredAmount(deliveredAmount any, memo *memoData) (*tokenInfo, error) {
	// Re-marshal to JSON to use the library's unmarshaler
	data, err := json.Marshal(deliveredAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal delivered amount: %w", err)
	}

	amount, err := types.UnmarshalCurrencyAmount(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal currency amount: %w", err)
	}

	var info *tokenInfo

	switch v := amount.(type) {
	case types.XRPCurrencyAmount:
		// XRP: validate memo.fromDecimals == 6
		if memo.fromDecimals != xrpDecimals {
			return nil, fmt.Errorf("fromDecimals mismatch for XRP: memo has %d, expected %d", memo.fromDecimals, xrpDecimals)
		}
		info = &tokenInfo{
			tokenType:    tokenTypeXRP,
			sourceToken:  [32]byte{}, // Zero address for XRP
			amount:       v.Uint64(),
			fromDecimals: xrpDecimals,
		}

	case types.IssuedCurrencyAmount:
		// Trust Lines: no fromDecimals validation (sender specifies arbitrarily)
		info, err = p.parseIssuedCurrencyAmount(v, memo.fromDecimals)
		if err != nil {
			return nil, err
		}

	case types.MPTCurrencyAmount:
		// MPT: validation happens inside parseMPTCurrencyAmount after fetching AssetScale
		info, err = p.parseMPTCurrencyAmount(v, memo.fromDecimals)
		if err != nil {
			return nil, err
		}

	default:
		// COVERAGE: This branch is unreachable because the xrpl-go library's UnmarshalCurrencyAmount
		// only returns three concrete types: XRPCurrencyAmount, IssuedCurrencyAmount, or MPTCurrencyAmount.
		// All three are handled above. This check exists as defensive programming against future
		// library changes that might add new currency types.
		return nil, fmt.Errorf("unexpected currency amount type: %T", amount)
	}

	// Reject zero amount transfers, consistent with other NTT implementations.
	// See: https://github.com/wormhole-foundation/native-token-transfers/blob/fbe42df37ba19d3c05db8bb77b56c47fc0467c0e/evm/src/NttManager/NttManager.sol#L391
	if info.amount == 0 {
		return nil, fmt.Errorf("zero amount transfers are not allowed")
	}

	return info, nil
}

// parseIssuedCurrencyAmount parses a Trust Line (issued currency) amount.
// Trust Lines have no fixed decimals, so the sender specifies the decimal precision
// in the memo's fromDecimals field. The value is scaled directly to this precision
// to avoid overflow when parsing high-precision values.
func (p *Parser) parseIssuedCurrencyAmount(issued types.IssuedCurrencyAmount, fromDecimals uint8) (*tokenInfo, error) {
	// Parse the value directly to the target precision to avoid overflow.
	// If we parsed to the string's natural precision first (could be 15 decimals),
	// we might overflow uint64 even if the final value at fromDecimals would fit.
	amount, err := p.parseDecimalToUint64(issued.Value, fromDecimals)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trust line value: %w", err)
	}

	// Calculate source token
	sourceToken, err := p.calculateTrustLineSourceToken(issued.Currency, string(issued.Issuer))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate trust line source token: %w", err)
	}

	return &tokenInfo{
		tokenType:    tokenTypeIssued,
		sourceToken:  sourceToken,
		amount:       amount,
		fromDecimals: fromDecimals,
	}, nil
}

// parseDecimalToUint64 parses a decimal string value (possibly in scientific notation)
// and returns the value scaled to the specified number of decimal places as uint64.
// This avoids overflow issues that could occur if we first parsed to the string's
// natural precision (could be 15 decimals) before scaling down.
func (p *Parser) parseDecimalToUint64(valueStr string, targetDecimals uint8) (uint64, error) {
	// Use big.Float for precise parsing of decimal values including scientific notation
	f, _, err := big.ParseFloat(valueStr, 10, 256, big.ToNearestEven)
	if err != nil {
		return 0, fmt.Errorf("failed to parse decimal value %q: %w", valueStr, err)
	}

	// Check for negative values
	if f.Sign() < 0 {
		return 0, fmt.Errorf("negative values not allowed: %s", valueStr)
	}

	// Scale directly to target decimals
	// This avoids overflow from parsing at high precision first
	multiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(targetDecimals)), nil))
	result := new(big.Float).Mul(f, multiplier)

	// Truncate to integer (floor) - any fractional part beyond targetDecimals is dust
	intVal, _ := result.Int(nil)
	// COVERAGE: This branch is mathematically unreachable. We check for negative input above (line 545),
	// and multiplying a non-negative number by a positive power of 10 cannot produce a negative result.
	// This check exists as defensive programming against potential floating-point edge cases.
	if intVal.Sign() < 0 {
		return 0, fmt.Errorf("negative result after scaling: %s", valueStr)
	}
	if !intVal.IsUint64() {
		return 0, fmt.Errorf("value %s at %d decimals exceeds uint64 max", valueStr, targetDecimals)
	}

	return intVal.Uint64(), nil
}

// =============================================================================
// Token identification
// =============================================================================

// normalizeCurrency converts a currency code to its canonical 20-byte internal format.
// Standard codes: [0x00][ASCII bytes][trailing zeros]
// Non-standard codes: [raw 160-bit value] (40-character hex string)
func (p *Parser) normalizeCurrency(currency string) ([20]byte, error) {
	var result [20]byte

	// XRP is disallowed as a currency code
	if strings.ToUpper(currency) == "XRP" {
		return result, fmt.Errorf("XRP is not a valid currency code for trust lines")
	}

	// Check if it's a hex string (40 characters = 20 bytes)
	if len(currency) == 40 {
		// Non-standard currency code (hex encoded)
		decoded, err := hex.DecodeString(currency)
		if err != nil {
			return result, fmt.Errorf("failed to decode hex currency: %w", err)
		}
		copy(result[:], decoded)
		return result, nil
	}

	// Standard currency code (3-character ASCII)
	if len(currency) < 1 || len(currency) > 3 {
		return result, fmt.Errorf("invalid standard currency code length: %d", len(currency))
	}

	// Standard format: [0x00][ASCII bytes][trailing zeros]
	result[0] = 0x00
	copy(result[12:12+len(currency)], []byte(currency))

	return result, nil
}

// calculateTrustLineSourceToken calculates the source token for a trust line.
// source_token[0] = 1, last 31 bytes = keccak256(normalizedCurrency + accountID)[1:]
func (p *Parser) calculateTrustLineSourceToken(currency, issuer string) ([32]byte, error) {
	var sourceToken [32]byte

	// Normalize currency
	normalizedCurrency, err := p.normalizeCurrency(currency)
	if err != nil {
		return sourceToken, err
	}

	// Decode issuer to account ID
	_, accountID, err := addresscodec.DecodeClassicAddressToAccountID(issuer)
	if err != nil {
		return sourceToken, fmt.Errorf("failed to decode issuer address: %w", err)
	}

	// Concatenate and hash
	data := make([]byte, 20+len(accountID))
	copy(data[:20], normalizedCurrency[:])
	copy(data[20:], accountID)

	hash := ethcrypto.Keccak256(data)

	// Set token type prefix and copy last 31 bytes of hash
	sourceToken[0] = tokenTypeIssued
	copy(sourceToken[1:], hash[1:])

	return sourceToken, nil
}

// parseMPTCurrencyAmount parses a Multi-Purpose Token amount.
// It validates that memoFromDecimals matches the token's AssetScale from the ledger.
func (p *Parser) parseMPTCurrencyAmount(mpt types.MPTCurrencyAmount, memoFromDecimals uint8) (*tokenInfo, error) {
	// Parse value as integer (MPT values are whole numbers like XRP drops)
	amount, err := strconv.ParseUint(mpt.Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MPT value %q: %w", mpt.Value, err)
	}

	// Fetch asset scale from the ledger
	assetScale, err := p.fetchMPTAssetScale(mpt.MPTIssuanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MPT asset scale: %w", err)
	}

	// Validate memo's fromDecimals matches the on-ledger AssetScale
	if memoFromDecimals != assetScale {
		return nil, fmt.Errorf("fromDecimals mismatch for MPT: memo has %d, AssetScale is %d", memoFromDecimals, assetScale)
	}

	// Calculate source token: [0x02][31-byte left-padded mpt_issuance_id]
	sourceToken, err := p.calculateMPTSourceToken(mpt.MPTIssuanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MPT source token: %w", err)
	}

	return &tokenInfo{
		tokenType:    tokenTypeMPT,
		sourceToken:  sourceToken,
		amount:       amount,
		fromDecimals: assetScale,
	}, nil
}

// calculateMPTSourceToken calculates the source token for an MPT.
// source_token[0] = 2, remaining 31 bytes = left-padded mpt_issuance_id (24 bytes)
func (p *Parser) calculateMPTSourceToken(mptID string) ([32]byte, error) {
	var sourceToken [32]byte

	// Decode the mpt_issuance_id (should be 24 bytes = 48 hex chars)
	mptIDBytes, err := hex.DecodeString(mptID)
	if err != nil {
		return sourceToken, fmt.Errorf("failed to decode mpt_issuance_id: %w", err)
	}

	if len(mptIDBytes) != 24 {
		return sourceToken, fmt.Errorf("invalid mpt_issuance_id length: got %d bytes, want 24", len(mptIDBytes))
	}

	// Set token type prefix and left-pad the ID
	sourceToken[0] = tokenTypeMPT
	// 31 bytes available after prefix, mptID is 24 bytes, so pad with 7 zeros
	copy(sourceToken[8:], mptIDBytes) // 1 (prefix) + 7 (padding) = 8

	return sourceToken, nil
}

// =============================================================================
// Amount scaling
// =============================================================================

// scaleAmount scales the amount to the target decimals.
// decimals = min(min(8, fromDecimals), toDecimals)
// Returns the scaled amount and the resulting decimals.
func (p *Parser) scaleAmount(amount uint64, fromDecimals, toDecimals uint8) (uint64, uint8) {
	// Calculate target decimals: min(min(8, fromDecimals), toDecimals)
	targetDecimals := min(min(maxNTTDecimals, fromDecimals), toDecimals)

	// If we need to reduce decimals, divide
	// Note: targetDecimals is always <= fromDecimals due to the min() formula above,
	// so we only ever scale down, never up.
	if fromDecimals > targetDecimals {
		divisor := uint64(1)
		for i := uint8(0); i < fromDecimals-targetDecimals; i++ {
			divisor *= 10
		}
		amount = amount / divisor
	}

	return amount, targetDecimals
}

// =============================================================================
// Emitter and payload construction
// =============================================================================

// calculateEmitterAddress calculates the emitter address from source NTT manager and source token.
// emitter = keccak256(source_ntt_manager_address + source_token)
func (p *Parser) calculateEmitterAddress(sourceNTTManager, sourceToken [32]byte) vaa.Address {
	data := make([]byte, 64)
	copy(data[:32], sourceNTTManager[:])
	copy(data[32:], sourceToken[:])

	hash := ethcrypto.Keccak256(data)

	var emitter vaa.Address
	copy(emitter[:], hash)
	return emitter
}

// buildNTTPayload builds the full NTT TransceiverMessage payload (~215 bytes).
func (p *Parser) buildNTTPayload(
	sourceNTTManager [32]byte,
	recipientNTTManager [32]byte,
	sequence uint64,
	sender [32]byte,
	decimals uint8,
	amount uint64,
	sourceToken [32]byte,
	recipientAddress [32]byte,
	recipientChain uint16,
) []byte {
	// Calculate total payload size:
	// TransceiverMessage header: 4 + 32 + 32 + 2 = 70 bytes
	// NTT Manager Payload: 32 + 32 + 4 + 1 + 8 + 32 + 32 + 2 = 143 bytes
	// Transceiver payload length: 2 bytes
	// Total: 70 + 143 + 2 = 215 bytes
	payload := make([]byte, 215)
	offset := 0

	// TransceiverMessage prefix (4 bytes)
	copy(payload[offset:], transceiverPrefix[:])
	offset += 4

	// source_ntt_manager_address (32 bytes)
	copy(payload[offset:], sourceNTTManager[:])
	offset += 32

	// recipient_ntt_manager_address (32 bytes)
	copy(payload[offset:], recipientNTTManager[:])
	offset += 32

	// ntt_manager_payload_length (2 bytes, big-endian)
	binary.BigEndian.PutUint16(payload[offset:], nttManagerPayloadLen)
	offset += 2

	// --- NTT Manager Payload starts here ---

	// id (32 bytes) - sequence as 32-byte big-endian
	binary.BigEndian.PutUint64(payload[offset+24:], sequence)
	offset += 32

	// sender (32 bytes)
	copy(payload[offset:], sender[:])
	offset += 32

	// NTT prefix (4 bytes)
	copy(payload[offset:], nttPrefix[:])
	offset += 4

	// decimals (1 byte)
	payload[offset] = decimals
	offset++

	// amount (8 bytes, big-endian)
	binary.BigEndian.PutUint64(payload[offset:], amount)
	offset += 8

	// source_token (32 bytes)
	copy(payload[offset:], sourceToken[:])
	offset += 32

	// recipient_address (32 bytes)
	copy(payload[offset:], recipientAddress[:])
	offset += 32

	// recipient_chain (2 bytes, big-endian)
	binary.BigEndian.PutUint16(payload[offset:], recipientChain)
	offset += 2

	// --- NTT Manager Payload ends here ---

	// transceiver_payload_length (2 bytes, big-endian) = 0
	binary.BigEndian.PutUint16(payload[offset:], 0)

	return payload
}
