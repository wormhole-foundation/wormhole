package tss

import (
	"fmt"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/xlabs/tss-lib/v2/tss"
)

func (t *Engine) parseBroadcast(m Incoming) (broadcastMessage, error) {
	broadcastMsg := m.toBroadcastMsg()
	if err := validateBroadcastCorrectForm(broadcastMsg); err != nil {
		return nil, err
	}

	senderId := SenderIndex(broadcastMsg.Message.Sender)

	if !t.GuardianStorage.contains(senderId) {
		return nil, fmt.Errorf("%w: %v", ErrUnkownSender, senderId)
	}

	switch v := broadcastMsg.Message.Content.(type) {
	case *tsscommv1.SignedMessage_Problem:
		if err := validateProblemMessageCorrectForm(v); err != nil {
			return nil, err
		}

		return &deliverableMessage{&parsedProblem{
			Problem: v.Problem,
			issuer:  senderId,
		}}, nil

	case *tsscommv1.SignedMessage_Announcement:
		if err := validateAnouncementCorrectForm(v); err != nil {
			return nil, err
		}

		return &deliverableMessage{&parsedAnnouncement{
			SawDigest: v.Announcement,
			issuer:    senderId,
		}}, nil

	case *tsscommv1.SignedMessage_TssContent:
		if err := validateContentCorrectForm(v.TssContent); err != nil {
			return nil, err
		}

		pid := t.GuardianStorage.getPartyIdFromIndex(senderId)

		p, err := tss.ParseWireMessage(v.TssContent.Payload, pid, true)
		if err != nil {
			return nil, err
		}

		parsed := &parsedTssContent{p, ""}
		res := &deliverableMessage{parsed}

		rnd, err := getRound(parsed)
		if err != nil {
			return res, fmt.Errorf("couldn't extract from broadcast message: %w", err)
		}

		parsed.signingRound = rnd

		// according to gg18 (tss ecdsa paper), unicasts are sent in these rounds.
		if rnd == round1Message1 || rnd == round2Message {
			return res, errBadRoundsInBroadcast
		}

		if err := t.validateTrackingIDForm(parsed.getTrackingID()); err != nil {
			return res, err
		}

		return res, nil
	case *tsscommv1.SignedMessage_HashEcho:
		if err := validateHashEchoMessageCorrectForm(v); err != nil {
			return nil, err
		}

		return &parsedHashEcho{
			HashEcho: v.HashEcho,
		}, nil
	default:
		return nil, fmt.Errorf("unknown content type: %T", v)
	}
}

func (t *Engine) parseTssContent(m *tsscommv1.TssContent, source *Identity) (*parsedTssContent, error) {
	if err := validateContentCorrectForm(m); err != nil {
		return nil, err
	}

	spid := source.getPidCopy()

	p, err := tss.ParseWireMessage(m.Payload, spid, false)
	if err != nil {
		return nil, err
	}

	parsed := &parsedTssContent{p, ""}

	// ensuring the reported source of the message matches the claimed source. (parsed.GetFrom() used by the tss-lib)
	if !equalPartyIds(parsed.GetFrom(), spid) {
		return parsed, fmt.Errorf("parsed message sender doesn't match the source of the message")
	}

	rnd, err := getRound(parsed)
	if err != nil {
		return parsed, fmt.Errorf("unicast parsing error: %w", err)
	}

	parsed.signingRound = rnd

	// only round 1 and round 2 are unicasts.
	if rnd != round1Message1 && rnd != round2Message {
		return parsed, errUnicastBadRound
	}

	if err := t.validateTrackingIDForm(parsed.getTrackingID()); err != nil {
		return parsed, err
	}

	return parsed, nil
}
