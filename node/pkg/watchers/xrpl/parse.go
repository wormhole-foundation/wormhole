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

// coreMemoFormat is the hex-encoded MemoFormat for generic Wormhole messages: "application/x-wormhole-publish"
const coreMemoFormat = "6170706C69636174696F6E2F782D776F726D686F6C652D7075626C697368"

// Prefixes for NTT payloads
var transceiverPrefix = [4]byte{0x99, 0x45, 0xFF, 0x10}
var nttPrefix = [4]byte{0x99, 0x4E, 0x54, 0x54}

// xtcfPrefix is the 4-byte prefix for XRPL ticket refill confirmation payloads
var xtcfPrefix = [4]byte{'X', 'T', 'C', 'F'}

// xackPrefix is the 4-byte prefix for XRPL transaction acknowledgement payloads
var xackPrefix = [4]byte{'X', 'A', 'C', 'K'}

// xackPayloadLen is the length of an XACK payload (14 bytes):
// prefix(4) + ticket_id(8) + success(1) + tx_type(1)
const xackPayloadLen = 14

// XACK transaction type constants
const (
	xackTxTypeRelease      = 0
	xackTxTypeTicketCreate = 1
	xackTxTypeBurn         = 2
)

// NTT constants
const (
	memoDataLength        = 72 // Length of memo data: prefix(4) + recipientNTTManager(32) + recipientAddress(32) + recipientChain(2) + fromDecimals(1) + toDecimals(1)
	tokenTypeXRP          = 0x00
	tokenTypeIssued       = 0x01
	tokenTypeMPT          = 0x02
	xrpDecimals           = 6
	maxNTTDecimals        = 8
	nttManagerPayloadLen  = 145 // Fixed length of NTT manager payload: id(32) + sender(32) + payload_length(2) + internal(79)
	nttInternalPayloadLen = 79  // Internal payload: prefix(4) + decimals(1) + amount(8) + source_token(32) + recipient_address(32) + recipient_chain(2)
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

// coreMessageData contains the parsed memo data for generic Wormhole messages
type coreMessageData struct {
	nonce   uint32
	payload []byte
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
	coreAccount        string          // Core Wormhole manager account — payments to this account are not NTT
	managedAccounts    map[string]bool // Managed accounts (NTT accounts) — TicketCreate on these emits XTCF
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
	MetaAffectedNodes     []transaction.AffectedNode
}

// NewParser creates a new Parser with the given core account, managed accounts, and MPT asset scale fetcher.
// Payments to coreAccount are skipped by parseNttTransaction (they are not NTT transfers).
// managedAccounts are the NTT accounts for which TicketCreate transactions generate XTCF messages.
func NewParser(coreAccount string, managedAccounts []string, fetchMPTAssetScale MPTAssetScaleFetcher) *Parser {
	managed := make(map[string]bool, len(managedAccounts))
	for _, addr := range managedAccounts {
		managed[addr] = true
	}
	return &Parser{
		coreAccount:        coreAccount,
		managedAccounts:    managed,
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
		MetaAffectedNodes:     tx.Meta.AffectedNodes,
		Timestamp:             timestamp,
	})
}

// ParseTxResponse converts a TxResponse (from reobservation) into a MessagePublication.
//
// SECURITY: This function does not verify that the transaction is included in a validated ledger.
// Callers MUST check tx.Validated before calling this function.
func (p *Parser) ParseTxResponse(tx *transactions.TxResponse) (*common.MessagePublication, error) {
	// Get the transaction date. In rippled API v2, the `date` field is inside
	// `tx_json` rather than at the top level, so TxResponse.Date may be 0.
	// Fall back to reading it from TxJSON when that happens.
	date := tx.Date
	if date == 0 {
		if d, ok := tx.TxJSON["date"]; ok {
			switch v := d.(type) {
			case float64:
				date = uint(v)
			case json.Number:
				n, err := v.Int64()
				if err == nil && n >= 0 {
					date = uint(n)
				}
			}
		}
	}

	if date == 0 {
		return nil, fmt.Errorf("transaction date is zero (not populated in API response)")
	}

	// Convert Ripple epoch timestamp to Unix timestamp
	if date > math.MaxInt64-rippleEpochOffset {
		return nil, fmt.Errorf("transaction date %d would overflow int64", date)
	}
	timestamp := time.Unix(int64(date)+rippleEpochOffset, 0)

	return p.parseTransaction(GenericTx{
		Transaction:           tx.TxJSON,
		Hash:                  tx.Hash.String(),
		LedgerIndex:           tx.LedgerIndex,
		MetaDeliveredAmount:   tx.Meta.DeliveredAmount,
		MetaTransactionIndex:  tx.Meta.TransactionIndex,
		MetaTransactionResult: tx.Meta.TransactionResult,
		MetaAffectedNodes:     tx.Meta.AffectedNodes,
		Timestamp:             timestamp,
	})
}

