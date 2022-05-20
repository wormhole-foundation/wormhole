package devnet

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestDeterministicEcdsaKeyByIndex(t *testing.T) {
	type test struct {
		index      uint64
		privKeyHex string
	}

	tests := []test{
		{index: 0, privKeyHex: "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"},
		{index: 1, privKeyHex: "c3b2e45c422a1602333a64078aeb42637370b0f48fe385f9cfa6ad54a8e0c47e"},
		{index: 2, privKeyHex: "9f790d3f08bc4b5cd910d4278f3deb406e57bb5e924906ccd52052bb078ccd47"},
		{index: 3, privKeyHex: "b20cc49d6f2c82a5e6519015fc18aa3e562867f85f872c58f1277cfbd2a0c8e4"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.index), func(t *testing.T) {
			privKey := InsecureDeterministicEcdsaKeyByIndex(crypto.S256(), tc.index)
			got := crypto.FromECDSA(privKey)
			assert.Equal(t, tc.privKeyHex, hex.EncodeToString(got))
		})
	}

}
