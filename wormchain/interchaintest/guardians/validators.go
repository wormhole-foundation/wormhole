package guardians

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type ValSet struct {
	Vals  []Val
	Total int
}

type Val struct {
	Addr []byte
	Priv *ecdsa.PrivateKey
}

func CreateVal(t *testing.T) *Val {
	priv, err := crypto.GenerateKey()
	require.NoError(t, err)

	signer := crypto.PubkeyToAddress(priv.PublicKey).Bytes()

	return &Val{
		Addr: signer,
		Priv: priv,
	}
}

func CreateValSet(t *testing.T, total int) *ValSet {
	// If total == 1, mirror the guardian keys in scripts/devnet-consts.json
	if total == 1 {
		privHex := "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"
		priv, err := crypto.HexToECDSA(privHex)
		require.NoError(t, err)
		signer := crypto.PubkeyToAddress(priv.PublicKey).Bytes()

		return &ValSet{
			Vals: []Val{
				{signer, priv},
			},
			Total: 1,
		}
	}

	var valSet ValSet
	for i := 0; i < total; i++ {
		valSet.Vals = append(valSet.Vals, *CreateVal(t))
	}

	valSet.Total = total
	return &valSet
}
