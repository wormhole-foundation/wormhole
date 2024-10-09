package tss

import (
	"fmt"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/yossigi/tss-lib/v2/ecdsa/signing"
	"github.com/yossigi/tss-lib/v2/tss"
	"go.uber.org/zap"
)

type logableError struct {
	cause      error
	trackingId []byte
	round      signingRound
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
		zapFields = append(zapFields, zap.String("trackingId", fmt.Sprintf("%x", informativeErr.trackingId)))
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

	if err := validateContentCorrectForm(m.Content); err != nil {
		return fmt.Errorf("signedMessage content error:%w", err)
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
