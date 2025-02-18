package tss

import (
	"fmt"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/xlabs/tss-lib/v2/tss"
)

func (t *Engine) parseEcho(m Incoming) (processedMessage, error) {
	echoMsg := m.toEcho()
	if err := vaidateEchoCorrectForm(echoMsg); err != nil {
		return nil, err
	}

	senderPid := protoToPartyId(echoMsg.Message.Sender)
	if !t.GuardianStorage.contains(senderPid) {
		return nil, fmt.Errorf("%w: %v", ErrUnkownSender, senderPid)
	}

	switch v := echoMsg.Message.Content.(type) {
	case *tsscommv1.SignedMessage_Problem:
		if err := validateProblemMessageCorrectForm(v); err != nil {
			return nil, err
		}

		return &parsedProblem{
			Problem: v.Problem,
			issuer:  echoMsg.Message.Sender,
		}, nil

	case *tsscommv1.SignedMessage_Announcement:
		if err := validateAnouncementCorrectForm(v); err != nil {
			return nil, err
		}

		return &parsedAnnouncement{
			SawDigest: v.Announcement,
			issuer:    echoMsg.Message.Sender,
		}, nil

	case *tsscommv1.SignedMessage_TssContent:
		if err := validateContentCorrectForm(v.TssContent); err != nil {
			return nil, err
		}

		p, err := tss.ParseWireMessage(v.TssContent.Payload, senderPid, true)
		if err != nil {
			return nil, err
		}

		parsed := &parsedTssContent{p, ""}

		rnd, err := getRound(parsed)
		if err != nil {
			return parsed, fmt.Errorf("couldn't extract round from echo: %w", err)
		}

		parsed.signingRound = rnd

		// according to gg18 (tss ecdsa paper), unicasts are sent in these rounds.
		if rnd == round1Message1 || rnd == round2Message {
			return parsed, errBadRoundsInEcho
		}

		if err := t.validateTrackingIDForm(parsed.getTrackingID()); err != nil {
			return parsed, err
		}

		return parsed, nil
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

func (t *Engine) parseTssContent(m *tsscommv1.TssContent, source *tsscommv1.PartyId) (*parsedTssContent, error) {
	if err := validateContentCorrectForm(m); err != nil {
		return nil, err
	}

	spid := protoToPartyId(source)
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
