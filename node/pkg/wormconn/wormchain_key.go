package wormconn

import (
	"os"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/cosmos/cosmos-sdk/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

func LoadWormchainPrivKey(path string, passPhrase string) (cryptotypes.PrivKey, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Validate the permissions on the open handle so we check the same file we
	// read, closing the gap between a separate stat and open.
	if err = common.ValidatePrivateKeyFilePermissions(f); err != nil {
		return nil, err
	}

	armor, err := common.SafeRead(f)
	if err != nil {
		return nil, err
	}

	key, _, err := crypto.UnarmorDecryptPrivKey(string(armor), passPhrase)
	return key, err
}
