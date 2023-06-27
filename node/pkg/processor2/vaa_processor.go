package processor2

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/accountant"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/processor2/reactor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// VAAConsensusProcessor is a reactor for performing consensus on v1 VAA messages. It sits on top of a reactor.Manager and implements
// VAA specific logic like persistence, signed VAA broadcasts, accounting and governor.
type VAAConsensusProcessor struct {
	// msgC is a channel of observed emitted messages
	msgC <-chan *common.MessagePublication
	// obsC is a channel of VAAs used to forward to the reactor manager
	obsC chan *vaaObservation
	// obsvReqSendC is a send-only channel of outbound re-observation requests to broadcast on p2p
	obsvReqSendC chan<- *gossipv1.ObservationRequest
	// signedInC is a channel of inbound signed VAA observations from p2p
	signedInC <-chan *gossipv1.SignedVAAWithQuorum
	// gossipSendC is a channel of outbound messages to broadcast on p2p
	gossipSendC chan<- []byte

	attestationEvents *reporter.AttestationEventReporter
	logger            *zap.Logger
	db                *db.Database

	manager *reactor.Manager[*vaaObservation]
	gst     *common.GuardianSetState

	governor        *governor.ChainGovernor
	acct            *accountant.Accountant
	acctReadC       <-chan *common.MessagePublication
	pythnetVaas     map[string]PythNetVaaEntry
	pythnetVAAsLock sync.Mutex
}

// NewVAAConsensusProcessor creates a new VAAConsensusProcessor.
func NewVAAConsensusProcessor(
	db *db.Database,
	msgC <-chan *common.MessagePublication,
	gossipSendC chan<- []byte,
	obsvC <-chan *gossipv1.SignedObservation,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	signedInC <-chan *gossipv1.SignedVAAWithQuorum,
	gk *ecdsa.PrivateKey,
	gst *common.GuardianSetState,
	attestationEvents *reporter.AttestationEventReporter,
	g *governor.ChainGovernor,
	acct *accountant.Accountant,
	acctReadC <-chan *common.MessagePublication,
) *VAAConsensusProcessor {
	// obsC is an internal buffer that is used to forward pre-filtered messages to the reactor manager.
	// See VAAConsensusProcessor.Run for filtering logic.
	obsC := make(chan *vaaObservation, 10)

	r := &VAAConsensusProcessor{
		obsvReqSendC:      obsvReqSendC,
		signedInC:         signedInC,
		msgC:              msgC,
		obsC:              obsC,
		gossipSendC:       gossipSendC,
		db:                db,
		attestationEvents: attestationEvents,
		governor:          g,
		acct:              acct,
		acctReadC:         acctReadC,
		pythnetVaas:       map[string]PythNetVaaEntry{},
		gst:               gst,
	}

	manager := reactor.NewManager[*vaaObservation]("vaa", obsC, obsvC, gst, reactor.Config{
		RetransmitFrequency: 5 * time.Minute,
		QuorumGracePeriod:   2 * time.Minute,
		QuorumTimeout:       24 * time.Hour,
		UnobservedTimeout:   24 * time.Hour,
		Signer:              reactor.NewEcdsaKeySigner(gk),
		NetworkAdapter:      reactor.NewChannelNetworkAdapter(gossipSendC),
	}, r, r)
	r.manager = manager

	return r
}

func (p *VAAConsensusProcessor) Run(ctx context.Context) error {
	p.logger = supervisor.Logger(ctx)

	err := supervisor.Run(ctx, "vaa-reactor-manager", p.manager.Run)
	if err != nil {
		return fmt.Errorf("failed to start reactor manager: %w", err)
	}
	wg := &sync.WaitGroup{}

	// Processor for local message observations
	for i := 0; i < runtime.NumCPU()/2; i++ {
		spawnChannelProcessor(ctx, wg, p.msgC, p.processMessageObservation)
	}

	// Processors for inbound signed VAAs
	for i := 0; i < runtime.NumCPU()/2; i++ {
		spawnChannelProcessor(ctx, wg, p.signedInC, p.handleSignedVAA)
	}

	// Processor for accountant messages
	spawnChannelProcessor(ctx, wg, p.acctReadC, p.processAccountantMessage)

	spawnTickerRunnable(ctx, wg, 30*time.Second, p.cleanPythnetVAAs)
	spawnTickerRunnable(ctx, wg, 5*time.Minute, p.checkReobservations)
	spawnTickerRunnable(ctx, wg, time.Minute, p.processGovernorPending)

	// Wait for all goroutines to exit.
	wg.Wait()

	return nil
}

