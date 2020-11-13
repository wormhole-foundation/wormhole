package key

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"

	tmcrypto "github.com/tendermint/tendermint/crypto"
)

// SigningAlgo - wrapper to expose interface
type SigningAlgo = keys.SigningAlgo

// StdPrivKey - wrapper to expose interface
type StdPrivKey = tmcrypto.PrivKey

const (
	// MultiAlgo implies that a pubkey is a multisignature
	MultiAlgo = keys.MultiAlgo
	// Secp256k1 uses the Bitcoin secp256k1 ECDSA parameters.
	Secp256k1 = keys.Secp256k1
	// Ed25519 represents the Ed25519 signature system.
	// It is currently not supported for end-user keys (wallets/ledgers).
	Ed25519 = keys.Ed25519
	// Sr25519 represents the Sr25519 signature system.
	Sr25519 = keys.Sr25519
)

func init() {
	// Set terra BIP44 coin type
	sdk.GetConfig().SetCoinType(330)
}

// CreateMnemonic - create new mnemonic
func CreateMnemonic() (string, error) {
	// Default number of words (24): This generates a mnemonic directly from the
	// number of words by reading system entropy.
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}

	return bip39.NewMnemonic(entropy)
}

// CreateHDPath returns BIP 44 object from account and index parameters.
func CreateHDPath(account uint32, index uint32) string {
	return keys.CreateHDPath(account, index).String()
}

// DerivePrivKey - derive prive key bytes
func DerivePrivKey(mnemonic string, hdPath string) ([]byte, error) {
	return keys.StdDeriveKey(mnemonic, "", hdPath, Secp256k1)
}

// StdPrivKeyGen is the default PrivKeyGen function in the keybase.
// For now, it only supports Secp256k1
func StdPrivKeyGen(bz []byte) (StdPrivKey, error) {
	return keys.StdPrivKeyGen(bz, Secp256k1)
}
