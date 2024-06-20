package processor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
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
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
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
		// IsReobservation returns whether this message is the result of a reobservation request.
		IsReobservation() bool
		// HandleQuorum finishes processing the observation once a quorum of signatures have
		// been received for it.
		HandleQuorum(sigs []*vaa.Signature, hash string, p *Processor)
	}

	// state represents the local view of a given observation
	state struct {
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
		signatures observationMap
	}
)

// LoggingID can be used to identify a state object in a log message. Note that it should not
// be used to uniquely identify an observation. It is only meant for logging purposes.
func (s *state) LoggingID() string {
	if s.ourObservation != nil {
		return s.ourObservation.MessageID()
	}

	return hex.EncodeToString(s.txHash)
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

	// gk is the node's guardian private key
	gk *ecdsa.PrivateKey

	logger *zap.Logger

	db *db.Database

	// Runtime state

	// gs is the currently valid guardian set
	gs *common.GuardianSet
	// gst is managed by the processor and allows concurrent access to the
	// guardian set by other components.
	gst *common.GuardianSetState

	// state is the current runtime VAA view
	state *aggregationState
	// gk pk as eth address
	ourAddr ethcommon.Address

	governor       *governor.ChainGovernor
	acct           *accountant.Accountant
	acctReadC      <-chan *common.MessagePublication
	pythnetVaas    map[string]PythNetVaaEntry
	gatewayRelayer *gwrelayer.GatewayRelayer
	updateVAALock  sync.Mutex
	updatedVAAs    map[string]*updateVaaEntry
}

// updateVaaEntry is used to queue up a VAA to be written to the database.
type updateVaaEntry struct {
	v         *vaa.VAA
	hashToLog string
	dirty     bool
}

var (
	observationChanDelay = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_signed_observation_channel_delay_us",
			Help:    "Latency histogram for delay of signed observations in channel",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10000.0, 100_000.0, 1_000_000.0, 10_000_000.0, 100_000_000.0, 1_000_000_000.0},
		})

	observationTotalDelay = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_signed_observation_total_delay_us",
			Help:    "Latency histogram for total time to process signed observations",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10_000.0, 100_000.0, 1_000_000.0, 10_000_000.0, 100_000_000.0, 1_000_000_000.0},
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
	signedInC <-chan *gossipv1.SignedVAAWithQuorum,
	gk *ecdsa.PrivateKey,
	gst *common.GuardianSetState,
	g *governor.ChainGovernor,
	acct *accountant.Accountant,
	acctReadC <-chan *common.MessagePublication,
	gatewayRelayer *gwrelayer.GatewayRelayer,
) *Processor {

	return &Processor{
		msgC:         msgC,
		setC:         setC,
		gossipSendC:  gossipSendC,
		obsvC:        obsvC,
		obsvReqSendC: obsvReqSendC,
		signedInC:    signedInC,
		gk:           gk,
		gst:          gst,
		db:           db,

		logger:         supervisor.Logger(ctx),
		state:          &aggregationState{observationMap{}},
		ourAddr:        crypto.PubkeyToAddress(gk.PublicKey),
		governor:       g,
		acct:           acct,
		acctReadC:      acctReadC,
		pythnetVaas:    make(map[string]PythNetVaaEntry),
		gatewayRelayer: gatewayRelayer,
		updatedVAAs:    make(map[string]*updateVaaEntry),
	}
}

func (p *Processor) Run(ctx context.Context) error {
	if err := supervisor.Run(ctx, "vaaWriter", common.WrapWithScissors(p.vaaWriter, "vaaWriter")); err != nil {
		return fmt.Errorf("failed to start vaa writer: %w", err)
	}

	cleanup := time.NewTicker(CleanupInterval)

	// Always initialize the timer so don't have a nil pointer in the case below. It won't get rearmed after that.
	govTimer := time.NewTimer(GovInterval)

	for {
		select {
		case <-ctx.Done():
			if p.acct != nil {
				p.acct.Close()
			}

			// Log these as warnings so they show up in the benchmark logs.
			metric := &dto.Metric{}
			_ = observationChanDelay.Write(metric)
			p.logger.Warn("PROCESSOR_METRICS", zap.Any("observationChannelDelay", metric.String()))

			metric = &dto.Metric{}
			_ = observationTotalDelay.Write(metric)
			p.logger.Warn("PROCESSOR_METRICS", zap.Any("observationProcessingDelay", metric.String()))

			return ctx.Err()
		case p.gs = <-p.setC:
			p.logger.Info("guardian set updated",
				zap.Strings("set", p.gs.KeysAsHexStrings()),
				zap.Uint32("index", p.gs.Index),
				zap.Int("quorum", p.gs.Quorum()),
			)
			p.gst.Set(p.gs)
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
			p.handleMessage(k)

		case k := <-p.acctReadC:
			if p.acct == nil {
				return fmt.Errorf("received an accountant event when accountant is not configured")
			}
			// SECURITY defense-in-depth: Make sure the accountant did not generate an unexpected message.
			if !p.acct.IsMessageCoveredByAccountant(k) {
				return fmt.Errorf("accountant published a message that is not covered by it: `%s`", k.MessageIDString())
			}
			p.handleMessage(k)
		case m := <-p.obsvC:
			observationChanDelay.Observe(float64(time.Since(m.Timestamp).Microseconds()))
			p.handleObservation(ctx, m)
		case m := <-p.signedInC:
			p.handleInboundSignedVAAWithQuorum(ctx, m)
		case <-cleanup.C:
			p.handleCleanup(ctx)
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
						p.handleMessage(k)
					}
				}
			}
			if (p.governor != nil) || (p.acct != nil) {
				govTimer.Reset(GovInterval)
			}
		}
	}
}

