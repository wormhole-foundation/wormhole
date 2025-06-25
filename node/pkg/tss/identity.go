package tss

import (
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"net"
	"strconv"

	"github.com/certusone/wormhole/node/pkg/tss/internal"
	ethcommon "github.com/ethereum/go-ethereum/common"
	common "github.com/xlabs/tss-common"
	"google.golang.org/protobuf/proto"
)

type Identity struct {
	Pid     *common.PartyID   `json:"Pid,inline"` // used for tss protocol.
	KeyPEM  PEM               `json:"KeyPEM"`     // the public key in PEM format.
	Key     *ecdsa.PublicKey  `json:"-"`          // ensuring this isn't stored in non-pem format.
	CertPem PEM               `json:"CertPem"`    // the certificate in PEM format.
	Cert    *x509.Certificate `json:"-"`          // ensuring this isn't stored in non-pem format.

	// the number representing the guardian when passing messages.
	CommunicationIndex SenderIndex `json:"CommunicationIndex"`
	// the hostname of the guardian, used to connect to it.
	Hostname string `json:"Hostname"`
	// the port the guardian is listening on. if 0 -> use the default port.
	Port int `json:"Port,omitempty"`
	// the combination of hostname and port. Used to establish a network connection.
	networkname string `json:"-"`

	// TODO: is this field mutable? in the future, when this field is set via guardian communications,
	// would it be set ONCE, or multiple times? (if once, we can use atomics to indicate whether it is set or not).
	// otherwise, we'll need a lock.
	VAAv1PubKey *ethcommon.Address `json:"VAAv1PubKey,omitempty"` // mapping between VaaV1 and PID (used in TSS)
}

func (id *Identity) Copy() *Identity {
	keypem := make([]byte, len(id.KeyPEM))
	copy(keypem, id.KeyPEM)

	certPem := make([]byte, len(id.CertPem))
	copy(certPem, id.CertPem)

	c, k, _ := extractCertAndKeyFromPem(certPem)
	cpy := &Identity{
		Pid:                id.getPidCopy(),
		KeyPEM:             keypem,
		CertPem:            certPem,
		CommunicationIndex: id.CommunicationIndex,
		Hostname:           id.Hostname,
		Port:               id.Port,
		Key:                k,
		Cert:               c,
		networkname:        id.networkname,
	}

	return cpy
}

func (id *Identity) NetworkName() string {
	if id.networkname != "" {
		return id.networkname
	}

	return id.portAndHostToNetName()
}

func (id *Identity) portAndHostToNetName() string {
	var port string
	if id.Port <= 0 || id.Port > (1<<16) {
		port = DefaultPort
	} else {
		port = strconv.Itoa(id.Port)
	}

	return net.JoinHostPort(id.Hostname, port)
}

func (id *Identity) getPidCopy() *common.PartyID {
	// return a copy, tss-lib might modify this object.
	return proto.CloneOf(id.Pid)
}

type IdentitiesKeep struct {
	// sorted by KeyPem.
	Identities []*Identity

	// maps and slices to ensure quick lookups.
	pemkeyToIndex      map[string]int
	vaav1PubToIdentity map[ethcommon.Address]int
	peerCerts          []*x509.Certificate
	partyIds           []*common.PartyID
}

// TODO: consider moving the following functions to the identity/identities responsibility.
func (ids *IdentitiesKeep) fetchIdentityFromPartyID(senderPid *common.PartyID) (*Identity, error) {
	return ids.fetchIdentityFromKeyPEM(PEM(senderPid.GetID()))
}

var errUnknownPubkey = fmt.Errorf("unknown public key")

func (ids *IdentitiesKeep) fetchIdentityFromKeyPEM(pk PEM) (*Identity, error) {
	pos, ok := ids.pemkeyToIndex[string(pk)]
	if !ok {
		return nil, errUnknownPubkey
	}

	return ids.fetchIdentityFromIndex(SenderIndex(pos))
}

// FetchIdentity implements ReliableTSS.
func (ids *IdentitiesKeep) FetchIdentity(cert *x509.Certificate) (*Identity, error) {
	var id *Identity

	switch key := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		publicKeyPem, err := internal.PublicKeyToPem(key)
		if err != nil {
			return nil, err
		}

		id, err = ids.fetchIdentityFromKeyPEM(publicKeyPem)
		if err != nil {
			return nil, fmt.Errorf("error fetching identity from public key: %w", err)
		}
	case []byte:
		var err error
		id, err = ids.fetchIdentityFromKeyPEM(key)
		if err != nil {
			return nil, fmt.Errorf("error fetching identity from public key bytes: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported public key type")
	}

	return id, nil
}

func (ids *IdentitiesKeep) contains(senderId SenderIndex) bool {
	if senderId < 0 || int(senderId) >= len(ids.Identities) {
		return false
	}

	return true
}

func (ids *IdentitiesKeep) fetchIdentityFromIndex(senderId SenderIndex) (*Identity, error) {
	if !ids.contains(senderId) {
		return nil, ErrUnkownSender
	}

	return ids.Identities[senderId], nil
}

func (ids *IdentitiesKeep) fetchIdentityFromVaav1Pubkey(pubkey ethcommon.Address) (*Identity, error) {
	index, ok := ids.vaav1PubToIdentity[pubkey]
	if !ok {
		return nil, fmt.Errorf("unknown vaav1 pubkey %s", pubkey.Hex())
	}

	return ids.fetchIdentityFromIndex(SenderIndex(index))

}

func (ids *IdentitiesKeep) GetPeers() []*x509.Certificate {
	return ids.peerCerts
}

func (ids *IdentitiesKeep) GetPartyIDs() []*common.PartyID {
	return ids.partyIds
}
