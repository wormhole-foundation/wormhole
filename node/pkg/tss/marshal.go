package tss

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/yossigi/tss-lib/v2/crypto"
	"github.com/yossigi/tss-lib/v2/tss"
)

func marshalEcdsaSecretkey(key *ecdsa.PrivateKey) []byte {
	return key.D.Bytes()
}

func unmarshalEcdsaSecretKey(bz []byte) *ecdsa.PrivateKey {
	// TODO: I straggled with unmarshalling ecdh.PrivateKey, as a result I resorted to "unsafe" and deprecated method:
	x, y := tss.S256().ScalarBaseMult(bz)
	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: tss.S256(),
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(bz),
	}
}

var errInvalidPublicKey = errors.New("invalid public key")

func unmarshalEcdsaPublickey(curve elliptic.Curve, bz []byte) (*ecdsa.PublicKey, error) {
	pnt := &crypto.ECPoint{}
	if err := pnt.GobDecode(bz); err != nil {
		return nil, err
	}

	if !curve.IsOnCurve(pnt.X(), pnt.Y()) {
		return nil, errInvalidPublicKey
	}

	pnt.SetCurve(curve)

	return pnt.ToECDSAPubKey(), nil
}

func marshalEcdsaPublickey(pk *ecdsa.PublicKey) ([]byte, error) {
	pnt, err := crypto.NewECPoint(pk.Curve, pk.X, pk.Y)
	if err != nil {
		return nil, err
	}
	return pnt.GobEncode()
}

func (s *GuardianStorage) unmarshalFromJSON(storageData []byte) error {
	if err := json.Unmarshal(storageData, &s); err != nil {
		return err
	}

	if s.SecretKey == nil {
		return fmt.Errorf("secretKey is nil")
	}

	if len(s.Guardians) == 0 {
		return fmt.Errorf("no guardians array given")
	}

	if s.Threshold > len(s.Guardians) {
		return fmt.Errorf("threshold is higher than the number of guardians")
	}

	return nil
}

func (s *GuardianStorage) load(storagePath string) error {
	if s == nil {
		return fmt.Errorf("GuardianStorage is nil")
	}

	storageData, err := os.ReadFile(storagePath)
	if err != nil {
		return err
	}

	if err := s.unmarshalFromJSON(storageData); err != nil {
		return err
	}

	s.signingKey = unmarshalEcdsaSecretKey(s.SecretKey)

	pk, err := unmarshalEcdsaPublickey(tss.S256(), s.Self.Key)
	if err != nil {
		return err
	}

	if !s.signingKey.PublicKey.Equal(pk) {
		return fmt.Errorf("signing key does not match the public key stored as Self partyId")
	}

	if !tss.S256().IsOnCurve(pk.X, pk.Y) {
		return fmt.Errorf("invalid public key, it isn't on the curve")
	}

	if len(s.Symkeys) != len(s.Guardians) {
		if err := s.createSharedSecrets(); err != nil {
			return err
		}
	}

	return nil
}

func (s *GuardianStorage) createSharedSecrets() error {
	curve := tss.S256()
	s.Symkeys = make([]symKey, len(s.Guardians))

	for i, g := range s.Guardians {
		gpk, err := unmarshalEcdsaPublickey(curve, g.Key)
		if err != nil {
			return errors.New("failed to unmarshal public key")
		}

		x, y := curve.ScalarMult(gpk.X, gpk.Y, s.SecretKey)
		sharedKey, err := crypto.NewECPoint(curve, x, y)
		if err != nil {
			return err
		}

		// TODO: Ensure that GobEncode is deterministic. otherwise symkey will be for each guardian in the pair.
		sharedKeyBytes, err := sharedKey.GobEncode()
		if err != nil {
			return err
		}

		// reducing the bytes of the sharedKey to 32 bytes (using sha512_256 to avoid collisions).
		tmp := sha512.Sum512_256(sharedKeyBytes)
		s.Symkeys[i] = tmp[:]
	}

	return nil
}
