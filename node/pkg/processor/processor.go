package processor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/gwrelayer"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	// dto "github.com/prometheus/client_model/go"
)

var GovInterval = time.Minute
var CleanupInterval = time.Second * 30

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
	defer s.signaturesLock.Unlock()

	created := false
	if _, ok := s.signatures[hash]; !ok {
		created = true
		s.signatures[hash] = &state{
			firstObserved: time.Now(),
			signatures:    make(map[ethcommon.Address][]byte),
		}
	}

	return s.signatures[hash], created
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
	obsvC chan *common.MsgWithTimeStamp[gossipv1.SignedObservation]

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
	gatewayRelayer *gwrelayer.GatewayRelayer
}

var (
	observationChanDelay = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_signed_observation_channel_delay_us",
			Help:    "Latency histogram for delay of signed observations in channel",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 250, 500, 750, 1000.0, 5000.0, 10000.0},
		})

	observationTotalDelay = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_signed_observation_total_delay_us",
			Help:    "Latency histogram for total time to process signed observations",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 250, 500, 750, 1000.0, 5000.0, 10000.0},
		})
)

func NewProcessor(
	ctx context.Context,
	db *db.Database,
	msgC <-chan *common.MessagePublication,
	setC <-chan *common.GuardianSet,
	gossipSendC chan<- []byte,
	obsvC chan *common.MsgWithTimeStamp[gossipv1.SignedObservation],
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	injectC <-chan *vaa.VAA,
	signedInC <-chan *gossipv1.SignedVAAWithQuorum,
	gk *ecdsa.PrivateKey,
	gst *common.GuardianSetState,
	attestationEvents *reporter.AttestationEventReporter,
	g *governor.ChainGovernor,
	acct *accountant.Accountant,
	acctReadC <-chan *common.MessagePublication,
	gatewayRelayer *gwrelayer.GatewayRelayer,
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

		logger:         supervisor.Logger(ctx).With(zap.String("component", "processor")),
		state:          &aggregationState{signatures: observationMap{}},
		ourAddr:        crypto.PubkeyToAddress(gk.PublicKey),
		governor:       g,
		acct:           acct,
		acctReadC:      acctReadC,
		pythnetVaas:    make(map[string]PythNetVaaEntry),
		workerFactor:   workerFactor,
		gatewayRelayer: gatewayRelayer,
	}
}

func (p *Processor) Run(ctx context.Context) error {
	errC := make(chan error) // errC receives errors from workers, which are then handled in this function

	if p.workerFactor < 0.0 {
		return fmt.Errorf("workerFactor must be positive or zero")
	}

	var numWorkers int
	if p.workerFactor == 0.0 {
		numWorkers = 1
		p.logger.Info("processor running in single worker mode", zap.Int("numWorkers", numWorkers), zap.Float64("workerFactor", p.workerFactor))
	} else {
		numWorkers = int(math.Ceil(float64(runtime.NumCPU()) * p.workerFactor))
		p.logger.Info("processor configured to use workers", zap.Int("numWorkers", numWorkers), zap.Float64("workerFactor", p.workerFactor))
	}

	// Start the routine to do housekeeping tasks that don't need to be distributed to the workers.
	go p.runHousekeeper(ctx, errC)

	// Start the workers.
	for workerId := 1; workerId <= numWorkers; workerId++ {
		go p.runWorker(ctx, errC)
	}

	for {
		select {
		case <-ctx.Done():

			// Leaving this here for easy debugging.
			// // Log these as warnings so they show up in the benchmark logs.
			// metric := &dto.Metric{}
			// _ = observationChanDelay.Write(metric)
			// p.logger.Warn("PROCESSOR_METRICS", zap.Any("observationChannelDelay", metric.String()))

			// metric = &dto.Metric{}
			// _ = observationTotalDelay.Write(metric)
			// p.logger.Warn("PROCESSOR_METRICS", zap.Any("observationProcessingDelay", metric.String()))

			return nil
		case e := <-errC:
			p.logger.Error("error in processor", zap.Error(e))
		}
	}
}

