package vaa

import (
	"github.com/certusone/wormhole/bridge/third_party/chainlink/cryptotest"
	"github.com/certusone/wormhole/bridge/third_party/chainlink/secp256k1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
	"time"
)

var randomStream = cryptotest.NewStream(&testing.T{}, 0)

func TestSerializeDeserialize(t *testing.T) {
	tests := []struct {
		name string
		vaa  *VAA
	}{
		{
			name: "BodyTransfer",
			vaa: &VAA{
				Version:          1,
				GuardianSetIndex: 9,
				Signature: &Signature{
					Sig:     [32]byte{2, 8},
					Address: common.Address{1, 2, 3, 4},
				},
				Timestamp: time.Unix(2837, 0),
				Payload: &BodyTransfer{
					Nonce:         38,
					SourceChain:   2,
					TargetChain:   1,
					TargetAddress: Address{2, 1, 3},
					Asset: &AssetMeta{
						Chain:   9,
						Address: Address{9, 2, 4},
					},
					Amount: big.NewInt(29),
				},
			},
		},
		{
			name: "GuardianSetUpdate",
			vaa: &VAA{
				Version:          1,
				GuardianSetIndex: 9,
				Signature: &Signature{
					Sig:     [32]byte{2, 8},
					Address: common.Address{1, 2, 3, 4},
				},
				Timestamp: time.Unix(2837, 0),
				Payload: &BodyGuardianSetUpdate{
					Key:      secp256k1.Generate(randomStream).Public,
					NewIndex: 2,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			vaaData, err := test.vaa.Serialize()
			require.NoError(t, err)

			vaaParsed, err := ParseVAA(vaaData)
			require.NoError(t, err)

			require.EqualValues(t, test.vaa, vaaParsed)
		})
	}
}

func TestVerifySignature(t *testing.T) {
	key := secp256k1.Generate(randomStream)

	v := &VAA{
		Version:          8,
		GuardianSetIndex: 9,
		Timestamp:        time.Unix(2837, 0),
		Payload: &BodyTransfer{
			SourceChain:   2,
			TargetChain:   1,
			TargetAddress: Address{2, 1, 3},
			Asset: &AssetMeta{
				Chain:   9,
				Address: Address{9, 2, 4},
			},
			Amount: big.NewInt(29),
		},
	}

	require.NoError(t, v.Sign(key))
	require.True(t, v.VerifySignature(key.Public))
}
