package processor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type (
	// Observation defines the interface for any events observed by the guardian.
	Observation interface {
		// GetEmitterChain returns the id of the chain where this event was observed.
		GetEmitterChain() vaa.ChainID
		// MessageID returns a human-readable emitter_chain/emitter_address/sequence tuple.
		MessageID() string
		// SigningDigest returns the hash of the hash signing body of the observation. This is used
		// for signature generation and verification.
		SigningDigest() ethcommon.Hash
		// IsReliable returns whether this message is considered reliable meaning it can be reobserved.
		IsReliable() bool
		// HandleQuorum finishes processing the observation once a quorum of signatures have
		// been received for it.
		HandleQuorum(sigs []*vaa.Signature, hash string, p *Processor)
	}

	// state represents the local view of a given observation
	state struct {
		// Mutex protecting this particular state entry.
		lock sync.Mutex

		// First time this digest was seen (possibly even before we observed it ourselves).
		firstObserved time.Time
		// A re-observation request shall not be sent before this time.
		nextRetry time.Time
		// Number of times we sent a re-observation request
		retryCtr uint
		// Copy of our observation.
		ourObservation Observation
		// Map of signatures seen by guardian. During guardian set updates, this may contain signatures belonging
		// to either the old or new guardian set.
		signatures map[ethcommon.Address][]byte
		// Flag set after reaching quorum and submitting the VAA.
		submitted bool
		// Flag set by the cleanup service after the settlement timeout has expired and misses were counted.
		settled bool
		// Human-readable description of the VAA's source, used for metrics.
		source string
		// Copy of the bytes we submitted (ourObservation, but signed and serialized). Used for retransmissions.
		ourMsg []byte
		// The hash of the transaction in which the observation was made.  Used for re-observation requests.
		txHash []byte
		// Copy of the guardian set valid at observation/injection time.
		gs *common.GuardianSet
	}

	observationMap map[string]*state

	// aggregationState represents the node's aggregation of guardian signatures.
	aggregationState struct {
		// signaturesLock should be held when inserting / deleting / iterating over the map, but not when working with a single entry.
		signaturesLock sync.Mutex
		signatures     observationMap
	}
)

// getOrCreateState returns the state for a given hash, creating it if it doesn't exist.  It grabs the lock.
func (s *aggregationState) getOrCreateState(hash string) (*state, bool) {
	s.signaturesLock.Lock()

	created := false
	st, ok := s.signatures[hash]
	if !ok {
		created = true
		st = &state{
			firstObserved: time.Now(),
			signatures:    make(map[ethcommon.Address][]byte),
		}
		s.signatures[hash] = st
	}
	s.signaturesLock.Unlock()

	return st, created
}

// delete removes a state entry from the map. It grabs the lock.
func (s *aggregationState) delete(hash string) {
	s.signaturesLock.Lock()
	delete(s.signatures, hash)
	s.signaturesLock.Unlock()
}

type PythNetVaaEntry struct {
	v          *vaa.VAA
	updateTime time.Time // Used for determining when to delete entries
}

type Processor struct {
	// msgC is a channel of observed emitted messages
	msgC <-chan *common.MessagePublication
	// setC is a channel of guardian set updates
	setC <-chan *common.GuardianSet
	// gossipSendC is a channel of outbound messages to broadcast on p2p
	gossipSendC chan<- []byte
	// obsvC is a channel of inbound decoded observations from p2p
	obsvC chan *gossipv1.SignedObservation

	// obsvReqSendC is a send-only channel of outbound re-observation requests to broadcast on p2p
	obsvReqSendC chan<- *gossipv1.ObservationRequest

	// signedInC is a channel of inbound signed VAA observations from p2p
	signedInC <-chan *gossipv1.SignedVAAWithQuorum

	// injectC is a channel of VAAs injected locally.
	injectC <-chan *vaa.VAA

	// gk is the node's guardian private key
	gk *ecdsa.PrivateKey

	attestationEvents *reporter.AttestationEventReporter

	logger *zap.Logger

	db *db.Database

	// Runtime state

	// gs is the currently valid guardian set
	gs atomic.Pointer[common.GuardianSet]
	// gst is managed by the processor and allows concurrent access to the
	// guardian set by other components.
	gst *common.GuardianSetState

	// state is the current runtime VAA view
	state *aggregationState
	// gk pk as eth address
	ourAddr ethcommon.Address

	governor  *governor.ChainGovernor
	acct      *accountant.Accountant
	acctReadC <-chan *common.MessagePublication

	pythnetVaaLock sync.Mutex
	pythnetVaas    map[string]PythNetVaaEntry
	workerFactor   float64
}

func NewProcessor(
	ctx context.Context,
	db *db.Database,
	msgC <-chan *common.MessagePublication,
	setC <-chan *common.GuardianSet,
	gossipSendC chan<- []byte,
	obsvC chan *gossipv1.SignedObservation,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	injectC <-chan *vaa.VAA,
	signedInC <-chan *gossipv1.SignedVAAWithQuorum,
	gk *ecdsa.PrivateKey,
	gst *common.GuardianSetState,
	attestationEvents *reporter.AttestationEventReporter,
	g *governor.ChainGovernor,
	acct *accountant.Accountant,
	acctReadC <-chan *common.MessagePublication,
	workerFactor float64,
) *Processor {
	return &Processor{
		msgC:         msgC,
		setC:         setC,
		gossipSendC:  gossipSendC,
		obsvC:        obsvC,
		obsvReqSendC: obsvReqSendC,
		signedInC:    signedInC,
		injectC:      injectC,
		gk:           gk,
		gst:          gst,
		db:           db,

		attestationEvents: attestationEvents,

		logger:       supervisor.Logger(ctx).With(zap.String("component", "processor")),
		state:        &aggregationState{signatures: observationMap{}},
		ourAddr:      crypto.PubkeyToAddress(gk.PublicKey),
		governor:     g,
		acct:         acct,
		acctReadC:    acctReadC,
		pythnetVaas:  make(map[string]PythNetVaaEntry),
		workerFactor: workerFactor,
	}
}

