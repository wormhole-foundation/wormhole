package tss

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
	"unsafe"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/yossigi/tss-lib/v2/common"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/tss"
)

// The following code follows Bracha's reliable broadcast algorithm.

// voterId is comprised from the id and key of the signer, should match the guardians (in GuardianStorage) id and key.
type voterId string

// messages that were processed and parsed.
type relbroadcastables interface {
	// We use the UUID to distinguish between messages the reliable
	// broadcast algorithm handles.
	// When supporting a new uuid, take careful considertaions.
	// for instance, TSS messages create their uuid from values that
	// make each message unique, but also ensure the reliable-broadcast can
	// detect equivication attacks.
	getUUID(loadDistKey []byte) (uuid, error)

	// can be used for tracking and managing messages and
	// cross referencing them across the Engine (not just reliable broadcast),
	// and is mainly used for cleanup.
	// Non-TSS messages can return nil.
	getTrackingID() *common.TrackingID
}

type processedMessage interface {
	relbroadcastables
	wrapError(error) error
}

type serialzeable interface {
	serialize() ([]byte, error)
}

type parsedProblem struct {
	*tsscommv1.Problem
	issuer *tsscommv1.PartyId
}

type parsedTssContent struct {
	tss.ParsedMessage
	signingRound
}

type parsedAnnouncement struct {
	*tsscommv1.SawDigest
	issuer *tsscommv1.PartyId
}

func serializeableToUUID(s serialzeable, loadDistKey []byte) (uuid, error) {
	bts, err := s.serialize()
	if err != nil {
		return uuid{}, err
	}

	return uuid(hash(append(bts, loadDistKey...))), nil

}

func (p *parsedProblem) getTrackingID() *common.TrackingID {
	// parsedProblem is not a tss.ParsedMessage, so it doesn't have a trackingID.
	// and as stated in the comment above, it can be nil.
	return nil
}

func (p *parsedProblem) wrapError(err error) error {
	return logableError{
		cause:      fmt.Errorf("error parsing problem, issuer %v: %w", p.issuer, err),
		trackingId: nil, // parsedProblem doesn't have a trackingID.
		round:      "",  // parsedProblem doesn't have a round.
	}
}

func (p *parsedProblem) serialize() ([]byte, error) {
	if p == nil {
		return []byte(parsedProblemDomain), fmt.Errorf("nil relbroadcastables")
	}
	unixtime := p.IssuingTime.AsTime().Unix()

	fromId := [hostnameSize]byte{}
	copy(fromId[:], []byte(p.issuer.Id))

	fromKey := [pemKeySize]byte{}
	copy(fromKey[:], p.issuer.Key)

	capacity := len(parsedProblemDomain) +
		hostnameSize +
		pemKeySize +
		auxiliaryDataSize +
		int(unsafe.Sizeof(unixtime))

	b := bytes.NewBuffer(make([]byte, 0, capacity))

	b.WriteString(parsedProblemDomain) // domain separation.
	b.Write(fromId[:])
	b.Write(fromKey[:])
	vaa.MustWrite(b, binary.BigEndian, p.ChainID)
	vaa.MustWrite(b, binary.BigEndian, unixtime)

	return b.Bytes(), nil
}

func (p *parsedProblem) getUUID(loadDistKey []byte) (uuid, error) {
	return serializeableToUUID(p, loadDistKey)
}

func (msg *parsedTssContent) getUUID(loadDistKey []byte) (uuid, error) {
	return getMessageUUID(msg.ParsedMessage, loadDistKey)
}

func (p *parsedTssContent) wrapError(err error) error {
	if p == nil {
		return err
	}

	return logableError{
		cause:      err,
		trackingId: p.getTrackingID(),
		round:      p.signingRound,
	}
}

func (p *parsedTssContent) getTrackingID() *common.TrackingID {
	if p == nil {
		return nil
	}

	if p.ParsedMessage == nil {
		return nil
	}

	if p.WireMsg() == nil {
		return nil
	}

	return p.WireMsg().GetTrackingID()
}

func (p *parsedAnnouncement) serialize() ([]byte, error) {
	if p == nil {
		return []byte(newAnouncementDomain), fmt.Errorf("nil relbroadcastables")
	}

	fromId := [hostnameSize]byte{}
	copy(fromId[:], []byte(p.issuer.Id))

	fromKey := [pemKeySize]byte{}
	copy(fromKey[:], p.issuer.Key)

	capacity := len(newAnouncementDomain) +
		(hostnameSize + pemKeySize) +
		auxiliaryDataSize +
		party.DigestSize

	b := bytes.NewBuffer(make([]byte, 0, capacity))

	b.WriteString(newAnouncementDomain) // domain separation.
	b.Write(fromId[:])
	b.Write(fromKey[:])
	b.Write(p.Digest[:])
	vaa.MustWrite(b, binary.BigEndian, p.ChainID)

	return b.Bytes(), nil
}

