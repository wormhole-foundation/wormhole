package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/altpub"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	guardianNotary "github.com/certusone/wormhole/node/pkg/notary"
	"github.com/certusone/wormhole/node/pkg/p2p"

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
)

var PollInterval = time.Minute
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
		// Our observation in case we need to resubmit it to the batch publisher.
		ourObs *gossipv1.Observation
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

	// delegateState represents the local view of a given delegate observation
	delegateState struct {
		// First time this digest was seen.
		firstObserved time.Time

		// Map of delegate observations sent by guardian.
		observations map[ethcommon.Address]*gossipv1.DelegateObservation

		// Flag set after reaching quorum and submitting the VAA.
		submitted bool
	}

	delegateObservationMap map[string]*delegateState

	// delegateAggregationState represents the node's aggregation of delegated guardian signatures.
	delegateAggregationState struct {
		observations delegateObservationMap
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

	// dgConfigC is a channel of delegated guardian config updates
	dgConfigC <-chan *DelegatedGuardianConfig

	// gossipAttestationSendC is a channel of outbound observation messages to broadcast on p2p
	gossipAttestationSendC chan<- []byte

	// gossipVaaSendC is a channel of outbound VAA messages to broadcast on p2p
	gossipVaaSendC chan<- []byte

	// batchObsvC is a channel of inbound decoded batches of observations from p2p
	batchObsvC <-chan *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch]

	// delegateObsvC is a channel of inbound delegate observations from p2p
	delegateObsvC <-chan *gossipv1.DelegateObservation

	// obsvReqSendC is a send-only channel of outbound re-observation requests to broadcast on p2p
	obsvReqSendC chan<- *gossipv1.ObservationRequest

	// delegateObsvSendC is a channel of outbound delegate observations to broadcast on p2p
	delegateObsvSendC chan<- *gossipv1.DelegateObservation

	// signedInC is a channel of inbound signed VAA observations from p2p
	signedInC <-chan *gossipv1.SignedVAAWithQuorum

	// guardianSigner is the guardian node's signer
	guardianSigner guardiansigner.GuardianSigner

	logger *zap.Logger

	db *guardianDB.Database

	alternatePublisher *altpub.AlternatePublisher

	// Runtime state

	// gs is the currently valid guardian set
	gs *common.GuardianSet
	// gst is managed by the processor and allows concurrent access to the
	// guardian set by other components.
	gst *common.GuardianSetState

	// dgc is the per-chain delegated guardian config
	dgc *DelegatedGuardianConfig

	// state is the current runtime VAA view
	state *aggregationState
	// delegateState is the current delegate observation view
	delegateState *delegateAggregationState
	// gk pk as eth address
	ourAddr ethcommon.Address

	governor       *governor.ChainGovernor
	acct           *accountant.Accountant
	acctReadC      <-chan *common.MessagePublication
	notary         *guardianNotary.Notary
	pythnetVaas    map[string]PythNetVaaEntry
	gatewayRelayer *gwrelayer.GatewayRelayer
	updateVAALock  sync.Mutex
	updatedVAAs    map[string]*updateVaaEntry
	networkID      string

	// batchObsvPubC is the internal channel used to publish observations to the batch processor for publishing.
	batchObsvPubC chan *gossipv1.Observation
}

// updateVaaEntry is used to queue up a VAA to be written to the database.
type updateVaaEntry struct {
	v     *vaa.VAA
	dirty bool
}

var (
	batchObservationChanDelay = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_batch_observation_channel_delay_us",
			Help:    "Latency histogram for delay of batched observations in channel",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10000.0},
		})

	batchObservationTotalDelay = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_batch_observation_total_delay_us",
			Help:    "Latency histogram for total time to process batched observations",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10000.0},
		})

	batchObservationChannelOverflow = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_batch_observation_channel_overflow",
			Help: "Total number of times a write to the batch observation publish channel failed",
		}, []string{"channel"})

	vaaPublishChannelOverflow = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_vaa_publish_channel_overflow",
			Help: "Total number of times a write to the vaa publish channel failed",
		})

	timeToHandleObservation = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_time_to_handle_observation_us",
			Help:    "Latency histogram for total time to handle observation on an observation",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10_000.0, 100_000.0, 1_000_000.0, 10_000_000.0, 100_000_000.0, 1_000_000_000.0},
		})

	timeToHandleQuorum = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_time_to_handle_quorum_us",
			Help:    "Latency histogram for total time to handle quorum on an observation",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10_000.0, 100_000.0, 1_000_000.0, 10_000_000.0, 100_000_000.0, 1_000_000_000.0},
		})

	// Transfer Verifier metrics
	msgVerificationStates = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_unusual_msg_verification_states_total",
			Help: "Total number of message verification state changes to unusual values",
		}, []string{"verification_state", "emitter_chain"})

	channelNilReceive = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_processor_channel_nil_receive_total",
			Help: "Total number of nil receives on processor channels.",
		},
		[]string{"channel"},
	)
)

