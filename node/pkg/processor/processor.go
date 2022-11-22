package processor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/notify/discord"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

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
		// SigningMsg returns the hash of the signing body of the observation. This is used
		// for signature generation and verification.
		SigningMsg() ethcommon.Hash
		// IsReliable returns whether this message is considered reliable meaning it can be reobserved.
		IsReliable() bool
		// HandleQuorum finishes processing the observation once a quorum of signatures have
		// been received for it.
		HandleQuorum(sigs []*vaa.Signature, hash string, p *Processor)
	}

	// a ObservationGroup is a group of messages (Observations) emitted by a single transaction.
	ObservationGroup interface {
		// GetEmitterChain returns the id of the chain where this event was observed.
		GetEmitterChain() vaa.ChainID
		// GetTransactionID returns the unique identifer of the transaction from the source chain.
		GetTransactionID() vaa.TransactionID
		// GetNonce returns the nonce of the observations that make up the Batch.
		GetNonce() vaa.Nonce
		// GetBatchID returns the string representation of the BatchID.
		GetBatchID() vaa.BatchID
		// SigningMsg returns the hash of the signing body of the observation. This is used
		// for signature generation and verification.
		SigningMsg() ethcommon.Hash
		// HandleQuorum finishes processing the observation once a quorum of signatures have
		// been received for it.
		HandleQuorum(sigs []*vaa.Signature, hash string, p *Processor)
	}

	// state represents the local view of a given observation
	state struct {
		// First time this digest was seen (possibly even before we observed it ourselves).
		firstObserved time.Time
		// The most recent time that a re-observation request was sent to the guardian network.
		lastRetry time.Time
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
		// Number of times the cleanup service has attempted to retransmit this VAA.
		retryCount uint
		// Copy of the bytes we submitted (ourObservation, but signed and serialized). Used for retransmissions.
		ourMsg []byte
		// The hash of the transaction in which the observation was made.  Used for re-observation requests.
		txHash []byte
		// Copy of the guardian set valid at observation/injection time.
		gs *common.GuardianSet
	}

	// batchState represents the local view of a group of Observations that are being considered for BatchVAAs.
	batchState struct {
		state
		ourObservation ObservationGroup
	}

	// observationMap holds Observations, and some metadata about them, as they progress toward quorum.
	// Messages get added when they are observed, or when they are first seen via a gossip message.
	// Keys in the map are the determinisic hashes (aka "digest") of the Observation.
	// Items are removed from the map (and memory) after they have reached quorum, or been stale for
	// some time, via the logic in cleanup.go
	observationMap map[string]*state

	// batchMap holds observed BatchMessages, keyed by BatchID.
	// The batches held in state are the observations (from watchers) of transactions
	// with multiple messages. This observation data is the reference for which messages a Batch
	// needs to include to be considered fully observed - ready to sign and broadcast.
	// This map is checked as quorum is reached for individual VAAs in the batch.
	batchMap map[vaa.BatchID][]*common.MessagePublication

	// observationsByBatchID maps the batchID to the messageIDs within the transaction.
	// This map collects observations seen by the guardian, by the message's BatchID, in order
	// to identify transactions that have multiple messages. As messages get observed they are
	// filed into this map, and when a BatchID has >= 2 messages, the transaction is flagged to
	// be watched and tracked to produce BatchVAAs. Stale items are periodically freed from memory.
	observationsByBatchID map[vaa.BatchID]map[string]*state

	// batchObservationMap collects SignedBatchObservation gossip toward BatchVAA quorum.
	// The paradigm is the exactly the same as collecting SignedObservations for v1 VAAs,
	// the BatchVAA was "observed" (all messages independently achieved quorum, we saw them, will attest it),
	// it's a matter of collecting peer signatures and testing for quorum.
	// Keys in the map are BatchVAA digests.
	batchObservationMap map[string]*batchState

	// aggregationState represents the node's aggregation of guardian signatures.
	aggregationState struct {
		signatures observationMap

		batches         batchMap              // reference for which messages make up a batch.
		batchMessages   observationsByBatchID // collect messages by BatchID, to identify batch transactions.
		batchSignatures batchObservationMap   // collects signatures on a fully-observed batch of messages.
	}
)

type PythNetVaaEntry struct {
	v          *vaa.VAA
	updateTime time.Time // Used for determining when to delete entries
}

type Processor struct {
	// lockC is a channel of observed emitted messages
	lockC chan *common.MessagePublication
	// setC is a channel of guardian set updates
	setC chan *common.GuardianSet

	// sendC is a channel of outbound messages to broadcast on p2p
	sendC chan []byte
	// obsvC is a channel of inbound decoded observations from p2p
	obsvC chan *gossipv1.SignedObservation

	// obsvReqSendC is a send-only channel of outbound re-observation requests to broadcast on p2p
	obsvReqSendC chan<- *gossipv1.ObservationRequest

	// signedInC is a channel of inbound signed VAA observations from p2p
	signedInC chan *gossipv1.SignedVAAWithQuorum

	// batchC is a ready-only channel of batched message data
	batchC chan *common.TransactionData

	// batchReqC is a send-only channel for requesting batch message data
	batchReqC chan *common.TransactionQuery

	// batchObsvC is a channel of inbound decoded batch observations from p2p
	batchObsvC chan *gossipv1.SignedBatchObservation

	// batchInC is a channel of inbound signed Batch VAAs from p2p
	batchSignedInC chan *gossipv1.SignedBatchVAAWithQuorum

	// batchVAAEnabled toggles handling or ignoring all things BatchVAA
	batchVAAEnabled bool

	// injectC is a channel of VAAs injected locally.
	injectC chan *vaa.VAA

	// gk is the node's guardian private key
	gk *ecdsa.PrivateKey

	// devnetMode specified whether to submit transactions to the hardcoded Ethereum devnet
	devnetMode         bool
	devnetNumGuardians uint
	devnetEthRPC       string

	wormchainLCD string

	attestationEvents *reporter.AttestationEventReporter

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
	// cleanup triggers periodic state cleanup
	cleanup *time.Ticker

	notifier    *discord.DiscordNotifier
	governor    *governor.ChainGovernor
	pythnetVaas map[string]PythNetVaaEntry
}

