package tss

import (
	"bytes"
	"fmt"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

var errNilGuardianSetState = fmt.Errorf("tss' guardianSetState nil")

// Assumes valid VAAs only.
func (t *Engine) WitnessNewVaa(v *vaa.VAA) error {
	if t == nil {
		return errNilTssEngine
	}

	if t.started.Load() != started {
		return errTssEngineNotStarted
	}

	if t.gst == nil {
		return errNilGuardianSetState
	}

	if !t.isleader {
		return nil
	}

	if v == nil {
		return fmt.Errorf("nil VAA")
	}

	if v.Version != vaa.VaaVersion1 {
		return fmt.Errorf("tss accepts VAA version 1 only. (received: %v)", v.Version)
	}

	bts, err := v.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal VAA: %w", err)
	}

	send := Unicast{
		Unicast: &tsscommv1.Unicast{
			Content: &tsscommv1.Unicast_Vaav1{
				Vaav1: &tsscommv1.VaaV1Info{
					Marshaled: bts,
				},
			},
		},
		Receipients: t.guardiansProtoIDs, // sending to all guardians.
	}

	select {
	case t.messageOutChan <- &send:
		t.logger.Info("leader sent VAAV1 to all guardians",
			zap.String("chainID", v.EmitterChain.String()),
			zap.String("digest", fmt.Sprintf("%x", v.SigningDigest())),
		)
	default:
		t.logger.Error(
			"leader failed to send new VAA seen to guardians, network output channel buffer is full",
			zap.String("chainID", v.EmitterChain.String()),
		)
	}

	return nil
}

// handleUnicastVaaV1 expects to receive valid Vaav1 messages.
// If the VAA is valid, it will trigger the TSS signing protocol too for that VAA (beginTSSSign, will ensure double signing for the same digest).
func (t *Engine) handleUnicastVaaV1(v *tsscommv1.Unicast_Vaav1, src *tsscommv1.PartyId) error {
	if t.gst == nil {
		return fmt.Errorf("no guardian set state")
	}

	if !bytes.Equal(t.LeaderIdentity, src.Key) {
		return fmt.Errorf("tss received a VAA unicast from a replica (non-leader): %s", src.Id)
	}

	if v == nil || v.Vaav1 == nil {
		return fmt.Errorf("tss received nil VAA")
	}

	newVaa, err := vaa.Unmarshal(v.Vaav1.Marshaled)
	if err != nil {
		return fmt.Errorf("tss failed to unmarshal VAA %w", err)
	}

	if newVaa.Version != vaa.VaaVersion1 {
		return fmt.Errorf("tss accepts VAA version 1 only. (received: %v)", newVaa.Version)
	}

	if err := newVaa.Verify(t.gst.Get().Keys); err != nil {
		return fmt.Errorf("tss received VAA that fails verification: %w", err)
	}

	dgst := newVaa.SigningDigest()

	return t.beginTSSSign(dgst[:], newVaa.EmitterChain, newVaa.ConsistencyLevel, signingMeta{isFromVaav1: true})
}
