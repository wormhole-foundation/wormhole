package tssmock

import (
	"context"
	"crypto/rand"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/tss"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/multi-party-sig/pkg/math/curve"
	"github.com/xlabs/multi-party-sig/pkg/math/sample"
	frostsign "github.com/xlabs/multi-party-sig/protocols/frost/sign"
	tsscommon "github.com/xlabs/tss-common"
	"github.com/xlabs/tss-common/service/signer"
)

// MockSigner is a mock implementation of the Signer interface for testing purposes.
type MockSigner struct {
	secret curve.Scalar
	pub    curve.Point
}

func (m *MockSigner) Sign(digest []byte, protocol tsscommon.ProtocolType) *signer.SignResponse {
	grp := curve.Secp256k1{}

	sig, err := frostsign.SignEcSchnorr(m.secret, digest)
	if err != nil {
		panic(err)
	}

	Rbits, _ := grp.MarshalPoint(sig.R)
	Sbits, _ := grp.MarshalScalar(sig.Z)
	resp := &signer.SignResponse{
		Response: &signer.SignResponse_Signature{
			Signature: &tsscommon.SignatureData{
				// Signature: []byte{},
				// SignatureRecovery: []byte{},
				R: Rbits,
				S: Sbits,
				M: digest,
				TrackingId: &tsscommon.TrackingID{
					Protocol:      uint32(protocol.ToInt()),
					Digest:        digest,
					PartiesState:  []byte{},
					AuxiliaryData: []byte{},
				},
			},
		},
	}
	return resp

}

// GetPublicKey implements tss.Signer.

// NewMockSigner returns a new MockSigner.
func NewMockSigner() *MockSigner {
	crv := curve.Secp256k1{}
	var secretKey curve.Scalar
	var pubkey curve.Point
	for range 10 {
		secretKey = sample.Scalar(rand.Reader, crv)
		pubKey := secretKey.ActOnBase()
		if !frostsign.PublicKeyValidForContract(pubKey) {
			continue
		}

		pubkey = pubKey
		break
	}

	if pubkey == nil {
		panic("failed to generate valid frost public key for mock signer")
	}

	return &MockSigner{
		secret: secretKey,
		pub:    pubkey,
	}
}

func (m *MockSigner) AsyncSign(rq *signer.SignRequest) error { return nil }

func (m *MockSigner) Response() <-chan *signer.SignResponse { return nil }

// AddResponse adds a response to the sign response channel.
func (m *MockSigner) AddResponse(resp *signer.SignResponse) {}

func (m *MockSigner) GetPublicData(ctx context.Context) (*signer.PublicData, error) {
	pubbts, err := m.pub.Curve().MarshalPoint(m.pub)
	if err != nil {
		return nil, err
	}

	return &signer.PublicData{FrostPublicData: pubbts}, nil
}

func (m *MockSigner) Verify(ctx context.Context, rq *signer.VerifySignatureRequest) error {
	return nil
}

func (m *MockSigner) WitnessNewVaaV1(ctx context.Context, v *vaa.VAA) error {
	return nil
}

func (m *MockSigner) GetPublicKey(ctx context.Context, protocol tsscommon.ProtocolType) (curve.Point, error) {
	return m.pub, nil
}

// MockSignerConnection is a mock implementation of the SignerConnection interface for testing purposes.
type MockSignerConnection struct {
	*MockSigner
}

func NewMockSignerConnection() *MockSignerConnection {
	// TODO: add frost. modified schnorr signature.
	return &MockSignerConnection{
		MockSigner: NewMockSigner(),
	}
}

func (m *MockSignerConnection) Connect(ctx context.Context) error {
	return nil
}

func (m *MockSignerConnection) Inform(msg *gossipv1.TSSGossipMessage) error {
	return nil
}

func (m *MockSignerConnection) Outbound() <-chan *gossipv1.TSSGossipMessage {
	return nil
}

// ensure the mock implements the interface
var (
	_ tss.Signer           = (*MockSigner)(nil)
	_ tss.SignerConnection = (*MockSignerConnection)(nil)
)
