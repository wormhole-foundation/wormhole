package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

const (
	EevaaMagic = "WHEV" // Preceeds every EEVAA message
	PostEEVAA  = 1      // Instruction kind for EEVAAs
)

type EEVAA struct {
	id      uint64
	payload []byte
}

func (e *EEVAA) String() string {
	return fmt.Sprintf("id: %d, payload: %s", e.id, hex.EncodeToString(e.payload))
}

// ParseEevaa ...
func ParseEevaa(data []byte) (*EEVAA, error) {
	ret := &EEVAA{}

	r := bytes.NewBuffer(data)

	magicBytes := make([]byte, len(EevaaMagic))
	if _, err := r.Read(magicBytes[:]); err != nil || bytes.Compare(magicBytes, []byte(EevaaMagic)) != 0 {
		return nil, fmt.Errorf("Invalid magic")
	}

	kindByte := make([]byte, 1)
	if _, err := r.Read(kindByte[:]); err != nil || bytes.Compare(kindByte, []byte{PostEEVAA}) != 0 {
		return nil, fmt.Errorf("Invalid instruction byte (expected %d)", PostEEVAA)
	}

	if err := binary.Read(r, binary.BigEndian, &ret.id); err != nil {
		return nil, fmt.Errorf("Could not read EEVAA id: %w", err);
	}

	var payloadLen uint16
	if err := binary.Read(r, binary.BigEndian, &payloadLen); err != nil {
		return nil, fmt.Errorf("Could not read EEVAA payload length: %w", err);
	}

	ret.payload = make([]byte, payloadLen)
	if _, err := r.Read(ret.payload[:]); err != nil {
		return nil, fmt.Errorf("Could not read EEVAA payload: %w", err);
	}

	return ret, nil
}
