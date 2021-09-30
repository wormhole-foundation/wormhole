package vaa

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSerializeDeserialize(t *testing.T) {
	tests := []struct {
		name string
		vaa  *VAA
	}{
		{
			name: "NormalVAA",
			vaa: &VAA{
				Version:          1,
				GuardianSetIndex: 9,
				Signatures: []*Signature{
					{
						Index:     1,
						Signature: [65]byte{},
					},
				},
				Timestamp:        time.Unix(2837, 0),
				Nonce:            10,
				Sequence:         3,
				ConsistencyLevel: 5,
				EmitterChain:     8,
				EmitterAddress:   Address{1, 2, 3},
				Payload:          []byte("abc"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			vaaData, err := test.vaa.Marshal()
			require.NoError(t, err)

			println(hex.EncodeToString(vaaData))
			vaaParsed, err := Unmarshal(vaaData)
			require.NoError(t, err)

			require.EqualValues(t, test.vaa, vaaParsed)
		})
	}
}

func TestVerifySignature(t *testing.T) {
	v := &VAA{
		Version:          8,
		GuardianSetIndex: 9,
		Timestamp:        time.Unix(2837, 0),
		Nonce:            5,
		Sequence:         10,
		ConsistencyLevel: 2,
		EmitterChain:     2,
		EmitterAddress:   Address{0, 1, 2, 3, 4},
		Payload:          []byte("abcd"),
	}

	data, err := v.SigningMsg()
	require.NoError(t, err)

	key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)

	sig, err := crypto.Sign(data.Bytes(), key)
	require.NoError(t, err)
	sigData := [65]byte{}
	copy(sigData[:], sig)

	v.Signatures = append(v.Signatures, &Signature{
		Index:     0,
		Signature: sigData,
	})
	addr := crypto.PubkeyToAddress(key.PublicKey)
	require.True(t, v.VerifySignatures([]common.Address{
		addr,
	}))
}

func TestBodyRegisterChain_Serialize(t *testing.T) {
	header, _ := hex.DecodeString("000000000000000000000000000000000000000000546f6b656e427269646765")
	require.Len(t, header, 32)

	var headerB [32]byte
	copy(headerB[:], header)
	msg := &BodyRegisterChain{
		Header:         headerB,
		ChainID:        8,
		EmitterAddress: Address{1, 2, 3, 4},
	}

	data := msg.Serialize()
	require.Equal(t, "000000000000000000000000000000000000000000546f6b656e42726964676501000000080102030400000000000000000000000000000000000000000000000000000000", hex.EncodeToString(data))
}
