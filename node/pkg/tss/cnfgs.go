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

	if len(s.Guardians.Identities) == 0 {
		return fmt.Errorf("no guardians array given")
	}

	if s.Threshold > len(s.Guardians.Identities) {
		return fmt.Errorf("threshold is higher than the number of guardians")
	}

	if s.TSSSecrets == nil {
		return fmt.Errorf("TSSSecrets is nil")
	}

	cnf := frost.EmptyConfig(curve.Secp256k1{})
	if err := cbor.Unmarshal(s.TSSSecrets, &cnf); err != nil { // TODO: find a way to remove cbor dependency
		return fmt.Errorf("error unmarshalling TSSSecrets: %v", err)
	}

	if len(cnf.VerificationShares.Points) != len(s.Guardians.Identities) {
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

	s.Guardians.peerCerts = make([]*x509.Certificate, s.Guardians.Len())
	s.Guardians.partyIds = make([]*common.PartyID, s.Guardians.Len())
	s.Guardians.pemkeyToIndex = make(map[string]int)
	s.Guardians.vaav1PubToIdentity = make(map[ethcommon.Address]int)
	// Since the guardians are sorted by key, we can use their position as their index.
	for i := range s.Guardians.Len() {
		s.Guardians.peerCerts[i] = s.Guardians.Identities[i].Cert
		s.Guardians.partyIds[i] = s.Guardians.Identities[i].Pid
		s.Guardians.pemkeyToIndex[string(s.Guardians.Identities[i].KeyPEM)] = i

		if s.Guardians.Identities[i].VAAv1PubKey != nil {
			s.Guardians.vaav1PubToIdentity[*(s.Guardians.Identities[i].VAAv1PubKey)] = i
		}
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

	for i, id := range s.Guardians.Identities {
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

// TODO: consider moving the following functions to the identity/identities responsibility.
func (s *GuardianStorage) fetchIdentityFromPartyID(senderPid *common.PartyID) (*Identity, error) {
	return s.fetchIdentityFromKeyPEM(PEM(senderPid.GetID()))
}

var errUnknownPubkey = fmt.Errorf("unknown public key")

func (st *GuardianStorage) fetchIdentityFromKeyPEM(pk PEM) (*Identity, error) {
	pos, ok := st.Guardians.pemkeyToIndex[string(pk)]
	if !ok {
		return nil, errUnknownPubkey
	}

	return st.fetchIdentityFromIndex(SenderIndex(pos))
}

// FetchIdentity implements ReliableTSS.
func (st *GuardianStorage) FetchIdentity(cert *x509.Certificate) (*Identity, error) {
	var id *Identity

	switch key := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		publicKeyPem, err := internal.PublicKeyToPem(key)
		if err != nil {
			return nil, err
		}

		id, err = st.fetchIdentityFromKeyPEM(publicKeyPem)
		if err != nil {
			return nil, fmt.Errorf("error fetching identity from public key: %w", err)
		}
	case []byte:
		var err error
		id, err = st.fetchIdentityFromKeyPEM(key)
		if err != nil {
			return nil, fmt.Errorf("error fetching identity from public key bytes: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported public key type")
	}

	return id, nil
}

func (s *GuardianStorage) contains(senderId SenderIndex) bool {
	if senderId < 0 || int(senderId) >= len(s.Guardians.Identities) {
		return false
	}

	return true
}

func (s *GuardianStorage) fetchIdentityFromIndex(senderId SenderIndex) (*Identity, error) {
	if !s.contains(senderId) {
		return nil, ErrUnkownSender
	}

	return s.Guardians.Identities[senderId], nil
}

func (s *GuardianStorage) fetchIdentityFromVaav1Pubkey(pubkey ethcommon.Address) (*Identity, error) {
	index, ok := s.Guardians.vaav1PubToIdentity[pubkey]
	if !ok {
		return nil, fmt.Errorf("unknown vaav1 pubkey %s", pubkey.Hex())
	}

	return s.fetchIdentityFromIndex(SenderIndex(index))

}