// batchObsvPubChanSize specifies the size of the channel used to publish observation batches. Allow five seconds worth.
const batchObsvPubChanSize = p2p.MaxObservationBatchSize * 5

func NewProcessor(
	ctx context.Context,
	db *guardianDB.Database,
	msgC <-chan *common.MessagePublication,
	setC <-chan *common.GuardianSet,
	dgConfigC <-chan *DelegatedGuardianConfig,
	gossipAttestationSendC chan<- []byte,
	gossipVaaSendC chan<- []byte,
	batchObsvC <-chan *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch],
	delegateObsvC <-chan *gossipv1.DelegateObservation,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	delegateObsvSendC chan<- *gossipv1.DelegateObservation,
	signedInC <-chan *gossipv1.SignedVAAWithQuorum,
	guardianSigner guardiansigner.GuardianSigner,
	gst *common.GuardianSetState,
	dgc *DelegatedGuardianConfig,
	g *governor.ChainGovernor,
	acct *accountant.Accountant,
	acctReadC <-chan *common.MessagePublication,
	notary *guardianNotary.Notary,
	gatewayRelayer *gwrelayer.GatewayRelayer,
	networkID string,
	alternatePublisher *altpub.AlternatePublisher,
) *Processor {

	return &Processor{
		msgC:                   msgC,
		setC:                   setC,
		dgConfigC:              dgConfigC,
		gossipAttestationSendC: gossipAttestationSendC,
		gossipVaaSendC:         gossipVaaSendC,
		batchObsvC:             batchObsvC,
		delegateObsvC:          delegateObsvC,
		obsvReqSendC:           obsvReqSendC,
		delegateObsvSendC:      delegateObsvSendC,
		signedInC:              signedInC,
		guardianSigner:         guardianSigner,
		gst:                    gst,
		db:                     db,
		alternatePublisher:     alternatePublisher,

		logger:         supervisor.Logger(ctx),
		state:          &aggregationState{observationMap{}},
		delegateState:  &delegateAggregationState{delegateObservationMap{}},
		ourAddr:        crypto.PubkeyToAddress(guardianSigner.PublicKey(ctx)),
		governor:       g,
		acct:           acct,
		acctReadC:      acctReadC,
		notary:         notary,
		pythnetVaas:    make(map[string]PythNetVaaEntry),
		gatewayRelayer: gatewayRelayer,
		batchObsvPubC:  make(chan *gossipv1.Observation, batchObsvPubChanSize),
		updatedVAAs:    make(map[string]*updateVaaEntry),
		networkID:      networkID,
		dgc:            dgc,
	}
}

