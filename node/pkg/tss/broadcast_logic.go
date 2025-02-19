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

type serialzeable interface {
	serialize() []byte
}

// Deliverable represents what the broadcast protocol returns.
// It is a message that its hash has been seen by at least 2f+1 guardians.
type deliverable interface {
	serialzeable

	deliver(*Engine) error
}

type deliverableMessage struct {
	deliverable
}

// serializeableMessage is a helper struct that converts a serializable message into a broadcastMessage.
type serializeableMessage struct {
	serialzeable
}

func (s *serializeableMessage) getUUID(loadDistKey []byte) uuid {
	return uuid(hash(append(s.serialize(), loadDistKey...)))
}

// implementing broadcastMessage interface for any deliverableMessage.
func (s *deliverableMessage) getUUID(loadDistKey []byte) uuid {
	return (&serializeableMessage{s}).getUUID(loadDistKey)
}

// serializeables:
type parsedProblem struct {
	*tsscommv1.Problem
	issuer *tsscommv1.PartyId
}

type tssMessageWrapper struct {
	tss.Message
}

func (t *tssMessageWrapper) serialize() []byte {
	return serializeTSSMessage(t.Message)
}

type parsedTssContent struct {
	tss.ParsedMessage
	signingRound
}

type parsedAnnouncement struct {
	*tsscommv1.SawDigest
	issuer *tsscommv1.PartyId
}

// broadcastable only struct (not deliverable or serializable):
type parsedHashEcho struct {
	*tsscommv1.HashEcho
}

func (p *parsedHashEcho) getUUID(loadDistKey []byte) uuid {
	uid := uuid{}
	copy(uid[:], p.HashEcho.SessionUuid)

	return uid
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

func (p *parsedProblem) deliver(t *Engine) error {
	return intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &reportProblemCommand{*p})
}

// This function sets a message's sessionID. It is crucial for SECURITY to ensure no equivocation
//
// We don't add the content of the message to the uuid, instead we collect all data that can put this message in a context.
// this is used by the broadcast protocol to check no two messages from the same sender will be used to update the full party
// in the same round for the specific session of the protocol.
func serializeTSSMessage(msg tss.Message) []byte {
	// The TackingID of a parsed message is tied to the run of the protocol for a single
	//  signature, thus we use it as a sessionID.
	messageTrackingID := [trackingIDHexStrSize]byte{}
	copy(messageTrackingID[:], []byte(msg.WireMsg().GetTrackingID().ToString()))

	fromId := [hostnameSize]byte{}
	copy(fromId[:], msg.GetFrom().Id)

	fromKey := [pemKeySize]byte{}
	copy(fromKey[:], msg.GetFrom().Key)

	// Adding the Message type allows the same sender to send messages for different rounds.
	// but, sender j is not allowed to send two different messages to the same round.
	tp := msg.Type()

	msgType := make([]byte, tssProtoMessageSize)
	copy(msgType[:], tp[:])

	d := make([]byte, 0, len(tssContentDomain)+int(trackingIDHexStrSize)+hostnameSize+pemKeySize)

	d = append(d, tssContentDomain...)
	d = append(d, messageTrackingID[:]...)
	d = append(d, fromId[:]...)
	d = append(d, fromKey[:]...)
	d = append(d, msgType[:]...)

	return d
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

func (p *parsedTssContent) deliver(t *Engine) error {
	if err := t.feedIncomingToFp(p.ParsedMessage); err != nil {
		return p.wrapError(fmt.Errorf("failed to update the full party: %w", err))
	}

	return nil
}

func (p *parsedTssContent) serialize() []byte {
	return serializeTSSMessage(p.ParsedMessage)
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

func (p *parsedAnnouncement) wrapError(err error) error {
	return logableError{cause: fmt.Errorf("error with digest announcement from %v: %w", p.issuer, err)}
}

func (p *parsedAnnouncement) deliver(t *Engine) error {
	return intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &newSeenDigestCommand{*p})
}

type broadcaststate struct {
	timeReceived   time.Time
	verifiedDigest *digest

	deliverable deliverable

	votes map[voterId]bool
	// if set to true: don't echo again, even if received from original sender.
	echoedAlready bool
	// if set to true: don't deliver again.
	alreadyDelivered bool

	mtx *sync.Mutex
}

func (t *Engine) getDeliverableIfAllowed(s *broadcaststate) deliverable {
	f := t.GuardianStorage.getMaxExpectedFaults()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.alreadyDelivered {
		return nil
	}

	if len(s.votes) < f*2+1 {
		return nil
	}

	if s.deliverable == nil {
		return nil
	}

	s.alreadyDelivered = true

	return s.deliverable
}

var ErrEquivicatingGuardian = fmt.Errorf("equivication, guardian sent two different messages for the same round and session")

func (s *broadcaststate) update(parsed broadcastMessage, unparsedContent Incoming) (shouldEcho bool, err error) {
	unparsedSignedMessage := unparsedContent.toEcho().Message
	echoer := unparsedContent.GetSource()

	isMsgSrc := equalPartyIds(protoToPartyId(echoer), protoToPartyId(unparsedSignedMessage.Sender))

	_, isEcho := unparsedSignedMessage.Content.(*tsscommv1.SignedMessage_HashEcho)

	s.mtx.Lock()
	defer s.mtx.Unlock()

	// the incoming message is valid when this function is reached.
	// So if the incomming message is not an echo, we can set the deliverable (which we'll return once we should deliver).
	if s.deliverable == nil && !isEcho {
		deliverable, ok := parsed.(deliverable)
		if ok { // has actual content.
			s.deliverable = deliverable
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

	return shouldEcho, err
}

func (st *GuardianStorage) getMaxExpectedFaults() int {
	// since threshold is 2/3*n+1, f = (st.Threshold - 1) / 2
	// in our case st.Threshold is not inclusive, so we don't need to subtract 1.
	return (st.Threshold) / 2 // this is the floor of the result.
}

// broadcastInspection is the main function that handles the hash-broadcast algorithm.
// it returns whether a message should be echoed, delivered, or an error.
// Once a deliverable is returned from this function, it can be used by the caller.
func (t *Engine) broadcastInspection(parsed broadcastMessage, msg Incoming) (bool, deliverable, error) {
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
		timeReceived:   time.Now(),
		verifiedDigest: nil,
		deliverable:    nil,

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

	// only non-echo messages should have the same sender as the source. (Echo messages should have different source then original sender).
	if _, ok := parsed.(deliverable); ok {
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
			return fmt.Errorf("caught bad behaviour: Echoer %v sent a digest that can't be verified", src.Id)
		}

		// no error and two different digests:
		return fmt.Errorf("equivication attack detected. Sender %v sent two different digests", unparsedSignedMessage.Sender.Id)
	}

	return nil
}
