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

const (
	HashLength    = 32
	AddressLength = 32

	// The minimum size of a marshaled message publication. It is the sum of the sizes of each of
	// the fields plus length information for fields with variable lengths (TxID and Payload).
	marshaledMsgSizeMin = 1 + // TxID length (uint8)
		20 + // TxID ([]byte), minimum length of 20 bytes
		8 + // Timestamp (int64)
		4 + // Nonce (uint32)
		8 + // Sequence (uint64)
		1 + // ConsistencyLevel (uint8)
		2 + // EmitterChain (uint16)
		32 + // EmitterAddress (32 bytes)
		1 + // IsReobservation (bool)
		1 + // Unreliable (bool)
		1 + // verificationState (uint8)
		2 // Payload length (uint16)

	// Deprecated: represents the minimum message length for a marshaled message publication
	// before the Unreliable and verificationState fields were added.
	// Use [minMarshaledMsgSize] instead.
	minMsgLength = 88
)

var (
	ErrBinaryWrite              = errors.New("failed to write binary data")
	ErrInvalidBinaryBool        = errors.New("invalid binary bool (neither 0x00 nor 0x01)")
	ErrInvalidVerificationState = errors.New("invalid verification state")
)

type ErrUnexpectedEndOfRead struct {
	expected int
	got      int
}

func (e ErrUnexpectedEndOfRead) Error() string {
	return fmt.Sprintf("data position after unmarshal does not match data length. expected: %d got: %d", e.expected, e.got)
}

// ErrInputSize is returned when the input size is not the expected size during marshaling.
type ErrInputSize struct {
	Msg string
}

func (i ErrInputSize) Error() string {
	return fmt.Sprintf("wrong size: %s", i.Msg)
}

// The `VerificationState` is the result of applying transfer verification to the transaction associated with the `MessagePublication`.
// While this could likely be extended to additional security controls in the future, it is only used for `txverifier` at present.
// Consequently, its status should be set to `NotVerified` or `NotApplicable` for all messages that aren't token transfers.
type VerificationState uint8

const (
	// The default state for a message. This can be used before verification occurs. If no verification is required, `NotApplicable` should be used instead.
	NotVerified VerificationState = iota
	// Represents a "known bad" status where a Message has been validated and the result indicates an erroneous or invalid message. The message should be discarded.
	Rejected
	// Represents a successful validation, neither confirmed to be good or bad, but unusual.
	Anomalous
	// Represents a "known good" status where a Message has been validated and the result is good. The message should be processed normally.
	Valid
	// Indicates that no verification is necessary.
	NotApplicable
	// The message could not complete the verification process.
	CouldNotVerify
)

// NumVariantsVerificationState is the number of variants in the VerificationState enum.
// Used to validate deserialization.
const NumVariantsVerificationState = 6

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
	case CouldNotVerify:
		return "CouldNotVerify"
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
	verificationState VerificationState
}

// TxIDString returns a hex-encoded representation of the TxID field, prefixed with '0x'.
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

// Deprecated: This function does not unmarshal the Unreliable or verificationState fields.
// Use [MessagePublication.MarshalBinary] instead.
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

// Deprecated: UnmarshalOldMessagePublicationWithTxHash deserializes a MessagePublication from when the TxHash was a fixed size ethCommon.Hash.
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

// MarshalBinary implements the BinaryMarshaler interface for MessagePublication.
func (msg *MessagePublication) MarshalBinary() ([]byte, error) {
	// Marshalled Message Publication layout:
	//
	// - TxID length
	// - TxID
	// - Timestamp
	// - Nonce
	// - Sequence
	// - ConsistencyLevel
	// - EmitterChain
	// - EmitterAddress
	// - IsReobservation
	// - Unreliable
	// - verificationState
	// - Payload length
	// - Payload

	const (
		// TODO: is this big enough?
		PayloadSizeMax = math.MaxUint16

		// TxID is an alias for []byte.
		TxIDSizeMin = 20 // 20 bytes is the minimum size of a txID as determined by the EVM address length.
		TxIDSizeMax = math.MaxUint8
	)

	// Check preconditions
	txIDLen := len(msg.TxID)
	if txIDLen > TxIDSizeMax {
		return nil, ErrInputSize{Msg: "TxID too long"}
	}

	if txIDLen < TxIDSizeMin {
		return nil, ErrInputSize{Msg: "TxID too short"}
	}

	payloadLen := len(msg.Payload)
	if payloadLen > PayloadSizeMax {
		return nil, ErrInputSize{Msg: "payload too long"}
	}

	// Set up for serialization
	var (
		be      = binary.BigEndian
		bufSize = marshaledMsgSizeMin + txIDLen + payloadLen
		buf     = make([]byte, 0, bufSize)
	)

	// TxID (and length)
	buf = append(buf, uint8(txIDLen))
	buf = append(buf, msg.TxID...)

	// Timestamp
	tsBytes := make([]byte, 8)
	//nolint:gosec // uint64 and int64 have the same number of bytes, and Unix time won't be negative.
	be.PutUint64(tsBytes, uint64(msg.Timestamp.Unix()))
	buf = append(buf, tsBytes...)

	// Nonce
	nonceBytes := make([]byte, 4)
	be.PutUint32(nonceBytes, msg.Nonce)
	buf = append(buf, nonceBytes...)

	// Sequence
	seqBytes := make([]byte, 8)
	be.PutUint64(seqBytes, msg.Sequence)
	buf = append(buf, seqBytes...)

	// ConsistencyLevel
	buf = append(buf, msg.ConsistencyLevel)

	// EmitterChain
	chainBytes := make([]byte, 2)
	be.PutUint16(chainBytes, uint16(msg.EmitterChain))
	buf = append(buf, chainBytes...)

	// EmitterAddress
	buf = append(buf, msg.EmitterAddress.Bytes()...)

	// IsReobservation
	if msg.IsReobservation {
		buf = append(buf, byte(1))
	} else {
		buf = append(buf, byte(0))
	}

	// Unreliable
	if msg.Unreliable {
		buf = append(buf, byte(1))
	} else {
		buf = append(buf, byte(0))
	}

	// verificationState
	buf = append(buf, uint8(msg.verificationState))

	// Payload (and length)
	payloadLenBytes := make([]byte, 2)
	be.PutUint16(payloadLenBytes, uint16(payloadLen))
	buf = append(buf, payloadLenBytes...)
	buf = append(buf, msg.Payload...)

	return buf, nil
}

