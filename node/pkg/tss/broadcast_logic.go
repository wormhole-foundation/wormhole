// The following code implements a hash broadcast protocol that guarantees that a
//
//	message has been seen by at least f+1 honest guardians.
//
// Unlike Bracha’s reliable broadcast, it doesn’t guarantee the message’s
// delivery (some honest nodes may deliver it, while others may not).
//
// In essence, the protocol ensures that if an honest guardian delivers a
// message `x`, no other honest guardian will deliver a different
// message `y` for the same round and session.

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

	"github.com/xlabs/tss-lib/v2/common"
	"github.com/xlabs/tss-lib/v2/ecdsa/party"
	"github.com/xlabs/tss-lib/v2/tss"
)

// voterId is comprised from the id and key of the signer, should match the guardians (in GuardianStorage) id and key.
type voterId string

type broadcastMessage interface {
	// We use the UUID to distinguish between messages the
	// broadcast algorithm handles.
	// When supporting a new uuid, take careful considertaions.
	// for instance, TSS messages create their uuid from values that
	// make each message unique, but also ensure the broadcast can
	// detect equivication attacks.
	getUUID(loadDistKey []byte) uuid
}

type processedMessage interface {
	broadcastMessage
	wrapError(error) error
}

type serialzeable interface {
	serialize() []byte
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

type parsedHashEcho struct {
	*tsscommv1.HashEcho
}

// getUUID implements processedMessage.
func (p *parsedHashEcho) getUUID(loadDistKey []byte) uuid {
	uid := uuid{}
	copy(uid[:], p.HashEcho.SessionUuid)

	return uid
}

// wrapError implements processedMessage.
func (p *parsedHashEcho) wrapError(e error) error {
	return logableError{cause: fmt.Errorf("error with hashEcho: %w", e)}
}

func serializeableToUUID(s serialzeable, loadDistKey []byte) uuid {
	return uuid(hash(append(s.serialize(), loadDistKey...)))

}

func (p *parsedProblem) wrapError(err error) error {
	return logableError{
		cause:      fmt.Errorf("error parsing problem, issuer %v: %w", p.issuer, err),
		trackingId: nil, // parsedProblem doesn't have a trackingID.
		round:      "",  // parsedProblem doesn't have a round.
	}
}

func (p *parsedProblem) serialize() []byte {
	if p == nil {
		return []byte(parsedProblemDomain)
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

	return b.Bytes()
}

func (p *parsedProblem) getUUID(loadDistKey []byte) uuid {
	return serializeableToUUID(p, loadDistKey)
}

func (msg *parsedTssContent) getUUID(loadDistKey []byte) uuid {
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

func (p *parsedAnnouncement) serialize() []byte {
	if p == nil {
		return []byte(newAnouncementDomain)
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

	return b.Bytes()
}

func (p *parsedAnnouncement) getUUID(loadDistKey []byte) uuid {
	return serializeableToUUID(p, loadDistKey)
}

func (p *parsedAnnouncement) wrapError(err error) error {
	return logableError{
		cause:      fmt.Errorf("error with digest announcement from %v: %w", p.issuer, err),
		trackingId: nil, // parsedAnnouncement doesn't have a trackingID.
		round:      "",  // parsedAnnouncement doesn't have a round.
	}
}

type broadcaststate struct {
	timeReceived   time.Time
	verifiedDigest *digest

	deliverableMessage broadcastMessage

	votes map[voterId]bool
	// if set to true: don't echo again, even if received from original sender.
	echoedAlready bool
	// if set to true: don't deliver again.
	alreadyDelivered bool

	mtx *sync.Mutex
}

func (t *Engine) getDeliverableIfAllowed(s *broadcaststate) broadcastMessage {
	f := t.GuardianStorage.getMaxExpectedFaults()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.alreadyDelivered {
		return nil
	}

	if len(s.votes) < f*2+1 {
		return nil
	}

	if s.deliverableMessage == nil {
		return nil
	}

	s.alreadyDelivered = true

	return s.deliverableMessage
}

var ErrEquivicatingGuardian = fmt.Errorf("equivication, guardian sent two different messages for the same round and session")

func wrapEquivErrWithTimestamp(err error, t time.Time) error {
	return fmt.Errorf("%w (first seen %v ago)", err, time.Since(t))
}

func (s *broadcaststate) update(parsed broadcastMessage, unparsedContent Incoming) (shouldEcho bool, err error) {
	unparsedSignedMessage := unparsedContent.toEcho().Message
	echoer := unparsedContent.GetSource()

	isMsgSrc := equalPartyIds(protoToPartyId(echoer), protoToPartyId(unparsedSignedMessage.Sender))

	_, ok1 := unparsedSignedMessage.Content.(*tsscommv1.SignedMessage_HashEcho)
	_, ok2 := parsed.(*parsedHashEcho)
	isEcho := ok1 || ok2

	s.mtx.Lock()
	defer s.mtx.Unlock()

	// the incoming message is valid when this function is reached.
	// So if the incomming message is not an echo, we can set the deliverable (which we'll return once we should deliver).
	if s.deliverableMessage == nil {
		if !isEcho { // has actual content.
			s.deliverableMessage = parsed
		}
	}

	s.votes[voterId(echoer.Key)] = true
	if s.echoedAlready {
		return shouldEcho, err
	}

	if isMsgSrc && !isEcho {
		s.echoedAlready = true
		shouldEcho = true

		return shouldEcho, err
	}

	// Unlike RB, we don't echo if we have enough votes.
	// that is, we don't want to echo something we don't hold the original message for.
	return shouldEcho, err
}

func (st *GuardianStorage) getMaxExpectedFaults() int {
	// since threshold is 2/3*n+1, f = (st.Threshold - 1) / 2
	// in our case st.Threshold is not inclusive, so we don't need to subtract 1.
	return (st.Threshold) / 2 // this is the floor of the result.
}

// broadcastInspection is the main function that handles the hash-broadcast algorithm.
// it returns whether a message should be re-broadcasted, a deiliverable message, or an error.
// Once a deliverable is returned from this function, it can be used by the caller.
func (t *Engine) broadcastInspection(parsed broadcastMessage, msg Incoming) (bool, broadcastMessage, error) {
	state := t.fetchOrCreateState(parsed)

	if err := t.validateBroadcastState(state, parsed, msg); err != nil {
		return false, nil, err
	}

	shouldBroadcast, err := state.update(parsed, msg)
	if err != nil {
		return false, nil, err
	}

	if shouldBroadcast && equalPartyIds(protoToPartyId(msg.toEcho().Message.Sender), t.Self) {
		shouldBroadcast = false // no need to echo if we're the original sender.
	}

	return shouldBroadcast, t.getDeliverableIfAllowed(state), nil
}

func (t *Engine) fetchOrCreateState(parsed broadcastMessage) *broadcaststate {
	uuid := parsed.getUUID(t.LoadDistributionKey)

	state := &broadcaststate{
		timeReceived:       time.Now(),
		verifiedDigest:     nil,
		deliverableMessage: nil,

		votes:            make(map[voterId]bool),
		echoedAlready:    false,
		alreadyDelivered: false,
		mtx:              &sync.Mutex{},
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()

	st, ok := t.received[uuid]
	if !ok {
		t.received[uuid] = state
		st = state
	}

	return st
}

func (t *Engine) validateBroadcastState(s *broadcaststate, parsed broadcastMessage, msg Incoming) error {
	unparsedSignedMessage := msg.toEcho().Message
	src := msg.GetSource()

	// locking a single state. Can be reached by multiple echoers.
	s.mtx.Lock()
	defer s.mtx.Unlock()

	// non-echo is a deliverable message. which only the original signer of the message can send.
	if _, ok := unparsedSignedMessage.Content.(*tsscommv1.SignedMessage_HashEcho); !ok {
		if _, ok := parsed.(*parsedHashEcho); ok {
			return fmt.Errorf("internal error. Parsed messsaage is a hash echo, but the signed message is not")
		}

		if !equalPartyIds(protoToPartyId(unparsedSignedMessage.Sender), protoToPartyId(src)) {
			return fmt.Errorf("any non echo message should have the same sender as the source")
		}
	}

	uid := parsed.getUUID(t.LoadDistributionKey)

	// verify incoming
	if s.verifiedDigest == nil {
		if err := t.verifySignedMessage(uid, unparsedSignedMessage); err != nil {
			return err
		}

		tmp := hashSignedMessage(unparsedSignedMessage)
		s.verifiedDigest = &tmp

	} else if *s.verifiedDigest != hashSignedMessage(unparsedSignedMessage) {
		if err := t.verifySignedMessage(uid, unparsedSignedMessage); err != nil {
			// two different digest and bad signature.
			return fmt.Errorf("Echoer %v sent a digest that can't be verified", src.Id)
		}

		// no error and two different digests:
		return fmt.Errorf("equivication attack detected. Sender %v sent two different digests", unparsedSignedMessage.Sender.Id)
	}

	return nil
}
