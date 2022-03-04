package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAESGCM(t *testing.T) {

	type test struct {
		data []byte
	}

	tests := []test{
		{data: []byte("lower")},
		{data: []byte("UPPER")},
		{data: []byte("Mixed")},
		{data: []byte("AlphaNum1")},
		{data: []byte("12345")},
	}

	key := []byte("01234567890123456789012345678901")

	for _, tc := range tests {
		// Test that we can encrypt data without error.
		enc, enc_err := EncryptAESGCM(tc.data, key)
		assert.Nil(t, enc_err)

		// Test that we can decrypt data without error.
		dec, dec_err := DecryptAESGCM(enc, key)
		assert.Nil(t, dec_err)

		// Test that our origin data and our decrypted data are exactly the same.
		assert.Equal(t, dec, tc.data)
	}
}