func (p *Processor) Run(ctx context.Context) error {
	if p.workerFactor < 0.0 {
		return fmt.Errorf("workerFactor must be positive or zero")
	}

	numWorkers := int(math.Ceil(float64(runtime.NumCPU()) * p.workerFactor))
	p.logger.Warn("processor configured to use workers", zap.Int("numWorkers", numWorkers), zap.Float64("workerFactor", p.workerFactor))

	go p.RunBasic(ctx) //nolint

	for workerId := 0; workerId < numWorkers; workerId++ {
		go p.RunConcurrent(ctx) //nolint
	}

	<-ctx.Done()

	return nil
}

func (p *Processor) RunBasic(ctx context.Context) error {
	cleanup := time.NewTimer(30 * time.Second)
	defer cleanup.Stop()

	govTimer := time.NewTimer(time.Minute)
	defer govTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			if p.acct != nil {
				p.acct.Close()
			}
			return ctx.Err()
		case v := <-p.injectC:
			p.handleInjection(ctx, v)
		case <-cleanup.C:
			cleanup = time.NewTimer(30 * time.Second)
			p.handleCleanup(ctx)
		case <-govTimer.C:
			if p.governor != nil {
				p.logger.Info("checking governor")
				toBePublished, err := p.governor.CheckPending()
				if err != nil {
					return err
				}
				if len(toBePublished) != 0 {
					for _, k := range toBePublished {
						// SECURITY defense-in-depth: Make sure the governor did not generate an unexpected message.
						if msgIsGoverned, err := p.governor.IsGovernedMsg(k); err != nil {
							return fmt.Errorf("governor failed to determine if message should be governed: `%s`: %w", k.MessageIDString(), err)
						} else if !msgIsGoverned {
							return fmt.Errorf("governor published a message that should not be governed: `%s`", k.MessageIDString())
						}
						if p.acct != nil {
							shouldPub, err := p.acct.SubmitObservation(k)
							if err != nil {
								return fmt.Errorf("failed to process message released by governor `%s`: %w", k.MessageIDString(), err)
							}
							if !shouldPub {
								continue
							}
						}
						p.handleMessage(ctx, k)
					}
				}
				govTimer = time.NewTimer(time.Minute)
			}
		}
	}
}

func (p *Processor) RunConcurrent(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			if p.acct != nil {
				p.acct.Close()
			}
			return ctx.Err()
		case gs := <-p.setC:
			p.gs.Store(gs)
			p.logger.Info("guardian set updated",
				zap.Strings("set", gs.KeysAsHexStrings()),
				zap.Uint32("index", gs.Index))
			p.gst.Set(gs)
		case k := <-p.msgC:
			if p.governor != nil {
				if !p.governor.ProcessMsg(k) {
					continue
				}
			}
			if p.acct != nil {
				shouldPub, err := p.acct.SubmitObservation(k)
				if err != nil {
					return fmt.Errorf("failed to process message `%s`: %w", k.MessageIDString(), err)
				}
				if !shouldPub {
					continue
				}
			}
			p.handleMessage(ctx, k)
		case k := <-p.acctReadC:
			if p.acct == nil {
				return fmt.Errorf("received an accountant event when accountant is not configured")
			}
			// SECURITY defense-in-depth: Make sure the accountant did not generate an unexpected message.
			if !p.acct.IsMessageCoveredByAccountant(k) {
				return fmt.Errorf("accountant published a message that is not covered by it: `%s`", k.MessageIDString())
			}
			p.handleMessage(ctx, k)
		case m := <-p.obsvC:
			p.handleObservation(ctx, m)
		case m := <-p.signedInC:
			p.handleInboundSignedVAAWithQuorum(ctx, m)
		}
	}
}

func (p *Processor) storeSignedVAA(v *vaa.VAA) error {
	if v.EmitterChain == vaa.ChainIDPythNet {
		key := fmt.Sprintf("%v/%v", v.EmitterAddress, v.Sequence)
		p.pythnetVaaLock.Lock()
		defer p.pythnetVaaLock.Unlock()
		p.pythnetVaas[key] = PythNetVaaEntry{v: v, updateTime: time.Now()}
		return nil
	}
	return p.db.StoreSignedVAA(v)
}

// haveSignedVAA returns true if we already have a VAA for the given VAAID
func (p *Processor) haveSignedVAA(id db.VAAID) bool {
	if id.EmitterChain == vaa.ChainIDPythNet {
		p.pythnetVaaLock.Lock()
		defer p.pythnetVaaLock.Unlock()
		key := fmt.Sprintf("%v/%v", id.EmitterAddress, id.Sequence)
		_, exists := p.pythnetVaas[key]
		return exists
	}

	return p.db.HasVAA(id)
}
