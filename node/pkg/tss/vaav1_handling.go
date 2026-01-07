package tss

import (
	"context"
	"errors"
	"fmt"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	errNilGuardianSetState      = errors.New("tss' guardianSetState nil")
	errNilVaa                   = errors.New("nil VAAv1")
	errNilGuardianSet           = errors.New("nil guardian set")
	errNotVaaV1                 = errors.New("not a VAAv1")
	errNetworkOutputChannelFull = errors.New("network output channel buffer is full")
	errCouldntGossipVaa         = errors.New("couldn't gossip VAAv1")
	errNilGuardianSigner        = errors.New("guardianSigner is nil")
)

func (s *signerClient) WitnessNewVaaV1(ctx context.Context, v *vaa.VAA) error {
	if s == nil {
		return ErrSignerClientNil
	}

	if v == nil {
		return errNilVaa
	}

	if !s.vaaData.isLeader {
		return nil
	}

	if s.vaaData.gst == nil {
		return errNilGuardianSetState
	}

	if s.vaaData.GuardianSigner == nil {
		return errNilGuardianSigner
	}

	if v.Version != vaa.VaaVersion1 {
		// not an error. but will not accept.
		return nil
	}

	gs := s.vaaData.gst.Get()
	if err := v.Verify(gs.Keys); err != nil {
		return nil // won't send invalid VAAs.
	}

	bts, err := v.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal VAA: %w", err)
	}

	// signing the content the leader will gossip.
	sig, err := s.vaaData.GuardianSigner.Sign(ctx, bts)
	if err != nil {
		return fmt.Errorf("failed to sign VAA: %w", err)
	}
}

// handleUnicastVaaV1 expects to receive valid Vaav1 messages.
// If the VAA is valid, it will trigger the TSS signing protocol too for that VAA (beginTSSSign, will ensure double signing for the same digest).
// func (t *Engine) handleUnicastVaaV1(v *tsscommv1.Unicast_Vaav1) error {
// if t == nil {
// 	return errNilTssEngine
// }

// if t.started.Load() != started {
// 	return errTssEngineNotStarted
// }

// if t.gst == nil {
// 	return errNilGuardianSetState
// }

// gs := t.gst.Get()
// if gs == nil {
// 	return errNilGuardianSet
// }

// if v == nil || v.Vaav1 == nil {
// 	return errNilVaa
// }

// newVaa, err := vaa.Unmarshal(v.Vaav1.Marshaled)
// if err != nil {
// 	return fmt.Errorf("unmarshal err: %w", err)
// }

// dgst := digest{}
// copy(dgst[:], newVaa.SigningDigest().Bytes())

// if newVaa.Version != vaa.VaaVersion1 {
// 	return errNotVaaV1
// }

// if err := newVaa.Verify(gs.Keys); err != nil {
// 	return err
// }

// signatureMeta := signingMeta{
// 	isFromVaav1:   true,
// 	verifiedVAAv1: newVaa,
// }

// }
