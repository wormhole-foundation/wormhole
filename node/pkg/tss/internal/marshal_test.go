package internal

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/test-go/testify/assert"
)

func TestMarshalSecretKey(t *testing.T) {
	a := assert.New(t)
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	a.NoError(err)

	bz := PrivateKeyToPem(sk)
	unmarshaled, err := PemToPrivateKey(bz)
	a.NoError(err)

	a.True(sk.PublicKey.Equal(&unmarshaled.PublicKey))
	a.True(sk.Equal(unmarshaled))

}

func TestMarshalPK(t *testing.T) {
	a := assert.New(t)
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	a.NoError(err)

	bz, err := PublicKeyToPem(&sk.PublicKey)
	a.NoError(err)

	unmarshaled, err := PemToPublicKey(bz)
	a.NoError(err)

	a.True(sk.PublicKey.Equal(unmarshaled))
}
