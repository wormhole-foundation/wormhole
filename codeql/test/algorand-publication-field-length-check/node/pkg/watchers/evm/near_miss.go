package evm

import "encoding/binary"

type MessagePublication struct {
	Nonce uint32
}

type Tx struct {
	ApplicationArgs [][]byte
}

func nonAlgorandUnguardedDecode(tx Tx) MessagePublication {
	return MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(tx.ApplicationArgs[2]))}
}
