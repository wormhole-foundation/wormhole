// Package xrpl implements XRPL transaction building and signing for the manager service.
package xrpl

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	addresscodec "github.com/Peersyst/xrpl-go/address-codec"
	binarycodec "github.com/Peersyst/xrpl-go/binary-codec"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"
	"github.com/certusone/wormhole/node/pkg/watchers/xrpl/currencycodec"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const (
	// xrplBaseFeeDrops is the XRPL base fee per signer in drops. The total fee
	// for a multisigned transaction is xrplBaseFeeDrops * (N+1) where N is the
	// number of signers.
	xrplBaseFeeDrops = 12
	// xrplBurnFeeDrops is the per-signer fee used for burn (no-op AccountSet)
	// transactions that replace a previously submitted transaction. It must be
	// at least 25% higher than the original fee.
	xrplBurnFeeDrops = 15
	// secp256k1CompressedPubKeyLen is the length of a compressed secp256k1 public key.
	secp256k1CompressedPubKeyLen = 33

	// DER signature encoding constants.
	derSequenceTag = 0x30 // ASN.1 SEQUENCE tag
	derIntegerTag  = 0x02 // ASN.1 INTEGER tag
	// derFixedOverhead is the fixed byte overhead inside the DER SEQUENCE body:
	// integer tag + length byte for r, plus integer tag + length byte for s.
	derFixedOverhead = 4
	// derSequenceHeaderLen is the SEQUENCE tag byte plus the total-length byte.
	derSequenceHeaderLen = 2
)

// BuildPaymentTransaction builds and flattens an XRPL Payment transaction for multisigning.
// The m parameter is the required number of signers in the manager set (used for fee calculation).
// The returned map is ready for binary-codec encoding.
func BuildPaymentTransaction(payload *vaa.XRPLReleasePayload, m uint8) (transaction.FlatTransaction, error) {
	// Convert 20-byte account IDs to r-addresses
	custodyAddress, err := AccountIDToAddress(payload.CustodyAccount[:])
	if err != nil {
		return nil, fmt.Errorf("failed to convert custody account ID to address: %w", err)
	}

	recipientAddress, err := AccountIDToAddress(payload.Recipient[:])
	if err != nil {
		return nil, fmt.Errorf("failed to convert recipient account ID to address: %w", err)
	}

	// Build Amount based on token type
	var amount types.CurrencyAmount
	switch payload.Token.Type {
	case vaa.XRPLTokenTypeXRP:
		// XRP amount in drops (already in smallest unit)
		amount = types.XRPCurrencyAmount(payload.Amount)
	case vaa.XRPLTokenTypeIOU:
		// IOU: convert integer amount to decimal string using token_decimals
		issuerAddr, err := AccountIDToAddress(payload.Token.Issuer[:])
		if err != nil {
			return nil, fmt.Errorf("failed to convert IOU issuer account ID to address: %w", err)
		}
		value := formatDecimalAmountForIOU(payload.Amount, payload.TokenDecimals)
		amount = types.IssuedCurrencyAmount{
			Currency: currencycodec.Encode(payload.Token.Currency),
			Issuer:   types.Address(issuerAddr),
			Value:    value,
		}
	case vaa.XRPLTokenTypeMPT:
		// MPT: integer string value
		amount = types.MPTCurrencyAmount{
			MPTIssuanceID: hex.EncodeToString(payload.Token.MPTID[:]),
			Value:         strconv.FormatUint(payload.Amount, 10),
		}
	default:
		return nil, fmt.Errorf("unsupported XRPL token type: 0x%02x", payload.Token.Type)
	}

	// Deterministic fee: 12 drops base fee * (N+1) for multisig
	fee := uint64(m+1) * xrplBaseFeeDrops

	// Convert memos
	memos := make([]types.MemoWrapper, 0, len(payload.Memos))
	for _, m := range payload.Memos {
		memo := types.Memo{}
		if len(m.Data) > 0 {
			memo.MemoData = hex.EncodeToString(m.Data)
		}
		if len(m.Format) > 0 {
			memo.MemoFormat = hex.EncodeToString(m.Format)
		}
		if len(m.Type) > 0 {
			memo.MemoType = hex.EncodeToString(m.Type)
		}
		memos = append(memos, types.MemoWrapper{Memo: memo})
	}

	// Build Payment transaction
	payment := &transaction.Payment{
		BaseTx: transaction.BaseTx{
			Account:        types.Address(custodyAddress),
			Fee:            types.XRPCurrencyAmount(fee),
			TicketSequence: uint32(payload.TicketID), // #nosec G115 -- TicketID is bounded by XRPL protocol
			Memos:          memos,
		},
		Amount:      amount,
		Destination: types.Address(recipientAddress),
	}

	// Flatten the transaction
	flatTx := payment.Flatten()

	// Manually set fields that Flatten() skips when zero-valued:
	// Sequence must be 0 when using TicketSequence
	flatTx["Sequence"] = uint32(0)
	// SigningPubKey must be empty string for multisig
	flatTx["SigningPubKey"] = ""
	// Fee as string (override in case Flatten used a different format)
	flatTx["Fee"] = strconv.FormatUint(fee, 10)

	return flatTx, nil
}

