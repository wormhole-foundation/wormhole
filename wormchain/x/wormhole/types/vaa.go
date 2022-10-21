package types

import "encoding/binary"

type GovernanceMessage struct {
	Module  [32]byte
	Action  byte
	Chain   uint16
	Payload []byte
}

func NewGovernanceMessage(module [32]byte, action byte, chain uint16, payload []byte) GovernanceMessage {
	return GovernanceMessage{
		Module:  module,
		Action:  action,
		Chain:   chain,
		Payload: payload,
	}
}

func (gm *GovernanceMessage) MarshalBinary() []byte {
	bz := []byte{}
	bz = append(bz, gm.Module[:]...)
	bz = append(bz, gm.Action)
	chain_bz := [2]byte{}
	binary.BigEndian.PutUint16(chain_bz[:], gm.Chain)
	bz = append(bz, chain_bz[:]...)
	// set update payload
	bz = append(bz, gm.Payload...)
	return bz
}
