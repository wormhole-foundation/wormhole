package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const (
	// TxIDLenMin is the minimum length of a txID.
	TxIDLenMin = 32
	// AddressLength is the length of a normalized Wormhole address in bytes.
	AddressLength = 32

	// Wormhole supports arbitrary payloads due to the variance in transaction and block sizes between chains.
	// However, during serialization, payload lengths are limited by Go slice length constraints and violations
	// of these limits can cause panics.
	// (https://go.dev/src/runtime/slice.go)
	// This limit is chosen to be large enough to prevent such panics but it should comfortably handle all payloads.
	// If not, the limit can be increased.
	PayloadLenMax = 1024 * 1024 * 1024 * 10 // 10 GB

	// The minimum size of a marshaled message publication. It is the sum of the sizes of each of
	// the fields plus length information for fields with variable lengths (TxID and Payload).
	marshaledMsgLenMin = 1 + // TxID length (uint8)
		TxIDLenMin + // TxID ([]byte), minimum length of 32 bytes (but may be longer)
		8 + // Timestamp (int64)
		4 + // Nonce (uint32)
		8 + // Sequence (uint64)
		1 + // ConsistencyLevel (uint8)
		2 + // EmitterChain (uint16)
		32 + // EmitterAddress (32 bytes)
		1 + // IsReobservation (bool)
		1 + // Unreliable (bool)
		1 + // verificationState (uint8)
		8 // Payload length (int64), may be zero

	// Deprecated: represents the minimum message length for a marshaled message publication
	// before the Unreliable and verificationState fields were added.
	// Use [marshaledMsgSizeMin] instead.
	minMsgLength = 88

	// minMsgIdLen is the minimum length of a message ID. It is used to uniquely identify
	// messages in the case of a duplicate message ID and is stored in the database.
	MinMsgIdLen = len("1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")
)

var (
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
	Msg  string
	Got  int
	Want int
}

func (e ErrInputSize) Error() string {
	if e.Got != 0 && e.Want != 0 {
		return fmt.Sprintf("wrong size: %s. expected %d bytes, got %d", e.Msg, e.Want, e.Got)
	}

	if e.Got != 0 {
		return fmt.Sprintf("wrong size: %s, got %d", e.Msg, e.Got)
	}

	return fmt.Sprintf("wrong size: %s", e.Msg)

}

// MaxSafeInputSize defines the maximum safe size for untrusted input from `io` Readers.
// It should be configured so that it can comfortably contain all valid reads while
// providing a strict upper bound to prevent unlimited reads.
const MaxSafeInputSize = 128 * 1024 * 1024 // 128MB (arbitrary)

var ErrInputTooLarge = errors.New("input data exceeds maximum allowed size")

