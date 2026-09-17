package algorand

import (
	bin "encoding/binary"
	"encoding/binary"
)

func positiveEmptyNonce(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	at.Txn.ApplicationArgs[2] = []byte{}
	publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(at.Txn.ApplicationArgs[2]))})
}

func positiveShortNonce(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	at.Txn.ApplicationArgs[2] = []byte{1, 2, 3, 4, 5, 6, 7}
	publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(at.Txn.ApplicationArgs[2]))})
}

func positiveOversizedNonce(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	at.Txn.ApplicationArgs[2] = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(at.Txn.ApplicationArgs[2]))})
}

func positiveEmptySequence(at SignedTxnWithAD) {
	if !isPublishMessage(at) || len(at.EvalDelta.Logs) == 0 {
		return
	}
	at.EvalDelta.Logs[0] = string([]byte{})
	publish(MessagePublication{Sequence: binary.BigEndian.Uint64([]byte(at.EvalDelta.Logs[0]))})
}

func positiveShortSequence(at SignedTxnWithAD) {
	if !isPublishMessage(at) || len(at.EvalDelta.Logs) == 0 {
		return
	}
	at.EvalDelta.Logs[0] = string([]byte{1, 2, 3, 4, 5, 6, 7})
	publish(MessagePublication{Sequence: binary.BigEndian.Uint64([]byte(at.EvalDelta.Logs[0]))})
}

func positiveOversizedSequence(at SignedTxnWithAD) {
	if !isPublishMessage(at) || len(at.EvalDelta.Logs) == 0 {
		return
	}
	at.EvalDelta.Logs[0] = string([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9})
	publish(MessagePublication{Sequence: binary.BigEndian.Uint64([]byte(at.EvalDelta.Logs[0]))})
}

func positiveNonceAliasNonExactGuard(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	nonceBytes := at.Txn.ApplicationArgs[2]
	if len(nonceBytes) > 0 {
		publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(nonceBytes))})
	}
}

func positiveSequenceAliasNonExactGuard(at SignedTxnWithAD) {
	if !isPublishMessage(at) || len(at.EvalDelta.Logs) == 0 {
		return
	}
	seqBytes := []byte(at.EvalDelta.Logs[0])
	if len(seqBytes) >= 8 {
		publish(MessagePublication{Sequence: binary.BigEndian.Uint64(seqBytes)})
	}
}

func positiveImportAlias(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	publish(MessagePublication{Nonce: uint32(bin.BigEndian.Uint64(at.Txn.ApplicationArgs[2]))})
}

type aliasedTxn = SignedTxnWithAD

func positiveTypeAlias(at aliasedTxn) {
	if !isPublishMessage(at) {
		return
	}
	publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(at.Txn.ApplicationArgs[2]))})
}

func decodeUint64Unsafe(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func positiveUnsafeHelper(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	publish(MessagePublication{Nonce: uint32(decodeUint64Unsafe(at.Txn.ApplicationArgs[2]))})
}

func readUint64Exact(b []byte) (uint64, error) {
	if len(b) != 8 {
		return 0, errBadLength
	}
	return binary.BigEndian.Uint64(b), nil
}

func positiveHelperErrorIgnored(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	nonce, _ := readUint64Exact(at.Txn.ApplicationArgs[2])
	publish(MessagePublication{Nonce: uint32(nonce)})
}

func positiveReassignedAfterExactGuard(at SignedTxnWithAD, other []byte) {
	if !isPublishMessage(at) {
		return
	}
	nonceBytes := at.Txn.ApplicationArgs[2]
	if len(nonceBytes) != 8 {
		return
	}
	nonceBytes = other
	publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(nonceBytes))})
}

func positiveApplicationArgMutatedAfterExactGuard(at SignedTxnWithAD, other []byte) {
	if !isPublishMessage(at) {
		return
	}
	if len(at.Txn.ApplicationArgs[2]) != 8 {
		return
	}
	at.Txn.ApplicationArgs[2] = other
	publish(MessagePublication{Nonce: uint32(binary.BigEndian.Uint64(at.Txn.ApplicationArgs[2]))})
}

func positiveLogMutatedAfterExactGuard(at SignedTxnWithAD, other string) {
	if !isPublishMessage(at) || len(at.EvalDelta.Logs) == 0 {
		return
	}
	if len([]byte(at.EvalDelta.Logs[0])) != 8 {
		return
	}
	at.EvalDelta.Logs[0] = other
	publish(MessagePublication{Sequence: binary.BigEndian.Uint64([]byte(at.EvalDelta.Logs[0]))})
}

func positiveLocalBigEndianAlias(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	order := binary.BigEndian
	publish(MessagePublication{Nonce: uint32(order.Uint64(at.Txn.ApplicationArgs[2]))})
}

func positiveHelperPublicationBeforeErrorCheck(at SignedTxnWithAD) {
	if !isPublishMessage(at) {
		return
	}
	nonce, err := readUint64Exact(at.Txn.ApplicationArgs[2])
	publish(MessagePublication{Nonce: uint32(nonce)})
	if err != nil {
		return
	}
	publish(MessagePublication{Nonce: uint32(nonce)})
}