func (p *Processor) Run(ctx context.Context) error {
	if err := supervisor.Run(ctx, "vaaWriter", common.WrapWithScissors(p.vaaWriter, "vaaWriter")); err != nil {
		return fmt.Errorf("failed to start vaa writer: %w", err)
	}

	if err := supervisor.Run(ctx, "batchProcessor", common.WrapWithScissors(p.batchProcessor, "batchProcessor")); err != nil {
		return fmt.Errorf("failed to start batch processor: %w", err)
	}

	cleanup := time.NewTicker(CleanupInterval)

	// Always initialize the timer so don't have a nil pointer in the case below. It won't get rearmed after that.
	pollTimer := time.NewTimer(PollInterval)

	for {
		select {
		case <-ctx.Done():
			if p.acct != nil {
				p.acct.Close()
			}
			return ctx.Err()
		case p.gs = <-p.setC:
			if p.gs == nil {
				p.logger.Error("received nil GuardianSet from setC channel")
				channelNilReceive.WithLabelValues("setC").Inc()
				continue
			}

			oldSize := 0
			oldGs := p.gst.Get()
			if oldGs != nil {
				oldSize = len(oldGs.Keys)
			}
			newSize := len(p.gs.Keys)

			// Log guardian set changes
			switch {
			case oldSize == 0 && newSize > 0:
				p.logger.Warn("guardian set populated",
					zap.Strings("set", p.gs.KeysAsHexStrings()),
					zap.Uint32("index", p.gs.Index),
					zap.Int("quorum", p.gs.Quorum()),
				)
			case oldSize > 0 && newSize == 0:
				p.logger.Error("guardian set emptied",
					zap.Int("old_size", oldSize),
					zap.Uint32("index", p.gs.Index),
				)
			case oldSize != newSize:
				p.logger.Warn("guardian set size changed",
					zap.Int("old_size", oldSize),
					zap.Int("new_size", newSize),
					zap.Strings("set", p.gs.KeysAsHexStrings()),
					zap.Uint32("index", p.gs.Index),
					zap.Int("quorum", p.gs.Quorum()),
				)
			default:
				p.logger.Info("guardian set updated",
					zap.Strings("set", p.gs.KeysAsHexStrings()),
					zap.Uint32("index", p.gs.Index),
					zap.Int("quorum", p.gs.Quorum()),
				)
			}

			p.gst.Set(p.gs)
		case dgConfig := <-p.dgConfigC:
			if dgConfig == nil {
				p.logger.Error("received nil DelegatedGuardianConfig from dgConfigC channel")
				channelNilReceive.WithLabelValues("dgConfigC").Inc()
				continue
			}

			var oldChains map[vaa.ChainID]*DelegatedGuardianChainConfig
			oldDgc := p.dgc
			if oldDgc != nil {
				oldChains = oldDgc.GetAll()
			}
			chains := dgConfig.Chains

			// Log details for removed chain configs
			for chain := range oldChains {
				if _, ok := chains[chain]; !ok {
					p.logger.Warn("delegated guardian config chain removed",
						zap.Stringer("chainID", chain),
					)
				}
			}

			// Log details for new/updated chain configs
			for chain, cfg := range chains {
				oldSize, oldQuorum := 0, 0
				oldCfg := oldChains[chain]
				if oldCfg != nil {
					oldSize = len(oldCfg.Keys)
					oldQuorum = oldCfg.Quorum()
				}
				newSize := len(cfg.Keys)
				newQuorum := cfg.Quorum()

				switch {
				case oldCfg == nil:
					p.logger.Warn("delegated guardian config chain added",
						zap.Stringer("chainID", chain),
						zap.Strings("set", cfg.KeysAsHexStrings()),
						zap.Int("quorum", newQuorum),
					)
				case oldSize != newSize:
					p.logger.Warn("delegated guardian config chain set size changed",
						zap.Stringer("chainID", chain),
						zap.Int("old_size", oldSize),
						zap.Int("new_size", newSize),
						zap.Strings("old_set", oldCfg.KeysAsHexStrings()),
						zap.Strings("new_set", cfg.KeysAsHexStrings()),
						zap.Int("quorum", newQuorum),
					)
				case oldQuorum != newQuorum:
					p.logger.Warn("delegated guardian config chain threshold changed",
						zap.Stringer("chainID", chain),
						zap.Int("old_quorum", oldQuorum),
						zap.Int("new_quorum", newQuorum),
						zap.Strings("set", cfg.KeysAsHexStrings()),
					)
				case !slices.Equal(oldCfg.Keys, cfg.Keys):
					p.logger.Warn("delegated guardian config chain set changed",
						zap.Stringer("chainID", chain),
						zap.Strings("old_set", oldCfg.KeysAsHexStrings()),
						zap.Strings("new_set", cfg.KeysAsHexStrings()),
						zap.Int("quorum", newQuorum),
					)
				default:
					p.logger.Debug("delegated guardian config chain unchanged",
						zap.Stringer("chainID", chain),
						zap.Strings("set", cfg.KeysAsHexStrings()),
						zap.Int("quorum", newQuorum),
					)
				}
			}

			p.dgc.Set(chains)
		case k := <-p.msgC:
			if k == nil {
				p.logger.Error("received nil MessagePublication from msgC channel")
				channelNilReceive.WithLabelValues("msgC").Inc()
				continue
			}

			p.logger.Debug("processor: received new message publication on message channel", k.ZapFields()...)

			cfg := p.dgc.GetChainConfig(k.EmitterChain)
			p.logger.Info("processor: checking delegation config for chain",
				zap.Uint32("emitter_chain", uint32(k.EmitterChain)),
				zap.Bool("has_config", cfg != nil),
				zap.String("our_addr", p.ourAddr.Hex()),
			)
			// len(cfg.Keys) > 0 is redundant, kept for extra safety
			if cfg != nil && len(cfg.Keys) > 0 {
				_, ok := cfg.KeyIndex(p.ourAddr)
				p.logger.Info("processor: delegation check result",
					zap.Uint32("emitter_chain", uint32(k.EmitterChain)),
					zap.Bool("is_delegated_guardian", ok),
					zap.Int("chain_quorum", cfg.Quorum()),
					zap.Strings("delegated_keys", cfg.KeysAsHexStrings()),
				)
				if ok {
					p.logger.Info("processor: process message publication using main processing loop")

					// Send messages to the Notary first. If messages are not approved, they should not continue
					// to the Governor or the Accountant.
					if !p.processWithNotary(k) {
						continue
					}

					p.logger.Info("processor: sending delegate observation as delegated guardian", k.ZapFields()...)
					if err := p.handleDelegateMessagePublication(k); err != nil {
						p.logger.Warn("failed to send delegate observation", k.ZapFields(zap.Error(err))...)
					} else {
						p.logger.Info("processor: successfully queued delegate observation", k.ZapFields()...)
					}

					// Send messages to the Governor and/or the Accountant
					if !p.processWithGovernor(k) {
						continue
					}

					if err := p.processWithAccountant(ctx, k); err != nil {
						return err
					}
				} else {
					p.logger.Info("processor: skipping message publication and delegate observation - not a delegated guardian for this chain",
						zap.Uint32("emitter_chain", uint32(k.EmitterChain)),
					)
				}
			} else {
				p.logger.Info("processor: no delegation config found for chain",
					zap.Uint32("emitter_chain", uint32(k.EmitterChain)),
				)
				p.logger.Info("processor: process message publication using main processing loop")
				if err := p.handleMessagePublication(ctx, k); err != nil {
					return err
				}
			}
		case k := <-p.acctReadC:
			if k == nil {
				p.logger.Error("received nil MessagePublication from acctReadC channel")
				channelNilReceive.WithLabelValues("acctReadC").Inc()
				continue
			}

			if p.acct == nil {
				return fmt.Errorf("received an accountant event when accountant is not configured")
			}
			// SECURITY defense-in-depth: Make sure the accountant did not generate an unexpected message.
			if !p.acct.IsMessageCoveredByAccountant(k) {
				return fmt.Errorf("accountant published a message that is not covered by it: `%s`", k.MessageIDString())
			}
			p.handleMessage(ctx, k)
		case m := <-p.batchObsvC:
			if m == nil {
				p.logger.Error("received nil MsgWithTimeStamp[SignedObservationBatch] from batchObsvC channel")
				channelNilReceive.WithLabelValues("batchObsvC").Inc()
				continue
			}
			batchObservationChanDelay.Observe(float64(time.Since(m.Timestamp).Microseconds()))
			p.handleBatchObservation(m)
		case m := <-p.signedInC:
			if m == nil {
				p.logger.Error("received nil SignedVAAWithQuorum from signedInC channel")
				channelNilReceive.WithLabelValues("signedInC").Inc()
				continue
			}
			p.handleInboundSignedVAAWithQuorum(m)
		case m := <-p.delegateObsvC:
			if m == nil {
				p.logger.Error("received nil DelegateObservation from delegateObsvC channel")
				channelNilReceive.WithLabelValues("delegateObsvC").Inc()
				continue
			}
			if err := p.handleDelegateObservation(ctx, m); err != nil {
				return err
			}
		case <-cleanup.C:
			p.handleCleanup(ctx)
		case <-pollTimer.C:
			// Poll the pending lists for messages that can be released. Both the Notary and the Governor
			// can delay messages.
			// As each of the Notary, Governor, and Accountant can be enabled separately, each must
			// be processed in a modular way.
			// When more than one of these features are enabled, messages should be processed
			// serially in the order: Notary -> Governor -> Accountant.
			// NOTE: The Accountant can signal to a channel that it is ready to publish a message via
			// writing to acctReadC so it is not handled here.
			if p.notary != nil {
				readyMsgs := p.notary.ReleaseReadyMessages()

				// Iterate over all ready messages. Hand-off to the Governor or the Accountant
				// if they're enabled. If not, publish.
				for _, msg := range readyMsgs {
					// TODO: Much of this is duplicated from the msgC branch. It might be a good
					// idea to refactor how we handle combinations of Notary, Governor, and Accountant being
					// enabled.

					// Publish DelegateObservation if we are a delegated guardian for the chain
					cfg := p.dgc.GetChainConfig(msg.EmitterChain)
					// len(cfg.Keys) > 0 is redundant, kept for extra safety
					if cfg != nil && len(cfg.Keys) > 0 {
						_, ok := cfg.KeyIndex(p.ourAddr)
						if ok {
							if err := p.handleDelegateMessagePublication(msg); err != nil {
								p.logger.Warn("failed to send delegate observation", msg.ZapFields(zap.Error(err))...)
							} else {
								p.logger.Info("processor: successfully queued delegate observation", msg.ZapFields()...)
							}
						}
					}

					// Hand-off to governor
					if !p.processWithGovernor(msg) {
						continue
					}

					// Hand-off to accountant. If we get here, both the Notary and the Governor
					// have signalled that the message is OK to publish.
					if err := p.processWithAccountant(ctx, msg); err != nil {
						return err
					}
				}
			}

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
						if err := p.processWithAccountant(ctx, k); err != nil {
							return err
						}
					}
				}
			}

			if (p.notary != nil) || (p.governor != nil) || (p.acct != nil) {
				pollTimer.Reset(PollInterval)
			}
		}
	}
}

