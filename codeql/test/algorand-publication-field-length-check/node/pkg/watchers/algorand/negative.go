package algorand

import "encoding/binary"

func negativeExactNonceGuard(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	if len(at.Txn.ApplicationArgs[2]) != 8 {
		return
	}
	publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(at.Txn.ApplicationArgs[2]))})
}

func negativeExactSequenceGuard(at SignedTxnWithAD) {
	if !isPublishMessage(at) || len(at.EvalDelta.Logs) == 0 {
		return
	}
	if len([]byte(at.EvalDelta.Logs[0])) != 8 {
		return
	}
	publish(MessagePublication{Sequence: binary.BigEndian.Uint64([]byte(at.EvalDelta.Logs[0]))})
}

func negativeAliasExactAcceptGuard(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	nonceBytes := at.Txn.ApplicationArgs[2]
	if len(nonceBytes) == 8 {
		publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(nonceBytes))})
	}
}

func negativeHelperErrorHandled(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	nonce, err := readUint64Exact(at.Txn.ApplicationArgs[2])
	if err != nil {
		return
	}
	publish(MessagePublication{Nonce: uint32(nonce)})
}

func negativeUnrelatedUint64(at SignedTxnWithAD, unrelated []byte) {
	if len(unrelated) != 8 {
		return
	}
	_ = binary.BigEndian.Uint64(unrelated)
}

func negativeContainerBoundsOnly(at SignedTxnWithAD) {
	if len(at.Txn.ApplicationArgs) <= 2 || len(at.EvalDelta.Logs) == 0 {
		return
	}
	publish(MessagePublication{})
}

func negativeLocalBigEndianAliasExactGuard(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	if len(at.Txn.ApplicationArgs[2]) != 8 {
		return
	}
	order := binary.BigEndian
	publish(MessagePublication{Nonce: uint32(order.Uint64(at.Txn.ApplicationArgs[2]))})
}

func negativeCheckedHelperResultUnusedNotPublished(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	nonce, err := readUint64Exact(at.Txn.ApplicationArgs[2])
	_ = nonce
	_ = err
	publish(MessagePublication{})
}
