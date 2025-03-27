package tss

import (
	"time"

	"github.com/certusone/wormhole/node/pkg/tss/internal"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/tss-lib/v2/common"
	"github.com/xlabs/tss-lib/v2/ecdsa/party"
	"github.com/xlabs/tss-lib/v2/tss"
	"go.uber.org/zap"
)

// The code in this file improves the fault tolerance of the TSS Engine
// such that honest-but-missing-current-blocks
// failures do not cause the protocol to halt on a signature.
// For instance, nodes that haven’t upgraded their binaries to match
// the most recent code may not receive new blocks or transactions, and as a
// result will refuse to participate in signing.
//
// While maintaining strict security guarantees, the fault tolerance mechanism
// will only deal with the case where a guardian is behind the network.
// That is, it relies on the guardian to report that it is behind the network.
//
// The guardian (Denote by G), that participates in the TSS protocol will see messages sent
// to and from other guardians (due to it being a part in the Reliable Broadcast protocol).
// As a result, G will witness the digests that other guardians started the signing protocol for.
// In case G witnesses f+1 other guardians working on a digest that it hasn't seen yet, it will assume
// it is behind the network, and will generate a problem report and broadcast it to the other guardians.
// That is, seeing f+1 other guardians guarantees that at least one honest guardian saw the message,
// and thus G can assume it is behind the network.
//
// TODO: Support crash-failures.
//
// FT Process: each guardian keeps track of the digest, signatures and trackingIDs it saw using the
// ftTracker and its goroutine. The ftTracker receives ftCommands from the Engine and update its state
// according to these commands. It will output a problem message to the other guardians if it detects
// that it is behind the network.
// below is the ftCommand interface and the commands that implement it.
//
// ftCommand represents commands that reach the ftTracker.
// to become ftCommand, the struct must implement the apply(*Engine, *ftTracker) method.
//
// the commands include signCommand, prepareToSignCommand, and reportProblemCommand.
//   - signCommand is used to inform the ftTracker that a guardian saw a digest, and what related information
//     it has about the digest.
//   - prepareToSignCommand is used to know which guardians aren't to be used in the protocol for
//     specific chainID.
//   - reportProblemCommand is used to deliver a problem message from another
//     guardian (after it was accepted by the reliable-broadcast protocol).
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

type inactives struct {
	partyIDs []*tss.PartyID

	downtimeEnding []*tss.PartyID
}

// getFaultiesLists returns a list of all relevant faulties lists.
// will drop from faulties each guardian that is in the downtimeEnding list.
// that is, generates a list of size (n choose n-1) + 1.
func (i *inactives) getFaultiesLists() [][]*tss.PartyID {
	listOfAllFaultiesLists := make([][]*tss.PartyID, 0, len(i.downtimeEnding)+1)
	for _, v := range append([]*tss.PartyID{nil}, i.downtimeEnding...) {
		listOfAllFaultiesLists = append(listOfAllFaultiesLists, i.getFaultiesWithout(v))
	}

	return listOfAllFaultiesLists
}

func (i *inactives) getFaultiesWithout(pid *tss.PartyID) []*tss.PartyID {
	if pid == nil {
		return i.partyIDs
	}

	if len(i.partyIDs) == 0 {
		return i.partyIDs
	}

	faulties := make([]*tss.PartyID, 0, len(i.partyIDs)-1)

	for _, p := range i.partyIDs {
		if equalPartyIds(p, pid) {
			continue
		}

		faulties = append(faulties, p)
	}

	return faulties
}

type sigPreparationInfo struct {
	inactives                        inactives
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
// the same struct is held by two different data structures:
//  1. a map so we can access and update the sigState easily.
//  2. a timedHeap that orders the signatures by the time they should be checked.
//     once the timedHeap timer expires we inspect the top (*sigState) and decide whether we should report
//     a problem to the other guardians, increase the timeout for this signature and check again later, or
//     drop the timer since we've seen the message.
type signatureState struct {
	chain vaa.ChainID // blockchain the message relates to (e.g. Ethereum, Solana, etc).

	digest party.Digest
	// States whether the guardian saw the digest and forwarded it to the engine to be signed by TSS.
	approvedToSign bool

	// each trackingId is a unique attempt to sign a message.
	// Once one of the trackidStr saw f+1 guardians and we haven't seent the digest yet, we can assume
	// we are behind the network and we should inform the others.
	trackidContext map[trackidStr]*tackingIDContext

	// consistansy states the level of finality floats over the digest.
	// Bad consistancy means the digest is not final, and even might not be seen by all guardians.
	digestconsistancy uint8
	isFromVaav1       bool

	alertTime time.Time

	beginTime time.Time // used to do cleanups.

}

// GetEndTime is in capital to support the HasTTl interface.
func (s *signatureState) GetEndTime() time.Time {
	return s.alertTime
}

type endDownTimeAlert struct {
	partyID   *tss.PartyID
	chain     vaa.ChainID
	alertTime time.Time
}

func (e *endDownTimeAlert) GetEndTime() time.Time {
	return e.alertTime
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
	ttlKeys        []keyAndTTL
	sigAlerts      internal.Ttlheap[*signatureState]
	sigsState      map[sigKey]*signatureState // TODO: sigState should include the chainID too, otherwise we might have two digest with  two differet chainIDs
	chainIdsToSigs map[vaa.ChainID]map[sigKey]*signatureState

	// for starters, we assume any fault is on all chains.
	membersData            map[strPartyId]*ftParty
	downtimeAlerts         internal.Ttlheap[*endDownTimeAlert]
	chainsWithNoSelfReport map[vaa.ChainID]bool
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
		sigAlerts:      internal.NewTtlHeap[*signatureState](),
		sigsState:      make(map[sigKey]*signatureState),
		membersData:    make(map[strPartyId]*ftParty),
		downtimeAlerts: internal.NewTtlHeap[*endDownTimeAlert](),

		chainIdsToSigs: map[vaa.ChainID]map[sigKey]*signatureState{},

		chainsWithNoSelfReport: make(map[vaa.ChainID]bool),
	}

	for _, pid := range t.GuardianStorage.Guardians.partyIds {
		strPid := strPartyId(partyIdToString(pid))
		f.membersData[strPid] = &ftParty{
			partyID:        pid,
			ftChainContext: map[vaa.ChainID]*ftChainContext{},
		}
	}

	for _, cid := range t.GuardianStorage.Configurations.ChainsWithNoSelfReport {
		f.chainsWithNoSelfReport[vaa.ChainID(cid)] = true
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
		alertTime:      alertTime,
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

// get the maximal amount of guardians that saw the digest and started signing.
func (s *signatureState) maxGuardianVotes() int {
	maxVotesSeen := 0

	for _, tidData := range s.trackidContext {
		if len(tidData.sawProtocolMessagesFrom) > maxVotesSeen {
			maxVotesSeen = len(tidData.sawProtocolMessagesFrom)
		}
	}

	return maxVotesSeen
}
