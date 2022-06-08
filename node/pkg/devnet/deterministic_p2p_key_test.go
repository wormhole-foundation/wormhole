package devnet

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeterministicP2PPrivKeyByIndex(t *testing.T) {
	type test struct {
		index      int64
		privKeyHex string
	}

	tests := []test{
		{index: 0, privKeyHex: "0194fdc2fa2ffcc041d3ff12045b73c86e4ff95ff662a5eee82abdf44a2d0b7597f3bd871315281e8b83edc7a9fd0541066154449070ccdb3cdd42cf69ccde88"},
		{index: 1, privKeyHex: "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c6496f1581709bb7b1ef030d210db18e3b0ba1c776fba65d8cdaad05415142d189f8"},
		{index: 2, privKeyHex: "2f8282cbe2f9696f3144c0aa4ced56dbd967dc2897806af3bed8a63aca16e18b8ed90420802c83b41e4a7fa94ce5f05792ea8bff3d7a63572e5c73454eaef51d"},
		{index: 3, privKeyHex: "85fbe72b6064289004a531f967898df5319ee02992fdd84021fa5052434bf6ee11bba3ed1721948cefb4e50b0a0bb5cad8a6b52dc7b1a40f4f6652105c91e2c4"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.index), func(t *testing.T) {
			privKey := DeterministicP2PPrivKeyByIndex(tc.index)
			got, _ := privKey.Raw()
			assert.Equal(t, tc.privKeyHex, hex.EncodeToString(got))
		})
	}

}
