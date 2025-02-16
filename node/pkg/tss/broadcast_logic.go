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
		return []byte(parsedProblemDomain), fmt.Errorf("nil parsedProblem")
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
		return []byte(newAnouncementDomain), fmt.Errorf("nil parsedAnnouncement")
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
	timeReceived   time.Time
	verifiedDigest *digest

	deliverableMessage relbroadcastables
	trackingId         *common.TrackingID

	votes map[voterId]bool
	// if set to true: don't echo again, even if received from original sender.
	echoedAlready bool
	// if set to true: don't deliver again.
	alreadyDelivered bool

	mtx *sync.Mutex
}

func (t *Engine) getDeliverableIfAllowed(s *broadcaststate) relbroadcastables {
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

func (t *Engine) updateState(s *broadcaststate, parsed relbroadcastables, msg *tsscommv1.SignedMessage, echoer *tsscommv1.PartyId) (shouldEcho bool, err error) {
	f := t.GuardianStorage.getMaxExpectedFaults()

	isMsgSrc := equalPartyIds(protoToPartyId(echoer), protoToPartyId(msg.Sender))

	s.mtx.Lock()
	defer s.mtx.Unlock()

	// the incoming message is valid when this function is reached.
	// So if the incomming message is not an echo, we can set the deliverable (which we'll return once we should deliver).
	if s.deliverableMessage == nil {
		if _, ok := msg.Content.(*tsscommv1.SignedMessage_HashEcho); !ok {
			s.deliverableMessage = parsed
		}
	}

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

// relbroadcastInspection is the main function that handles the hash-broadcast algorithm.
// it returns whether a message should be re-broadcasted, a deiliverable message, or an error.
// Once a deliverable is returned from this function, it can be used by the caller.
func (t *Engine) relbroadcastInspection(parsed relbroadcastables, msg Incoming) (bool, relbroadcastables, error) {
	// No need to check input: it was already checked before reaching this point
	signed := msg.toEcho().Message
	echoer := msg.GetSource()

	state, err := t.fetchOrCreateState(msg, parsed, signed)
	if err != nil {
		return false, nil, err
	}

	if err := t.validateBroadcastState(state, signed, msg.GetSource()); err != nil {
		return false, nil, err
	}

	shouldBroadcast, err := t.updateState(state, parsed, signed, echoer)
	if err != nil {
		return false, nil, err
	}

	return shouldBroadcast, t.getDeliverableIfAllowed(state), nil
}

func (t *Engine) fetchOrCreateState(msg Incoming, parsed relbroadcastables, signed *tsscommv1.SignedMessage) (*broadcaststate, error) {
	uuid, err := parsed.getUUID(t.LoadDistributionKey)
	if err != nil {
		return nil, err
	}

	t.mtx.Lock()
	state, ok := t.received[uuid]
	if !ok {
		state = &broadcaststate{
			timeReceived:       time.Now(),
			verifiedDigest:     nil,
			deliverableMessage: nil,
			trackingId:         parsed.getTrackingID(),

			votes:            make(map[voterId]bool),
			echoedAlready:    false,
			alreadyDelivered: false,
			mtx:              &sync.Mutex{},
		}

		t.received[uuid] = state
	}
	t.mtx.Unlock()

	return state, nil
}

func (t *Engine) validateBroadcastState(s *broadcaststate, signed *tsscommv1.SignedMessage, source *tsscommv1.PartyId) error {
	// locking a single state. Can be reached by multiple echoers.
	s.mtx.Lock()
	defer s.mtx.Unlock()

	// non echo is a deliverable message. which only the original signer of the message can send.
	if _, ok := signed.Content.(*tsscommv1.SignedMessage_HashEcho); !ok {
		if !equalPartyIds(protoToPartyId(signed.Sender), protoToPartyId(source)) {
			return fmt.Errorf("any non echo message should have the same sender as the source")
		}
	}

	// verify incoming
	if s.verifiedDigest == nil {
		// TODO: Ensure the session UUID is also signed!
		// 		If it isn't someone can copy a meesage and send a signed message with a different session UUID.
		if err := t.verifySignedMessage(signed); err != nil {
			return err
		}

		tmp := hashSignedMessage(signed)
		s.verifiedDigest = &tmp

	} else if *s.verifiedDigest != hashSignedMessage(signed) {
		if err := t.verifySignedMessage(signed); err != nil {
			// two different digest and bad signature.
			return fmt.Errorf("equivication attack detected. echoer %v sent a digest that can't be verified", source.Id)
		}

		// no error and two different digests:
		return fmt.Errorf("equivication attack detected. sender %v sent two different digests", signed.Sender.Id)
	}

	return nil
}
