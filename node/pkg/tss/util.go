package tss

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/multi-party-sig/protocols/frost/keygen"
	"github.com/xlabs/multi-party-sig/protocols/frost/sign"
	common "github.com/xlabs/tss-common"
	"github.com/xlabs/tss-lib/v2/party"
	"go.uber.org/zap"
)

type logableError struct {
	cause      error
	trackingId *common.TrackingID
	round      signingRound
}

type set[T comparable] map[T]struct{}

type strPartyId string

// activeSigCounter is a helper struct to keep track of active signatures.
// Each signature has a digest, and each guardian is allowed to be active
// for a certain number of signatures.
// a guardian is allowed to send how many messages it want per signature, but not allowed to
// participate in more than maxActiveSignaturesPerGuardian signatures at a time.
type activeSigCounter struct {
	mtx sync.RWMutex

	digestToGuardians map[sigKey]set[strPartyId]
	guardianToDigests map[strPartyId]set[sigKey]
	firstSeen         map[sigKey]time.Time
}

func newSigCounter() activeSigCounter {
	return activeSigCounter{
		mtx: sync.RWMutex{},

		digestToGuardians: make(map[sigKey]set[strPartyId]),
		guardianToDigests: make(map[strPartyId]set[sigKey]),
		firstSeen:         make(map[sigKey]time.Time),
	}
}

// Add adds a guardian to the counter for a given digest.
// returns false if this guardian is active for too many signatures ( > maxActiveSignaturesPerGuardian).
func (c *activeSigCounter) add(trackId *common.TrackingID, guardian *common.PartyID, maxActiveSignaturesPerGuardian int) bool {
	if trackId == nil || guardian == nil {
		return false
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	sgkey := trackingIdIntoSigKey(trackId)

	if _, ok := c.digestToGuardians[sgkey]; !ok {
		c.digestToGuardians[sgkey] = make(set[strPartyId])
	}

	strPartyId := strPartyId(guardian.ToString())

	if _, ok := c.guardianToDigests[strPartyId]; !ok {
		c.guardianToDigests[strPartyId] = make(set[sigKey])
	}

	// if already an active signature for this guardian, then it doesn't count as an additional signature
	if _, ok := c.guardianToDigests[strPartyId][sgkey]; ok {
		return true
	}

	// the guardian hasn't yet participated in this signing for the digest, we must ensure an additional signature is allowed
	if len(c.guardianToDigests[strPartyId])+1 > maxActiveSignaturesPerGuardian {
		return false
	}

	c.digestToGuardians[sgkey][strPartyId] = struct{}{}
	c.guardianToDigests[strPartyId][sgkey] = struct{}{}
	if _, ok := c.firstSeen[sgkey]; !ok {
		c.firstSeen[sgkey] = time.Now()
	}

	return true
}

func (c *activeSigCounter) remove(trackid *common.TrackingID) {
	if trackid == nil {
		return
	}

	key := trackingIdIntoSigKey(trackid)

	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.unlockedRemover(key)
}

func (c *activeSigCounter) unlockedRemover(key sigKey) {
	guardians := c.digestToGuardians[key]
	delete(c.digestToGuardians, key)

	for g := range guardians {
		delete(c.guardianToDigests[g], key)
	}

	delete(c.firstSeen, key)
}

func (c *activeSigCounter) cleanSelf(maxDuration time.Duration) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	for k, v := range c.firstSeen {
		if time.Since(v).Abs() > maxDuration {
			c.unlockedRemover(k)
		}
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
		zapFields = append(zapFields, zap.String("chainID", extractChainIDFromTrackingID(informativeErr.trackingId).String()))
	}

	if informativeErr.round != "" {
		zapFields = append(zapFields, zap.String("round", string(informativeErr.round)))
	}

	l.Error(informativeErr.Error(), zapFields...)
}

var (
	ErrBroadcastIsNil     = fmt.Errorf("broadcast is nil")
	ErrNilPartyId         = fmt.Errorf("party id is nil")
	ErrEmptyIDInPID       = fmt.Errorf("partyId identifier is empty")
	ErrEmptyKeyInPID      = fmt.Errorf("partyId doesn't contain a key")
	ErrSignedMessageIsNil = fmt.Errorf("SignedMessage is nil")
	ErrNoContent          = fmt.Errorf("SignedMessage doesn't contain a content")
	ErrNilPayload         = fmt.Errorf("SignedMessage doesn't contain a payload")
	ErrMissingTimestamp   = fmt.Errorf("problem struct missing timestamp field")
)

func validateBroadcastCorrectForm(e *tsscommv1.Echo) error {
	if e == nil {
		return ErrBroadcastIsNil
	}

	m := e.Message
	if m == nil {
		return ErrSignedMessageIsNil
	}

	if m.Content == nil {
		return ErrNoContent
	}

	if len(m.Signature) == 0 {
		return errEmptySignature
	}

	return nil
}

var (
	errNilEcho           = errors.New("echo is nil")
	errEchoDigestBadSize = errors.New("digest is not the correct size")
	errEchoSessionUUID   = errors.New("echo sessionUUID is not the correct size")
)

func validateHashEchoMessageCorrectForm(v *tsscommv1.SignedMessage_HashEcho) error {
	if v == nil || v.HashEcho == nil {
		return errNilEcho
	}

	if len(v.HashEcho.OriginalContetDigest) != len(digest{}) {
		return errEchoDigestBadSize
	}

	if len(v.HashEcho.SessionUuid) != len(uuid{}) {
		return errEchoSessionUUID
	}

	return nil
}

func validateUnicastCorrectForm(m *tsscommv1.Unicast) error {
	if m == nil {
		return ErrNoContent
	}

	if m.Content == nil {
		return ErrNoContent
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

type signingRound string

const (
	round1Message signingRound = "round1"
	round2Message signingRound = "round2"
	round3Message signingRound = "round3"
)

var _intToRoundArr = []signingRound{
	round1Message,
	round2Message,
	round3Message,
}

func intToRound(i int) signingRound {
	if i < 0 || i > 2 {
		return ""
	}

	return _intToRoundArr[i-1]
}

func getRound(m common.ParsedMessage) (signingRound, error) {
	if m == nil {
		return "", fmt.Errorf("message is nil")
	}

	if m.Content() == nil {
		return "", fmt.Errorf("message content is nil")
	}

	rnd := m.Content().RoundNumber()
	if rnd < 1 || rnd > len(_intToRoundArr) {
		return "", fmt.Errorf("message content round number is out of range")
	}

	return _intToRoundArr[m.Content().RoundNumber()-1], nil
}

func isBroadcastMsg(m common.ParsedMessage) bool {
	switch m.Content().(type) {
	case *sign.Broadcast2:
		return true
	case *sign.Broadcast3:
		return true
	case *keygen.Broadcast2:
		return true
	case *keygen.Broadcast3:
		return true
	default:
		return false
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
	if tid == nil {
		return fmt.Errorf("trackingID is nil")
	}

	if len(tid.Digest) != party.DigestSize {
		return fmt.Errorf("trackingID digest is not in correct size")
	}

	// checking that the byte array is the correct size
	if len(tid.PartiesState) < (st.NumGuardians()+7)/8 {
		return fmt.Errorf("trackingID partiesState is too short")
	}

	// TODO: expecting AuxiliaryData to be set.

	return nil
}

func extractChainIDFromTrackingID(tid *common.TrackingID) vaa.ChainID {
	bts := [2]byte{}
	copy(bts[:], tid.AuxiliaryData)

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

type SenderIndex uint32

func (s SenderIndex) toProto() uint32 {
	return uint32(s)
}
