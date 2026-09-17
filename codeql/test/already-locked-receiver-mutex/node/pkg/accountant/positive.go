package accountant

func positiveDirectNoLock(acct *Accountant, pe *pendingEntry) {
	acct.publishTransferAlreadyLocked(pe)
}

func positiveRLockWrongLockType(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.RLock()
	defer acct.pendingTransfersLock.RUnlock()
	acct.addPendingTransferAlreadyLocked(pe)
}

func positiveWrongField(acct *Accountant, pe *pendingEntry) {
	acct.otherLock.Lock()
	defer acct.otherLock.Unlock()
	acct.deletePendingTransferAlreadyLocked(pe.msg)
}

func positiveUnlockedBeforeCall(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	acct.pendingTransfersLock.Unlock()
	acct.publishTransferAlreadyLocked(pe)
}

func positiveDifferentReceiver(acct *Accountant, other *Accountant, pe *pendingEntry) {
	other.pendingTransfersLock.Lock()
	defer other.pendingTransfersLock.Unlock()
	acct.addPendingTransferAlreadyLocked(pe)
}

func positiveRelockDifferentReceiver(acct *Accountant, other *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	acct.pendingTransfersLock.Unlock()
	other.pendingTransfersLock.Lock()
	defer other.pendingTransfersLock.Unlock()
	acct.deletePendingTransferAlreadyLocked(pe.msg)
}

func TestPositiveNonTestFileStillChecked(acct *Accountant, pe *pendingEntry) {
	acct.publishTransferAlreadyLocked(pe)
}

func positiveNilReceiverAfterDifferentLock(acct *Accountant, pe *pendingEntry) {
	var nilAcct *Accountant
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	nilAcct.addPendingTransferAlreadyLocked(pe)
}

func positiveConditionalUnlockBeforeCall(acct *Accountant, pe *pendingEntry, release bool) {
	acct.pendingTransfersLock.Lock()
	if release {
		acct.pendingTransfersLock.Unlock()
	}
	acct.publishTransferAlreadyLocked(pe)
}

func positiveRelockBypassedByBreak(acct *Accountant, pe *pendingEntry, release bool) {
	acct.pendingTransfersLock.Lock()
	for release {
		acct.pendingTransfersLock.Unlock()
		break
		acct.pendingTransfersLock.Lock()
	}
	acct.publishTransferAlreadyLocked(pe)
}

func positiveRelockBypassedByGoto(acct *Accountant, pe *pendingEntry, release bool) {
	acct.pendingTransfersLock.Lock()
	if release {
		acct.pendingTransfersLock.Unlock()
		goto publish
		acct.pendingTransfersLock.Lock()
	}
publish:
	acct.publishTransferAlreadyLocked(pe)
}

func positiveDeferredInitialLock(acct *Accountant, pe *pendingEntry) {
	defer acct.pendingTransfersLock.Lock()
	acct.publishTransferAlreadyLocked(pe)
}

func positiveGoInitialLock(acct *Accountant, pe *pendingEntry) {
	go acct.pendingTransfersLock.Lock()
	acct.publishTransferAlreadyLocked(pe)
}

func positiveDeferredRelock(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	acct.pendingTransfersLock.Unlock()
	defer acct.pendingTransfersLock.Lock()
	acct.publishTransferAlreadyLocked(pe)
}

func positiveGoRelock(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	acct.pendingTransfersLock.Unlock()
	go acct.pendingTransfersLock.Lock()
	acct.publishTransferAlreadyLocked(pe)
}

func positiveReceiverReassignedAfterLock(acct *Accountant, other *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	acct = other
	acct.publishTransferAlreadyLocked(pe)
}

func positiveAliasReassignedAfterLock(acct *Accountant, other *Accountant, pe *pendingEntry) {
	alias := acct
	alias.pendingTransfersLock.Lock()
	alias = other
	alias.publishTransferAlreadyLocked(pe)
}

func positiveGoHelperAfterLock(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	go acct.publishTransferAlreadyLocked(pe)
	acct.pendingTransfersLock.Unlock()
}

func positiveDeferredHelperAfterUnlock(acct *Accountant, pe *pendingEntry) {
	acct.pendingTransfersLock.Lock()
	defer acct.publishTransferAlreadyLocked(pe)
	acct.pendingTransfersLock.Unlock()
}