// BuildTicketCreateTransaction builds and flattens an XRPL TicketCreate transaction for multisigning.
// The m parameter is the required number of signers in the manager set (used for fee calculation).
// The returned map is ready for binary-codec encoding.
func BuildTicketCreateTransaction(payload *vaa.XRPLTicketRefillPayload, m uint8) (transaction.FlatTransaction, error) {
	// Convert 20-byte account ID to r-address
	accountAddress, err := AccountIDToAddress(payload.Account[:])
	if err != nil {
		return nil, fmt.Errorf("failed to convert account ID to address: %w", err)
	}

	// Deterministic fee: 12 drops base fee * (M+1) for multisig
	fee := uint64(m+1) * xrplBaseFeeDrops

	// Build TicketCreate transaction
	ticketCreate := &transaction.TicketCreate{
		BaseTx: transaction.BaseTx{
			Account:        types.Address(accountAddress),
			Fee:            types.XRPCurrencyAmount(fee),
			TicketSequence: uint32(payload.UseTicket), // #nosec G115 -- UseTicket is bounded by XRPL protocol
		},
		TicketCount: uint32(payload.RequestCount), // #nosec G115 -- RequestCount is bounded (1-250)
	}

	// Flatten the transaction
	flatTx := ticketCreate.Flatten()

	// Manually set fields that Flatten() skips when zero-valued:
	// Sequence must be 0 when using TicketSequence
	flatTx["Sequence"] = uint32(0)
	// SigningPubKey must be empty string for multisig
	flatTx["SigningPubKey"] = ""
	// Fee as string (override in case Flatten used a different format)
	flatTx["Fee"] = strconv.FormatUint(fee, 10)

	return flatTx, nil
}

// BuildBurnTicketTransaction builds and flattens an XRPL AccountSet no-op transaction
// that consumes a ticket. The m parameter is the required number of signers in the
// manager set (used for fee calculation). The returned map is ready for binary-codec encoding.
func BuildBurnTicketTransaction(payload *vaa.XRPLBurnTicketPayload, m uint8) (transaction.FlatTransaction, error) {
	// Convert 20-byte account ID to r-address
	accountAddress, err := AccountIDToAddress(payload.Account[:])
	if err != nil {
		return nil, fmt.Errorf("failed to convert account ID to address: %w", err)
	}

	// Deterministic fee: 15 drops base fee * (M+1) for multisig
	// NOTE: this must be at least 25% higher than the fee of the transaction being replaced
	fee := uint64(m+1) * xrplBurnFeeDrops

	// Build AccountSet no-op transaction (no flags or fields set)
	accountSet := &transaction.AccountSet{
		BaseTx: transaction.BaseTx{
			Account:        types.Address(accountAddress),
			Fee:            types.XRPCurrencyAmount(fee),
			TicketSequence: uint32(payload.TicketID), // #nosec G115 -- TicketID is bounded by XRPL protocol
		},
	}

	// Flatten the transaction
	flatTx := accountSet.Flatten()

	// Manually set fields that Flatten() skips when zero-valued:
	// Sequence must be 0 when using TicketSequence
	flatTx["Sequence"] = uint32(0)
	// SigningPubKey must be empty string for multisig
	flatTx["SigningPubKey"] = ""
	// Fee as string (override in case Flatten used a different format)
	flatTx["Fee"] = strconv.FormatUint(fee, 10)

	return flatTx, nil
}

// ComputeMultisignHash computes the hash that a signer must sign for XRPL multisigning.
// It encodes the transaction with the signer's account ID appended, then returns SHA512Half.
func ComputeMultisignHash(flatTx transaction.FlatTransaction, signerAddress string) ([]byte, error) {
	// EncodeForMultisigning prepends "SMT\0" prefix, encodes the tx, and appends the signer's account ID
	encoded, err := binarycodec.EncodeForMultisigning(flatTx, signerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to encode for multisigning: %w", err)
	}

	// Decode the hex string to bytes
	encodedBytes, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encoded hex: %w", err)
	}

	return SHA512Half(encodedBytes), nil
}