func (p *VAAConsensusProcessor) cleanPythnetVAAs() {
	// Clean up old pythnet VAAs.
	oldestTime := time.Now().Add(-time.Hour)
	p.pythnetVAAsLock.Lock()
	for key, pe := range p.pythnetVaas {
		if pe.updateTime.Before(oldestTime) {
			delete(p.pythnetVaas, key)
		}
	}
	p.pythnetVAAsLock.Unlock()
}

func (p *VAAConsensusProcessor) processAccountantMessage(k *common.MessagePublication) {
	if p.acct == nil {
		panic("acct: received an accountant event when accountant is not configured")
	}
	// SECURITY defense-in-depth: Make sure the accountant did not generate an unexpected message.
	if !p.acct.IsMessageCoveredByAccountant(k) {
		p.logger.Error("acct: accountant published a message that is not covered by it", zap.String("message_id", k.MessageIDString()))
		return
	}
	gs := p.gst.Get()
	if gs == nil {
		p.logger.Warn("received observation before guardian set was known - skipping")
		return
	}
	p.obsC <- &vaaObservation{
		MessagePublication: k,
		gsIndex:            gs.Index,
	}
}

func (p *VAAConsensusProcessor) processMessageObservation(k *common.MessagePublication) {
	if k.EmitterAddress == vaa.GovernanceEmitter && k.EmitterChain == vaa.GovernanceChain {
		p.logger.Error(
			"EMERGENCY: PLEASE REPORT THIS IMMEDIATELY! A Solana message was emitted from the governance emitter. This should never be possible.",
			zap.Stringer("emitter_chain", k.EmitterChain),
			zap.Stringer("emitter_address", k.EmitterAddress),
			zap.Uint32("nonce", k.Nonce),
			zap.Stringer("txhash", k.TxHash),
			zap.Time("timestamp", k.Timestamp))
		return
	}

	if p.governor != nil {
		if !p.governor.ProcessMsg(k) {
			return
		}
	}
	if p.acct != nil {
		shouldPub, err := p.acct.SubmitObservation(k)
		if err != nil {
			p.logger.Error("acct: failed to process message", zap.String("message_id", k.MessageIDString()), zap.Error(err))
			return
		}
		if !shouldPub {
			return
		}
	}

	// Ignore incoming observations when our database already has a quorum VAA for it.
	// This can occur when we're receiving late observations due to node catchup.
	if existing, err := p.getSignedVAA(*db.VaaIDFromVAA(k.CreateVAA(0))); err == nil {
		p.logger.Debug("ignoring observation since we already have a quorum VAA for it",
			zap.Stringer("emitter_chain", k.EmitterChain),
			zap.Stringer("emitter_address", k.EmitterAddress),
			zap.Uint32("nonce", k.Nonce),
			zap.Stringer("txhash", k.TxHash),
			zap.Time("timestamp", k.Timestamp),
			zap.String("message_id", existing.MessageID()),
		)
		return
	}

	gs := p.gst.Get()
	if gs == nil {
		p.logger.Warn("received observation before guardian set was known - skipping")
		return
	}

	v := k.CreateVAA(gs.Index)
	p.attestationEvents.ReportMessagePublication(&reporter.MessagePublication{
		VAA:            *v,
		InitiatingTxID: k.TxHash,
	})
	p.obsC <- &vaaObservation{
		MessagePublication: k,
		gsIndex:            gs.Index,
	}
}

// processGovernorPending checks the governor for pending messages and publishes them.
func (p *VAAConsensusProcessor) processGovernorPending() {
	if p.governor == nil {
		return
	}
	toBePublished, err := p.governor.CheckPending()
	if err != nil {
		p.logger.Error("failed to check for pending messages on governor", zap.Error(err))
		return
	}
	if len(toBePublished) != 0 {
		for _, k := range toBePublished {
			// SECURITY defense-in-depth: Make sure the governor did not generate an unexpected message.
			if msgIsGoverned, err := p.governor.IsGovernedMsg(k); err != nil {
				p.logger.Error("cgov: governor failed to determine if message should be governed", zap.String("message_id", k.MessageIDString()), zap.Error(err))
				continue
			} else if !msgIsGoverned {
				p.logger.Error("cgov: governor published a message that should not be governed", zap.String("message_id", k.MessageIDString()))
				continue
			}
			if p.acct != nil {
				shouldPub, err := p.acct.SubmitObservation(k)
				if err != nil {
					p.logger.Error("acct: failed to process message released by governor", zap.String("message_id", k.MessageIDString()), zap.Error(err))
					continue
				}
				if !shouldPub {
					continue
				}
			}
			gs := p.gst.Get()
			if gs == nil {
				p.logger.Warn("received observation before guardian set was known - skipping")
				continue
			}
			p.obsC <- &vaaObservation{
				MessagePublication: k,
				gsIndex:            gs.Index,
			}
		}
	}
}

