package tss

import (
	"fmt"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

var errNilGuardianSetState = fmt.Errorf("tss' guardianSetState nil")
var errNilVaa = fmt.Errorf("nil VAA")

// Assumes valid VAAs only.
func (t *Engine) WitnessNewVaa(v *vaa.VAA) (err error) {
	if t == nil {
		return errNilTssEngine
	}

	if t.started.Load() != started {
		return errTssEngineNotStarted
	}

	// consider removing this check: Going leaderless, and letting everyone send VAAs they see
	// adds a layer of availability (no need to worry about leader missing VAAs or going offline),
	// but will spam the network with duplicated VAAs.
	if !t.isleader {
		return nil
	}

	if v == nil {
		err = errNilVaa

		return
	}

	// at end of function check if logging is needed too.
	defer func() {
		if err == nil {
			return
		}

		t.logger.Error("issue sending VAAv1 to others", zap.Error(err))
	}()

	if t.gst == nil {
		err = errNilGuardianSetState

		return
	}

	dgst := digest{}
	copy(dgst[:], v.SigningDigest().Bytes())

	if v.Version != vaa.VaaVersion1 {
		// not an error. but will not accept.
		return nil
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
		t.logger.Info("informed all guardians on vaav1",
			zap.String("chainID", v.EmitterChain.String()),
			zap.String("digest", fmt.Sprintf("%x", v.SigningDigest())),
		)
	default:
		err = fmt.Errorf("network output channel buffer is full")
	}

	return nil
}

var errNilGuardianSet = fmt.Errorf("nil guardian set")
var errNotVaaV1 = fmt.Errorf("not a v1 VAA")

// handleUnicastVaaV1 expects to receive valid Vaav1 messages.
// If the VAA is valid, it will trigger the TSS signing protocol too for that VAA (beginTSSSign, will ensure double signing for the same digest).
func (t *Engine) handleUnicastVaaV1(v *tsscommv1.Unicast_Vaav1) error {
	if t == nil {
		return errNilTssEngine
	}

	if t.started.Load() != started {
		return errTssEngineNotStarted
	}

	if t.gst == nil {
		return errNilGuardianSetState
	}

	gs := t.gst.Get()
	if gs == nil {
		return errNilGuardianSet
	}

	if v == nil || v.Vaav1 == nil {
		return errNilVaa
	}

	newVaa, err := vaa.Unmarshal(v.Vaav1.Marshaled)
	if err != nil {
		return fmt.Errorf("unmarshal err: %w", err)
	}

	dgst := digest{}
	copy(dgst[:], newVaa.SigningDigest().Bytes())

	if newVaa.Version != vaa.VaaVersion1 {
		return errNotVaaV1
	}

	if err := newVaa.Verify(gs.Keys); err != nil {
		return err
	}

	return t.beginTSSSign(dgst[:], newVaa.EmitterChain, newVaa.ConsistencyLevel, signingMeta{isFromVaav1: true})
}