// EncodeDERSignature DER-encodes an ECDSA signature (r, s) without sighash type byte.
// Unlike Dogecoin, XRPL does not append a sighash type byte.
func EncodeDERSignature(r, s []byte) []byte {
	// DER format: 0x30 [total-length] 0x02 [r-length] [r] 0x02 [s-length] [s]

	// Remove leading zeros from r and s, but ensure they don't look negative
	r = canonicalizeInt(r)
	s = canonicalizeInt(s)

	rLen := len(r)
	sLen := len(s)
	totalLen := derFixedOverhead + rLen + sLen // 0x02 + rLen + r + 0x02 + sLen + s

	sig := make([]byte, 0, totalLen+derSequenceHeaderLen) // +2 for 0x30 and totalLen
	sig = append(sig, derSequenceTag)                     // DER sequence tag
	sig = append(sig, byte(totalLen))                     // Total length
	sig = append(sig, derIntegerTag)                      // Integer tag for r
	sig = append(sig, byte(rLen))                         // r length
	sig = append(sig, r...)                               // r value
	sig = append(sig, derIntegerTag)                      // Integer tag for s
	sig = append(sig, byte(sLen))                         // s length
	sig = append(sig, s...)                               // s value

	return sig
}

// canonicalizeInt removes leading zeros and adds a zero byte if the high bit is set.
func canonicalizeInt(b []byte) []byte {
	// Remove leading zeros
	for len(b) > 1 && b[0] == 0 && b[1]&0x80 == 0 {
		b = b[1:]
	}
	// Add zero byte if high bit is set (to prevent negative interpretation)
	if len(b) > 0 && b[0]&0x80 != 0 {
		b = append([]byte{0}, b...)
	}
	return b
}

// AccountIDToAddress converts a 20-byte XRPL account ID to a classic r-address.
func AccountIDToAddress(accountID []byte) (string, error) {
	if len(accountID) != addresscodec.AccountAddressLength {
		return "", fmt.Errorf("invalid account ID length: expected %d, got %d", addresscodec.AccountAddressLength, len(accountID))
	}
	return addresscodec.EncodeAccountIDToClassicAddress(accountID)
}

// CompressedPubKeyToAddress derives an XRPL classic address from a compressed secp256k1 public key.
func CompressedPubKeyToAddress(compressedPubKey []byte) (string, error) {
	if len(compressedPubKey) != secp256k1CompressedPubKeyLen {
		return "", fmt.Errorf("invalid compressed public key length: expected %d, got %d", secp256k1CompressedPubKeyLen, len(compressedPubKey))
	}
	pubKeyHex := hex.EncodeToString(compressedPubKey)
	return addresscodec.EncodeClassicAddressFromPublicKeyHex(pubKeyHex)
}

// SHA512Half computes the first 32 bytes of SHA-512.
func SHA512Half(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:32]
}

// xrplMaxSignificantDigits is the maximum number of significant digits
// allowed in an XRPL IOU amount value.
const xrplMaxSignificantDigits = 15

// formatDecimalAmountForIOU converts an integer amount and decimals count to a decimal string.
// For example, formatDecimalAmountForIOU(12345, 3) returns "12.345".
// It uses string-based manipulation (not floating-point arithmetic) to avoid
// precision loss, and truncates the result to 15 significant digits as required
// by the XRPL IOU format.
func formatDecimalAmountForIOU(amount uint64, decimals uint8) string {
	if decimals == 0 {
		return strconv.FormatUint(amount, 10)
	}

	// Convert to string and insert the decimal point via string manipulation.
	digits := strconv.FormatUint(amount, 10)
	d := int(decimals)

	var intPart, fracPart string
	if len(digits) <= d {
		// Entire value is fractional: pad with leading zeros.
		intPart = "0"
		fracPart = strings.Repeat("0", d-len(digits)) + digits
	} else {
		intPart = digits[:len(digits)-d]
		fracPart = digits[len(digits)-d:]
	}

	// Truncate to xrplMaxSignificantDigits significant digits.
	full := intPart + fracPart
	sigCount := 0
	started := false
	truncAt := len(full) // index at which to stop keeping digits
	for i, c := range full {
		if c != '0' {
			started = true
		}
		if started {
			sigCount++
			if sigCount == xrplMaxSignificantDigits {
				truncAt = i + 1
				break
			}
		}
	}

	// Rebuild intPart and fracPart from the truncated digit string.
	if truncAt < len(full) {
		truncated := full[:truncAt]
		if truncAt <= len(intPart) {
			intPart = truncated + strings.Repeat("0", len(intPart)-truncAt)
			fracPart = ""
		} else {
			fracPart = truncated[len(intPart):]
		}
	}

	// Remove trailing zeros from fractional part.
	fracPart = strings.TrimRight(fracPart, "0")

	if fracPart == "" {
		return intPart
	}
	return intPart + "." + fracPart
}
