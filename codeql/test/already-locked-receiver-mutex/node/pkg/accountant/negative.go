package accountant

func negativeSameReceiverLocked(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	acct.publishTransferAlreadyLocked(pe)
	acct.pendingTransfersLock.Unlock()
}

func negativeDeferUnlock(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	acct.addPendingTransferAlreadyLocked(pe)
}

func negativeTwoCallsProtected(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	acct.addPendingTransferAlreadyLocked(pe)
	acct.publishTransferAlreadyLocked(pe)
}

func negativeDominatedAfterEarlyReturn(acct *Accountant, pe *pendingEntry, err error) {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	if err != nil {
		return
	}
	acct.deletePendingTransferAlreadyLocked(pe.msg)
}

func negativeAliasReceiver(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	alias := acct
	alias.publishTransferAlreadyLocked(pe)
}

func negativeDifferentReceiversEachProtected(acct *Accountant, other *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	acct.publishTransferAlreadyLocked(pe)
	acct.pendingTransfersLock.Unlock()
	other.pendingTransfersLock.Lock()
	defer other.pendingTransfersLock.Unlock()
	other.deletePendingTransferAlreadyLocked(pe.msg)
}

func negativeConditionalUnlockThenRelock(acct *Accountant, pe *pendingEntry, release bool) {
	acct.pendingTransfersLock.Lock()
	if release {
		acct.pendingTransfersLock.Unlock()
		acct.pendingTransfersLock.Lock()
	}
	acct.publishTransferAlreadyLocked(pe)
	acct.pendingTransfersLock.Unlock()
}
