package processor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/p2p"

	"github.com/certusone/wormhole/node/pkg/accountant"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/processor/reactor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type VAAReactor struct {
	// msgC is a channel of observed emitted messages
	msgC <-chan *common.SinglePublication
	// obsC is a channel of VAAs used to forward to the reactor manager
	obsC chan *vaaObservation
	// obsvReqSendC is a send-only channel of outbound re-observation requests to broadcast on p2p
	obsvReqSendC chan<- *gossipv1.ObservationRequest

	consensusIO p2p.GossipIO
	vaaIO       p2p.GossipIO

	attestationEvents *reporter.AttestationEventReporter
	logger            *zap.Logger
	db                *db.Database

	manager *reactor.Manager[*vaaObservation]

	// gst is managed by the processor and allows concurrent access to the
	// guardian set by other components.
	gst *common.GuardianSetState

	governor        *governor.ChainGovernor
	acct            *accountant.Accountant
	acctReadC       <-chan *common.SinglePublication
	pythnetVaas     map[string]PythNetVaaEntry
	pythnetVAAsLock sync.Mutex
}

func NewVAAReactor(
	db *db.Database,
	msgC <-chan *common.SinglePublication,
	setC <-chan *common.GuardianSet,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	consensusIO p2p.GossipIO,
	vaaIO p2p.GossipIO,
	gk *ecdsa.PrivateKey,
	gst *common.GuardianSetState,
	attestationEvents *reporter.AttestationEventReporter,
	g *governor.ChainGovernor,
	acct *accountant.Accountant,
	acctReadC <-chan *common.SinglePublication,
) *VAAReactor {
	obsC := make(chan *vaaObservation, 10)

	r := &VAAReactor{
		obsvReqSendC:      obsvReqSendC,
		consensusIO:       consensusIO,
		vaaIO:             vaaIO,
		msgC:              msgC,
		obsC:              obsC,
		gst:               gst,
		db:                db,
		attestationEvents: attestationEvents,
		governor:          g,
		acct:              acct,
		acctReadC:         acctReadC,
		pythnetVaas:       map[string]PythNetVaaEntry{},
	}

	manager := reactor.NewManager[*vaaObservation]("vaa", obsC, consensusIO, setC, gst, reactor.Config{
		RetransmitFrequency: 5 * time.Minute,
		QuorumGracePeriod:   2 * time.Minute,
		QuorumTimeout:       24 * time.Hour,
		UnobservedTimeout:   24 * time.Hour,
		Signer:              reactor.NewEcdsaKeySigner(gk),
	}, r, r)
	r.manager = manager

	return r
}

func (p *VAAReactor) Run(ctx context.Context) error {
	p.logger = supervisor.Logger(ctx)

	err := supervisor.Run(ctx, "vaa-reactor-manager", p.manager.Run)
	if err != nil {
		return fmt.Errorf("failed to start reactor manager: %w", err)
	}

	cleanup := time.NewTicker(30 * time.Second)
	defer cleanup.Stop()

	reobservationTicker := time.NewTicker(5 * time.Minute)
	defer reobservationTicker.Stop()

	// Always initialize the timer so don't have a nil pointer in the case below. It won't get rearmed after that.
	govTimer := time.NewTicker(time.Minute)
	defer govTimer.Stop()

	signedInProducer, signedInConsumer := p2p.MeteredBufferedChannelPair[*gossipv1.GossipMessage_SignedVaaWithQuorum](ctx, 50, "vaa_processor_signed_in")
	err = p2p.SubscribeFiltered(ctx, p.vaaIO, signedInProducer)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			if p.acct != nil {
				p.acct.Close()
			}
			return ctx.Err()
		case k := <-p.msgC:
			if p.governor != nil {
				if !p.governor.ProcessMsg(k) {
					continue
				}
			}
			if p.acct != nil {
				shouldPub, err := p.acct.SubmitObservation(k)
				if err != nil {
					return fmt.Errorf("acct: failed to process message `%s`: %w", k.MessageIDString(), err)
				}
				if !shouldPub {
					continue
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
				continue
			}

			gs := p.manager.GuardianSet()
			if gs == nil {
				p.logger.Warn("received observation before guardian set was known - skipping")
				continue
			}

			v := k.CreateVAA(gs.Index)
			p.attestationEvents.ReportMessagePublication(&reporter.MessagePublication{
				VAA:            *v,
				InitiatingTxID: k.TxHash,
			})
			p.obsC <- &vaaObservation{
				SinglePublication: k,
				gsIndex:           gs.Index,
			}
		case k := <-p.acctReadC:
			if p.acct == nil {
				panic("acct: received an accountant event when accountant is not configured")
			}
			gs := p.manager.GuardianSet()
			if gs == nil {
				p.logger.Warn("received observation before guardian set was known - skipping")
				continue
			}
			p.obsC <- &vaaObservation{
				SinglePublication: k,
				gsIndex:           gs.Index,
			}
		case <-cleanup.C:
			// Clean up old pythnet VAAs.
			oldestTime := time.Now().Add(-time.Hour)
			p.pythnetVAAsLock.Lock()
			for key, pe := range p.pythnetVaas {
				if pe.updateTime.Before(oldestTime) {
					delete(p.pythnetVaas, key)
				}
			}
			p.pythnetVAAsLock.Unlock()
		case <-reobservationTicker.C:
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
		case s := <-signedInConsumer:
			p.handleSignedVAA(s.SignedVaaWithQuorum)
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
							return fmt.Errorf("cgov: governor failed to determine if message should be governed: `%s`: %w", k.MessageIDString(), err)
						} else if !msgIsGoverned {
							return fmt.Errorf("cgov: governor published a message that should not be governed: `%s`", k.MessageIDString())
						}
						if p.acct != nil {
							shouldPub, err := p.acct.SubmitObservation(k)
							if err != nil {
								return fmt.Errorf("acct: failed to process message released by governor `%s`: %w", k.MessageIDString(), err)
							}
							if !shouldPub {
								continue
							}
						}
						gs := p.manager.GuardianSet()
						if gs == nil {
							p.logger.Warn("received observation before guardian set was known - skipping")
							continue
						}
						p.obsC <- &vaaObservation{
							SinglePublication: k,
							gsIndex:           gs.Index,
						}
					}
				}
			}
		}
	}
}

