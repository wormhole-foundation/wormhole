package wormconn

import (
	"os"

	"github.com/cosmos/cosmos-sdk/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

func LoadWormchainPrivKey(path string, passPhrase string) (cryptotypes.PrivKey, error) {
	armor, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	key, _, err := crypto.UnarmorDecryptPrivKey(string(armor), passPhrase)
	return key, err
}
