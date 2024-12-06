package tss

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/yossigi/tss-lib/v2/common"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/ecdsa/signing"
	"github.com/yossigi/tss-lib/v2/tss"
	"go.uber.org/zap"
)

type logableError struct {
	cause      error
	trackingId *common.TrackingID
	round      signingRound
}

type set[T comparable] map[T]struct{}

type strDigest string
type strPartyId string

// activeSigCounter is a helper struct to keep track of active signatures.
// Each signature has a digest, and each guardian is allowed to be active
// for a certain number of signatures.
// a guardian is allowed to send how many messages it want per signature, but not allowed to
// participate in more than maxActiveSignaturesPerGuardian signatures at a time.
type activeSigCounter struct {
	mtx sync.RWMutex

	digestToGuardians map[strDigest]set[strPartyId]
	guardianToDigests map[strPartyId]set[strDigest]
}

func newSigCounter() activeSigCounter {
	return activeSigCounter{
		mtx: sync.RWMutex{},

		digestToGuardians: make(map[strDigest]set[strPartyId]),
		guardianToDigests: make(map[strPartyId]set[strDigest]),
	}
}

func idToString(id *tss.PartyID) strPartyId {
	return strPartyId(fmt.Sprintf("%s%x", id.Id, id.Key))
}

// Add adds a guardian to the counter for a given digest.
// returns false if this guardian is active for too many signatures ( > maxActiveSignaturesPerGuardian).
func (c *activeSigCounter) add(trackId *common.TrackingID, guardian *tss.PartyID, maxActiveSignaturesPerGuardian int) bool {
	if trackId == nil || guardian == nil {
		return false
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	if _, ok := c.digestToGuardians[strDigest(trackId.Digest)]; !ok {
		c.digestToGuardians[strDigest(trackId.Digest)] = make(set[strPartyId])
	}

	strPartyId := idToString(guardian)

	if _, ok := c.guardianToDigests[strPartyId]; !ok {
		c.guardianToDigests[strPartyId] = make(set[strDigest])
	}

	// if already an active signature for this guardian, then it doesn't count as an additional signature
	if _, ok := c.guardianToDigests[strPartyId][strDigest(trackId.Digest)]; ok {
		return true
	}

	// the guardian hasn't yet participated in this signing for the digest, we must ensure an additional signature is allowed
	if len(c.guardianToDigests[strPartyId])+1 > maxActiveSignaturesPerGuardian {
		return false
	}

	c.digestToGuardians[strDigest(trackId.Digest)][strPartyId] = struct{}{}
	c.guardianToDigests[strPartyId][strDigest(trackId.Digest)] = struct{}{}

	return true
}

func (c *activeSigCounter) remove(trackid *common.TrackingID) {
	if trackid == nil {
		return
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	guardians := c.digestToGuardians[strDigest(trackid.Digest)]
	delete(c.digestToGuardians, strDigest(trackid.Digest))

	for g := range guardians {
		delete(c.guardianToDigests[g], strDigest(trackid.Digest))
	}
}

func (l logableError) Error() string {
	if l.cause == nil {
		return ""
	}

	return l.cause.Error()
}

// Unwrap ensures logableError supports errors.Is and errors.As methods.
func (l logableError) Unwrap() error {
	return l.cause
}

func logErr(l *zap.Logger, err error) {
	if l == nil {
		return
	}

	if err == nil {
		return
	}

	informativeErr, ok := err.(logableError)
	if !ok {
		l.Error(err.Error())

		return
	}

	var zapFields []zap.Field
	if informativeErr.trackingId != nil {
		zapFields = append(zapFields, zap.String("trackingId", informativeErr.trackingId.ToString()))
	}

	if informativeErr.round != "" {
		zapFields = append(zapFields, zap.String("round", string(informativeErr.round)))
	}

	l.Error(informativeErr.Error(), zapFields...)
}

func equalPartyIds(a, b *tss.PartyID) bool {
	return a.Id == b.Id && string(a.Key) == string(b.Key)
}

func protoToPartyId(pid *tsscommv1.PartyId) *tss.PartyID {
	return &tss.PartyID{
		MessageWrapper_PartyID: &tss.MessageWrapper_PartyID{
			Id:      pid.Id,
			Moniker: pid.Moniker,
			Key:     pid.Key,
		},
		Index: int(pid.Index),
	}
}

func partyIdToProto(pid *tss.PartyID) *tsscommv1.PartyId {
	return &tsscommv1.PartyId{
		Id:      pid.Id,
		Moniker: pid.Moniker,
		Key:     pid.Key,
		Index:   uint32(pid.Index),
	}
}

func partyIdToString(guardian *tss.PartyID) string {
	return fmt.Sprintf("%s%x", guardian.Id, guardian.Key)
}

var (
	ErrEchoIsNil             = fmt.Errorf("echo is nil")
	ErrNoAuthenticationField = fmt.Errorf("SignedMessage doesn't contain an authentication field")
	ErrNilPartyId            = fmt.Errorf("party id is nil")
	ErrEmptyIDInPID          = fmt.Errorf("partyId identifier is empty")
	ErrEmptyKeyInPID         = fmt.Errorf("partyId doesn't contain a key")
	ErrSignedMessageIsNil    = fmt.Errorf("SignedMessage is nil")
	ErrNoContent             = fmt.Errorf("SignedMessage doesn't contain a content")
	ErrNilPayload            = fmt.Errorf("SignedMessage doesn't contain a payload")
	ErrMissingTimestamp      = fmt.Errorf("problem struct missing timestamp field")
)

func vaidateEchoCorrectForm(e *tsscommv1.Echo) error {
	if e == nil {
		return ErrEchoIsNil
	}

	m := e.Message
	if m == nil {
		return ErrSignedMessageIsNil
	}

	if err := validatePartIdProtoCorrectForm(m.Sender); err != nil {
		return fmt.Errorf("signedMessage sender pID error:%w", err)
	}

	switch v := m.Content.(type) {
	case *tsscommv1.SignedMessage_TssContent:
		if err := validateContentCorrectForm(v.TssContent); err != nil {
			return fmt.Errorf("signedMessage content error:%w", err)
		}
	case *tsscommv1.SignedMessage_Problem:
		if err := validateProblemCorrectForm(v.Problem); err != nil {
			return err
		}

		if time.Since(v.Problem.IssuingTime.AsTime()).Abs() > maxHeartbeatInterval {
			return fmt.Errorf("problem's timestamp is too old")
		}
	case nil:
		return ErrNoContent
	default:
		return fmt.Errorf("unknown content type: %T", v)
	}

	if len(m.Signature) == 0 {
		return ErrNoAuthenticationField
	}

	return nil
}

func validatePartIdProtoCorrectForm(p *tsscommv1.PartyId) error {
	if p == nil {
		return ErrNilPartyId
	}

	if p.Id == "" {
		return ErrEmptyIDInPID
	}

	if len(p.Key) == 0 {
		return ErrEmptyKeyInPID
	}

	return nil
}

func validateContentCorrectForm(m *tsscommv1.TssContent) error {
	if m == nil {
		return ErrNoContent
	}

	if m.Payload == nil {
		return ErrNilPayload
	}

	return nil
}

func validateProblemCorrectForm(p *tsscommv1.Problem) error {
	if p == nil {
		return ErrNoContent
	}

	if p.IssuingTime == nil {
		return ErrMissingTimestamp
	}

	return nil
}

type signingRound string

const (
	round1Message1 signingRound = "round1M1"
	round1Message2 signingRound = "round1M2"
	round2Message  signingRound = "round2"
	round3Message  signingRound = "round3"
	round4Message  signingRound = "round4"
	round5Message  signingRound = "round5"
	round6Message  signingRound = "round6"
	round7Message  signingRound = "round7"
	round8Message  signingRound = "round8"
	round9Message  signingRound = "round9"
)

var _intToRoundArr = []signingRound{
	"round1",
	round2Message,
	round3Message,
	round4Message,
	round5Message,
	round6Message,
	round7Message,
	round8Message,
	round9Message,
}

func intToRound(i int) signingRound {
	if i < 1 || i > 9 {
		return ""
	}

	return _intToRoundArr[i-1]
}

func getRound(m tss.ParsedMessage) (signingRound, error) {
	switch m.Content().(type) {
	case *signing.SignRound1Message1:
		return round1Message1, nil
	case *signing.SignRound1Message2:
		return round1Message2, nil
	case *signing.SignRound2Message:
		return round2Message, nil
	case *signing.SignRound3Message:
		return round3Message, nil
	case *signing.SignRound4Message:
		return round4Message, nil
	case *signing.SignRound5Message:
		return round5Message, nil
	case *signing.SignRound6Message:
		return round6Message, nil
	case *signing.SignRound7Message:
		return round7Message, nil
	case *signing.SignRound8Message:
		return round8Message, nil
	case *signing.SignRound9Message:
		return round9Message, nil
	default:
		return "", fmt.Errorf("unknown message type")
	}
}

func intoChannelOrDone[T any](ctx context.Context, c chan T, v T) error {
	select {
	case c <- v:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("error sending to channel: %w", ctx.Err())
	}
}

func outOfChannelOrDone[T any](ctx context.Context, c chan T) (T, error) {
	var v T
	select {
	case v = <-c:
		return v, nil
	case <-ctx.Done():
		return v, ctx.Err()
	}
}

func (st *GuardianStorage) validateTrackingIDForm(tid *common.TrackingID) error {
	if len(tid.Digest) != party.DigestSize {
		return fmt.Errorf("trackingID digest is not in correct size")
	}

	// checking that the byte array is the correct size
	if len(tid.PartiesState) < (len(st.Guardians)+7)/8 {
		return fmt.Errorf("trackingID partiesState is too short")
	}

	// TODO: expecting auxilaryData to be set.

	return nil
}

func getCommitteeIDs(pids []*tss.PartyID) []string {
	ids := make([]string, 0, len(pids))
	for _, v := range pids {
		ids = append(ids, v.Id)
	}

	return ids
}

func extractChainIDFromTrackingID(tid *common.TrackingID) vaa.ChainID {
	bts := [2]byte{}
	copy(bts[:], tid.AuxilaryData)
	return vaa.ChainID(binary.BigEndian.Uint16(bts[:]))
}

func chainIDToBytes(chainID vaa.ChainID) []byte {
	bts := [2]byte{}
	binary.BigEndian.PutUint16(bts[:], uint16(chainID))
	return bts[:]
}

// sigKey contains two main parts of common.TrackID: the digest and the chainID.
// it doesan't contain the faulty bitmap since we want to point to the same signature even if the faulty bitmap changes.
type sigKey [party.DigestSize + auxiliaryDataSize]byte

func intoSigKey(dgst party.Digest, chain vaa.ChainID) sigKey {
	var key sigKey
	copy(key[:party.DigestSize], dgst[:])
	copy(key[party.DigestSize:], chainIDToBytes(chain))

	return key
}

func trackingIdIntoSigKey(tid *common.TrackingID) sigKey {
	dgst := party.Digest{}
	copy(dgst[:], tid.Digest)

	return intoSigKey(dgst, extractChainIDFromTrackingID(tid))
}

func numFaultiesInTrackingID(tid *common.TrackingID, numGuardians int) int {
	boolarr := common.ConvertByteArrayToBoolArray(tid.PartiesState, numGuardians)
	faulties := 0
	for _, v := range boolarr {
		if !v {
			faulties++
		}
	}

	return faulties
}
