package algorand

import "encoding/binary"

func testFixtureUnguardedDecode(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(at.Txn.ApplicationArgs[2]))})
}