// checkReobservations checks if any observations have been waiting on consensus for more than 5 minutes and
// re-requests them.
func (p *VAAConsensusProcessor) checkReobservations() {
	p.manager.IterateReactors(func(digest ethcommon.Hash, r *reactor.ConsensusReactor[*vaaObservation]) {
		if r.State() != reactor.StateObserved || r.HasQuorum() || time.Since(r.LastObservation()) < time.Minute*5 {
			return
		}

		observation := r.Observation()
		if observation.Unreliable {
			return
		}

		req := &gossipv1.ObservationRequest{
			ChainId: uint32(observation.EmitterChain),
			TxHash:  observation.TxHash.Bytes(),
		}
		if err := common.PostObservationRequest(p.obsvReqSendC, req); err != nil {
			p.logger.Warn("failed to broadcast re-observation request", zap.Error(err))
		}
	})
}

func (p *VAAConsensusProcessor) handleSignedVAA(m *gossipv1.SignedVAAWithQuorum) {
	v, err := vaa.Unmarshal(m.Vaa)
	if err != nil {
		p.logger.Warn("received invalid VAA in SignedVAAWithQuorum message",
			zap.Error(err), zap.Any("message", m))
		return
	}

	// Calculate digest for logging
	digest := v.SigningDigest()
	hash := hex.EncodeToString(digest.Bytes())

	gs := p.gst.Get()
	if gs == nil {
		p.logger.Warn("dropping SignedVAAWithQuorum message since we haven't initialized our guardian set yet",
			zap.String("digest", hash),
			zap.Any("message", m),
		)
		return
	}

	if err := v.Verify(gs.Keys); err != nil {
		p.logger.Warn("dropping SignedVAAWithQuorum message because it failed verification", zap.Error(err))
		return
	}

	// We now established that:
	//  - all signatures on the VAA are valid
	//  - the signature's addresses match the node's current guardian set
	//  - enough signatures are present for the VAA to reach quorum

	// Check if we already store this VAA
	_, err = p.getSignedVAA(*db.VaaIDFromVAA(v))
	if err == nil {
		p.logger.Debug("ignored SignedVAAWithQuorum message for VAA we already store",
			zap.String("digest", hash),
		)
		return
	} else if err != db.ErrVAANotFound {
		p.logger.Error("failed to look up VAA in database",
			zap.String("digest", hash),
			zap.Error(err),
		)
		return
	}

	// Store signed VAA in database.
	p.logger.Info("storing inbound signed VAA with quorum",
		zap.String("digest", hash),
		zap.Any("vaa", v),
		zap.String("bytes", hex.EncodeToString(m.Vaa)),
		zap.String("message_id", v.MessageID()))

	if err := p.storeSignedVAA(v); err != nil {
		p.logger.Error("failed to store signed VAA", zap.Error(err))
		return
	}
	p.attestationEvents.ReportVAAQuorum(v)
}

func (p *VAAConsensusProcessor) HandleQuorum(observation *vaaObservation, signatures []*vaa.Signature) {
	p.logger.Info("reached quorum on VAA", zap.String("message_id", observation.MessageID()), zap.Stringer("digest", observation.SigningDigest()))

	v := observation.VAA()
	v.Signatures = signatures

	p.broadcastSignedVAA(v)
	p.attestationEvents.ReportVAAQuorum(v)
}

func (p *VAAConsensusProcessor) HandleFinalization(observation *vaaObservation, signatures []*vaa.Signature) {
	p.logger.Info("finalized VAA", zap.String("message_id", observation.MessageID()), zap.Stringer("digest", observation.SigningDigest()), zap.Int("num_signatures", len(signatures)))
}

