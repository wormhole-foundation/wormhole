package wormchain

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/vaa"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestStringToUInt(t *testing.T) {
	tests := []struct {
		label     string
		str       string
		num       uint64
		willError bool
	}{
		{label: "preceding slash", str: "\"1", num: 1, willError: false},
		{label: "trailing slash", str: "1\"", num: 1, willError: false},
		{label: "max range", str: "1", num: 1, willError: false},
		{label: "max range", str: "18446744073709551615", num: 18446744073709551615, willError: false},
		{label: "negative number", str: "-1", num: 0, willError: true},
		{label: "max range plus one", str: "18446744073709551616", num: 0, willError: true},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			num, err := stringToUint(tc.str)

			if tc.willError == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.num, num)

		})
	}
}

func TestSecondDecode(t *testing.T) {
	tests := []struct {
		label     string
		str       string
		bytes     []byte
		willError bool
	}{
		{label: "simple", str: "Zm9vCg==", bytes: []byte{0x66, 0x6f, 0x6f, 0xa}, willError: false},
		{label: "corrupted", str: "XXXXXaGVsbG8=", bytes: []byte(nil), willError: true},
		{label: "address", str: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=", bytes: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4}, willError: false},
		{label: "address", str: "\"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=\"", bytes: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4}, willError: false},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			decodedBytes, err := secondDecode(tc.str)

			if tc.willError == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.bytes, decodedBytes)
		})
	}
}

func TestStringToAddress(t *testing.T) {
	tests := []struct {
		label     string
		str       string
		addr      vaa.Address
		willError bool
	}{
		{label: "simple", str: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=", addr: vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}, willError: false},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			addr, err := StringToAddress(tc.str)

			if tc.willError == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.addr, addr)
		})
	}
}

func TestStringToHash(t *testing.T) {
	tests := []struct {
		label     string
		str       string
		hash      eth_common.Hash
		willError bool
	}{
		{label: "simple", str: "4fae136bb1fd782fe1b5180ba735cdc83bcece3f9b7fd0e5e35300a61c8acd8f", hash: eth_common.HexToHash("4fae136bb1fd782fe1b5180ba735cdc83bcece3f9b7fd0e5e35300a61c8acd8f"), willError: false},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			hash, err := StringToHash(tc.str)

			if tc.willError == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.hash, hash)
		})
	}
}
