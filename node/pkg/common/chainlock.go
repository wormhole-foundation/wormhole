package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const HashLength = 32
const AddressLength = 32

// The `VerificationState` is the result of applying transfer verification to the transaction associated with the `MessagePublication`.
// While this could likely be extended to additional security controls in the future, it is only used for `txverifier` at present.
// Consequently, its status should be set to `NotVerified` or `NotApplicable` for all messages that aren't token transfers.
type VerificationState uint8

const (
	// The default state for a message. This can be used before verification occurs. If no verification is required, `NotApplicable` should be used instead.
	NotVerified VerificationState = iota
	// Represents a "known bad" status where a Message has been validated and the result indicates an erroneous or invalid message. The message should be discarded.
	Rejected
	// Represents an unusual state after validation, neither confirmed to be good or bad.
	Anomalous
	// Represents a "known good" status where a Message has been validated and the result is good. The message should be processed normally.
	Valid
	// Indicates that no verification is necessary.
	NotApplicable
)

func (v VerificationState) String() string {
	switch v {
	case NotVerified:
		return "NotVerified"
	case Valid:
		return "Valid"
	case Anomalous:
		return "Anomalous"
	case Rejected:
		return "Rejected"
	case NotApplicable:
		return "NotApplicable"
	default:
		return ""
	}
}

type MessagePublication struct {
	TxID      []byte
	Timestamp time.Time

	Nonce            uint32
	Sequence         uint64
	ConsistencyLevel uint8
	EmitterChain     vaa.ChainID
	EmitterAddress   vaa.Address
	Payload          []byte
	IsReobservation  bool

	// Unreliable indicates if this message can be reobserved. If a message is considered unreliable it cannot be
	// reobserved.
	// This field is not marshalled/serialized.
	Unreliable bool

	// The `VerificationState` is the result of applying transfer
	// verification to the transaction associated with the
	// `MessagePublication`. While this could likely be extended to
	// additional security controls in the future, it is only used for
	// `txverifier` at present. Consequently, its status should be set to
	// `NotVerified` or `NotApplicable` for all messages that aren't token
	// transfers.
	// This field is intentionally private so that it must be
	// updated using the setter, which performs verification on the new
	// state.
	// This field is not marshalled/serialized.
	verificationState VerificationState
}

func (msg *MessagePublication) TxIDString() string {
	return "0x" + hex.EncodeToString(msg.TxID)
}

func (msg *MessagePublication) MessageID() []byte {
	return []byte(msg.MessageIDString())
}

func (msg *MessagePublication) MessageIDString() string {
	return fmt.Sprintf("%v/%v/%v", uint16(msg.EmitterChain), msg.EmitterAddress, msg.Sequence)
}

func (msg *MessagePublication) VerificationState() VerificationState {
	return msg.verificationState
}

// SetVerificationState is the setter for verificationState. Returns an error if called in a way that likely indicates a programming mistake.
// This includes cases where:
// - an existing state would be overwritten by the NotVerified state
// - the argument is equal to the existing value
func (msg *MessagePublication) SetVerificationState(s VerificationState) error {
	// Avoid rewriting an existing state with the default value. There shouldn't be a reason to overwrite an existing verification,
	// and if it happens it's probably a bug.
	if s == NotVerified && msg.verificationState != NotVerified {
		return fmt.Errorf("SetVerificationState: refusing to overwrite existing VerificationState %s to NotVerified state", s)

	}
	// Not a problem per se but likely indicates a programming error.
	if s == msg.verificationState {
		return fmt.Errorf("SetVerificationState: called with value %s but Message Publication already has this value", s)
	}
	msg.verificationState = s
	return nil
}

const minMsgLength = 88 // Marshalled length with empty payload

func (msg *MessagePublication) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	if len(msg.TxID) > math.MaxUint8 {
		return nil, errors.New("TxID too long")
	}
	vaa.MustWrite(buf, binary.BigEndian, uint8(len(msg.TxID))) // #nosec G115 -- This is validated above
	buf.Write(msg.TxID)

	vaa.MustWrite(buf, binary.BigEndian, uint32(msg.Timestamp.Unix())) // #nosec G115 -- This conversion is safe until year 2106
	vaa.MustWrite(buf, binary.BigEndian, msg.Nonce)
	vaa.MustWrite(buf, binary.BigEndian, msg.Sequence)
	vaa.MustWrite(buf, binary.BigEndian, msg.ConsistencyLevel)
	vaa.MustWrite(buf, binary.BigEndian, msg.EmitterChain)
	buf.Write(msg.EmitterAddress[:])
	vaa.MustWrite(buf, binary.BigEndian, msg.IsReobservation)
	// Unreliable and verificationState are not marshalled because they are not used in the Governor code,
	// which is currently the only place in the node where marshalling this struct is done.
	buf.Write(msg.Payload)

	return buf.Bytes(), nil
}

