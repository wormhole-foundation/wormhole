package tss

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"os"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/yossigi/tss-lib/v2/ecdsa/keygen"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/tss"
)

const (
	Participants = 5
	Threshold    = 3 // 13 players are needed if threshold is 12, since it isn't inclusive
)

type symKey []byte

// Engine is a wrapper for the tss-lib fullParty, adds reliable broadcast logic to the message sending and receiving.
type Engine struct {
	GuardianStorage

	fp party.FullParty

	signingKey ecdsa.PrivateKey
	secretKeys []symKey
	// need to receive specific files, one for the partyIds
	// one for the keygen data.
	// TODO: consider using something different than this:
	// one for its partId.
}

// BeginAsyncThresholdSigningProtocol implements ReliableTSS.
func (t *Engine) BeginAsyncThresholdSigningProtocol(msg *gossipv1.ObservationRequest) error {
	panic("unimplemented")
}

// ProducedOutputMessages implements ReliableTSS.
func (t *Engine) ProducedOutputMessages() <-chan *gossipv1.GossipMessage_TssMessage {
	panic("unimplemented")
}

type GuardianStorage struct {
	Self *tss.PartyID

	//Stored sorted by Key. include Self.
	Guardians []*tss.PartyID

	// secret key used by a single guardian.
	SecretKey []byte

	Threshold int
	// all secret keys should be generated with specific value.
	SavedSecretParameters *keygen.LocalPartySaveData

	signingKey *ecdsa.PrivateKey // should be the unmarshalled value of signing key.
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
func GuardianStorageFromFile(storagePath string) (*GuardianStorage, error) {
	var storage GuardianStorage
	if err := storage.load(storagePath); err != nil {
		return nil, err
	}

	return &storage, nil
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

	return nil
}

func NewReliableTSS(ctx context.Context, storage *GuardianStorage) (*Engine, error) {
	if storage == nil {
		return nil, fmt.Errorf("the guardian's tss storage is nil")
	}

	// TODO: do DH with every guardian to get symKeys.
	computeSharedSecrets(storage)

	// set up new party, and Start it.
	return &Engine{
		fp:         nil,
		signingKey: ecdsa.PrivateKey{},
		secretKeys: []symKey{},
	}, nil
}

func computeSharedSecrets(storage *GuardianStorage) ([]symKey, error) {
	return nil, nil
	// curve := ecdh.X25519()
	// secretKey, err := curve.NewPrivateKey(storage.SecretKey)
	// if err != nil {
	// 	return nil, err
	// }
	// // symKeys := make([]symKey, 0, len(storage.GuardianIDs))
	// for _, v := range storage.Guardians {
	// 	pk, err := curve.NewPublicKey(v.Key)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	// s,k := ecdh.X25519().GenerateKey()
	// 	// x, y := elliptic.UnmarshalCompressed(tss.S256(), v.Key)
	// 	// ecPoint, err := crypto.NewECPoint(tss.S256(), x, y)
	// 	// if err != nil {
	// 	// 	return symKeys, err
	// 	// }
	// 	// storage.SecretKey.ScalarMult(x, y, v.Key)
	// }
}

func (t *Engine) HandleIncomingTssMessage(msg *gossipv1.GossipMessage_TssMessage) {
	if t == nil {
		return
	}
	party.NewFullParty(nil)

	fmt.Println("Engine:incoming")
}

func (t *Engine) produceOutGoing() <-chan struct{} {
	if t == nil {
		return nil
	}

	fmt.Println("Engine:outgoing")
	return nil
}

func (t *Engine) runSigningProtocol(msg *gossipv1.ObservationRequest) {
	if t == nil {
		return
	}
	fmt.Println("Engine:start signing")
}

func (t *Engine) Close() {
	fmt.Println("Engine:close")
}
