package common

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAESGCM(t *testing.T) {

	type test struct {
		data []byte
		enc  string
	}

	key := []byte("01234567890123456789012345678901")

	tests := []test{
		{data: []byte("lower"), enc: "7b3023dde90ca9598eff203d92145d0c363c61743edf23110bee39c1b01200fc6f"},
		{data: []byte("UPPER"), enc: "f70a6549fd8a004551f2f3aeebc74d03efd3520aace6bd51f1b38cac8a94aae352"},
		{data: []byte("Mixed"), enc: "95af0109040796bda46a223acebca97305aff135bd628ce70812d59a13c1a821ac"},
		{data: []byte("AlphaNum1"), enc: "6385f1d2ae006fc7fa2b5e0b74f12b71bf25e2ec3d9ee70baf142703a7c187de08a22806d3"},
		{data: []byte("12345"), enc: "bd083e01ca72af788a04866ddd9c0d061f64bc0a0e0d23432fbf1d6fcc724c126c"},
	}

	for _, testCase := range tests {
		// Verify that we can encrypt plain text
		t.Run(string(testCase.data), func(t *testing.T) {
			enc, err := EncryptAESGCM(testCase.data, key)
			assert.Nil(t, err)
			assert.NotNil(t, enc)
			// AESGCM is non-deterministic, so we cannot expect consistent cipher text
		})

		// Verify that we can decrypt cipher text
		t.Run(string(testCase.data), func(t *testing.T) {
			// Convert the test hexified cipher text back to bytes, mostly for testcase readability purposes
			enc, err := hex.DecodeString(testCase.enc)
			assert.Nil(t, err)
			assert.NotNil(t, enc)

			dec, err := DecryptAESGCM(enc, key)
			assert.Nil(t, err)
			assert.NotNil(t, dec)
			assert.Equal(t, testCase.data, dec)
		})

		// Verify that we can encrypt plain text, decrypt cipher text, and verify that both match
		t.Run(string(testCase.data), func(t *testing.T) {
			enc, err := EncryptAESGCM(testCase.data, key)
			assert.Nil(t, err)
			assert.NotNil(t, enc)

			dec, err := DecryptAESGCM(enc, key)
			assert.Nil(t, err)
			assert.NotNil(t, dec)
			assert.Equal(t, testCase.data, dec)
		})
	}
}