// UnmarshalOldMessagePublicationWithTxHash deserializes a MessagePublication from when the TxHash was a fixed size ethCommon.Hash.
// This function can be deleted once all guardians have been upgraded. That's why the code is just duplicated.
func UnmarshalOldMessagePublicationWithTxHash(data []byte) (*MessagePublication, error) {
	if len(data) < minMsgLength {
		return nil, errors.New("message is too short")
	}

	msg := &MessagePublication{}

	reader := bytes.NewReader(data[:])

	txHash := common.Hash{}
	if n, err := reader.Read(txHash[:]); err != nil || n != HashLength {
		return nil, fmt.Errorf("failed to read TxHash [%d]: %w", n, err)
	}
	msg.TxID = txHash.Bytes()

	unixSeconds := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &unixSeconds); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	msg.Timestamp = time.Unix(int64(unixSeconds), 0)

	if err := binary.Read(reader, binary.BigEndian, &msg.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &msg.Sequence); err != nil {
		return nil, fmt.Errorf("failed to read sequence: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &msg.ConsistencyLevel); err != nil {
		return nil, fmt.Errorf("failed to read consistency level: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &msg.EmitterChain); err != nil {
		return nil, fmt.Errorf("failed to read emitter chain: %w", err)
	}

	emitterAddress := vaa.Address{}
	if n, err := reader.Read(emitterAddress[:]); err != nil || n != AddressLength {
		return nil, fmt.Errorf("failed to read emitter address [%d]: %w", n, err)
	}
	msg.EmitterAddress = emitterAddress

	if err := binary.Read(reader, binary.BigEndian, &msg.IsReobservation); err != nil {
		return nil, fmt.Errorf("failed to read isReobservation: %w", err)
	}

	payload := make([]byte, reader.Len())
	n, err := reader.Read(payload)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read payload [%d]: %w", n, err)
	}
	msg.Payload = payload[:n]

	return msg, nil
}

// UnmarshalMessagePublication deserializes a MessagePublication
func UnmarshalMessagePublication(data []byte) (*MessagePublication, error) {
	if len(data) < minMsgLength {
		return nil, errors.New("message is too short")
	}

	msg := &MessagePublication{}

	reader := bytes.NewReader(data[:])

	txIdLen := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &txIdLen); err != nil {
		return nil, fmt.Errorf("failed to read TxID len: %w", err)
	}

	msg.TxID = make([]byte, txIdLen)
	if n, err := reader.Read(msg.TxID[:]); err != nil || n != int(txIdLen) {
		return nil, fmt.Errorf("failed to read TxID [%d]: %w", n, err)
	}

	unixSeconds := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &unixSeconds); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	msg.Timestamp = time.Unix(int64(unixSeconds), 0)

	if err := binary.Read(reader, binary.BigEndian, &msg.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &msg.Sequence); err != nil {
		return nil, fmt.Errorf("failed to read sequence: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &msg.ConsistencyLevel); err != nil {
		return nil, fmt.Errorf("failed to read consistency level: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &msg.EmitterChain); err != nil {
		return nil, fmt.Errorf("failed to read emitter chain: %w", err)
	}

	emitterAddress := vaa.Address{}
	if n, err := reader.Read(emitterAddress[:]); err != nil || n != AddressLength {
		return nil, fmt.Errorf("failed to read emitter address [%d]: %w", n, err)
	}
	msg.EmitterAddress = emitterAddress

	if err := binary.Read(reader, binary.BigEndian, &msg.IsReobservation); err != nil {
		return nil, fmt.Errorf("failed to read isReobservation: %w", err)
	}

	// Unreliable and verificationState are not unmarshalled because they are not used in the Governor code,
	// which is currently the only place in the node where unmarshalling this struct is done.

	payload := make([]byte, reader.Len())
	n, err := reader.Read(payload)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read payload [%d]: %w", n, err)
	}
	msg.Payload = payload[:n]

	return msg, nil
}

// The standard json Marshal / Unmarshal of time.Time gets confused between local and UTC time.
func (msg *MessagePublication) MarshalJSON() ([]byte, error) {
	type Alias MessagePublication
	return json.Marshal(&struct {
		Timestamp int64
		*Alias
	}{
		Timestamp: msg.Timestamp.Unix(),
		Alias:     (*Alias)(msg),
	})
}

func (msg *MessagePublication) UnmarshalJSON(data []byte) error {
	type Alias MessagePublication
	aux := &struct {
		Timestamp int64
		*Alias
	}{
		Alias: (*Alias)(msg),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	msg.Timestamp = time.Unix(aux.Timestamp, 0)
	return nil
}

func (msg *MessagePublication) CreateVAA(gsIndex uint32) *vaa.VAA {
	return &vaa.VAA{
		Version:          vaa.SupportedVAAVersion,
		GuardianSetIndex: gsIndex,
		Signatures:       nil,
		Timestamp:        msg.Timestamp,
		Nonce:            msg.Nonce,
		EmitterChain:     msg.EmitterChain,
		EmitterAddress:   msg.EmitterAddress,
		Payload:          msg.Payload,
		Sequence:         msg.Sequence,
		ConsistencyLevel: msg.ConsistencyLevel,
	}
}

func (msg *MessagePublication) CreateDigest() string {
	v := msg.CreateVAA(0) // The guardian set index is not part of the digest, so we can pass in zero.
	db := v.SigningDigest()
	return hex.EncodeToString(db.Bytes())
}

// ZapFields takes some zap fields and appends zap fields related to the message. Example usage:
// `logger.Info("logging something with a message", msg.ZapFields(zap.Int("some_other_field", 100))...)“
// TODO refactor the codebase to use this function instead of manually logging the message with inconsistent fields
func (msg *MessagePublication) ZapFields(fields ...zap.Field) []zap.Field {
	return append(fields,
		zap.String("tx", msg.TxIDString()),
		zap.Time("timestamp", msg.Timestamp),
		zap.Uint32("nonce", msg.Nonce),
		zap.Uint8("consistency", msg.ConsistencyLevel),
		zap.String("message_id", string(msg.MessageID())),
		zap.Bool("unreliable", msg.Unreliable),
		zap.String("verificationState", msg.verificationState.String()),
	)
}
