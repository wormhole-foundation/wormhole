package tss

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	whcommon "github.com/certusone/wormhole/node/pkg/common"
	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/multi-party-sig/pkg/math/curve"
	"github.com/xlabs/multi-party-sig/protocols/frost"
	common "github.com/xlabs/tss-common"
)

type message interface {
	IsBroadcast() bool
	GetNetworkMessage() *tsscommv1.PropagatedMessage
}

type Sendable interface {
	message
	GetDestinations() []*Identity

	cloneSelf() Sendable // deep copy to avoid race condition in tests (ensuring no one shares the same sendable).
}

type Incoming interface {
	message
	IsUnicast() bool
	GetSource() *Identity

	toUnicast() *tsscommv1.Unicast
	toBroadcastMsg() *tsscommv1.Echo
}

// ReliableMessenger is a component of tss, where it knows how to handle incoming tsscommv1.PropagatedMessage,
// it may produce messages (of type Sendable), which should be delivered to other guardians.
// these Sendable messages are produced by the tss engine, and are needed by the other guardians to
// complete a TSS round. In addition it supplies a server with certificates of any
// party member, including itself.
type ReliableMessenger interface {
	// HandleIncomingTssMessage receives a network `message`` and process it using a reliable-broadcast protocol.
	HandleIncomingTssMessage(msg Incoming)
	ProducedOutputMessages() <-chan Sendable // just need to propagate this through the p2p network

	// Utilities for servers:
	GetCertificate() *tls.Certificate // containing secret key.
	GetPeers() []*x509.Certificate    // containing public keys.

	// FetchPartyId returns the PartyId for a given certificate, it'll use the public key
	// in the certificate and match it to the public key expected to be found in `*tsscommv1.PartyId`.
	FetchIdentity(cert *x509.Certificate) (*Identity, error)
}

// Signer is the interface to give any component with the ability to authorise a new threshold signature over a message.
type Signer interface {
	// for consistency level see https://wormhole.com/docs/build/reference/consistency-levels/
	BeginAsyncThresholdSigningProtocol(vaaDigest []byte, chainID vaa.ChainID, vaaconsistency uint8) error
	ProducedSignature() <-chan *common.SignatureData

	GetPublicKey() curve.Point
	GetEthAddress() ethcommon.Address

	// tells the maximal duration one might wait on a signature to be produced
	// (realisticly, it should be produced within a few seconds).
	MaxTTL() time.Duration

	// WitnessNewVaa is a method to witness a new VAA, andto start a signing protocol for it.
	WitnessNewVaa(v *vaa.VAA) error
}

type KeyGenerator interface {
	// StartDKG starts a distributed key generation protocol, which will produce a frost.Config.
	// Using this config, one can create a frost.Signer.
	// this function doesn't change disk stored state.
	StartDKG() (chan *frost.Config, error)
}

// ReliableTSS represents a TSS engine that can fully support logic of
// reliable broadcast needed for the security of TSS over the network.
type ReliableTSS interface {
	ReliableMessenger
	Signer
	KeyGenerator

	SetGuardianSetState(gs *whcommon.GuardianSetState) error
	Start(ctx context.Context) error
}
