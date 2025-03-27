package tss

import (
	"time"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/tss-lib/v2/common"
	"github.com/xlabs/tss-lib/v2/ecdsa/party"
	"github.com/xlabs/tss-lib/v2/tss"
	"go.uber.org/zap"
)

// This file represents tracking mechanism for signatures, their trackingID, and who is currently working on them.
// It provide the Engine with information, for instance whether if the guardian saw the digest already.
//
// ftCommand represents commands that reach the ftTracker.
// to become ftCommand, the struct must implement the apply(*Engine, *ftTracker) method.
//
// the commands include signCommand, prepareToSignCommand, and reportProblemCommand.
//   - signCommand is used to inform the ftTracker that a guardian saw a digest, and what related information
//     it has about the digest.
//   - prepareToSignCommand is used to know which guardians aren't to be used in the protocol for
//     specific chainID.
type ftCommand interface {
	apply(*Engine, *ftTracker)
}

type trackidStr string

type signCommand struct {
	SigningInfo       *party.SigningInfo
	passedToFP        bool
	digestconsistancy uint8
	signingMeta       signingMeta
}

type SigEndCommand struct {
	*common.TrackingID // either from error, or from success.
}

type sigPreparationInfo struct {
	alreadyStartedSigningTrackingIDs map[trackidStr]bool
}

// Used to know which guardians aren't to be used in the protocol for specific chainID.
type prepareToSignCommand struct {
	ChainID vaa.ChainID
	Digest  party.Digest

	reply chan sigPreparationInfo
}

type tackingIDContext struct {
	sawProtocolMessagesFrom map[strPartyId]bool
	alreadyPassedTIDtoFP    bool
}

// the signatureState struct is used to keep track of a signature.
type signatureState struct {
	chain vaa.ChainID // blockchain the message relates to (e.g. Ethereum, Solana, etc).

	digest party.Digest
	// States whether the guardian saw the digest and forwarded it to the engine to be signed by TSS.
	approvedToSign bool

	// each trackingId is a unique attempt to sign a message.
	trackidContext map[trackidStr]*tackingIDContext

	// consistansy states the level of finality floats over the digest.
	// Bad consistancy means the digest is not final, and even might not be seen by all guardians.
	digestconsistancy uint8
	isFromVaav1       bool

	beginTime time.Time // used to do cleanups.
}

type keyAndTTL struct {
	key sigKey
	ttl time.Time
}

type ftChainContext struct {
	timeToRevive                time.Time                  // the time this party is expected to come back and be part of the protocol again.
	liveSigsWaitingForThisParty map[sigKey]*signatureState // sigs that once the revive time expire should be retried.
}

// Describes a specfic party's data in terms of fault tolerance.
type ftParty struct {
	partyID        *tss.PartyID
	ftChainContext map[vaa.ChainID]*ftChainContext
}

type ftTracker struct {
	ttlKeys []keyAndTTL

	sigsState      map[sigKey]*signatureState
	chainIdsToSigs map[vaa.ChainID]map[sigKey]*signatureState

	membersData map[strPartyId]*ftParty
}

func newChainContext() *ftChainContext {
	return &ftChainContext{
		// ensuring the first time we see this party, we don't assume it's down.
		timeToRevive: time.Time{},

		liveSigsWaitingForThisParty: map[sigKey]*signatureState{},
	}
}

// a single threaded env, that inspects incoming signatures request, message deliveries etc.
func (t *Engine) ftTracker() {
	f := &ftTracker{
		sigsState:   make(map[sigKey]*signatureState),
		membersData: make(map[strPartyId]*ftParty),

		chainIdsToSigs: map[vaa.ChainID]map[sigKey]*signatureState{},
	}

	for _, pid := range t.GuardianStorage.Guardians.partyIds {
		strPid := strPartyId(partyIdToString(pid))
		f.membersData[strPid] = &ftParty{
			partyID:        pid,
			ftChainContext: map[vaa.ChainID]*ftChainContext{},
		}
	}

	maxttl := t.GuardianStorage.maxSignerTTL()

	ticker := time.NewTicker(maxttl)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case cmd := <-t.ftCommandChan:
			cmd.apply(t, f)

			if len(f.ttlKeys) > sigStateRateLimit {
				f.cleanup(t, maxttl)
			}

		case <-ticker.C:
			f.cleanup(t, maxttl)
		}
	}
}

func (f *ftTracker) cleanup(t *Engine, maxttl time.Duration) {
	now := time.Now()

	toRemove := []sigKey{}

	if len(f.ttlKeys) > sigStateRateLimit {
		diff := len(f.ttlKeys) - sigStateRateLimit

		toRemove = make([]sigKey, diff)

		for i, keyAndTime := range f.ttlKeys[:diff] {
			toRemove[i] = keyAndTime.key
		}

		f.ttlKeys = f.ttlKeys[diff:] // remove the first diff elements.

		t.logger.Warn("ftTracker's limit reached, removing the oldest stored signature states", zap.Int("amount", diff))
	}

	cutoff := 0

	for i, keyAndTtl := range f.ttlKeys {
		if now.Sub(keyAndTtl.ttl) >= maxttl {
			toRemove = append(toRemove, keyAndTtl.key)
		} else {
			cutoff = i

			break
		}
	}

	f.ttlKeys = f.ttlKeys[cutoff:] // remove the first cutoff elements.

	for _, key := range toRemove {
		sigState, ok := f.sigsState[key]
		if !ok {
			continue
		}

		f.remove(sigState)
	}

}