func (p *VAAConsensusProcessor) HandleTimeout(previousState reactor.State, digest ethcommon.Hash, observation *vaaObservation, signatures []*vaa.Signature) {
	p.logger.Info("VAA consensus timed out", zap.String("timeout_state", string(previousState)), zap.Stringer("digest", digest), zap.Bool("observed", observation != nil), zap.Int("num_signatures", len(signatures)))
}

func (p *VAAConsensusProcessor) StoreSignedObservation(observation *vaaObservation, signatures []*vaa.Signature) error {
	v := observation.VAA()
	v.Signatures = signatures

	err := p.storeSignedVAA(v)
	if err != nil {
		return fmt.Errorf("failed to store signed observation: %w", err)
	}

	return nil
}

func (p *VAAConsensusProcessor) GetSignedObservation(id string) (observation *vaaObservation, signatures []*vaa.Signature, found bool, err error) {
	vID, err := db.VaaIDFromString(id)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to parse message id: %w", err)
	}

	v, err := p.getSignedVAA(*vID)
	if err != nil {
		if err == db.ErrVAANotFound {
			return nil, nil, false, nil
		}
		return nil, nil, false, err
	}

	// This is lossy and loses the txHash and reliability status. This is not an issue at this point because the
	// data is only used to persist completed VAAs.
	return &vaaObservation{
		MessagePublication: &common.MessagePublication{
			TxHash:           ethcommon.Hash{},
			Timestamp:        v.Timestamp,
			Nonce:            v.Nonce,
			Sequence:         v.Sequence,
			ConsistencyLevel: v.ConsistencyLevel,
			EmitterChain:     v.EmitterChain,
			EmitterAddress:   v.EmitterAddress,
			Payload:          v.Payload,
			Unreliable:       false,
		},
		gsIndex: v.GuardianSetIndex,
	}, v.Signatures, true, nil
}

type PythNetVaaEntry struct {
	v          *vaa.VAA
	updateTime time.Time // Used for determining when to delete entries
}

func (p *VAAConsensusProcessor) storeSignedVAA(v *vaa.VAA) error {
	if v.EmitterChain == vaa.ChainIDPythNet {
		key := fmt.Sprintf("%v/%v", v.EmitterAddress, v.Sequence)
		p.pythnetVAAsLock.Lock()
		p.pythnetVaas[key] = PythNetVaaEntry{v: v, updateTime: time.Now()}
		p.pythnetVAAsLock.Unlock()
		return nil
	}
	return p.db.StoreSignedVAA(v)
}

func (p *VAAConsensusProcessor) getSignedVAA(id db.VAAID) (*vaa.VAA, error) {
	if id.EmitterChain == vaa.ChainIDPythNet {
		key := fmt.Sprintf("%v/%v", id.EmitterAddress, id.Sequence)
		p.pythnetVAAsLock.Lock()
		ret, exists := p.pythnetVaas[key]
		p.pythnetVAAsLock.Unlock()
		if exists {
			return ret.v, nil
		}

		return nil, db.ErrVAANotFound
	}

	vb, err := p.db.GetSignedVAABytes(id)
	if err != nil {
		return nil, err
	}

	v, err := vaa.Unmarshal(vb)
	if err != nil {
		panic("failed to unmarshal VAA from db")
	}

	return v, err
}

func (p *VAAConsensusProcessor) broadcastSignedVAA(v *vaa.VAA) {
	b, err := v.Marshal()
	if err != nil {
		panic(err)
	}

	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedVaaWithQuorum{
		SignedVaaWithQuorum: &gossipv1.SignedVAAWithQuorum{Vaa: b},
	}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		panic(err)
	}

	p.gossipSendC <- msg
}

// vaaObservation is used to wrap common.MessagePublication as a reactor.Observation.
type vaaObservation struct {
	*common.MessagePublication
	gsIndex uint32
}

func (v *vaaObservation) MessageID() string {
	return v.MessagePublication.MessageIDString()
}

func (v *vaaObservation) SigningDigest() ethcommon.Hash {
	return v.CreateVAA(0).SigningDigest()
}

func (v *vaaObservation) VAA() *vaa.VAA {
	return v.CreateVAA(v.gsIndex)
}

func spawnChannelProcessor[K any](ctx context.Context, wg *sync.WaitGroup, c <-chan K, f func(K)) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case v := <-c:
				f(v)
			}
		}
	}()
}

func spawnTickerRunnable(ctx context.Context, wg *sync.WaitGroup, interval time.Duration, f func()) {
	ticker := time.NewTicker(interval)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				f()
			}
		}
	}()
}
