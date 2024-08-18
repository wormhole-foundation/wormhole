package tss

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/yossigi/tss-lib/v2/common"
	"github.com/yossigi/tss-lib/v2/ecdsa/keygen"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/tss"
)

type symKey []byte

// Engine is the implementation of reliableTSS, it is a wrapper for the tss-lib fullParty and adds reliable broadcast logic
// to the message sending and receiving.
type Engine struct {
	ctx context.Context

	GuardianStorage

	fp party.FullParty
	//
	fpOutChan    chan tss.Message // this one must listen on it, and output to the p2p network.
	fpSigOutChan chan *common.SignatureData
	fpErrChannel chan *tss.Error

	gossipOutChan chan *gossipv1.GossipMessage_TssMessage
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

// BeginAsyncThresholdSigningProtocol used to start the TSS protocol over a specific msg.
func (t *Engine) BeginAsyncThresholdSigningProtocol(digest []byte) error {
	if t == nil {
		return fmt.Errorf("tss engine is nil")
	}

	if t.fp == nil {
		return fmt.Errorf("tss engine is not set up correctly, use NewReliableTSS to create a new engine")
	}

	if len(digest) != 32 {
		return fmt.Errorf("digest length is not 32 bytes")
	}

	d := party.Digest{}
	copy(d[:], digest)

	return t.fp.AsyncRequestNewSignature(d)
}

// ProducedOutputMessages implements ReliableTSS.
func (t *Engine) ProducedOutputMessages() <-chan *gossipv1.GossipMessage_TssMessage {
	return t.gossipOutChan
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

	fpParams := party.Parameters{
		SavedSecrets: storage.SavedSecretParameters,
		PartyIDs:     storage.Guardians,
		Self:         storage.Self,
		Threshold:    storage.Threshold,
		WorkDir:      "",
		MaxSignerTTL: time.Minute * 5,
	}

	fp, err := party.NewFullParty(&fpParams)
	if err != nil {
		return nil, err
	}

	fpOutChan := make(chan tss.Message) // this one must listen on it, and output to the p2p network.
	fpSigOutChan := make(chan *common.SignatureData)
	fpErrChannel := make(chan *tss.Error)

	if err := fp.Start(fpOutChan, fpSigOutChan, fpErrChannel); err != nil {
		return nil, err
	}

	// ignoring returned func since i want fp.Stop to be tied to the ctx forever.
	_ = context.AfterFunc(ctx, func() {
		fp.Stop()
	})

	t := &Engine{
		GuardianStorage: *storage,

		fp:           fp,
		fpOutChan:    fpOutChan,
		fpSigOutChan: fpSigOutChan,
		fpErrChannel: fpErrChannel,

		gossipOutChan: make(chan *gossipv1.GossipMessage_TssMessage),
	}

	go t.fpListener()

	return t, nil
}

// fpListener serves as a listining loop for the full party outputs.
// ensures the FP isn't being blocked on writing to fpOutChan, and wraps the result into a gossip message.
func (t *Engine) fpListener() {
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-t.fpOutChan:
			//todo: wrap the message into a gossip message and output it to the network, sign/mac it and send it.
		case <-t.fpSigOutChan:
			// todo: find out who should get the signature.
		case <-t.fpErrChannel:
			// todo: log the error?
		}
	}
}

func (t *Engine) HandleIncomingTssMessage(msg *gossipv1.GossipMessage_TssMessage) {
	if t == nil {
		return
	}

	fmt.Println("Engine:incoming")
}

func (t *Engine) Close() {
	fmt.Println("Engine:close")
}
