package devnet

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeterministicEcdsaKeyByIndex(t *testing.T) {
	type test struct {
		index            uint64
		privKeyHexSubStr string
	}

	tests := []test{
		{index: 0, privKeyHexSubStr: "cfb12303a1"},
		{index: 1, privKeyHexSubStr: "c3b2e45c42"},
		{index: 2, privKeyHexSubStr: "9f790d3f08"},
		{index: 3, privKeyHexSubStr: "b20cc49d6f"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.index), func(t *testing.T) {
			privKey := DeterministicEcdsaKeyByIndex(crypto.S256(), tc.index)
			got := crypto.FromECDSA(privKey)
			assert.Equal(t, tc.privKeyHexSubStr, hex.EncodeToString(got)[0:10])
		})
	}

}