// storeSignedVAA schedules a database update for a VAA.
func (p *Processor) storeSignedVAA(v *vaa.VAA) {
	p.postUpdate(v, "")
}

// postSignedVAA schedules handling of a newly signed VAA. This includes
// writing it to the database, broadcasting it and generating a log message.
func (p *Processor) postSignedVAA(v *vaa.VAA, hash string) {
	p.postUpdate(v, hash)
}

// postUpdate adds a VAA to the map of VAAs to be updated by the vaa writer routine.
func (p *Processor) postUpdate(v *vaa.VAA, hash string) {
	if v.EmitterChain == vaa.ChainIDPythNet {
		key := fmt.Sprintf("%v/%v", v.EmitterAddress, v.Sequence)
		p.pythnetVaas[key] = PythNetVaaEntry{v: v, updateTime: time.Now()}
		return
	}
	key := fmt.Sprintf("%d/%v/%v", v.EmitterChain, v.EmitterAddress, v.Sequence)
	p.updateVAALock.Lock()
	p.updatedVAAs[key] = &updateVaaEntry{v: v, hashToLog: hash, dirty: true}
	p.updateVAALock.Unlock()
}

// haveSignedVAA returns true if we already have a VAA for the given VAAID
func (p *Processor) haveSignedVAA(id db.VAAID) bool {
	if id.EmitterChain == vaa.ChainIDPythNet {
		if p.pythnetVaas == nil {
			return false
		}
		key := fmt.Sprintf("%v/%v", id.EmitterAddress, id.Sequence)
		_, exists := p.pythnetVaas[key]
		return exists
	}

	key := fmt.Sprintf("%d/%v/%v", id.EmitterChain, id.EmitterAddress, id.Sequence)
	if p.getVaaFromUpdateMap(key) != nil {
		return true
	}

	if p.db == nil {
		return false
	}

	ok, err := p.db.HasVAA(id)
	if err != nil {
		p.logger.Error("failed to look up VAA in database",
			zap.String("vaaID", string(id.Bytes())),
			zap.Error(err),
		)
		return false
	}

	return ok
}

// getVaaFromUpdateMap gets the VAA from the local map. If it's not there, it returns nil.
func (p *Processor) getVaaFromUpdateMap(key string) *vaa.VAA {
	p.updateVAALock.Lock()
	entry, exists := p.updatedVAAs[key]
	p.updateVAALock.Unlock()
	if !exists {
		return nil
	}
	return entry.v
}

// vaaWriter is the routine that writes VAAs to the database once per second. It creates a local copy of the map
// being used by the processor to reduce lock contention. It uses a dirty flag to handle the case where the VAA
// gets updated again while we are in the process of writing it to the database. This routine also handles broadcasting
// a newly signed VAA and writing the "signed VAA with quorum" info log.
func (p *Processor) vaaWriter(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			var updatedVAAs map[string]*updateVaaEntry
			p.updateVAALock.Lock()
			if len(p.updatedVAAs) != 0 {
				// There's something to write. Create a local copy of the map so we can release the lock.
				updatedVAAs = make(map[string]*updateVaaEntry)
				for key, entry := range p.updatedVAAs {
					updatedVAAs[key] = entry
					entry.dirty = false
				}
			}
			p.updateVAALock.Unlock()
			if updatedVAAs != nil {
				// If there's anything to write, do that.
				for _, entry := range updatedVAAs {
					if err := p.db.StoreSignedVAA(entry.v); err != nil {
						p.logger.Error("failed to write VAA to database", zap.Error(err))
					}
					if entry.hashToLog != "" {
						p.logger.Info("signed VAA with quorum",
							zap.String("message_id", entry.v.MessageID()),
							zap.String("digest", entry.hashToLog),
						)
						p.broadcastSignedVAA(entry.v)
					}
				}

				// Go through the map and delete anything we have written that hasn't been updated again.
				// If something has been updated again, it will get written next interval.
				p.updateVAALock.Lock()
				for key, entry := range p.updatedVAAs {
					if !entry.dirty {
						delete(p.updatedVAAs, key)
					}
				}
				p.updateVAALock.Unlock()
			}
		}
	}
}
