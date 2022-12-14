package devnet

import (
	"os"

	"github.com/cosmos/cosmos-sdk/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

var wormchainKeyPassphrase = "test0000"

func LoadWormchainPrivKey(path string) (cryptotypes.PrivKey, error) {
	armor, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	key, _, err := crypto.UnarmorDecryptPrivKey(string(armor), wormchainKeyPassphrase)
	return key, err
}
