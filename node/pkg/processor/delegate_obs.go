package processor

import (
	"fmt"
	"math"
	"time"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"google.golang.org/protobuf/proto"
)

// publishDelegateObservation populates and marshals the delegate observation and publishes it to gossip.
func (p *Processor) publishDelegateObservation(d *gossipv1.DelegateObservation) {
	// Populate the missing fields in the delegate observation
	d.GuardianAddr = p.ourAddr.Bytes()
	d.SentTimestamp = time.Now().Unix()

	b, err := proto.Marshal(d)
	if err != nil {
		panic(err)
	}

	select {
	case p.gossipDelegatedAttestationSendC <- b:
	default:
		delegateObservationChannelOverflow.WithLabelValues("gossipSend").Inc()
	}
}

// delegateObservationToMessagePublication converts a DelegateObservation into a MessagePublication that can be passed through the normal processor pipeline.
// Returns error on invalid delegate observation input.
//
// VERSION SKEW: the NumVariantsVerificationState check below is
// forward-incompatible in a way worth flagging: a delegate guardian on a
// newer build that adds a VerificationState variant will produce observations
// the canonical rejects here, even though the canonical's consensus path
// normalizes this field away (NormalizeForDelegateConsensus → NotVerified).
// If enough delegates upgrade ahead of canonicals, the silently-dropped
// observations reduce the effective quorum count and reintroduce the stall
// pattern — operators only see the per-observation "failed to convert
// delegate observation to message publication" warn, with no aggregated
// signal that quorum is being denied. When extending VerificationState,
// either deploy canonicals first or relax this check.
//
// The KnownChainIDFromNumber check has the same forward-incompatibility
// shape but is not flagged separately: a canonical that doesn't know an
// emitter chain can't reason about it downstream anyway (governor wouldn't
// catch transfers from it, etc.), so the deploy-ordering constraint isn't
// specific to delegated guardians.
func delegateObservationToMessagePublication(d *gossipv1.DelegateObservation) (*node_common.MessagePublication, error) {
	if d == nil {
		return nil, fmt.Errorf("nil delegate observation")
	}

	const TxIDSizeMax = math.MaxUint8
	txIDLen := len(d.TxHash)
	if txIDLen > TxIDSizeMax {
		return nil, fmt.Errorf("delegate observation tx_hash too long: got %d; want at most %d", txIDLen, TxIDSizeMax)
	}

	if d.ConsistencyLevel > math.MaxUint8 {
		return nil, fmt.Errorf("invalid delegate observation consistency : %d", d.ConsistencyLevel)
	}

	c, err := vaa.KnownChainIDFromNumber(d.EmitterChain)
	if err != nil {
		return nil, fmt.Errorf("invalid delegate observation emitter chain: %w", err)
	}

	addr, err := vaa.BytesToAddress(d.EmitterAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid delegate observation emitter address: %w", err)
	}

	if d.VerificationState >= node_common.NumVariantsVerificationState {
		return nil, fmt.Errorf("invalid verification state: %d", d.VerificationState)
	}

	mp := &node_common.MessagePublication{
		TxID:             d.TxHash,
		Timestamp:        time.Unix(int64(d.Timestamp), 0), // Timestamp is uint32 representing seconds since UNIX epoch so is safe to convert.
		Nonce:            d.Nonce,
		Sequence:         d.Sequence,
		ConsistencyLevel: uint8(d.ConsistencyLevel),
		EmitterChain:     c,
		EmitterAddress:   addr,
		Payload:          d.Payload,
		IsReobservation:  d.IsReobservation,
		Unreliable:       d.Unreliable,
	}

	// only set verification state if it's not the default value (NotVerified)
	if d.VerificationState != uint32(node_common.NotVerified) {
		if err = mp.SetVerificationState(node_common.VerificationState(d.VerificationState)); err != nil {
			return nil, fmt.Errorf("could not set verification state: %w", err)
		}
	}

	return mp, nil
}

// messagePublicationToDelegateObservation converts a MessagePublication into a DelegateObservation to be sent by a delegated guardian.
// This does not populate the GuardianAddr and SentTimestamp fields.
// Returns error on invalid message publication input
func messagePublicationToDelegateObservation(m *node_common.MessagePublication) (*gossipv1.DelegateObservation, error) {
	if m == nil {
		return nil, fmt.Errorf("nil message publication")
	}

	const TxIDSizeMax = math.MaxUint8
	txIDLen := len(m.TxID)
	if txIDLen > TxIDSizeMax {
		return nil, fmt.Errorf("message publication tx_hash too long: got %d; want at most %d", txIDLen, TxIDSizeMax)
	}

	// Check if payload length is within max message size for p2p
	if len(m.Payload) > node_common.DelegatedPayloadLenMax {
		return nil, fmt.Errorf("message publication payload length too large: got %d; want at most %d", len(m.Payload), node_common.DelegatedPayloadLenMax)
	}

	d := &gossipv1.DelegateObservation{
		Timestamp:         uint32(m.Timestamp.Unix()), // #nosec G115 -- This conversion is safe until year 2106
		Nonce:             m.Nonce,
		EmitterChain:      uint32(m.EmitterChain),
		EmitterAddress:    m.EmitterAddress.Bytes(),
		Sequence:          m.Sequence,
		ConsistencyLevel:  uint32(m.ConsistencyLevel),
		Payload:           m.Payload,
		TxHash:            m.TxID,
		Unreliable:        m.Unreliable,
		IsReobservation:   m.IsReobservation,
		VerificationState: uint32(m.VerificationState()),
		// GuardianAddr and SentTimestamp will be populated in publishDelegateObservation before p2p broadcast.
	}

	return d, nil
}