// runHousekeeper performs general tasks that do not need to be distributed to the workers. There will always be exactly one instance of this.
func (p *Processor) runHousekeeper(ctx context.Context, errC chan<- error) {
	// Always start the timers to avoid nil pointer dereferences below. They will only be rearmed on worker 1.
	cleanup := time.NewTimer(CleanupInterval)
	defer cleanup.Stop()

	// Always initialize the timer so don't have a nil pointer in the case below. It won't get rearmed after that.
	govTimer := time.NewTimer(GovInterval)
	defer govTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case gs := <-p.setC:
			p.logger.Info("guardian set updated",
				zap.Strings("set", gs.KeysAsHexStrings()),
				zap.Uint32("index", gs.Index))
			p.gst.Set(gs)
		case <-cleanup.C:
			cleanup.Reset(CleanupInterval)
			p.handleCleanup(ctx)
		case <-govTimer.C:
			if p.governor != nil {
				toBePublished, err := p.governor.CheckPending()
				if err != nil {
					common.SendOnChannel(ctx, errC, err)
					continue
				}
				if len(toBePublished) != 0 {
					for _, k := range toBePublished {
						// SECURITY defense-in-depth: Make sure the governor did not generate an unexpected message.
						if msgIsGoverned, err := p.governor.IsGovernedMsg(k); err != nil {
							common.SendOnChannel(ctx, errC, fmt.Errorf("governor failed to determine if message should be governed: `%s`: %w", k.MessageIDString(), err))
							continue
						} else if !msgIsGoverned {
							common.SendOnChannel(ctx, errC, fmt.Errorf("governor published a message that should not be governed: `%s`", k.MessageIDString()))
							continue
						}
						if p.acct != nil {
							shouldPub, err := p.acct.SubmitObservation(k)
							if err != nil {
								common.SendOnChannel(ctx, errC, fmt.Errorf("failed to process message released by governor `%s`: %w", k.MessageIDString(), err))
								continue
							}
							if !shouldPub {
								continue
							}
						}
						p.handleMessage(ctx, k)
					}
				}
				govTimer.Reset(GovInterval)
			}
		}
	}
}

// runWorker performs the per-observation tasks that can be distributed to the workers. There will be at least one of these.
func (p *Processor) runWorker(ctx context.Context, errC chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case k := <-p.msgC:
			if p.governor != nil {
				if !p.governor.ProcessMsg(k) {
					continue
				}
			}
			if p.acct != nil {
				shouldPub, err := p.acct.SubmitObservation(k)
				if err != nil {
					common.SendOnChannel(ctx, errC, fmt.Errorf("failed to process message `%s`: %w", k.MessageIDString(), err))
					continue
				}
				if !shouldPub {
					continue
				}
			}
			p.handleMessage(ctx, k)
		case k := <-p.acctReadC:
			if p.acct == nil {
				common.SendOnChannel(ctx, errC, fmt.Errorf("received an accountant event when accountant is not configured"))
				continue
			}
			// SECURITY defense-in-depth: Make sure the accountant did not generate an unexpected message.
			if !p.acct.IsMessageCoveredByAccountant(k) {
				common.SendOnChannel(ctx, errC, fmt.Errorf("accountant published a message that is not covered by it: `%s`", k.MessageIDString()))
				continue
			}
			p.handleMessage(ctx, k)
		case v := <-p.injectC:
			p.handleInjection(ctx, v)
		case m := <-p.obsvC:
			observationChanDelay.Observe(float64(time.Since(m.Timestamp).Microseconds()))
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

func (p *Processor) getSignedVAA(id db.VAAID) (*vaa.VAA, error) {
	if id.EmitterChain == vaa.ChainIDPythNet {
		p.pythnetVaaLock.Lock()
		defer p.pythnetVaaLock.Unlock()
		key := fmt.Sprintf("%v/%v", id.EmitterAddress, id.Sequence)
		ret, exists := p.pythnetVaas[key]
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
