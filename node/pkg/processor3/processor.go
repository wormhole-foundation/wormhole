package processor3

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var EnableLeaderSets = false

// observationMap maps the hex-encoded hash of an observation to its accumulation state
type observationMap map[string]*state

type observationProcessingJob struct {
	o     *gossipv1.SignedObservation
	state *state
}

type selfObservationEvent struct {
	m     *common.MessagePublication
	state *state
}

type ConcurrentProcessor struct {

	// EXTERNALLY SHARED CHANNELS
	// msgC is a channel of observed emitted messages
	msgC <-chan *common.MessagePublication

	// gossipSendC is a channel of outbound messages to broadcast on p2p
	gossipSendC chan<- []byte
	// obsvC is a channel of inbound decoded observations from p2p
	obsvC chan *gossipv1.SignedObservation

	// obsvReqSendC is a send-only channel of outbound re-observation requests to broadcast on p2p
	obsvReqSendC chan<- *gossipv1.ObservationRequest

	// signedInC is a channel of inbound signed VAA observations from p2p
	signedInC <-chan *gossipv1.SignedVAAWithQuorum

	// INTERNAL CHANNELS
	// we write our own observations to this channel such that they can populate the local message state
	msgSelfObservedC chan *common.MessagePublication

	// collections of channels to fan-out. Related events will be in the same channel.
	selfObservationChannels []chan selfObservationEvent
	observationChannels     []chan observationProcessingJob
	inboundVaaChannels      []chan *vaa.VAA

	// once quorum is reached on a message, this channel is notified and the state is subsequently deleted
	reachedQuorumC chan string

	// gk is the node's guardian private key
	gk *ecdsa.PrivateKey

	attestationEvents *reporter.AttestationEventReporter

	db       *db.Database
	pythVaas pythVaaDb // in-memory store of pythnet VAAs to avoid writing to disk too much

	// gk pk as eth address
	ourAddr ethcommon.Address

	governor  *governor.ChainGovernor
	acct      *accountant.Accountant
	acctReadC <-chan *common.MessagePublication

	// gst is managed by the processor and allows concurrent access to the
	// guardian set by other components.
	gst *common.GuardianSetState
}

type Processor struct {
	*ConcurrentProcessor

	numCPU int

	// Runtime state
	state observationMap
}

func NewProcessor3(
	numCPU int,
	db *db.Database,
	msgC <-chan *common.MessagePublication,
	gossipSendC chan<- []byte,
	obsvC chan *gossipv1.SignedObservation,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	signedInC <-chan *gossipv1.SignedVAAWithQuorum,
	gk *ecdsa.PrivateKey,
	gst *common.GuardianSetState,
	attestationEvents *reporter.AttestationEventReporter,
	g *governor.ChainGovernor,
	acct *accountant.Accountant,
	acctReadC <-chan *common.MessagePublication,
) *Processor {

	numPriorityBuckets = numCPU
	numProcessingBuckets = numCPU + 2

	cp := &ConcurrentProcessor{
		msgC:         msgC,
		gossipSendC:  gossipSendC,
		obsvC:        obsvC,
		obsvReqSendC: obsvReqSendC,
		signedInC:    signedInC,
		gk:           gk,
		db:           db,
		pythVaas:     NewPythVaaDb(),

		attestationEvents: attestationEvents,

		ourAddr:   crypto.PubkeyToAddress(gk.PublicKey),
		governor:  g,
		acct:      acct,
		acctReadC: acctReadC,
		gst:       gst,

		msgSelfObservedC:        make(chan *common.MessagePublication, 500),
		selfObservationChannels: make([]chan selfObservationEvent, numProcessingBuckets),
		observationChannels:     make([]chan observationProcessingJob, numProcessingBuckets),
		inboundVaaChannels:      make([]chan *vaa.VAA, numProcessingBuckets),
		reachedQuorumC:          make(chan string, 500),
	}

	return &Processor{
		ConcurrentProcessor: cp,
		state:               make(observationMap),
		numCPU:              numCPU,
	}
}

func (p *ConcurrentProcessor) processMsg(ctx context.Context, logger *zap.Logger, msg *common.MessagePublication) error {
	if p.governor != nil {
		if !p.governor.ProcessMsg(msg) {
			return nil
		}
	}
	if p.acct != nil {
		shouldPub, err := p.acct.SubmitObservation(msg)
		if err != nil {
			return err
		}
		if !shouldPub {
			return nil
		}
	}
	p.handleMessage(ctx, logger, msg)
	return nil
}