func (p *parsedAnnouncement) getUUID(loadDistKey []byte) (uuid, error) {
	return serializeableToUUID(p, loadDistKey)
}

func (p *parsedAnnouncement) getTrackingID() *common.TrackingID { return nil }

func (p *parsedAnnouncement) wrapError(err error) error {
	return logableError{
		cause:      fmt.Errorf("error with digest announcement from %v: %w", p.issuer, err),
		trackingId: nil, // parsedAnnouncement doesn't have a trackingID.
		round:      "",  // parsedAnnouncement doesn't have a round.
	}
}

type broadcaststate struct {
	// The following three fields should not be changed after creation of broadcaststate:
	timeReceived  time.Time
	messageDigest digest
	trackingId    *common.TrackingID

	votes map[voterId]bool
	// if set to true: don't echo again, even if received from original sender.
	echoedAlready bool
	// if set to true: don't deliver again.
	alreadyDelivered bool

	mtx *sync.Mutex
}

func (t *Engine) shouldDeliver(s *broadcaststate) bool {
	f := t.GuardianStorage.getMaxExpectedFaults()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.alreadyDelivered {
		return false
	}

	if len(s.votes) < f*2+1 {
		return false
	}

	s.alreadyDelivered = true

	return true
}

var ErrEquivicatingGuardian = fmt.Errorf("equivication, guardian sent two different messages for the same round and session")

func (t *Engine) updateState(s *broadcaststate, msg *tsscommv1.SignedMessage, echoer *tsscommv1.PartyId) (shouldEcho bool, err error) {
	// this is a SECURITY measure to prevent equivication attacks:
	// It is possible that the same guardian sends two different messages for the same round and session.
	// We do not accept messages with the same uuid and different content.
	if s.messageDigest != hashSignedMessage(msg) {
		if err := t.verifySignedMessage(msg); err == nil { // no error means the sender is the equivicator.
			return false, fmt.Errorf("%w:%v", ErrEquivicatingGuardian, msg.Sender)
		}

		return false, fmt.Errorf("%w:%v", ErrEquivicatingGuardian, echoer)
	}

	f := t.GuardianStorage.getMaxExpectedFaults()

	return s.update(echoer, msg, f)
}

func (s *broadcaststate) update(echoer *tsscommv1.PartyId, msg *tsscommv1.SignedMessage, f int) (shouldEcho bool, err error) {
	isMsgSrc := equalPartyIds(protoToPartyId(echoer), protoToPartyId(msg.Sender))

	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.votes[voterId(echoer.Id)] = true
	if s.echoedAlready {
		return shouldEcho, err
	}

	if isMsgSrc {
		s.echoedAlready = true
		shouldEcho = true

		return shouldEcho, err
	}

	// at least one honest guardian heard this echo (meaning all honests will hear this message eventually).
	if len(s.votes) >= f+1 {
		s.echoedAlready = true
		shouldEcho = true

		return shouldEcho, err
	}

	return shouldEcho, err
}

func (st *GuardianStorage) getMaxExpectedFaults() int {
	// since threshold is 2/3*n+1, f = (st.Threshold - 1) / 2
	// in our case st.Threshold is not inclusive, so we don't need to subtract 1.
	return (st.Threshold) / 2 // this is the floor of the result.
}

func (t *Engine) relbroadcastInspection(parsed relbroadcastables, msg Incoming) (shouldEcho bool, shouldDeliver bool, err error) {
	// No need to check input: it was already checked before reaching this point
	signed := msg.toEcho().Message
	echoer := msg.GetSource()

	state, err := t.fetchOrCreateState(parsed, signed)
	if err != nil {
		return false, false, err
	}

	// If we weren't using TLS - at this point we would have to verify the
	// signature of the echoer (sender).

	allowedToBroadcast, err := t.updateState(state, signed, echoer)
	if err != nil {
		return false, false, err
	}

	if t.shouldDeliver(state) {
		return allowedToBroadcast, true, nil
	}

	return allowedToBroadcast, false, nil
}

func (t *Engine) fetchOrCreateState(parsed relbroadcastables, signed *tsscommv1.SignedMessage) (*broadcaststate, error) {
	uuid, err := parsed.getUUID(t.LoadDistributionKey)
	if err != nil {
		return nil, err
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()
	state, ok := t.received[uuid]

	if ok {
		return state, nil
	}

	if err := t.verifySignedMessage(signed); err != nil {
		return nil, err
	}

	state = &broadcaststate{
		timeReceived:  time.Now(),
		messageDigest: hashSignedMessage(signed),

		trackingId: parsed.getTrackingID(),

		votes:            make(map[voterId]bool),
		echoedAlready:    false,
		alreadyDelivered: false,
		mtx:              &sync.Mutex{},
	}

	t.received[uuid] = state

	return state, nil
}
