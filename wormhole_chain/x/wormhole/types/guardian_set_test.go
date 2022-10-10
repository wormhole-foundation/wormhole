package types

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestKeysAsAddresses(t *testing.T) {
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	addr1 := crypto.PubkeyToAddress(privKey1.PublicKey)
	addr2 := crypto.PubkeyToAddress(privKey2.PublicKey)

	addrsBytes := [][]byte{}
	addrsBytes = append(addrsBytes, addr1.Bytes())
	addrsBytes = append(addrsBytes, addr2.Bytes())

	addrs := []common.Address{}
	addrs = append(addrs, addr1)
	addrs = append(addrs, addr2)

	guardianSet := GuardianSet{
		Index:          1,
		Keys:           addrsBytes,
		ExpirationTime: 0,
	}

	assert.Equal(t, addrs, guardianSet.KeysAsAddresses())
}

func TestContainsKey(t *testing.T) {
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey3, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	addr1 := crypto.PubkeyToAddress(privKey1.PublicKey)
	addr2 := crypto.PubkeyToAddress(privKey2.PublicKey)
	addr3 := crypto.PubkeyToAddress(privKey3.PublicKey)

	addrsBytes := [][]byte{}
	addrsBytes = append(addrsBytes, addr1.Bytes())
	addrsBytes = append(addrsBytes, addr2.Bytes())

	guardianSet := GuardianSet{
		Index:          1,
		Keys:           addrsBytes,
		ExpirationTime: 0,
	}

	assert.Equal(t, true, guardianSet.ContainsKey(addr1))
	assert.Equal(t, true, guardianSet.ContainsKey(addr2))
	assert.Equal(t, false, guardianSet.ContainsKey(addr3))
}

func TestValidateBasic(t *testing.T) {
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	addr1 := crypto.PubkeyToAddress(privKey1.PublicKey)
	addr2 := crypto.PubkeyToAddress(privKey2.PublicKey)

	addrsBytes := [][]byte{}
	addrsBytes = append(addrsBytes, addr1.Bytes())
	addrsBytes = append(addrsBytes, addr2.Bytes())

	largeAddrsBytes := [][]byte{}
	for i := 0; i < 256; i++ {
		largeAddrsBytes = append(largeAddrsBytes, addr1.Bytes())
	}

	tests := []struct {
		label     string
		gs        GuardianSet
		valid     bool
		willError bool
	}{
		{label: "ValidSet", gs: GuardianSet{Keys: addrsBytes, Index: 1, ExpirationTime: 0}, willError: false},
		{label: "EmptySet", gs: GuardianSet{}, willError: true},
		{label: "TooLargeSet", gs: GuardianSet{Keys: largeAddrsBytes, Index: 1, ExpirationTime: 0}, willError: true},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			err := tc.gs.ValidateBasic()

			if tc.willError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}
