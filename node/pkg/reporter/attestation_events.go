package reporter

import (
	"math/rand"
	"sync"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/vaa"

	"github.com/ethereum/go-ethereum/common"
)

type (
	// MessagePublication is a VAA along with a transaction identifer from the EmiterChain
	MessagePublication struct {
		VAA vaa.VAA
		// The native transaction identifier from the EmitterAddress interaction.
		InitiatingTxID common.Hash
	}

	// VerifiedPeerSignature is a message observation from a Guardian that has been verified
	// to be authentic and authorized to contribute to VAA quorum (ie. within the Guardian set).
	VerifiedPeerSignature struct {
		// The chain the transaction took place on
		EmitterChain vaa.ChainID
		// EmitterAddress of the contract that emitted the Message
		EmitterAddress vaa.Address
		// Sequence of the VAA
		Sequence uint64
		// The address of the Guardian that observed and signed the message
		GuardianAddress common.Address
		// Transaction Identifier of the initiating event
		Signature []byte
	}
)

type lifecycleEventChannels struct {
	// channel for each event
	MessagePublicationC chan *MessagePublication
	VAAStateUpdateC     chan *vaa.VAA
	VerifiedSignatureC  chan *VerifiedPeerSignature
	VAAQuorumC          chan *vaa.VAA
}

type AttestationEventReporter struct {
	mu     sync.RWMutex
	logger *zap.Logger

	subs map[int]*lifecycleEventChannels
}
type activeSubscription struct {
	ClientId int
	Channels *lifecycleEventChannels
}

func EventListener(logger *zap.Logger) *AttestationEventReporter {
	events := &AttestationEventReporter{
		logger: logger.Named("eventlistener"),
		subs:   map[int]*lifecycleEventChannels{},
	}
	return events
}

// getUniqueClientId loops to generate & test integers for existence as key of map. returns an int that is not a key in map.
func (re *AttestationEventReporter) getUniqueClientId() int {
	clientId := rand.Intn(1e6)
	found := false
	for found {
		clientId = rand.Intn(1e6)
		_, found = re.subs[clientId]
	}
	return clientId
}

func (re *AttestationEventReporter) Subscribe() *activeSubscription {
	re.mu.Lock()
	defer re.mu.Unlock()

	clientId := re.getUniqueClientId()
	re.logger.Debug("Subscribe for client", zap.Int("clientId", clientId))
	channels := &lifecycleEventChannels{
		MessagePublicationC: make(chan *MessagePublication, 50),
		VAAStateUpdateC:     make(chan *vaa.VAA, 50),
		VerifiedSignatureC:  make(chan *VerifiedPeerSignature, 50),
		VAAQuorumC:          make(chan *vaa.VAA, 50),
	}
	re.subs[clientId] = channels
	sub := &activeSubscription{ClientId: clientId, Channels: channels}
	return sub
}

func (re *AttestationEventReporter) Unsubscribe(clientId int) {
	re.mu.Lock()
	defer re.mu.Unlock()

	re.logger.Debug("Unsubscribe for client", zap.Int("clientId", clientId))
	delete(re.subs, clientId)
}

// ReportMessagePublication is invoked when an on-chain message is observed.
func (re *AttestationEventReporter) ReportMessagePublication(msg *MessagePublication) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	for client, sub := range re.subs {
		select {
		case sub.MessagePublicationC <- msg:
			re.logger.Debug("published MessagePublication to client", zap.Int("client", client))
		default:
			re.logger.Debug("buffer overrun when attempting to publish message", zap.Int("client", client))
		}
	}
}

// ReportVerifiedPeerSignature is invoked after a SignedObservation is verified.
func (re *AttestationEventReporter) ReportVerifiedPeerSignature(msg *VerifiedPeerSignature) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	for client, sub := range re.subs {
		select {
		case sub.VerifiedSignatureC <- msg:
			re.logger.Debug("published VerifiedPeerSignature to client", zap.Int("client", client))
		default:
			re.logger.Debug("buffer overrun when attempting to publish message", zap.Int("client", client))
		}
	}
}

// ReportVAAStateUpdate is invoked each time the local VAAState is updated.
func (re *AttestationEventReporter) ReportVAAStateUpdate(msg *vaa.VAA) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	for client, sub := range re.subs {
		select {
		case sub.VAAStateUpdateC <- msg:
			re.logger.Debug("published VAAStateUpdate to client", zap.Int("client", client))
		default:
			re.logger.Debug("buffer overrun when attempting to publish message", zap.Int("client", client))
		}
	}
}

// ReportVAAQuorum is invoked when quorum is reached.
func (re *AttestationEventReporter) ReportVAAQuorum(msg *vaa.VAA) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	for client, sub := range re.subs {
		select {
		case sub.VAAQuorumC <- msg:
			re.logger.Debug("published VAAQuorum to client", zap.Int("client", client))
		default:
			re.logger.Debug("buffer overrun when attempting to publish message", zap.Int("client", client))

		}
	}
}
