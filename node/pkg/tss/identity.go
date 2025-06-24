package tss

import (
	"crypto/ecdsa"
	"crypto/x509"
	"net"
	"strconv"

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

type Identities struct {
	// sorted by KeyPem.
	Identities []*Identity

	// maps and slices to ensure quick lookups.
	pemkeyToIndex      map[string]int
	vaav1PubToIdentity map[ethcommon.Address]int
	peerCerts          []*x509.Certificate
	partyIds           []*common.PartyID
}
