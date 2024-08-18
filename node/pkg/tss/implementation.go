package tss

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/yossigi/tss-lib/v2/ecdsa/keygen"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/tss"
)

type symKey []byte

// Engine is the implementation of reliableTSS, it is a wrapper for the tss-lib fullParty and adds reliable broadcast logic
// to the message sending and receiving.
type Engine struct {
	GuardianStorage

	fp party.FullParty
}

// GuardianStorage is a struct that holds the data needed for a guardian to participate in the TSS protocol
// including its signing key, and the shared symmetric keys with other guardians.
// should be loaded from a file.
type GuardianStorage struct {
	Self *tss.PartyID

	//Stored sorted by Key. include Self.
	Guardians []*tss.PartyID

	// SecretKey is the marshaled secret key of ReliableTSS, used to genereate SymKeys and signingKey.
	SecretKey []byte

	Threshold int

	// all secret keys should be generated with specific value.
	SavedSecretParameters *keygen.LocalPartySaveData

	signingKey *ecdsa.PrivateKey // should be the unmarshalled value of signing key.
	Symkeys    []symKey          // should be generated upon creation using DH shared key protocol if nil.
}

// BeginAsyncThresholdSigningProtocol implements ReliableTSS.
func (t *Engine) BeginAsyncThresholdSigningProtocol(msg *gossipv1.ObservationRequest) error {
	// TODO:.
	return nil
}

// ProducedOutputMessages implements ReliableTSS.
func (t *Engine) ProducedOutputMessages() <-chan *gossipv1.GossipMessage_TssMessage {
	// TODO:.
	return nil
}

// GuardianStorageFromFile loads a guardian storage from a file.
// If the storage file hadn't contained symetric keys, it'll compute them.
func GuardianStorageFromFile(storagePath string) (*GuardianStorage, error) {
	var storage GuardianStorage
	if err := storage.load(storagePath); err != nil {
		return nil, err
	}

	return &storage, nil
}

func NewReliableTSS(ctx context.Context, storage *GuardianStorage) (*Engine, error) {
	if storage == nil {
		return nil, fmt.Errorf("the guardian's tss storage is nil")
	}

	// set up new party, and Start it.
	return &Engine{
		fp:         nil,
		signingKey: ecdsa.PrivateKey{},
		secretKeys: []symKey{},
	}, nil
}

func (t *Engine) HandleIncomingTssMessage(msg *gossipv1.GossipMessage_TssMessage) {
	if t == nil {
		return
	}
	party.NewFullParty(nil)

	fmt.Println("Engine:incoming")
}

func (t *Engine) Close() {
	fmt.Println("Engine:close")
}
