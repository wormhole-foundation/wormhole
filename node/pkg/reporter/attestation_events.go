package reporter

import (
	"math/rand"
	"sync"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/vaa"

	"github.com/ethereum/go-ethereum/common"
)

const maxClientId = 1e6

type (
	// MessagePublication is a VAA along with a transaction identifer from the EmiterChain
	MessagePublication struct {
		VAA vaa.VAA
		// The native transaction identifier from the EmitterAddress interaction.
		InitiatingTxID common.Hash
	}
)

type lifecycleEventChannels struct {
	// channel for each event
	MessagePublicationC chan *MessagePublication
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
	clientId := 0
	found := true
	for found {
		clientId = rand.Intn(maxClientId) //#nosec G404 The clientIds don't need to be unpredictable. They just need to be unique.
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
			re.logger.Error("buffer overrun when attempting to publish message", zap.Int("client", client))
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
			re.logger.Error("buffer overrun when attempting to publish message", zap.Int("client", client))

		}
	}
}
