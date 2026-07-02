package wormconn

import (
	"os"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/cosmos/cosmos-sdk/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

func LoadWormchainPrivKey(path string, passPhrase string) (cryptotypes.PrivKey, error) {
	if err := common.ValidatePrivateKeyFilePermissions(path); err != nil {
		return nil, err
	}

	armor, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	key, _, err := crypto.UnarmorDecryptPrivKey(string(armor), passPhrase)
	return key, err
}
