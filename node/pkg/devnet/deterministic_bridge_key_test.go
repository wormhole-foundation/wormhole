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
		{index: 4, privKeyHex: "eded5a2fdcb5bbbfa5b07f2a91393813420e7ac30a72fc935b6df36f8294b855"},
		{index: 5, privKeyHex: "00d39587c3556f289677a837c7f3c0817cb7541ce6e38a243a4bdc761d534c5e"},
		{index: 6, privKeyHex: "da534d61a8da77b232f3a2cee55c0125e2b3e33a5cd8247f3fe9e72379445c3b"},
		{index: 7, privKeyHex: "cdbabfc2118eb00bc62c88845f3bbd03cb67a9e18a055101588ca9b36387006c"},
		{index: 8, privKeyHex: "c83d36423820e7350428dc4abe645cb2904459b7d7128adefe16472fdac397ba"},
		{index: 9, privKeyHex: "1cbf4e1388b81c9020500fefc83a7a81f707091bb899074db1bfce4537428112"},
		{index: 10, privKeyHex: "17646a6ba14a541957fc7112cc973c0b3f04fce59484a92c09bb45a0b57eb740"},
		{index: 11, privKeyHex: "eb94ff04accbfc8195d44b45e7c7da4c6993b2fbbfc4ef166a7675a905df9891"},
		{index: 12, privKeyHex: "053a6527124b309d914a47f5257a995e9b0ad17f14659f90ed42af5e6e262b6a"},
		{index: 13, privKeyHex: "3fbf1e46f6da69e62aed5670f279e818889aa7d8f1beb7fd730770fd4f8ea3d7"},
		{index: 14, privKeyHex: "53b05697596ba04067e40be8100c9194cbae59c90e7870997de57337497172e9"},
		{index: 15, privKeyHex: "4e95cb2ff3f7d5e963631ad85c28b1b79cb370f21c67cbdd4c2ffb0bf664aa06"},
		{index: 16, privKeyHex: "01b8c448ce2c1d43cfc5938d3a57086f88e3dc43bb8b08028ecb7a7924f4676f"},
		{index: 17, privKeyHex: "1db31a6ba3bcd54d2e8a64f8a2415064265d291593450c6eb7e9a6a986bd9400"},
		{index: 18, privKeyHex: "70d8f1c9534a0ab61a020366b831a494057a289441c07be67e4288c44bc6cd5d"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.index), func(t *testing.T) {
			privKey := InsecureDeterministicEcdsaKeyByIndex(crypto.S256(), tc.index)
			got := crypto.FromECDSA(privKey)
			assert.Equal(t, tc.privKeyHex, hex.EncodeToString(got))
		})
	}

}
