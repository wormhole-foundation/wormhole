package guardiand

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAuthProof(t *testing.T) {
	// Create some private/public keys for testing with
	ethPrivateKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	ethPublicAddress := crypto.PubkeyToAddress(ethPrivateKey.PublicKey)
	guardianPrivateKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	guardianPublicAddress := crypto.PubkeyToAddress(guardianPrivateKey.PublicKey)

	// Guardian Side
	//
	// generates proof that ethPublic address was signed by the Guardian Private Key
	//
	digest := crypto.Keccak256Hash(ethPublicAddress.Bytes())
	ethProof, err := crypto.Sign(digest.Bytes(), guardianPrivateKey)
	assert.Nil(t, err)
	assert.NotNil(t, ethProof)

	// Contract Side
	//
	// verifies proof that ethPublic address was signed by the Guardian Private Key
	//
	digest2 := common.BytesToHash(crypto.Keccak256(ethPublicAddress.Bytes()))
	guardianPublicAddressBytes, _ := crypto.Ecrecover(digest2.Bytes(), ethProof)
	guardianPublicAddress2 := common.BytesToAddress(crypto.Keccak256(guardianPublicAddressBytes[1:])[12:])

	// Assert the digests are the same
	assert.Equal(t, digest, digest2)

	// Assert that the guardianPublicAddress from the proof is valid
	assert.Equal(t, guardianPublicAddress, guardianPublicAddress2)
}
