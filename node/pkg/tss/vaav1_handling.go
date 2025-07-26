package tss

import (
	"errors"
	"fmt"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

var errNilGuardianSetState = errors.New("tss' guardianSetState nil")
var errNilVaa = errors.New("nil VAA")
var errNetworkOutputChannelFull = errors.New("network output channel buffer is full")

// Assumes valid VAAs only.
func (t *Engine) WitnessNewVaa(v *vaa.VAA) error {
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
		return errNilVaa
	}

	if t.gst == nil {
		t.logger.Error("issue sending VAAv1 to others", zap.Error(errNilGuardianSetState))

		return errNilGuardianSetState
	}

	dgst := digest{}
	copy(dgst[:], v.SigningDigest().Bytes())

	if v.Version != vaa.VaaVersion1 {
		// not an error. but will not accept.
		return nil
	}

	gs := t.gst.Get()
	if err := v.Verify(gs.Keys); err != nil {
		return nil // won't send invalid VAAs.
	}

	bts, err := v.Marshal()
	if err != nil {
		err := fmt.Errorf("failed to marshal VAA: %w", err)

		t.logger.Error("issue sending VAAv1 to others", zap.Error(err))
		return err
	}

	send := Unicast{
		Unicast: &tsscommv1.Unicast{
			Content: &tsscommv1.Unicast_Vaav1{
				Vaav1: &tsscommv1.VaaV1Info{
					Marshaled: bts,
				},
			},
		},
		Receipients: t.GuardianStorage.Identities, // sending to all guardians.
	}

	select {
	case t.messageOutChan <- &send:
		t.logger.Info("informed all guardians on vaav1",
			zap.String("chainID", v.EmitterChain.String()),
			zap.String("digest", fmt.Sprintf("%x", v.SigningDigest())),
		)
	default:
		t.logger.Error("issue sending VAAv1 to others", zap.Error(errNetworkOutputChannelFull))
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

	signatureMeta := signingMeta{
		isFromVaav1:   true,
		verifiedVAAv1: newVaa,
	}

	return t.beginTSSSign(dgst[:], newVaa.EmitterChain, newVaa.ConsistencyLevel, signatureMeta)
}

var errCouldNotMapAllVaaV1Signers = fmt.Errorf("could not map all VAAv1 signers to senderIndexes")

func (t *Engine) translateVaaV1Signers(v *vaa.VAA) (map[SenderIndex]*Identity, error) {
	gs := t.gst.Get()
	// translating the VAAv1 signers to senderIndexes to use.
	signersID := make(map[SenderIndex]*Identity, len(gs.Keys))
	for _, s := range v.Signatures {
		if int(s.Index) >= len(gs.Keys) {
			// shouldn't happen, since the signature was verified. but just in case.
			return nil, fmt.Errorf("signature index %d out of bounds for guardian set size %d", s.Index, len(gs.Keys))
		}

		id, err := t.GuardianStorage.fetchIdentityFromVaav1Pubkey(gs.Keys[s.Index])
		if err != nil {
			continue
		}

		signersID[id.CommunicationIndex] = id
	}

	if len(signersID) != len(v.Signatures) {
		// make into an error:
		return nil, errCouldNotMapAllVaaV1Signers

	}

	return signersID, nil
}
