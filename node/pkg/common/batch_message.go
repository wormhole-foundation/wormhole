package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/ethereum/go-ethereum/common"
)

type BatchMessageID struct {
	EmitterChain  vaa.ChainID
	TransactionID common.Hash
}

type BatchMessage struct {
	BatchMessageID
	Messages []*MessagePublication
}

func (b BatchMessageID) String() string {
	return fmt.Sprintf("%d/%s",
		b.EmitterChain,
		hex.EncodeToString(b.TransactionID.Bytes()))
}

// 35 bytes for EmitterChain + TransactionID, 88 for MessagePublication
const minBatchMsgLength = 123

func (batch *BatchMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, batch.EmitterChain)

	buf.Write(batch.TransactionID[:])

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(batch.Messages)))

	for _, msg := range batch.Messages {
		msgBytes, err := msg.Marshal()
		if err != nil {
			return nil, err
		}
		lenBytes := len(msgBytes)

		vaa.MustWrite(buf, binary.BigEndian, uint8(lenBytes))

		buf.Write(msgBytes)
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a BatchMessage
func UnmarshalBatchMessage(data []byte) (*BatchMessage, error) {
	if len(data) < minBatchMsgLength {
		return nil, fmt.Errorf("message is too short")
	}

	batch := &BatchMessage{}

	reader := bytes.NewReader(data[:])

	if err := binary.Read(reader, binary.BigEndian, &batch.EmitterChain); err != nil {
		return nil, fmt.Errorf("failed to read emitter chain: %w", err)
	}

	txID := common.Hash{}
	if n, err := reader.Read(txID[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read TxHash [%d]: %w", n, err)
	}
	batch.TransactionID = txID

	lenMessages, err := reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read Messages length: %w", err)
	}

	batch.Messages = make([]*MessagePublication, int(lenMessages))

	for i := 0; i < int(lenMessages); i++ {
		msgLength, err := reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("failed to read Message length [%d]", i)
		}
		numBytes := int(msgLength)

		msg := make([]byte, numBytes)

		if n, err := reader.Read(msg[:]); err != nil || n == 0 {
			return nil, fmt.Errorf("failed to read Message bytes [%d]: %w", n, err)
		}

		msgPub, err := UnmarshalMessagePublication(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal MessagePublication. %w", err)
		}

		batch.Messages[i] = msgPub

	}

	return batch, nil
}
