// Package xrpl implements XRPL transaction building and signing for the manager service.
package xrpl

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"

	addresscodec "github.com/Peersyst/xrpl-go/address-codec"
	binarycodec "github.com/Peersyst/xrpl-go/binary-codec"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
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
		value := formatDecimalAmount(payload.Amount, payload.TokenDecimals)
		amount = types.IssuedCurrencyAmount{
			Currency: encodeCurrency(payload.Token.Currency),
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
	fee := uint64(m+1) * 12

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
	fee := uint64(m+1) * 12

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
	totalLen := 4 + rLen + sLen // 0x02 + rLen + r + 0x02 + sLen + s

	sig := make([]byte, 0, totalLen+2) // +2 for 0x30 and totalLen
	sig = append(sig, 0x30)            // DER sequence tag
	sig = append(sig, byte(totalLen))  // Total length
	sig = append(sig, 0x02)            // Integer tag for r
	sig = append(sig, byte(rLen))      // r length
	sig = append(sig, r...)            // r value
	sig = append(sig, 0x02)            // Integer tag for s
	sig = append(sig, byte(sLen))      // s length
	sig = append(sig, s...)            // s value

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
	if len(compressedPubKey) != 33 {
		return "", fmt.Errorf("invalid compressed public key length: expected 33, got %d", len(compressedPubKey))
	}
	pubKeyHex := hex.EncodeToString(compressedPubKey)
	return addresscodec.EncodeClassicAddressFromPublicKeyHex(pubKeyHex)
}

// SHA512Half computes the first 32 bytes of SHA-512.
func SHA512Half(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:32]
}

// formatDecimalAmount converts an integer amount and decimals count to a decimal string.
// For example, formatDecimalAmount(12345, 3) returns "12.345".
func formatDecimalAmount(amount uint64, decimals uint8) string {
	if decimals == 0 {
		return strconv.FormatUint(amount, 10)
	}

	// Use big.Float for precision
	f := new(big.Float).SetUint64(amount)
	divisor := new(big.Float).SetFloat64(math.Pow10(int(decimals)))
	f.Quo(f, divisor)

	// Format with exact number of decimal places
	return f.Text('f', int(decimals))
}

// encodeCurrency converts a 20-byte XRPL currency code to the string representation.
// Standard 3-character currencies are stored as ASCII in bytes 12-14 with zeros elsewhere.
// Non-standard (160-bit) currencies are returned as 40-character hex strings.
func encodeCurrency(currency [20]byte) string {
	// Check if this is a standard 3-character currency code
	// Standard format: 12 zero bytes, 3 ASCII bytes, 5 zero bytes
	isStandard := true
	for i := 0; i < 12; i++ {
		if currency[i] != 0 {
			isStandard = false
			break
		}
	}
	if isStandard {
		for i := 15; i < 20; i++ {
			if currency[i] != 0 {
				isStandard = false
				break
			}
		}
	}
	if isStandard && currency[12] != 0 {
		return string(currency[12:15])
	}

	// Non-standard: return as hex
	return hex.EncodeToString(currency[:])
}