func NewProcessor(
	ctx context.Context,
	db *db.Database,
	lockC chan *common.MessagePublication,
	setC chan *common.GuardianSet,
	sendC chan []byte,
	obsvC chan *gossipv1.SignedObservation,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	injectC chan *vaa.VAA,
	signedInC chan *gossipv1.SignedVAAWithQuorum,
	batchC chan *common.TransactionData,
	batchReqC chan *common.TransactionQuery,
	batchObsvC chan *gossipv1.SignedBatchObservation,
	batchSignedInC chan *gossipv1.SignedBatchVAAWithQuorum,
	batchVAAEnabled bool,
	gk *ecdsa.PrivateKey,
	gst *common.GuardianSetState,
	devnetMode bool,
	devnetNumGuardians uint,
	devnetEthRPC string,
	wormchainLCD string,
	attestationEvents *reporter.AttestationEventReporter,
	notifier *discord.DiscordNotifier,
	g *governor.ChainGovernor,
) *Processor {

	return &Processor{
		lockC:              lockC,
		setC:               setC,
		sendC:              sendC,
		obsvC:              obsvC,
		obsvReqSendC:       obsvReqSendC,
		signedInC:          signedInC,
		batchC:             batchC,
		batchReqC:          batchReqC,
		batchObsvC:         batchObsvC,
		batchSignedInC:     batchSignedInC,
		batchVAAEnabled:    batchVAAEnabled,
		injectC:            injectC,
		gk:                 gk,
		gst:                gst,
		devnetMode:         devnetMode,
		devnetNumGuardians: devnetNumGuardians,
		devnetEthRPC:       devnetEthRPC,
		db:                 db,

		wormchainLCD: wormchainLCD,

		attestationEvents: attestationEvents,

		notifier: notifier,

		logger: supervisor.Logger(ctx),
		state: &aggregationState{
			observationMap{},
			batchMap{},
			observationsByBatchID{},
			batchObservationMap{},
		},
		ourAddr:     crypto.PubkeyToAddress(gk.PublicKey),
		governor:    g,
		pythnetVaas: make(map[string]PythNetVaaEntry),
	}
}

func (p *Processor) Run(ctx context.Context) error {
	p.cleanup = time.NewTicker(30 * time.Second)

	// Always initialize the timer so don't have a nil pointer in the case below. It won't get rearmed after that.
	govTimer := time.NewTimer(time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p.gs = <-p.setC:
			p.logger.Info("guardian set updated",
				zap.Strings("set", p.gs.KeysAsHexStrings()),
				zap.Uint32("index", p.gs.Index))
			p.gst.Set(p.gs)
		case k := <-p.lockC:
			if p.governor != nil {
				if !p.governor.ProcessMsg(k) {
					continue
				}
			}
			p.handleMessage(ctx, k)
		case v := <-p.injectC:
			p.handleInjection(ctx, v)
		case m := <-p.obsvC:
			p.handleObservation(ctx, m)
		case m := <-p.signedInC:
			p.handleInboundSignedVAAWithQuorum(ctx, m)
		case m := <-p.batchC:
			p.handleBatchMessage(ctx, m)
		case m := <-p.batchObsvC:
			p.handleBatchObservation(ctx, m)
		case m := <-p.batchSignedInC:
			p.handleInboundSignedBatchVAAWithQuorum(ctx, m)
		case <-p.cleanup.C:
			// run the Batch cleanup first, as batches are composed of observations.
			p.handleBatchCleanup(ctx)
			p.handleCleanup(ctx)
		case <-govTimer.C:
			if p.governor != nil {
				toBePublished, err := p.governor.CheckPending()
				if err != nil {
					return err
				}
				if len(toBePublished) != 0 {
					for _, k := range toBePublished {
						p.handleMessage(ctx, k)
					}
				}
				govTimer = time.NewTimer(time.Minute)
			}
		}
	}
}

func (p *Processor) storeSignedVAA(v *vaa.VAA) error {
	if v.EmitterChain == vaa.ChainIDPythNet {
		key := fmt.Sprintf("%v/%v", v.EmitterAddress, v.Sequence)
		p.pythnetVaas[key] = PythNetVaaEntry{v: v, updateTime: time.Now()}
		return nil
	}
	return p.db.StoreSignedVAA(v)
}

func (p *Processor) getSignedVAA(id db.VAAID) (*vaa.VAA, error) {
	if id.EmitterChain == vaa.ChainIDPythNet {
		key := fmt.Sprintf("%v/%v", id.EmitterAddress, id.Sequence)
		ret, exists := p.pythnetVaas[key]
		if exists {
			return ret.v, nil
		}

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
