package tss

import (
	"encoding/binary"
	"fmt"
	"time"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/yossigi/tss-lib/v2/common"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/tss"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
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
// the commands include signCommand, deliveryCommand, getInactiveGuardiansCommand, and reportProblemCommand.
//   - signCommand is used to inform the ftTracker that a guardian saw a digest, and what related information
//     it has about the digest.
//   - deliveryCommand is used to inform the ftTracker that a guardian saw a message and forwarded it
//     to the fullParty.
//   - getInactiveGuardiansCommand is used to know which guardians aren't to be used in the protocol for
//     specific chainID.
//   - reportProblemCommand is used to deliver a problem message from another
//     guardian (after it was accepted by the reliable-broadcast protocol).
type ftCommand interface {
	apply(*Engine, *ftTracker)
}

type trackidStr string

type signCommand struct {
	SigningInfo *party.SigningInfo
}

type deliveryCommand struct {
	parsedMsg tss.Message
	from      *tss.PartyID
}

type SigEndCommand struct {
	*common.TrackingID // either from error, or from success.
}

type reportProblemCommand struct {
	parsedProblem
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

// Used to know which guardians aren't to be used in the protocol for specific chainID.
type getInactiveGuardiansCommand struct {
	digest  party.Digest
	ChainID vaa.ChainID
	reply   chan inactives
}

type tackingIDContext struct {
	sawProtocolMessagesFrom map[strPartyId]bool
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

// sigStateKey contains two main parts of common.TrackID: the digest and the chainID.
// it doesan't contain the faulty bitmap since we want to point to the same signature even if the faulty bitmap changes.
type sigStateKey [party.DigestSize + auxiliaryDataSize]byte

type ftChainContext struct {
	timeToRevive                time.Time                       // the time this party is expected to come back and be part of the protocol again.
	liveSigsWaitingForThisParty map[sigStateKey]*signatureState // sigs that once the revive time expire should be retried.
}

// Describes a specfic party's data in terms of fault tolerance.
type ftParty struct {
	partyID        *tss.PartyID
	ftChainContext map[vaa.ChainID]*ftChainContext
}

type ftTracker struct {
	sigAlerts      internal.Ttlheap[*signatureState]
	sigsState      map[sigStateKey]*signatureState // TODO: sigState should include the chainID too, otherwise we might have two digest with  two differet chainIDs
	chainIdsToSigs map[vaa.ChainID]map[sigStateKey]*signatureState

	// for starters, we assume any fault is on all chains.
	membersData    map[strPartyId]*ftParty
	downtimeAlerts internal.Ttlheap[*endDownTimeAlert]
}

func newChainContext() *ftChainContext {
	return &ftChainContext{
		// ensuring the first time we see this party, we don't assume it's down.
		timeToRevive: time.Time{},

		liveSigsWaitingForThisParty: map[sigStateKey]*signatureState{},
	}
}

// a single threaded env, that inspects incoming signatures request, message deliveries etc.
func (t *Engine) ftTracker() {
	f := &ftTracker{
		sigAlerts:      internal.NewTtlHeap[*signatureState](),
		sigsState:      make(map[sigStateKey]*signatureState),
		membersData:    make(map[strPartyId]*ftParty),
		downtimeAlerts: internal.NewTtlHeap[*endDownTimeAlert](),
		chainIdsToSigs: map[vaa.ChainID]map[sigStateKey]*signatureState{},
	}

	for _, pid := range t.GuardianStorage.Guardians {
		strPid := strPartyId(partyIdToString(pid))
		f.membersData[strPid] = &ftParty{
			partyID:        pid,
			ftChainContext: map[vaa.ChainID]*ftChainContext{},
		}
	}

	maxttl := t.GuardianStorage.maxSignerTTL()

	ticker := time.NewTicker(maxttl)
	defer ticker.Stop()

	debugTicker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-debugTicker.C:
			t.logger.Info("ftTracker tick")

		case cmd := <-t.ftCommandChan:
			cmd.apply(t, f)
		case <-f.sigAlerts.WaitOnTimer():
			f.inspectAlertHeapsTop(t)

		case <-f.downtimeAlerts.WaitOnTimer():
			continue
			f.inspectDowntimeAlertHeapsTop(t)

		case <-ticker.C:
			f.cleanup(maxttl)
		}
	}
}

func (f *ftTracker) cleanup(maxttl time.Duration) {
	now := time.Now()

	for _, sigState := range f.sigsState {
		if now.Sub(sigState.beginTime) < maxttl {
			continue
		}

		f.remove(sigState)
	}
}

func (f *ftTracker) remove(sigState *signatureState) {
	if sigState == nil {
		return
	}

	key := intoSigStateKey(sigState.digest, sigState.chain)
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

func intoSigStateKey(dgst party.Digest, chain vaa.ChainID) sigStateKey {
	var key sigStateKey
	copy(key[:party.DigestSize], dgst[:])
	copy(key[party.DigestSize:], chainIDToBytes(chain))

	return key
}

func (cmd *reportProblemCommand) deteministicJitter(maxjitter time.Duration) time.Duration {
	bts, err := cmd.serialize()
	if err != nil {
		return 0
	}

	jitterBytes := hash(bts)
	nanoJitter := binary.BigEndian.Uint64(jitterBytes[:8])
	return (time.Duration(nanoJitter).Abs() % maxjitter).Abs()
}

func (cmd *reportProblemCommand) apply(t *Engine, f *ftTracker) {
	// the incoming command is assumed to be from a reliable-broadcast protocol and to be valid:
	// not too old (less than maxHeartbeatInterval), signed by the correct party, etc.
	pid := protoToPartyId(cmd.issuer)

	m := f.membersData[strPartyId(partyIdToString(pid))]

	jitter := cmd.deteministicJitter(t.GuardianStorage.maxJitter)

	now := time.Now()
	// Adds some deterministic jitter to the time to revive, so reportProblemCommand messages that arrive at the same time
	// won't have the same revival time.
	reviveTime := now.Add(t.GuardianStorage.guardianDownTime + jitter)

	chainID := vaa.ChainID(cmd.ChainID)
	chainData, ok := m.ftChainContext[chainID]
	if !ok {
		chainData = newChainContext()
		m.ftChainContext[chainID] = chainData
	}

	// we update the revival time only if the revival time had passed
	if now.After(chainData.timeToRevive) {
		chainData.timeToRevive = reviveTime
		f.downtimeAlerts.Enqueue(&endDownTimeAlert{
			partyID:   pid,
			chain:     chainID,
			alertTime: reviveTime,
		})
	}

	t.logger.Info("received a problem message from guardian",
		zap.String("problem issuer", cmd.issuer.Id),
		zap.String("chainID", vaa.ChainID(cmd.ChainID).String()),
		zap.Int("number of inactives on chain", len(f.getIncatives(vaa.ChainID(cmd.ChainID)).partyIDs)),
	)

	// if the problem is about this guardian, then there is no reason to retry the sigs since it won't
	// be part of the protocol.
	// we do let this guardian know that it is faulty and it's time so it can collect correct data
	// from signingInfo, which should be synchronised with the other guardians (if it attempts to sign later sigs).
	if equalPartyIds(pid, t.Self) {
		return
	}

	retryNow := chainData.liveSigsWaitingForThisParty
	chainData.liveSigsWaitingForThisParty = map[sigStateKey]*signatureState{} // clear the live sigs.

	go func() {
		for _, sig := range retryNow {
			// TODO: maybe find something smarter to do here.
			if err := t.BeginAsyncThresholdSigningProtocol(sig.digest[:], chainID); err != nil {
				t.logger.Error("failed to retry a signature", zap.Error(err))
			}
		}
	}()
}

func (cmd *getInactiveGuardiansCommand) apply(t *Engine, f *ftTracker) {
	if cmd.reply == nil {
		t.logger.Error("reply channel is nil")
		return
	}

	reply := f.getIncatives(cmd.ChainID)

	if err := intoChannelOrDone(t.ctx, cmd.reply, reply); err != nil {
		t.logger.Error("error on telling on inactive guardians on specific chain", zap.Error(err))
	}

	close(cmd.reply)
}

func (f *ftTracker) getIncatives(chainID vaa.ChainID) inactives {
	reply := inactives{}

	for _, m := range f.membersData {
		chainData, ok := m.ftChainContext[chainID]
		if !ok {
			chainData = newChainContext()
			m.ftChainContext[chainID] = chainData

			continue // never seen before, so it's active.
		}

		diff := time.Until(chainData.timeToRevive)
		//  |revive_time - now| < synchronsingInterval, then its time to revive comes soon.
		if diff.Abs() < synchronsingInterval {
			reply.downtimeEnding = append(reply.downtimeEnding, m.partyID)
		}

		//there is time to wait until the guardian is back, so it's inactive.
		if diff > 0 {
			reply.partyIDs = append(reply.partyIDs, m.partyID)
		}
	}

	return reply
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
	// TODO: Ensure the digest contains the auxilaryData. otherwise, there can be two signatures witth the same digest? I doubt it.
	state, ok := f.sigsState[intoSigStateKey(dgst, chain)]
	if !ok {
		state = f.setNewSigState(dgst, chain, time.Now())
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

		chainData.liveSigsWaitingForThisParty[intoSigStateKey(dgst, state.chain)] = state
	}
}

func (cmd *SigEndCommand) apply(t *Engine, f *ftTracker) {
	dgst := party.Digest{}
	copy(dgst[:], cmd.Digest[:])

	chain := extractChainIDFromTrackingID(cmd.TrackingID)
	key := intoSigStateKey(dgst, chain)

	if sigstate, ok := f.sigsState[key]; ok {
		f.remove(sigstate)
	}
}

func (cmd *deliveryCommand) apply(t *Engine, f *ftTracker) {
	wmsg := cmd.parsedMsg.WireMsg()
	if wmsg == nil {
		t.logger.Error("deliveryCommand: wire message is nil")
		return
	}

	tid := wmsg.GetTrackingID()
	if tid == nil {
		t.logger.Error("deliveryCommand: tracking id is nil")
		return
	}

	dgst := party.Digest{}
	copy(dgst[:], tid.GetDigest())

	chain := extractChainIDFromTrackingID(tid)

	state, ok := f.sigsState[intoSigStateKey(dgst, chain)]
	if !ok {
		alertTime := time.Now().Add(t.GuardianStorage.DelayGraceTime)
		state = f.setNewSigState(dgst, chain, alertTime)

		// Since this is a delivery and not a sign command, we add this to the alert heap.
		f.sigAlerts.Enqueue(state)
	}

	tidData, ok := state.trackidContext[trackidStr(tid.ToString())]
	if !ok {
		tidData = &tackingIDContext{
			sawProtocolMessagesFrom: map[strPartyId]bool{},
		}

		state.trackidContext[trackidStr(tid.ToString())] = tidData
	}

	tidData.sawProtocolMessagesFrom[strPartyId(partyIdToString(cmd.from))] = true
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

	sigkey := intoSigStateKey(digest, chain)
	f.sigsState[sigkey] = state

	chn, ok := f.chainIdsToSigs[chain]
	if !ok {
		chn = map[sigStateKey]*signatureState{}
		f.chainIdsToSigs[chain] = chn
	}
	chn[sigkey] = state

	return state
}
func (f *ftTracker) inspectAlertHeapsTop(t *Engine) {
	sigState := f.sigAlerts.Dequeue()

	if sigState.approvedToSign {
		return
	}

	if _, exists := f.sigsState[intoSigStateKey(sigState.digest, sigState.chain)]; !exists {
		// sig is removed (either old, or finished signing), and we don't need to do anything.
		return
	}

	// At least one honest guardian saw the message, but I didn't (I'm probablt behined the network).
	if sigState.maxGuardianVotes() >= t.GuardianStorage.getMaxExpectedFaults()+1 {
		t.logger.Info("Hadn't seen digest to sign yet, but f+1 attempted to sign it, reporting issue",
			zap.String("chainID", sigState.chain.String()),
			zap.Duration("Time since signature started", time.Since(sigState.beginTime)),
			zap.Int("Number of guardians that saw the message", sigState.maxGuardianVotes()),
		)

		t.reportProblem(sigState.chain)

		return
	}

	// haven't seen the message, but not behind the network (yet).
	// increasing timeout for this signature.
	sigState.alertTime = time.Now().Add(t.DelayGraceTime / 2) // TODO: consider some logic on reducing the time.
	f.sigAlerts.Enqueue(sigState)
}

func (t *Engine) reportProblem(chain vaa.ChainID) {
	sm := &tsscommv1.SignedMessage{
		Content: &tsscommv1.SignedMessage_Problem{
			Problem: &tsscommv1.Problem{
				ChainID:     uint32(chain),
				IssuingTime: timestamppb.Now(),
			},
		},

		Sender:    partyIdToProto(t.Self),
		Signature: []byte{},
	}

	if err := t.sign(sm); err != nil {
		t.logger.Error("failed to report a problem to the other guardians", zap.Error(err))

		return
	}

	intoChannelOrDone[Sendable](t.ctx, t.messageOutChan, newEcho(sm, t.guardiansProtoIDs))
}

// get the maximal amount of guardians that saw the digest and started signing.
func (s *signatureState) maxGuardianVotes() int {
	max := 0
	for _, tidData := range s.trackidContext {
		if len(tidData.sawProtocolMessagesFrom) > max {
			max = len(tidData.sawProtocolMessagesFrom)
		}
	}

	return max
}

func (f *ftTracker) inspectDowntimeAlertHeapsTop(t *Engine) {
	alert := f.downtimeAlerts.Dequeue()
	if alert == nil {
		return
	}

	liveSigsInChain, ok := f.chainIdsToSigs[alert.chain]
	if !ok {
		return
	}

	// we don't have to change the revival time for this party, since it should be the same as the alert.
	// instead we start collecting the signatures that should be retried.

	inactives := f.getIncatives(alert.chain)

	allReleveantFaulties := inactives.getFaultiesLists()

	var toSign []*signatureState

	for _, sigState := range liveSigsInChain {
		if !sigState.approvedToSign {
			continue // no need to retry this signature.
		}

		for _, faulties := range allReleveantFaulties {
			// create signingTask and ask: am i one of the signers?
			info, err := t.fp.GetSigningInfo(makeSigningRequest(sigState.digest, faulties, sigState.chain))
			if err != nil {
				t.logger.Error(
					"couldn't retry signing digest",
					zap.Error(err),
					zap.String("digest", fmt.Sprintf("%x", sigState.digest)),
					zap.String("chainID", sigState.chain.String()),
					zap.Strings("faulties", getCommitteeIDs(faulties)),
				)

				continue
			}

			// this guardian is a signer once this server revives? if it is: retry the signature.
			if info.IsSigner {
				toSign = append(toSign, sigState)
			}
		}
	}

	// retry signatures...
	go func() {
		for _, sig := range toSign {
			if err := t.BeginAsyncThresholdSigningProtocol(sig.digest[:], sig.chain); err != nil {
				t.logger.Error("failed to retry a signature", zap.Error(err))
			}
		}
	}()
}
