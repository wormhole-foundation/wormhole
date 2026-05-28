// Package currencycodec encodes and decodes XRPL currency codes between their
// JSON string representation and the canonical 20-byte internal form used in
// NTT source token derivation.
package currencycodec

import (
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// NonStandardHexLen is the length of a non-standard (160-bit) currency code
	// when expressed as a hex string: 20 bytes * 2 = 40 characters.
	NonStandardHexLen = 40
	// NormalizedLen is the length in bytes of the canonical currency representation.
	NormalizedLen = 20
)

// Decode converts an XRPL currency string to its canonical 20-byte form.
// Standard codes: [0x00][ASCII bytes (3)][trailing zeros]
// Non-standard codes: [raw 160-bit value] (40-character hex string)
// "XRP" is rejected because it is not a valid trust-line currency.
func Decode(currency string) ([NormalizedLen]byte, error) {
	var result [NormalizedLen]byte

	// XRP is disallowed as a currency code
	if strings.ToUpper(currency) == "XRP" {
		return result, fmt.Errorf("XRP is not a valid currency code for trust lines")
	}

	// Non-standard currency code (40-character hex)
	if len(currency) == NonStandardHexLen {
		decoded, err := hex.DecodeString(currency)
		if err != nil {
			return result, fmt.Errorf("failed to decode hex currency: %w", err)
		}
		copy(result[:], decoded)
		return result, nil
	}

	// Standard currency code (1-3 character ASCII)
	if len(currency) == 0 || len(currency) > 3 {
		return result, fmt.Errorf("invalid standard currency code length: %d", len(currency))
	}

	// Standard format: [0x00][ASCII bytes][trailing zeros]
	result[0] = 0x00
	copy(result[12:12+len(currency)], []byte(currency))

	return result, nil
}

// Encode converts a canonical 20-byte XRPL currency to its JSON string form.
// Standard 3-character currencies are stored as ASCII in bytes 12-14 with
// zeros elsewhere. Non-standard (160-bit) currencies are returned as
// 40-character hex strings.
func Encode(currency [NormalizedLen]byte) string {
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