var (
	ErrBinaryWrite         = errors.New("failed to write binary data")
	ErrTxIDTooLong         = errors.New("field TxID too long")
	ErrTxIDTooShort        = errors.New("field TxID too short")
	ErrInvalidPayload      = errors.New("field payload too long")
	ErrDataTooShort        = errors.New("data too short")
	ErrTimestampTooShort   = errors.New("data too short for timestamp")
	ErrNonceTooShort       = errors.New("data too short for nonce")
	ErrSequenceTooShort    = errors.New("data too short for sequence")
	ErrConsistencyTooShort = errors.New("data too short for consistency level")
	ErrChainTooShort       = errors.New("data too short for emitter chain")
	ErrAddressTooShort     = errors.New("data too short for emitter address")
	ErrReobsTooShort       = errors.New("data too short for IsReobservation")
	ErrUnreliableTooShort  = errors.New("data too short for Unreliable")
	ErrVerStateTooShort    = errors.New("data too short for verification state")
	ErrPayloadLenTooShort  = errors.New("data too short for payload length")
	ErrPayloadTooShort     = errors.New("data too short for payload")
)

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
	// NOTE: there is no upper bound on the size of the payload. Wormhole supports arbitrary payloads
	// due to the variance in transaction and block sizes between chains. However, during deserialization,
	// payload lengths are bounds-checked against [PayloadLenMax] to prevent makeslice panics from malformed input.
	Payload         []byte
	IsReobservation bool

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

	// TxID is an alias for []byte.
	const TxIDSizeMax = math.MaxUint8

	// Check preconditions
	txIDLen := len(msg.TxID)
	if txIDLen > TxIDSizeMax {
		return nil, ErrInputSize{Msg: "TxID too long", Want: TxIDSizeMax, Got: txIDLen}
	}

	if txIDLen < TxIDLenMin {
		return nil, ErrInputSize{Msg: "TxID too short", Want: TxIDLenMin, Got: txIDLen}
	}

	payloadLen := len(msg.Payload)
	// Set up for serialization
	var (
		be = binary.BigEndian
		// Size of the buffer needed to hold the serialized message.
		// TxIDLenMin is already accounted for in the marshaledMsgLenMin calculation.
		bufSize = (marshaledMsgLenMin - TxIDLenMin) + txIDLen + payloadLen
		buf     = make([]byte, 0, bufSize)
	)

	// TxID (and length)
	buf = append(buf, uint8(txIDLen))
	buf = append(buf, msg.TxID...)

	// Timestamp
	tsBytes := make([]byte, 8)
	// #nosec G115  -- int64 and uint64 have the same number of bytes, and Unix time won't be negative.
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
	// There is no upper bound on the size of the payload as Wormhole supports arbitrary payloads. A uint64 should suffice to hold the length of the payload.
	payloadLenBytes := make([]byte, 8)
	be.PutUint64(payloadLenBytes, uint64(payloadLen))
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
func (m *MessagePublication) UnmarshalBinary(data []byte) error {

	// fixedFieldsLen is the minimum length of the fixed portion of a message publication.
	// It is the sum of the sizes of each of the fields plus length information for the Payload.
	// This is used to check that the data is long enough for the rest of the message after reading the TxID.
	const fixedFieldsLen = 8 + // Timestamp (int64)
		4 + // Nonce (uint32)
		8 + // Sequence (uint64)
		1 + // ConsistencyLevel (uint8)
		2 + // EmitterChain (uint16)
		32 + // EmitterAddress (32 bytes)
		1 + // IsReobservation (bool)
		1 + // Unreliable (bool)
		8 // Payload length (uint64)

	// Calculate minimum required length for the fixed portion
	// (excluding variable-length fields: TxID and Payload)
	if len(data) < marshaledMsgLenMin {
		return ErrInputSize{Msg: "data too short", Got: len(data), Want: marshaledMsgLenMin}
	}

	mp := &MessagePublication{}

	// Set up deserialization
	be := binary.BigEndian
	pos := 0

	// TxID length
	txIDLen := uint8(data[pos])
	pos++

	// Bounds checks. TxID length should be at least TxIDLenMin, but not larger than the length of the data.
	// The second check is to avoid panics.
	if int(txIDLen) < TxIDLenMin {
		return ErrInputSize{Msg: "TxID length is too short", Got: int(txIDLen), Want: TxIDLenMin}
	}

	if int(txIDLen) > len(data) {
		return ErrInputSize{Msg: "TxID length is longer than bytes", Got: int(txIDLen)}
	}

	// Read TxID
	mp.TxID = make([]byte, txIDLen)
	copy(mp.TxID, data[pos:pos+int(txIDLen)])
	pos += int(txIDLen)

	// TxID has a dynamic length, so now that we've read it, check that the remaining data is long enough for the rest of the message. This means that all fixed-length fields can be parsed with a payload of 0 or more bytes.
	// Concretely, we're checking that the data is at least long enough to contain information for all of
	// the fields except for the Payload itself.
	if len(data)-pos < fixedFieldsLen {
		return ErrInputSize{Msg: "data too short after reading TxID", Got: len(data)}
	}

	// Timestamp
	timestamp := be.Uint64(data[pos : pos+8])
	// Nanoseconds are not serialized as they are not used in Wormhole, so set them to zero.
	// #nosec G115  -- int64 and uint64 have the same number of bytes, and Unix time won't be negative.
	mp.Timestamp = time.Unix(int64(timestamp), 0)
	pos += 8

	// Nonce
	mp.Nonce = be.Uint32(data[pos : pos+4])
	pos += 4

	// Sequence
	mp.Sequence = be.Uint64(data[pos : pos+8])
	pos += 8

	// ConsistencyLevel
	// TODO: This could be validated against the valid range of values for ConsistencyLevel.
	mp.ConsistencyLevel = data[pos]
	pos++

	// EmitterChain
	mp.EmitterChain = vaa.ChainID(be.Uint16(data[pos : pos+2]))
	pos += 2

	// EmitterAddress
	copy(mp.EmitterAddress[:], data[pos:pos+32])
	pos += 32

	// IsReobservation
	if !validBinaryBool(data[pos]) {
		return ErrInvalidBinaryBool
	}
	mp.IsReobservation = data[pos] != 0
	pos++

	// Unreliable
	if !validBinaryBool(data[pos]) {
		return ErrInvalidBinaryBool
	}
	mp.Unreliable = data[pos] != 0
	pos++

	// verificationState. NumVariantsVerificationState is the number of variants of the enum,
	// which begins at 0. This means the valid range is [0, NumVariantsVerificationState-1].
	if data[pos] > NumVariantsVerificationState-1 {
		return ErrInvalidVerificationState
	}
	mp.verificationState = VerificationState(data[pos])
	pos++

	// Payload length
	payloadLen := be.Uint64(data[pos : pos+8])
	pos += 8

	// Check if payload length is within reasonable bounds to prevent makeslice panic.
	// Since payloadLen is read as uint64 from untrusted input, it could potentially
	// exceed this limit and cause a runtime panic when passed to make([]byte, payloadLen).
	// This bounds check prevents such panics by rejecting oversized payload lengths early.
	if payloadLen > PayloadLenMax {
		return ErrInputSize{Msg: "payload length too large", Got: len(data)}
	}

	// Check if we have enough data for the payload
	// #nosec G115 -- payloadLen is read from data, bounds checked above
	if len(data) < pos+int(payloadLen) {
		return ErrInputSize{Msg: "invalid payload length"}
	}

	// Read payload
	mp.Payload = make([]byte, payloadLen)
	// #nosec G115 -- payloadLen is read from data, bounds checked above
	copy(mp.Payload, data[pos:pos+int(payloadLen)])
	// #nosec G115 -- payloadLen is read from data, bounds checked above
	pos += int(payloadLen)

	// Check that exactly the correct number of bytes was read.
	if pos != len(data) {
		return ErrUnexpectedEndOfRead{expected: len(data), got: pos}
	}

	// Overwrite the receiver with the deserialized message.
	*m = *mp
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

// SafeRead reads from r with a size limit to prevent memory exhaustion attacks.
// It returns an error if the input exceeds MaxSafeInputSize.
func SafeRead(r io.Reader) ([]byte, error) {
	// Create a LimitReader that allows reading up to MaxSafeInputSize + 1 bytes.
	// The extra byte is specifically to detect if the input stream *exceeds* MaxSafeInputSize.
	lr := io.LimitReader(r, MaxSafeInputSize+1)

	//nolint:forbidigo // SafeRead is intended as a convenient and safe wrapper for ReadAll.
	b, err := io.ReadAll(lr)
	if err != nil {
		// Propagate any actual read errors from the underlying reader.
		return nil, err
	}

	// If the length of the read bytes is greater than MaxSafeInputSize,
	// it means the original reader contained more data than allowed.
	// In this case, we return an error instead of silently truncating.
	if len(b) > MaxSafeInputSize {
		return nil, ErrInputTooLarge
	}

	// If err was nil and len(b) <= MaxSafeInputSize, it means we read all
	// available input (or up to the limit) without exceeding the maximum.
	return b, nil
}