// parseNttTransaction contains the shared logic for parsing both TransactionStream and TxResponse.
// Returns (nil, nil) if no NTT memo is found or if the payment is sent to the core account.
//
// SECURITY: This function does not verify that the transaction is included in a validated ledger.
// Callers MUST check the Validated field before calling this function.
func (p *Parser) parseNttTransaction(
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

	// Skip payments to the core account — those are not NTT transfers
	if p.coreAccount != "" && destination == p.coreAccount {
		return nil, nil
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

	txHash, sequence, err := p.extractTxHashAndSequence(tx)
	if err != nil {
		return nil, err
	}

	// Convert destination (payment recipient) to source NTT manager (32-byte left-padded)
	sourceNTTManager, err := p.addressToEmitter(destination)
	if err != nil {
		return nil, fmt.Errorf("failed to convert source NTT manager address: %w", err)
	}

	// Calculate emitter address: keccak256("ntt" + source_ntt_manager + source_token)
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

// parseTransaction dispatches to parseTicketCreateTransaction, parseCoreTransaction, and parseNttTransaction.
// TicketCreate is checked first because it has no Destination field and would fail in the other parsers.
// Returns (nil, nil) if none matched.
func (p *Parser) parseTransaction(tx GenericTx) (*common.MessagePublication, error) {
	msg, err := p.parseTicketCreateTransaction(tx)
	if msg != nil || err != nil {
		return msg, err
	}

	msg, err = p.parseXACKTransaction(tx)
	if msg != nil || err != nil {
		return msg, err
	}

	msg, err = p.parseCoreTransaction(tx)
	if msg != nil || err != nil {
		return msg, err
	}

	return p.parseNttTransaction(tx)
}

// parseCoreTransaction parses a generic Wormhole message (payment to the core account).
// Returns (nil, nil) if the payment is not to the core account or has no core memo.
func (p *Parser) parseCoreTransaction(tx GenericTx) (*common.MessagePublication, error) {
	if p.coreAccount == "" {
		return nil, nil
	}

	// Check destination is the core account
	destination, err := p.extractDestination(tx.Transaction)
	if err != nil {
		return nil, err
	}
	if destination != p.coreAccount {
		return nil, nil
	}

	// Parse core memo data — if no matching memo, not a core message
	coreMemo, err := p.parseCoreMessageMemoData(tx.Transaction)
	if err != nil {
		return nil, err
	}
	if coreMemo == nil {
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

	// Extract sender address as emitter
	sender, err := p.extractSender(tx.Transaction)
	if err != nil {
		return nil, err
	}

	txHash, sequence, err := p.extractTxHashAndSequence(tx)
	if err != nil {
		return nil, err
	}

	return &common.MessagePublication{
		TxID:             txHash,
		Timestamp:        tx.Timestamp,
		Nonce:            coreMemo.nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDXRPL,
		EmitterAddress:   sender,
		Payload:          coreMemo.payload,
		ConsistencyLevel: 0,
		IsReobservation:  false,
	}, nil
}

// parseCoreMessageMemoData extracts and parses the generic Wormhole message memo data
// from the first memo (index 0) in the transaction's Memos array.
// Returns (nil, nil) if no matching memo is found (not an error, just not a core message).
//
// MemoData format (hex-decoded):
//   - uint8   version (must be 1)
//   - uint32  nonce (big-endian)
//   - []byte  payload (remaining bytes)
func (p *Parser) parseCoreMessageMemoData(tx transaction.FlatTransaction) (*coreMessageData, error) {
	memosRaw, ok := tx["Memos"]
	if !ok {
		return nil, nil
	}

	memos, ok := memosRaw.([]any)
	if !ok {
		return nil, nil
	}

	if len(memos) == 0 {
		return nil, nil
	}

	// Only check the first memo (index 0)
	// SECURITY: this makes it safe to add future memo support without the possibility for message confusion
	memoWrapper, ok := memos[0].(map[string]any)
	if !ok {
		return nil, nil
	}

	memoRaw, ok := memoWrapper["Memo"]
	if !ok {
		return nil, nil
	}

	memo, ok := memoRaw.(map[string]any)
	if !ok {
		return nil, nil
	}

	// Check MemoFormat matches core message MIME type
	memoFormatStr, ok := memo["MemoFormat"].(string)
	if !ok || memoFormatStr != coreMemoFormat {
		return nil, nil
	}

	// Extract and decode MemoData (hex-encoded payload)
	memoDataStr, ok := memo["MemoData"].(string)
	if !ok {
		return nil, nil
	}

	data, err := hex.DecodeString(memoDataStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode core MemoData: %w", err)
	}

	// Minimum length: 1 (version) + 4 (nonce) = 5 bytes
	if len(data) < 5 {
		return nil, fmt.Errorf("core memo data too short: got %d bytes, need at least 5", len(data))
	}

	// Validate version
	if data[0] != 1 {
		return nil, fmt.Errorf("unsupported core memo version: %d", data[0])
	}

	nonce := binary.BigEndian.Uint32(data[1:5])
	payload := data[5:]

	return &coreMessageData{
		nonce:   nonce,
		payload: payload,
	}, nil
}

// parseTicketCreateTransaction parses a TicketCreate transaction on a managed account.
// When tickets are created, it emits an XTCF (ticket refill confirmation) message so the
// sequencer can track the newly created ticket range.
// Returns (nil, nil) if this is not a TicketCreate on a managed account.
func (p *Parser) parseTicketCreateTransaction(tx GenericTx) (*common.MessagePublication, error) {
	// Check transaction type is TicketCreate
	txTypeRaw, ok := tx.Transaction["TransactionType"]
	if !ok {
		return nil, nil
	}
	txType, ok := txTypeRaw.(string)
	if !ok || txType != "TicketCreate" {
		return nil, nil
	}

	// Check the Account is a managed account
	accountRaw, ok := tx.Transaction["Account"]
	if !ok {
		return nil, nil
	}
	account, ok := accountRaw.(string)
	if !ok {
		return nil, nil
	}
	if !p.managedAccounts[account] {
		return nil, nil
	}

	// Validate transaction result is tesSUCCESS
	// Only handle successful TicketCreate here (XTCF).
	// Failed TicketCreate falls through to parseXACKTransaction.
	if err := p.validateTransactionResult(tx); err != nil {
		return nil, nil //nolint:nilerr // Intentional: failed TicketCreate falls through to XACK handler
	}

	// Extract created ticket sequences from AffectedNodes metadata
	var ticketSequences []uint64
	for _, node := range tx.MetaAffectedNodes {
		if node.CreatedNode == nil {
			continue
		}
		if string(node.CreatedNode.LedgerEntryType) != "Ticket" {
			continue
		}
		// NewFields is a FlatLedgerObject (map[string]interface{})
		seqRaw, ok := node.CreatedNode.NewFields["TicketSequence"]
		if !ok {
			continue
		}
		// JSON numbers deserialize as float64
		switch v := seqRaw.(type) {
		case float64:
			ticketSequences = append(ticketSequences, uint64(v))
		case json.Number:
			n, err := v.Int64()
			if err != nil {
				return nil, fmt.Errorf("failed to parse TicketSequence from json.Number: %w", err)
			}
			if n < 0 {
				return nil, fmt.Errorf("negative TicketSequence: %d", n)
			}
			ticketSequences = append(ticketSequences, uint64(n)) // #nosec G115 -- validated non-negative above
		default:
			return nil, fmt.Errorf("unexpected TicketSequence type: %T", seqRaw)
		}
	}

	if len(ticketSequences) == 0 {
		return nil, fmt.Errorf("TicketCreate transaction has no created Ticket entries")
	}

	// Find the minimum ticket sequence (ticket_start) and count
	ticketStart := ticketSequences[0]
	for _, seq := range ticketSequences[1:] {
		if seq < ticketStart {
			ticketStart = seq
		}
	}
	ticketCount := uint64(len(ticketSequences))

	// Build XTCF payload (20 bytes)
	payload := make([]byte, 20)
	copy(payload[0:4], xtcfPrefix[:])
	binary.BigEndian.PutUint64(payload[4:12], ticketStart)
	binary.BigEndian.PutUint64(payload[12:20], ticketCount)

	txHash, sequence, err := p.extractTxHashAndSequence(tx)
	if err != nil {
		return nil, err
	}

	// Emitter is the managed account (left-padded to 32 bytes)
	emitterAddress, err := p.addressToEmitter(account)
	if err != nil {
		return nil, fmt.Errorf("failed to convert account to emitter: %w", err)
	}

	return &common.MessagePublication{
		TxID:             txHash,
		Timestamp:        tx.Timestamp,
		Nonce:            0,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDXRPL,
		EmitterAddress:   emitterAddress,
		Payload:          payload,
		ConsistencyLevel: 0,
		IsReobservation:  false,
	}, nil
}

// accountSetOptionalFields lists the optional fields on an XRPL AccountSet transaction.
// A no-op burn AccountSet must have none of these set.
var accountSetOptionalFields = []string{
	"ClearFlag", "SetFlag", "Domain", "EmailHash", "MessageKey",
	"NFTokenMinter", "TransferRate", "TickSize", "WalletLocator", "WalletSize",
}

// isNoOpAccountSet returns true if the transaction contains no optional AccountSet fields,
// meaning it is a no-op used purely to consume a ticket (burn).
func isNoOpAccountSet(tx transaction.FlatTransaction) bool {
	for _, field := range accountSetOptionalFields {
		if _, ok := tx[field]; ok {
			return false
		}
	}
	return true
}

// parseXACKTransaction parses a ticket-based transaction on a managed account and emits
// an XACK (transaction acknowledgement) message. This handles:
// - Release Payments (tx_type=0, success=true/false)
// - Failed TicketCreate (tx_type=1, success=false)
// - Burn/AccountSet (tx_type=2, success=true/false)
// Returns (nil, nil) if this is not an XACK-eligible transaction.
func (p *Parser) parseXACKTransaction(tx GenericTx) (*common.MessagePublication, error) {
	// Check the Account is a managed account
	accountRaw, ok := tx.Transaction["Account"]
	if !ok {
		return nil, nil
	}
	account, ok := accountRaw.(string)
	if !ok {
		return nil, nil
	}
	if !p.managedAccounts[account] {
		return nil, nil
	}

	// Check transaction has a TicketSequence field (ticket-based transactions only)
	ticketSeqRaw, ok := tx.Transaction["TicketSequence"]
	if !ok {
		return nil, nil
	}
	var ticketSequence uint64
	switch v := ticketSeqRaw.(type) {
	case float64:
		ticketSequence = uint64(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return nil, fmt.Errorf("failed to parse TicketSequence from json.Number: %w", err)
		}
		if n < 0 {
			return nil, fmt.Errorf("negative TicketSequence: %d", n)
		}
		ticketSequence = uint64(n) // #nosec G115 -- validated non-negative above
	default:
		return nil, fmt.Errorf("unexpected TicketSequence type: %T", ticketSeqRaw)
	}

	// Determine tx_type from TransactionType
	txTypeRaw, ok := tx.Transaction["TransactionType"]
	if !ok {
		return nil, nil
	}
	txTypeStr, ok := txTypeRaw.(string)
	if !ok {
		return nil, nil
	}

	var txType uint8
	switch txTypeStr {
	case "Payment":
		txType = xackTxTypeRelease
	case "TicketCreate":
		txType = xackTxTypeTicketCreate
	case "AccountSet":
		if !isNoOpAccountSet(tx.Transaction) {
			return nil, nil // Non-empty AccountSet is not a burn no-op
		}
		txType = xackTxTypeBurn
	default:
		return nil, nil // Unknown transaction type, skip
	}

	// Determine success
	var success uint8
	if tx.MetaTransactionResult == tesSUCCESS {
		success = 1
	}

	// Build 14-byte XACK payload
	payload := make([]byte, xackPayloadLen)
	copy(payload[0:4], xackPrefix[:])
	binary.BigEndian.PutUint64(payload[4:12], ticketSequence)
	payload[12] = success
	payload[13] = txType

	txHash, sequence, err := p.extractTxHashAndSequence(tx)
	if err != nil {
		return nil, err
	}

	// Emitter is the managed account (left-padded to 32 bytes)
	emitterAddress, err := p.addressToEmitter(account)
	if err != nil {
		return nil, fmt.Errorf("failed to convert account to emitter: %w", err)
	}

	return &common.MessagePublication{
		TxID:             txHash,
		Timestamp:        tx.Timestamp,
		Nonce:            0,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDXRPL,
		EmitterAddress:   emitterAddress,
		Payload:          payload,
		ConsistencyLevel: 0,
		IsReobservation:  false,
	}, nil
}

// =============================================================================
// Transaction validation helpers
// =============================================================================

// extractTxHashAndSequence decodes the transaction hash and computes the sequence number
// from the ledger index and transaction index: (ledgerIndex << 32) | txIndex.
func (p *Parser) extractTxHashAndSequence(tx GenericTx) ([]byte, uint64, error) {
	txHash, err := hex.DecodeString(tx.Hash)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode tx hash: %w", err)
	}

	// This should never happen based on the current rippled implementation
	// https://github.com/XRPLF/rippled/blob/677758b1cc9d8afc190582a75160425096708f54/include/xrpl/protocol/detail/sfields.macro#L77
	if tx.MetaTransactionIndex > math.MaxUint32 {
		return nil, 0, fmt.Errorf("invalid transaction index: %d", tx.MetaTransactionIndex)
	}

	sequence := (uint64(tx.LedgerIndex.Uint32()) << 32) | tx.MetaTransactionIndex
	return txHash, sequence, nil
}

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

// parseMemoData extracts and parses the 72-byte NTT memo data from the first memo (index 0)
// in the transaction's Memos array.
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

	if len(memos) == 0 {
		return nil, nil
	}

	// Only check the first memo (index 0)
	// SECURITY: this makes it safe to add future memo support without the possibility for message confusion
	memoWrapper, ok := memos[0].(map[string]any)
	if !ok {
		return nil, nil
	}

	memoRaw, ok := memoWrapper["Memo"]
	if !ok {
		return nil, nil
	}

	memo, ok := memoRaw.(map[string]any)
	if !ok {
		return nil, nil
	}

	// Check MemoFormat matches NTT transfer MIME type
	memoFormatStr, ok := memo["MemoFormat"].(string)
	if !ok || memoFormatStr != nttMemoFormat {
		return nil, nil
	}

	// Extract and decode MemoData (hex-encoded payload)
	memoDataStr, ok := memo["MemoData"].(string)
	if !ok {
		return nil, nil
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
		return nil, nil
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

	// If we need to reduce decimals, divide by 10 iteratively to avoid
	// uint64 overflow when computing a large divisor (e.g., 10^21).
	// Note: targetDecimals is always <= fromDecimals due to the min() formula above,
	// so we only ever scale down, never up.
	if fromDecimals > targetDecimals {
		decimalsToShift := fromDecimals - targetDecimals
		for i := uint8(0); i < decimalsToShift; i++ {
			amount /= 10
		}
	}

	return amount, targetDecimals
}

// =============================================================================
// Emitter and payload construction
// =============================================================================

// calculateEmitterAddress calculates the emitter address from source NTT manager and source token.
// emitter = keccak256("ntt" + source_ntt_manager_address + source_token)
func (p *Parser) calculateEmitterAddress(sourceNTTManager, sourceToken [32]byte) vaa.Address {
	data := make([]byte, 3+64)
	copy(data[:3], "ntt")
	copy(data[3:35], sourceNTTManager[:])
	copy(data[35:], sourceToken[:])

	hash := ethcrypto.Keccak256(data)

	var emitter vaa.Address
	copy(emitter[:], hash)
	return emitter
}

// buildNTTPayload builds the full NTT TransceiverMessage payload (217 bytes).
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
	// NTT Manager Payload: 32 + 32 + 2 + 4 + 1 + 8 + 32 + 32 + 2 = 145 bytes
	// Transceiver payload length: 2 bytes
	// Total: 70 + 145 + 2 = 217 bytes
	payload := make([]byte, 217)
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

	// payload_length (2 bytes, big-endian) - length of internal NTT payload
	binary.BigEndian.PutUint16(payload[offset:], nttInternalPayloadLen)
	offset += 2

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