func (p *Processor) Run(ctx context.Context) error {
	for i := 0; i < p.numCPU; i++ {
		go p.RunConcurrently(ctx) //nolint
	}

	for i := 0; i < numProcessingBuckets; i++ {
		p.selfObservationChannels[i] = make(chan selfObservationEvent, 500)
		p.observationChannels[i] = make(chan observationProcessingJob, 500)
		p.inboundVaaChannels[i] = make(chan *vaa.VAA, 500)
		go p.RunObservationProcessor(ctx, p.observationChannels[i], p.inboundVaaChannels[i], p.selfObservationChannels[i]) //nolint
	}

	p.RunSequential(ctx) //nolint
	return nil
}

func (p *ConcurrentProcessor) RunObservationProcessor(ctx context.Context, obsJobC <-chan observationProcessingJob, inboundVaaC <-chan *vaa.VAA, msgSelfC chan selfObservationEvent) error {
	logger := supervisor.Logger(ctx)

	for {
		select { // this nested select implements a priority selection for the inboundVaaC and msgSelfC channels
		case <-ctx.Done():
			return nil
		case v := <-inboundVaaC:
			p.handleInboundSignedVAAWithQuorum(ctx, logger, v)
		case m := <-msgSelfC:
			p.handleSelfObservation(ctx, logger, m)
		default:
			select {
			case <-ctx.Done():
				return nil
			case v := <-inboundVaaC:
				p.handleInboundSignedVAAWithQuorum(ctx, logger, v)
			case m := <-msgSelfC:
				p.handleSelfObservation(ctx, logger, m)
			case m := <-obsJobC:
				p.handleObservation(ctx, logger, m)
			}
		}
	}
}

func (p *ConcurrentProcessor) RunConcurrently(ctx context.Context) error {
	logger := supervisor.Logger(ctx)
	for {
		select {
		case msg := <-p.msgC:
			if err := p.processMsg(ctx, logger, msg); err != nil {
				return fmt.Errorf("failed to process message `%s`: %w", msg.MessageIDString(), err)
			}
		case k := <-p.acctReadC:
			if p.acct == nil {
				return fmt.Errorf("received an accountant event when accountant is not configured")
			}
			// SECURITY defense-in-depth: Make sure the accountant did not generate an unexpected message.
			if !p.acct.IsMessageCoveredByAccountant(k) {
				return fmt.Errorf("accountant published a message that is not covered by it: `%s`", k.MessageIDString())
			}
			p.handleMessage(ctx, logger, k)
		case m := <-p.signedInC:
			p.dispatchInboundSignedVAAWithQuorum(ctx, logger, m)
		}
	}
}

func (p *Processor) RunSequential(ctx context.Context) error {
	cleanup := time.NewTicker(30 * time.Second)

	// Always initialize the timer so don't have a nil pointer in the case below. It won't get rearmed after that.
	govTimer := time.NewTimer(time.Minute)

	logger := supervisor.Logger(ctx)

	for {
		select {
		case <-ctx.Done():
			if p.acct != nil {
				p.acct.Close()
			}
			return ctx.Err()
		case k := <-p.obsvC:
			p.dispatchObservation(ctx, logger, k)
		case <-cleanup.C:
			p.handleCleanup(ctx, logger)
		case hash := <-p.reachedQuorumC:
			delete(p.state, hash)
		case m := <-p.msgSelfObservedC:
			p.dispatchSelfObservation(ctx, logger, m)
		case <-govTimer.C:
			if p.governor != nil {
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
						p.handleMessage(ctx, logger, k)
					}
				}
			}
			if (p.governor != nil) || (p.acct != nil) {
				govTimer = time.NewTimer(time.Minute)
			}
		}
	}
}

func (p *ConcurrentProcessor) storeSignedVAA(v *vaa.VAA) error {
	if v.EmitterChain == vaa.ChainIDPythNet {
		key := *db.VaaIDFromVAA(v)
		p.pythVaas.put(key, v)
		return nil
	}
	return p.db.StoreSignedVAA(v)
}

// haveSignedVAA returns true if we already have a VAA for the given VAAID
func (p *ConcurrentProcessor) haveSignedVAA(id db.VAAID) bool {
	if id.EmitterChain == vaa.ChainIDPythNet {
		_, exists := p.pythVaas.get(id)
		return exists
	}

	return p.db.HasVAA(id)
}

func (p *ConcurrentProcessor) getSignedVAA(id db.VAAID) (*vaa.VAA, error) {

	if id.EmitterChain == vaa.ChainIDPythNet {
		ret, exists := p.pythVaas.get(id)
		if exists {
			return ret.v, nil
		}

		return nil, db.ErrVAANotFound
	}

	if p.db == nil {
		return nil, db.ErrVAANotFound
	}

	vb, err := p.db.GetSignedVAABytes(id)
	if err != nil {
		return nil, err
	}

	vaa, err := vaa.Unmarshal(vb)
	if err != nil {
		panic("failed to unmarshal VAA from db")
	}

	return vaa, err
}
