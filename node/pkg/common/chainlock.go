package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/vaa"

	"github.com/ethereum/go-ethereum/common"
)

type MessagePublication struct {
	TxHash    common.Hash // TODO: rename to identifier? on Solana, this isn't actually the tx hash
	Timestamp time.Time

	Nonce            uint32
	Sequence         uint64
	ConsistencyLevel uint8
	EmitterChain     vaa.ChainID
	EmitterAddress   vaa.Address
	Payload          []byte
}

func (k *MessagePublication) MessageID() []byte {
	return []byte(k.MessageIDString())
}

func (k *MessagePublication) MessageIDString() string {
	return fmt.Sprintf("%v/%v/%v", uint16(k.EmitterChain), k.EmitterAddress, k.Sequence)
}

const minMsgLength = 88

func (k *MessagePublication) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(k.TxHash[:])
	vaa.MustWrite(buf, binary.BigEndian, uint32(k.Timestamp.Unix()))
	vaa.MustWrite(buf, binary.BigEndian, k.Nonce)
	vaa.MustWrite(buf, binary.BigEndian, k.Sequence)
	vaa.MustWrite(buf, binary.BigEndian, k.ConsistencyLevel)
	vaa.MustWrite(buf, binary.BigEndian, k.EmitterChain)
	buf.Write(k.EmitterAddress[:])
	buf.Write(k.Payload)

	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a VAA
func UnmarshalMessagePublication(data []byte) (*MessagePublication, error) {
	if len(data) < minMsgLength {
		return nil, fmt.Errorf("message is too short")
	}

	k := &MessagePublication{}

	reader := bytes.NewReader(data[:])

	txHash := common.Hash{}
	if n, err := reader.Read(txHash[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read TxHash [%d]: %w", n, err)
	}
	k.TxHash = txHash

	unixSeconds := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &unixSeconds); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	k.Timestamp = time.Unix(int64(unixSeconds), 0)

	if err := binary.Read(reader, binary.BigEndian, &k.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &k.Sequence); err != nil {
		return nil, fmt.Errorf("failed to read sequence: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &k.ConsistencyLevel); err != nil {
		return nil, fmt.Errorf("failed to read consistency level: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &k.EmitterChain); err != nil {
		return nil, fmt.Errorf("failed to read emitter chain: %w", err)
	}

	emitterAddress := vaa.Address{}
	if n, err := reader.Read(emitterAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read emitter address [%d]: %w", n, err)
	}
	k.EmitterAddress = emitterAddress

	payload := make([]byte, 1000)
	n, err := reader.Read(payload)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read payload [%d]: %w", n, err)
	}
	k.Payload = payload[:n]

	return k, nil
}