// Deprecated: UnmarshalMessagePublication deserializes a MessagePublication.
// This function does not unmarshal the Unreliable or verificationState fields.
// Use [MessagePublication.UnmarshalBinary] instead.
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

// UnmarshalBinary implements the BinaryUnmarshaler interface for MessagePublication.
func (msg *MessagePublication) UnmarshalBinary(data []byte) error {
	// Calculate minimum required length for the fixed portion
	// (excluding variable-length fields: TxID and Payload)

	// Initial check for minimum data length
	if len(data) < marshaledMsgSizeMin {
		return ErrInputSize{Msg: "data too short"}
	}

	// Set up deserialization
	be := binary.BigEndian
	pos := 0

	// TxID length
	txIDLen := uint8(data[pos])
	pos++

	// Check if we have enough data for TxID and the rest
	if len(data) < marshaledMsgSizeMin+int(txIDLen) {
		return ErrInputSize{Msg: "data too short"}
	}

	// Read TxID
	msg.TxID = make([]byte, txIDLen)
	copy(msg.TxID, data[pos:pos+int(txIDLen)])
	pos += int(txIDLen)

	// Timestamp
	timestamp := be.Uint64(data[pos : pos+8])
	// Nanoseconds are not serialized
	//nolint:gosec // uint64 and int64 have the same number of bytes, and Unix time won't be negative.
	msg.Timestamp = time.Unix(int64(timestamp), 0)
	pos += 8

	// Nonce
	msg.Nonce = be.Uint32(data[pos : pos+4])
	pos += 4

	// Sequence
	msg.Sequence = be.Uint64(data[pos : pos+8])
	pos += 8

	// ConsistencyLevel
	// TODO: This could be validated against the valid range of values for ConsistencyLevel.
	msg.ConsistencyLevel = data[pos]
	pos++

	// EmitterChain
	msg.EmitterChain = vaa.ChainID(be.Uint16(data[pos : pos+2]))
	pos += 2

	// EmitterAddress
	copy(msg.EmitterAddress[:], data[pos:pos+32])
	pos += 32

	// IsReobservation
	if !validBinaryBool(data[pos]) {
		return ErrInvalidBinaryBool
	}
	msg.IsReobservation = data[pos] != 0
	pos++

	// Unreliable
	if !validBinaryBool(data[pos]) {
		return ErrInvalidBinaryBool
	}
	msg.Unreliable = data[pos] != 0
	pos++

	// verificationState
	if data[pos] > NumVariantsVerificationState {
		return ErrInvalidVerificationState
	}
	msg.verificationState = VerificationState(data[pos])
	pos++

	// Payload length
	payloadLen := be.Uint16(data[pos : pos+2])
	pos += 2

	// Check if we have enough data for the payload
	if len(data) < pos+int(payloadLen) {
		return ErrInputSize{Msg: "payload too short"}
	}

	// Read payload
	msg.Payload = make([]byte, payloadLen)
	copy(msg.Payload, data[pos:pos+int(payloadLen)])
	pos += int(payloadLen)

	// Check that exactly the correct number of bytes was read.
	if pos != len(data) {
		return ErrUnexpectedEndOfRead{expected: len(data), got: pos}
	}

	return nil
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
		// MessageID contains EmitterChain, EmitterAddress, and Sequence
		zap.String("msgID", string(msg.MessageID())),
		zap.String("txID", msg.TxIDString()),
		zap.Time("timestamp", msg.Timestamp),
		zap.Uint32("nonce", msg.Nonce),
		zap.Uint8("consistency", msg.ConsistencyLevel),
		zap.Bool("unreliable", msg.Unreliable),
		zap.Bool("isReobservation", msg.IsReobservation),
		zap.String("verificationState", msg.verificationState.String()),
	)
}

// VAAHash returns a hash corresponding to the fields of the Message Publication that are ultimately
// encoded in a VAA. This is a helper function used to uniquely identify a Message Publication.
func (msg *MessagePublication) VAAHash() string {
	v := msg.CreateVAA(0) // We can pass zero in as the guardian set index because it is not part of the digest.
	digest := v.SigningDigest()
	return hex.EncodeToString(digest.Bytes())
}

// validBinaryBool returns true if the byte is either 0x00 or 0x01.
// Go marshals booleans as strictly 0x00 or 0x01, so this function is used to validate
// that a given byte is a valid boolean. When reading, any non-zero value is considered true,
// but here we want to validate that the value is strictly either 0x00 or 0x01.
func validBinaryBool(b byte) bool {
	return b == 0x00 || b == 0x01
}
