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
	var valSet ValSet
	for i := 0; i < total; i++ {
		valSet.Vals = append(valSet.Vals, *CreateVal(t))
	}

	valSet.Total = total

	return &valSet
}
