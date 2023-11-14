package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
)

const HashLength = 32
const AddressLength = 32

type MessagePublication struct {
	TxHash    common.Hash // TODO: rename to identifier? on Solana, this isn't actually the tx hash
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
}

func (msg *MessagePublication) MessageID() []byte {
	return []byte(msg.MessageIDString())
}

func (msg *MessagePublication) MessageIDString() string {
	return fmt.Sprintf("%v/%v/%v", uint16(msg.EmitterChain), msg.EmitterAddress, msg.Sequence)
}

const minMsgLength = 88 // Marshalled length with empty payload

func (msg *MessagePublication) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(msg.TxHash[:])
	vaa.MustWrite(buf, binary.BigEndian, uint32(msg.Timestamp.Unix()))
	vaa.MustWrite(buf, binary.BigEndian, msg.Nonce)
	vaa.MustWrite(buf, binary.BigEndian, msg.Sequence)
	vaa.MustWrite(buf, binary.BigEndian, msg.ConsistencyLevel)
	vaa.MustWrite(buf, binary.BigEndian, msg.EmitterChain)
	buf.Write(msg.EmitterAddress[:])
	vaa.MustWrite(buf, binary.BigEndian, msg.IsReobservation)
	buf.Write(msg.Payload)

	return buf.Bytes(), nil
}

const oldMinMsgLength = 83 // Old marshalled length with empty payload

// UnmarshalOldMessagePublicationBeforeIsReobservation deserializes a MessagePublication from prior to the addition of IsReobservation.
// This function can be deleted once all guardians have been upgraded. That's why the code is just duplicated.
func UnmarshalOldMessagePublicationBeforeIsReobservation(data []byte) (*MessagePublication, error) {
	if len(data) < oldMinMsgLength {
		return nil, errors.New("message is too short")
	}

	msg := &MessagePublication{}

	reader := bytes.NewReader(data[:])

	txHash := common.Hash{}
	if n, err := reader.Read(txHash[:]); err != nil || n != HashLength {
		return nil, fmt.Errorf("failed to read TxHash [%d]: %w", n, err)
	}
	msg.TxHash = txHash

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
		return nil, fmt.Errorf("message is too short")
	}

	msg := &MessagePublication{}

	reader := bytes.NewReader(data[:])

	txHash := common.Hash{}
	if n, err := reader.Read(txHash[:]); err != nil || n != HashLength {
		return nil, fmt.Errorf("failed to read TxHash [%d]: %w", n, err)
	}
	msg.TxHash = txHash

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
		zap.Stringer("tx", msg.TxHash),
		zap.Time("timestamp", msg.Timestamp),
		zap.Uint32("nonce", msg.Nonce),
		zap.Uint8("consistency", msg.ConsistencyLevel),
		zap.String("message_id", string(msg.MessageID())),
		zap.Bool("unreliable", msg.Unreliable),
	)
}