func (f *ftTracker) remove(sigState *signatureState) {
	if sigState == nil {
		return
	}

	key := intoSigKey(sigState.digest, sigState.chain)

	for _, m := range f.membersData {
		if chainData, ok := m.ftChainContext[sigState.chain]; ok {
			delete(chainData.liveSigsWaitingForThisParty, key)
		}
	}

	chn, ok := f.chainIdsToSigs[sigState.chain]
	if ok {
		delete(chn, key)
	}

	delete(f.sigsState, key)
}

func (cmd *prepareToSignCommand) apply(t *Engine, f *ftTracker) {
	if cmd.reply == nil {
		t.logger.Error("reply channel is nil")

		return
	}

	reply := sigPreparationInfo{
		alreadyStartedSigningTrackingIDs: map[trackidStr]bool{},
	}

	sigKey := intoSigKey(cmd.Digest, cmd.ChainID)

	sigState, ok := f.sigsState[sigKey]
	if ok {
		for tidStr, ctx := range sigState.trackidContext {
			if ctx.alreadyPassedTIDtoFP {
				reply.alreadyStartedSigningTrackingIDs[tidStr] = true
			}
		}
	}

	if err := intoChannelOrDone(t.ctx, cmd.reply, reply); err != nil {
		t.logger.Error("error on telling on inactive guardians on specific chain", zap.Error(err))
	}

	close(cmd.reply)
}

// This changes the signatureState and sets it as Seen/Started/Approved to sign.
// As a result, once the alertHeap expires, it will not report a problem after
// f+1 guardians started signing since this guardian started as well.
func (cmd *signCommand) apply(t *Engine, f *ftTracker) {
	tid := cmd.SigningInfo.TrackingID
	if tid == nil {
		t.logger.Error("signCommand: tracking id is nil")

		return
	}

	dgst := party.Digest{}
	copy(dgst[:], tid.Digest[:])

	chain := extractChainIDFromTrackingID(tid)

	state, ok := f.sigsState[intoSigKey(dgst, chain)]
	if !ok {
		state = f.setNewSigState(dgst, chain, time.Now())

		state.digestconsistancy = cmd.digestconsistancy
		state.isFromVaav1 = cmd.signingMeta.isFromVaav1 // this is only interesting in case the guardian first saw this message via VAAv1.
	}

	state.approvedToSign = true

	for _, pid := range cmd.SigningInfo.SigningCommittee {
		m, ok := f.membersData[strPartyId(partyIdToString(pid))]
		if !ok {
			t.logger.Error("signCommand: party not found in the members data")

			continue
		}

		chainData, ok := m.ftChainContext[state.chain]
		if !ok {
			chainData = newChainContext()
			m.ftChainContext[state.chain] = chainData
		}

		chainData.liveSigsWaitingForThisParty[intoSigKey(dgst, state.chain)] = state
	}

	// if this guardian has request a signature for this TID, then we store it to ensure it doesn't attempt to sign again later.
	if cmd.passedToFP {
		tidStr := trackidStr(tid.ToString())

		tidData, ok := state.trackidContext[tidStr]
		if !ok {
			tidData = &tackingIDContext{
				sawProtocolMessagesFrom: map[strPartyId]bool{},
				alreadyPassedTIDtoFP:    false,
			}

			state.trackidContext[tidStr] = tidData
		}

		tidData.alreadyPassedTIDtoFP = true
	}
}

func (cmd *SigEndCommand) apply(t *Engine, f *ftTracker) {
	dgst := party.Digest{}
	copy(dgst[:], cmd.Digest[:])

	chain := extractChainIDFromTrackingID(cmd.TrackingID)
	key := intoSigKey(dgst, chain)

	if sigstate, ok := f.sigsState[key]; ok {
		f.remove(sigstate)
	}
}

func (f *ftTracker) setNewSigState(digest party.Digest, chain vaa.ChainID, alertTime time.Time) *signatureState {
	state := &signatureState{
		chain:          chain,
		digest:         digest,
		approvedToSign: false,
		trackidContext: map[trackidStr]*tackingIDContext{},
		beginTime:      time.Now(),
	}

	sigkey := intoSigKey(digest, chain)
	f.sigsState[sigkey] = state

	chn, ok := f.chainIdsToSigs[chain]
	if !ok {
		chn = map[sigKey]*signatureState{}
		f.chainIdsToSigs[chain] = chn
	}

	chn[sigkey] = state

	f.ttlKeys = append(f.ttlKeys, keyAndTTL{key: sigkey, ttl: alertTime})

	return state
}
