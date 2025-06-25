package tss

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"

	"github.com/certusone/wormhole/node/pkg/tss/internal"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/fxamacker/cbor/v2"
	"github.com/xlabs/multi-party-sig/pkg/math/curve"
	"github.com/xlabs/multi-party-sig/protocols/frost"
	common "github.com/xlabs/tss-common"
)

func (s *GuardianStorage) unmarshalFromJSON(storageData []byte) error {
	if err := json.Unmarshal(storageData, &s); err != nil {
		return err
	}

	if s.PrivateKey == nil {
		return fmt.Errorf("TlsPrivateKey is nil")
	}

	if len(s.IdentitiesKeep.Identities) == 0 {
		return fmt.Errorf("no guardians array given")
	}

	if s.Threshold > len(s.IdentitiesKeep.Identities) {
		return fmt.Errorf("threshold is higher than the number of guardians")
	}

	if s.TSSSecrets == nil {
		return fmt.Errorf("TSSSecrets is nil")
	}

	cnf := frost.EmptyConfig(curve.Secp256k1{})
	if err := cbor.Unmarshal(s.TSSSecrets, &cnf); err != nil { // TODO: find a way to remove cbor dependency
		return fmt.Errorf("error unmarshalling TSSSecrets: %v", err)
	}

	if len(cnf.VerificationShares.Points) != len(s.IdentitiesKeep.Identities) {
		return fmt.Errorf("number of verification shares does not match number of guardians")
	}

	s.frostconf = cnf

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

	pk, err := internal.PemToPublicKey(s.Self.KeyPEM)
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

	if err := s.fillAndValidateStoredIdentities(); err != nil {
		return err
	}

	numGuardians := len(s.IdentitiesKeep.Identities)

	s.IdentitiesKeep.peerCerts = make([]*x509.Certificate, numGuardians)
	s.IdentitiesKeep.partyIds = make([]*common.PartyID, numGuardians)
	s.IdentitiesKeep.pemkeyToIndex = make(map[string]int)
	s.IdentitiesKeep.vaav1PubToIdentity = make(map[ethcommon.Address]int)
	// Since the guardians are sorted by key, we can use their position as their index.
	for i := range numGuardians {
		s.IdentitiesKeep.peerCerts[i] = s.IdentitiesKeep.Identities[i].Cert
		s.IdentitiesKeep.partyIds[i] = s.IdentitiesKeep.Identities[i].Pid
		s.IdentitiesKeep.pemkeyToIndex[string(s.IdentitiesKeep.Identities[i].KeyPEM)] = i

		if s.IdentitiesKeep.Identities[i].VAAv1PubKey != nil {
			s.IdentitiesKeep.vaav1PubToIdentity[*(s.IdentitiesKeep.Identities[i].VAAv1PubKey)] = i
		}
	}

	if s.LeaderIdentity == nil {
		// since the guardians are expected to be sorted already, the first guardian is the leader.
		s.LeaderIdentity = s.IdentitiesKeep.Identities[0].KeyPEM
	}

	s.isleader = bytes.Equal(s.Self.KeyPEM, s.LeaderIdentity)

	return nil
}

// validates the stored Identity structs. Ensures that the cert and key are valid and match.
// ensures no nil values are stored. Verifies that the tss-lib.PartyIDs are unique.
func (s *GuardianStorage) fillAndValidateStoredIdentities() error {
	uniquePidIDs := make(map[string]struct{})

	for i, id := range s.Identities {
		if id == nil {
			return fmt.Errorf("error guardian %v is nil", i)
		}

		c, key, err := extractCertAndKeyFromPem(id.CertPem)
		if err != nil {
			return fmt.Errorf("error parsing guardian %v: %w", i, err)
		}

		if id.Pid == nil {
			return fmt.Errorf("error guardian %v PartyID is nil", i)
		}

		if len(id.Hostname) == 0 {
			return fmt.Errorf("error guardian %v hostname is empty", i)
		}

		if len(id.Pid.GetID()) == 0 {
			return fmt.Errorf("error guardian %v PartyID.Id is empty", i)
		}

		if _, ok := uniquePidIDs[id.Pid.GetID()]; ok {
			return fmt.Errorf("error guardian %v PartyID.Id is not unique", i)
		}
		uniquePidIDs[id.Pid.GetID()] = struct{}{}

		// storing the cert and key in the identity struct.
		id.Key = key
		id.Cert = c

		keypem, err := internal.PublicKeyToPem(key)
		if err != nil {
			return fmt.Errorf("error converting guardian %v  cert's PK  to pem: %v", i, err)
		}

		id.KeyPEM = keypem

		id.CommunicationIndex = SenderIndex(i)
		id.networkname = id.portAndHostToNetName()
	}

	return nil
}

func extractCertAndKeyFromPem(pem PEM) (*x509.Certificate, *ecdsa.PublicKey, error) {
	c, err := internal.PemToCert(pem)
	if err != nil {
		return nil, nil, err
	}

	key, ok := c.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, fmt.Errorf("cert stored with non-ecdsa publickey")
	}

	return c, key, nil
}

func (s *GuardianStorage) NumGuardians() int {
	if s == nil {
		return 0
	}

	return len(s.Identities)
}
