package currencycodec

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecode_StandardCode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		currency string
	}{
		{"USD", "USD"},
		{"EUR", "EUR"},
		{"BTC", "BTC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Decode(tc.currency)
			require.NoError(t, err)
			assert.Equal(t, byte(0x00), result[0])
			assert.Equal(t, []byte(tc.currency), result[12:12+len(tc.currency)])
		})
	}
}

func TestDecode_HexCode(t *testing.T) {
	hexCurrency := "524C555344000000000000000000000000000000" // RLUSD
	result, err := Decode(hexCurrency)
	require.NoError(t, err)
	expected, _ := hex.DecodeString(hexCurrency)
	assert.Equal(t, expected, result[:])
}

func TestDecode_XRPDisallowed(t *testing.T) {
	_, err := Decode("XRP")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "XRP is not a valid currency code")

	_, err = Decode("xrp")
	require.Error(t, err)
}

func TestDecode_SingleCharCode(t *testing.T) {
	result, err := Decode("X")
	require.NoError(t, err)
	assert.Equal(t, byte(0x00), result[0])
	assert.Equal(t, []byte("X"), result[12:13])
}

func TestDecode_InvalidHexLength(t *testing.T) {
	// 38 characters - not 1-3 ASCII and not 40-char hex
	_, err := Decode("524C5553440000000000000000000000000000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid standard currency code length")
}

func TestDecode_InvalidHexChars(t *testing.T) {
	_, err := Decode("524C55534400000000000000000000000000ZZZZ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode hex currency")
}

func TestDecode_EmptyString(t *testing.T) {
	_, err := Decode("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid standard currency code length")
}

func TestEncode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		currency [20]byte
		expected string
	}{
		{
			name: "standard USD",
			currency: [20]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x55, 0x53, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			expected: "USD",
		},
		{
			name: "standard EUR",
			currency: [20]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x45, 0x55, 0x52, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			expected: "EUR",
		},
		{
			name: "non-standard hex currency",
			currency: [20]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
				0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14,
			},
			expected: "0102030405060708090a0b0c0d0e0f1011121314",
		},
		{
			// Looks like a standard code in bytes 12-14 but has a non-zero tail —
			// must be treated as non-standard hex.
			name: "non-standard with zero prefix and nonzero tail",
			currency: [20]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x55, 0x53, 0x44, 0xff, 0x00, 0x00, 0x00, 0x00,
			},
			expected: "000000000000000000000000555344ff00000000",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Encode(tc.currency))
		})
	}
}

// TestRoundTrip verifies that Encode and Decode are inverses for the inputs
// they are expected to emit on the wire: 3-character standard currency codes
// and lowercase 40-char hex non-standard codes. (Decode tolerates 1-2 char
// inputs and uppercase hex, but Encode always emits canonical forms; those
// are covered by the separate Encode/Decode tests above.)
func TestRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{"USD", "USD"},
		{"EUR", "EUR"},
		{"BTC", "BTC"},
		{"non-standard hex - RLUSD", "524c555344000000000000000000000000000000"},
		{"non-standard hex - arbitrary", "0102030405060708090a0b0c0d0e0f1011121314"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := Decode(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.input, Encode(decoded))
		})
	}
}