func (p *VAAReactor) handleSignedVAA(m *gossipv1.SignedVAAWithQuorum) {
	v, err := vaa.Unmarshal(m.Vaa)
	if err != nil {
		p.logger.Warn("received invalid VAA in SignedVAAWithQuorum message",
			zap.Error(err), zap.Any("message", m))
		return
	}

	// Calculate digest for logging
	digest := v.SigningMsg()
	hash := hex.EncodeToString(digest.Bytes())

	gs := p.manager.GuardianSet()
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

func (p *VAAReactor) HandleQuorum(observation *vaaObservation, signatures []*vaa.Signature) {
	p.logger.Info("reached quorum on VAA", zap.String("message_id", observation.MessageID()), zap.Stringer("digest", observation.SigningMsg()))

	v := observation.VAA()
	v.Signatures = signatures

	timeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := p.broadcastSignedVAA(timeout, v); err != nil {
		p.logger.Warn("failed to broadcast signed VAA", zap.Error(err))
	}
	p.attestationEvents.ReportVAAQuorum(v)
}

func (p *VAAReactor) HandleFinalization(observation *vaaObservation, signatures []*vaa.Signature) {
	p.logger.Info("finalized VAA", zap.String("message_id", observation.MessageID()), zap.Stringer("digest", observation.SigningMsg()), zap.Int("num_signatures", len(signatures)))
}

func (p *VAAReactor) HandleTimeout(previousState reactor.State, digest ethcommon.Hash, observation *vaaObservation, signatures []*vaa.Signature) {
	p.logger.Info("VAA consensus timed out", zap.String("timeout_state", string(previousState)), zap.Stringer("digest", digest), zap.Bool("observed", observation != nil), zap.Int("num_signatures", len(signatures)))
}

func (p *VAAReactor) StoreSignedObservation(observation *vaaObservation, signatures []*vaa.Signature) error {
	v := observation.VAA()
	v.Signatures = signatures

	err := p.storeSignedVAA(v)
	if err != nil {
		return fmt.Errorf("failed to store signed observation: %w", err)
	}

	return nil
}

func (p *VAAReactor) GetSignedObservation(id string) (observation *vaaObservation, signatures []*vaa.Signature, found bool, err error) {
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
		SinglePublication: &common.SinglePublication{
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

func (p *VAAReactor) storeSignedVAA(v *vaa.VAA) error {
	if v.EmitterChain == vaa.ChainIDPythNet {
		key := fmt.Sprintf("%v/%v", v.EmitterAddress, v.Sequence)
		p.pythnetVAAsLock.Lock()
		p.pythnetVaas[key] = PythNetVaaEntry{v: v, updateTime: time.Now()}
		p.pythnetVAAsLock.Unlock()
		return nil
	}
	return p.db.StoreSignedVAA(v)
}

func (p *VAAReactor) getSignedVAA(id db.VAAID) (*vaa.VAA, error) {
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

func (p *VAAReactor) broadcastSignedVAA(ctx context.Context, v *vaa.VAA) error {
	b, err := v.Marshal()
	if err != nil {
		panic(err)
	}

	err = p.vaaIO.Send(ctx, &gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedVaaWithQuorum{
		SignedVaaWithQuorum: &gossipv1.SignedVAAWithQuorum{Vaa: b},
	}})
	if err != nil {
		return err
	}

	return nil
}

// vaaObservation is used to wrap common.SinglePublication as a reactor.Observation.
type vaaObservation struct {
	*common.SinglePublication
	gsIndex uint32
}

func (v *vaaObservation) MessageID() string {
	return v.SinglePublication.MessageIDString()
}

func (v *vaaObservation) SigningMsg() ethcommon.Hash {
	return v.CreateVAA(v.gsIndex).SigningMsg()
}

func (v *vaaObservation) VAA() *vaa.VAA {
	return v.CreateVAA(v.gsIndex)
}
