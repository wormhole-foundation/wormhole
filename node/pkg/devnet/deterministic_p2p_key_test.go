package devnet

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeterministicP2PPrivKeyByIndex(t *testing.T) {
	type test struct {
		index            int64
		privKeyHexSubStr string
	}

	tests := []test{
		{index: 0, privKeyHexSubStr: "0194fdc2fa"},
		{index: 1, privKeyHexSubStr: "52fdfc0721"},
		{index: 2, privKeyHexSubStr: "2f8282cbe2"},
		{index: 3, privKeyHexSubStr: "85fbe72b60"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.index), func(t *testing.T) {
			privKey := DeterministicP2PPrivKeyByIndex(tc.index)
			got, _ := privKey.Raw()
			assert.Equal(t, tc.privKeyHexSubStr, hex.EncodeToString(got)[0:10])
		})
	}

}
