package reactor

import (
	"context"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"

	"github.com/benbjohnson/clock"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Manager handles the creation and maintenance of reactors for incoming local and foreign observations
type Manager[K Observation] struct {
	// group of the manager. This is a label for this manager and all associated reactors.
	group string
	// observationC is a channel of observed emitted messages
	observationC <-chan K

	// confirmationC is a channel of inbound decoded observations from p2p
	confirmationC <-chan *gossipv1.SignedObservation

	// reactors are the current live reactors. This list may contain prepared reactors in StateInitialized. This field
	// may only be touched by removeReactor and loadOrCreateReactor.
	reactors              map[ethcommon.Hash]*ConsensusReactor[K]
	reactorsReverseLookup map[*ConsensusReactor[K]]ethcommon.Hash
	reactorsLock          sync.Mutex

	// stateTransitionChan is a channel of state transitions. This channel is used to communicate state transitions from
	// the reactors to the manager.
	stateTransitionChan chan *StateTransition[K]

	// gst is managed by the processor and allows concurrent access to the
	// guardian set by other components.
	gst *common.GuardianSetState

	// config template for the reactors
	config Config
	// handler for manager events
	handler ManagerEventHandler[K]

	// storage for storing state and signed observations
	storage ConsensusStorage[K]

	logger *zap.Logger
	clock  clock.Clock
}

// ManagerEventHandler handles significant consensus event from reactors
type ManagerEventHandler[K Observation] interface {
	HandleQuorum(observation K, signatures []*vaa.Signature)
	HandleFinalization(observation K, signatures []*vaa.Signature)
	HandleTimeout(previousState State, digest ethcommon.Hash, observation K, signatures []*vaa.Signature)
}

// NewManager creates a new reactor manager
func NewManager[K Observation](group string, observationC <-chan K, confirmationC <-chan *gossipv1.SignedObservation, gst *common.GuardianSetState, config Config, handler ManagerEventHandler[K], storage ConsensusStorage[K]) *Manager[K] {
	m := &Manager[K]{
		group:                 group,
		observationC:          observationC,
		confirmationC:         confirmationC,
		reactors:              map[ethcommon.Hash]*ConsensusReactor[K]{},
		reactorsReverseLookup: map[*ConsensusReactor[K]]ethcommon.Hash{},
		gst:                   gst,
		config:                config,
		handler:               handler,
		storage:               storage,
		stateTransitionChan:   make(chan *StateTransition[K], 1000),
		clock:                 clock.New(),
	}

	return m
}

func (p *Manager[K]) Run(ctx context.Context) error {
	// Share a logger between runners
	p.logger = supervisor.Logger(ctx)
	p.logger = p.logger.With(zap.String("group", p.group))

	wg := sync.WaitGroup{}
	for i := 0; i < runtime.NumCPU()/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case s := <-p.stateTransitionChan:
					p.reactorsLock.Lock()
					digest, exists := p.reactorsReverseLookup[s.Reactor]
					p.reactorsLock.Unlock()
					if !exists {
						p.logger.Warn("received state transition for unknown reactor - dropping", zap.Any("transition", s))
						continue
					}
					p.transitionHook(digest, s)
				case k := <-p.observationC:
					messagesObservedTotal.WithLabelValues(p.group).Inc()

					digest := k.SigningDigest()
					p.logger.Debug("received observation", zap.Stringer("digest", digest), zap.Any("observation", k))

					r, err := p.loadOrCreateReactor(ctx, digest, p.gst.Get(), p.storageObservationFilter(k.MessageID()))
					if err != nil {
						p.logger.Error("failed to load or create reactor", zap.Error(err), zap.Stringer("digest", digest))
						continue
					}
					r.ObservationChannel() <- k
				case m := <-p.confirmationC:
					digest := ethcommon.BytesToHash(m.Hash)

					gs := p.gst.Get()

					r, err := p.loadOrCreateReactor(ctx, digest, gs, chainNewReactorFilters(
						// Signed observations have to be verified before creating reactors to prevent DoS.
						// They will also be verified in the reactor; This duplication is intended as the overhead of verifying
						// signatures twice is worth the reduced complexity and security risk from not having a code
						// path in the reactor that skips verification. It also only evaluates if a new reactor would be created.
						p.signedObservationSignatureFilter(gs, m),
						p.storageObservationFilter(m.MessageId),
					))
					if err != nil {
						p.logger.Error("failed to load or create reactor", zap.Error(err), zap.Stringer("digest", digest))
						continue
					}
					r.ForeignObservationChannel() <- m
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

func (p *Manager[K]) IterateReactors(iterF func(digest ethcommon.Hash, reactor *ConsensusReactor[K])) {
	p.reactorsLock.Lock()
	defer p.reactorsLock.Unlock()

	for digest, reactor := range p.reactors {
		iterF(digest, reactor)
	}
}

func (p *Manager[K]) transitionHook(digest ethcommon.Hash, s *StateTransition[K]) {
	switch s.ToState {
	case StateQuorum:
		p.logger.Debug("reactor reached quorum", zap.Stringer("digest", digest), zap.String("message_id", s.Reactor.Observation().MessageID()))

		// Store in the database
		if p.storage != nil {
			err := p.storage.StoreSignedObservation(s.Reactor.Observation(), s.Reactor.VAASignatures())
			if err != nil {
				p.logger.Error("failed to store signed observation on quorum", zap.String("message_id", s.Reactor.Observation().MessageID()), zap.Stringer("digest", s.Reactor.Observation().SigningDigest()), zap.Error(err))
			}
		}

		// Handle consensus
		go p.handler.HandleQuorum(s.Reactor.Observation(), s.Reactor.VAASignatures())
	case StateFinalized:
		// Remove from the reactors list
		p.removeReactor(digest)

		// Store in the database. The Signed Observation will already be in storage from reaching StateQuorum. It is
		// stored again because more signatures may have been collected before reaching finalization.
		if p.storage != nil {
			err := p.storage.StoreSignedObservation(s.Reactor.Observation(), s.Reactor.VAASignatures())
			if err != nil {
				p.logger.Error("failed to store signed observation on finalization", zap.String("message_id", s.Reactor.Observation().MessageID()), zap.Stringer("digest", s.Reactor.Observation().SigningDigest()), zap.Error(err))
			}
		}

		p.logger.Debug("reactor finalized and removed from manager", zap.Stringer("digest", digest), zap.String("message_id", s.Reactor.Observation().MessageID()))
		go p.handler.HandleFinalization(s.Reactor.Observation(), s.Reactor.VAASignatures())
	case StateTimedOut:
		// Remove from the reactors list
		p.removeReactor(digest)

		p.logger.Debug("reactor timed out and removed from manager", zap.Stringer("digest", digest))
		go p.handler.HandleTimeout(s.FromState, digest, s.Reactor.Observation(), s.Reactor.VAASignatures())
	}
}

func (p *Manager[K]) loadOrCreateReactor(ctx context.Context, digest ethcommon.Hash, gs *common.GuardianSet, filter func(digest ethcommon.Hash) bool) (*ConsensusReactor[K], error) {
	p.reactorsLock.Lock()
	defer p.reactorsLock.Unlock()

	var r *ConsensusReactor[K]
	if reactor, exists := p.reactors[digest]; exists {
		p.logger.Debug("found existing reactor", zap.Stringer("digest", digest))
		r = reactor
	} else {
		if filter != nil && !filter(digest) {
			return nil, fmt.Errorf("filter prevented creation of new reactor")
		}

		p.logger.Debug("creating new reactor", zap.Stringer("digest", digest))
		r = NewReactor[K](p.group, &p.config, gs, p.stateTransitionChan)
		r.clock = p.clock
		err := supervisor.Run(ctx, fmt.Sprintf("reactor-%s", digest.String()), r.Run)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn reactor routine: %w", err)
		}
		p.reactors[digest] = r
		p.reactorsReverseLookup[r] = digest
		reactorNum.WithLabelValues(p.group).Set(float64(len(p.reactors)))
	}

	return r, nil
}

func (p *Manager[K]) removeReactor(digest ethcommon.Hash) {
	p.reactorsLock.Lock()
	defer p.reactorsLock.Unlock()
	r, exists := p.reactors[digest]
	if exists {
		delete(p.reactors, digest)
		delete(p.reactorsReverseLookup, r)
	}
	reactorNum.WithLabelValues(p.group).Set(float64(len(p.reactors)))
}

// newReactorFilter is a filter function that returns true if a reactor should be created for a given digest.
type newReactorFilter func(digest ethcommon.Hash) bool

// chainNewReactorFilters returns a newReactorFilter that chains multiple filters together.
func chainNewReactorFilters(filters ...newReactorFilter) newReactorFilter {
	return func(digest ethcommon.Hash) bool {
		for _, f := range filters {
			if !f(digest) {
				return false
			}
		}
		return true
	}
}

// signedObservationSignatureFilter returns a newReactorFilter that filters out signed observations that are not signed by
// a guardian in the guardian set.
func (p *Manager[K]) signedObservationSignatureFilter(gs *common.GuardianSet, m *gossipv1.SignedObservation) newReactorFilter {
	return func(digest ethcommon.Hash) bool {
		err := verifySignedObservation(p.group, m, gs)
		if err != nil {
			p.logger.Debug("failed to verify signed observation - dropping",
				zap.Error(err),
				zap.Stringer("digest", digest),
				zap.String("signature", hex.EncodeToString(m.Signature)),
				zap.String("addr", hex.EncodeToString(m.Addr)),
			)
			return false
		}

		return true
	}
}

// storageObservationFilter returns a newReactorFilter that checks if a message is already in storage.
func (p *Manager[K]) storageObservationFilter(messageID string) newReactorFilter {
	return func(digest ethcommon.Hash) bool {
		if p.storage != nil {
			existingObservation, _, found, err := p.storage.GetSignedObservation(messageID)
			if err != nil {
				p.logger.Warn("failed to check db for existing signed observation", zap.String("message_id", messageID), zap.Stringer("digest", digest), zap.Error(err))
			}
			if found {
				p.logger.Debug("ignoring confirmation - already in storage and no live reactor", zap.String("message_id", messageID), zap.Stringer("digest", digest), zap.Stringer("digest_stored", existingObservation.SigningDigest()))
				return false
			}
		}

		return true
	}
}
