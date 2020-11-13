package key

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CreateMnemonic(t *testing.T) {
	_, err := CreateMnemonic()
	assert.NoError(t, err)
}

func Test_DrivePrivKey(t *testing.T) {
	mnemonic, err := CreateMnemonic()
	assert.NoError(t, err)

	// Only Secp256k1 is supported
	_, err = DerivePrivKey(mnemonic, CreateHDPath(1, 1))
	assert.NoError(t, err)
}
