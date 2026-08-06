package accountant

import "sync"

type Accountant struct {
	pendingTransfersLock sync.RWMutex
	otherLock            sync.Mutex
	msgChan              chan *struct{}
	pendingTransfers     map[string]*pendingEntry
}

type pendingEntry struct{ msg string }

var otherAcct *Accountant

func (acct *Accountant) publishTransferAlreadyLocked(pe *pendingEntry) {
	acct.deletePendingTransferAlreadyLocked(pe.msg)
	otherAcct.deletePendingTransferAlreadyLocked(pe.msg)
}

func (acct *Accountant) addPendingTransferAlreadyLocked(pe *pendingEntry) {
	acct = otherAcct
	acct.deletePendingTransferAlreadyLocked(pe.msg)
}

func (acct *Accountant) deletePendingTransferAlreadyLocked(msgId string) {
	alias := acct
	alias = otherAcct
	alias.publishTransferAlreadyLocked(&pendingEntry{msg: msgId})
	// Conservatively do not inherit the enclosing helper precondition into closures.
	func() {
		acct.publishTransferAlreadyLocked(&pendingEntry{msg: msgId})
	}()
}

func (acct *Accountant) cleanupAlreadyLocked(pe *pendingEntry) {
	acct.deletePendingTransferAlreadyLocked(pe.msg)
}
