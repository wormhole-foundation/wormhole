package tss

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	"github.com/xlabs/tss-lib/v2/tss"
)

func (s *GuardianStorage) unmarshalFromJSON(storageData []byte) error {
	if err := json.Unmarshal(storageData, &s); err != nil {
		return err
	}

	if s.PrivateKey == nil {
		return fmt.Errorf("TlsPrivateKey is nil")
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

	return s.SetInnerFields()
}

func (s *GuardianStorage) SetInnerFields() error {
	signingKey, err := internal.PemToPrivateKey(s.PrivateKey)
	if err != nil {
		return fmt.Errorf("error parsing tls private key: %v", err)
	}

	s.signingKey = signingKey

	pk, err := internal.PemToPublicKey(s.Self.Key)
	if err != nil {
		return err
	}

	if !s.signingKey.PublicKey.Equal(pk) {
		return fmt.Errorf("signing key does not match the public key stored in Self.Key")
	}

	if !s.signingKey.Curve.IsOnCurve(pk.X, pk.Y) {
		return fmt.Errorf("invalid public key, it isn't on the curve")
	}

	tlsCert, err := tls.X509KeyPair(s.TlsX509, s.PrivateKey)
	if err != nil {
		return fmt.Errorf("error loading tls cert: %v", err)
	}

	s.tlsCert = &tlsCert

	if len(s.GuardianCerts) != len(s.Guardians) {
		return fmt.Errorf("number of guardians and guardiansCerts do not match")
	}

	if err := s.parseCerts(); err != nil {
		return err
	}

	s.guardiansProtoIDs = make([]*tsscommv1.PartyId, len(s.Guardians))
	for i, guardian := range s.Guardians {
		s.guardiansProtoIDs[i] = partyIdToProto(guardian)
	}

	s.guardianToCert = make(map[string]*x509.Certificate)
	for i, guardian := range s.Guardians {
		s.guardianToCert[partyIdToString(guardian)] = s.guardiansCerts[i]
	}

	return s.setCertToGuardian()
}

func (s *GuardianStorage) setCertToGuardian() error {
	s.pemkeyToGuardian = make(map[string]*tss.PartyID)
	for i, crt := range s.guardiansCerts {
		var byteKey []byte

		switch m := crt.PublicKey.(type) {
		case *ecdsa.PublicKey:
			bts, err := internal.PublicKeyToPem(m)
			if err != nil {
				return fmt.Errorf("error parsing guardian %v cert: %v", i, err)
			}

			byteKey = bts
		case []byte:
			byteKey = m
		default:
			return fmt.Errorf("error guardian %v cert stored with non-ecdsa publickey", i)
		}

		s.pemkeyToGuardian[string(byteKey)] = s.Guardians[i]
	}

	return nil
}

func (s *GuardianStorage) parseCerts() error {
	s.guardiansCerts = make([]*x509.Certificate, len(s.Guardians))
	for i, cert := range s.GuardianCerts {
		c, err := internal.PemToCert(cert)
		if err != nil {
			return fmt.Errorf("error parsing guardian %v cert: %v", i, err)
		}

		if _, ok := c.PublicKey.(*ecdsa.PublicKey); !ok {
			return fmt.Errorf("error guardian %v cert stored with non-ecdsa publickey", i)
		}

		s.guardiansCerts[i] = c
	}

	return nil
}
