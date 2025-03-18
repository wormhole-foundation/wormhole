package tss

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	"github.com/xlabs/tss-lib/v2/tss"
	"google.golang.org/protobuf/proto"
)

func (s *GuardianStorage) unmarshalFromJSON(storageData []byte) error {
	if err := json.Unmarshal(storageData, &s); err != nil {
		return err
	}

	if s.PrivateKey == nil {
		return fmt.Errorf("TlsPrivateKey is nil")
	}

	if len(s.Guardians.Identities) == 0 {
		return fmt.Errorf("no guardians array given")
	}

	if s.Threshold > len(s.Guardians.Identities) {
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

	s.guardiansProtoIDs = make([]*tsscommv1.PartyId, s.Guardians.Len())
	s.Guardians.peerCerts = make([]*x509.Certificate, s.Guardians.Len())
	s.Guardians.partyIds = make([]*tss.PartyID, s.Guardians.Len())
	s.Guardians.pemkeyToGuardian = make(map[string]*Identity)
	s.Guardians.indexToIdendity = map[SenderIndex]*Identity{}
	// Since the guardians are sorted by key, we can use their position as their index.
	for i := range s.Guardians.Len() {
		s.guardiansProtoIDs[i] = partyIdToProto(s.Guardians.Identities[i].Pid)
		s.Guardians.peerCerts[i] = s.Guardians.Identities[i].Cert
		s.Guardians.partyIds[i] = s.Guardians.Identities[i].Pid
		s.Guardians.pemkeyToGuardian[string(s.Guardians.Identities[i].KeyPEM)] = s.Guardians.Identities[i]
		s.Guardians.indexToIdendity[SenderIndex(i)] = s.Guardians.Identities[i]
	}

	if s.LeaderIdentity == nil {
		// since the guardians are expected to be sorted already, the first guardian is the leader.
		s.LeaderIdentity = s.Guardians.Identities[0].KeyPEM
	}

	s.isleader = bytes.Equal(s.Self.KeyPEM, s.LeaderIdentity)

	return nil
}

// validates the stored Identity structs. Ensures that the cert and key are valid and match.
// ensures no nil values are stored. Verifies that the tss-lib.PartyIDs are unique.
func (s *GuardianStorage) fillAndValidateStoredIdentities() error {
	uniquePidIDs := make(map[string]struct{})
	uniquePidKey := make(map[string]struct{})

	for i, id := range s.Guardians.Identities {
		if id == nil {
			return fmt.Errorf("error guardian %v is nil", i)
		}

		c, err := internal.PemToCert(id.CertPem)
		if err != nil {
			return fmt.Errorf("error parsing guardian %v cert: %v", i, err)
		}

		key, ok := c.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("error guardian %v cert stored with non-ecdsa publickey", i)
		}

		pem, err := internal.PublicKeyToPem(key)
		if err != nil {
			return fmt.Errorf("error converting guardian %v  cert's PK  to pem: %v", i, err)
		}

		if id.Pid == nil {
			return fmt.Errorf("error guardian %v PartyID is nil", i)
		}

		if !bytes.Equal(id.Pid.Key, pem) {
			return fmt.Errorf("error guardian %v cert's PK does not match the PartyID.Key stored", i)
		}

		if id.Hostname == "" {
			return fmt.Errorf("error guardian %v hostname is empty", i)
		}

		if id.Pid.Id == "" {
			return fmt.Errorf("error guardian %v PartyID.Id is empty", i)
		}

		if _, ok := uniquePidIDs[id.Pid.Id]; ok {
			return fmt.Errorf("error guardian %v PartyID.Id is not unique", i)
		}
		uniquePidIDs[id.Pid.Id] = struct{}{}

		if _, ok := uniquePidKey[string(id.Pid.Key)]; ok {
			return fmt.Errorf("error guardian %v PartyID.Key is not unique", i)
		}
		uniquePidKey[string(id.Pid.Key)] = struct{}{}

		// storing the cert and key in the identity struct.
		id.Key = key
		id.Cert = c
		id.CommunicationIndex = SenderIndex(i)
	}

	return nil
}

func (s *GuardianStorage) getSortedFirst() (*tss.PartyID, error) {
	guardians := make([]*tss.PartyID, s.Guardians.Len())
	for i := range s.Guardians.Len() {
		pid, ok := proto.Clone(s.Guardians.partyIds[i].MessageWrapper_PartyID).(*tss.MessageWrapper_PartyID)
		if !ok {
			return nil, fmt.Errorf("error cloning guardian %v", i)
		}

		guardians[i] = &tss.PartyID{
			MessageWrapper_PartyID: pid,
			// Index:                  i,
		}
	}

	slices.SortFunc(guardians, func(a, b *tss.PartyID) int {
		return bytes.Compare(a.Key, b.Key)
	})

	for i, g := range guardians {
		g.Index = i
	}

	return guardians[0], nil
}

var errInternalNoCert = errors.New("internal error. no certificate found")

func (s *GuardianStorage) fetchCertificate(sender SenderIndex) (*x509.Certificate, error) {
	id, ok := s.Guardians.indexToIdendity[sender]
	if !ok {
		return nil, ErrUnkownSender
	}

	return id.Cert, nil
}

func (g *GuardianStorage) contains(sender SenderIndex) bool {
	_, ok := g.Guardians.indexToIdendity[sender]

	return ok
}

func (s *GuardianStorage) getPartyIdFromIndex(senderId SenderIndex) *tss.PartyID {
	id, ok := s.Guardians.indexToIdendity[senderId]
	if !ok {
		return nil
	}

	keyCpy := make([]byte, len(id.Pid.Key))
	copy(keyCpy, id.Pid.Key)

	// return a copy, tss-lib might modify this object.
	return &tss.PartyID{
		MessageWrapper_PartyID: &tss.MessageWrapper_PartyID{
			Id:      id.Pid.Id,
			Moniker: id.Pid.Moniker,
			Key:     keyCpy,
		},

		Index: id.Pid.Index,
	}
}