// storeSignedVAA schedules a database update for a VAA.
func (p *Processor) storeSignedVAA(v *vaa.VAA) {
	if v.EmitterChain == vaa.ChainIDPythNet {
		key := fmt.Sprintf("%v/%v", v.EmitterAddress, v.Sequence)
		p.pythnetVaas[key] = PythNetVaaEntry{v: v, updateTime: time.Now()}
		return
	}
	key := fmt.Sprintf("%d/%v/%v", v.EmitterChain, v.EmitterAddress, v.Sequence)
	p.updateVAALock.Lock()
	p.updatedVAAs[key] = &updateVaaEntry{v: v, dirty: true}
	p.updateVAALock.Unlock()
}

// haveSignedVAA returns true if we already have a VAA for the given VAAID
func (p *Processor) haveSignedVAA(id guardianDB.VAAID) bool {
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
// gets updated again while we are in the process of writing it to the database.
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
				vaaBatch := make([]*vaa.VAA, 0, len(updatedVAAs))
				for _, entry := range updatedVAAs {
					vaaBatch = append(vaaBatch, entry.v)
				}

				if err := p.db.StoreSignedVAABatch(vaaBatch); err != nil {
					p.logger.Error("failed to write VAAs to database", zap.Int("numVAAs", len(vaaBatch)), zap.Error(err))
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

// trackVerificationState tracks transfer verification states for analytics and logs unusual states.
// This provides centralized observability for the Transfer Verifier across all watchers.
func (p *Processor) trackVerificationState(msg *common.MessagePublication) {
	state := msg.VerificationState()

	// Only track states that have been verified (skip NotVerified and NotApplicable)
	if state == common.NotVerified || state == common.NotApplicable {
		return
	}

	emitterChainStr := fmt.Sprintf("%d", msg.EmitterChain)

	// Track unusual states (anything other than Valid)
	if state != common.Valid {
		msgVerificationStates.WithLabelValues(state.String(), emitterChainStr).Inc()

		// Log unusual states for visibility
		p.logger.Warn("transfer verifier returned unusual state",
			msg.ZapFields(
				zap.String("verification_state", state.String()),
			)...)
	}
}
