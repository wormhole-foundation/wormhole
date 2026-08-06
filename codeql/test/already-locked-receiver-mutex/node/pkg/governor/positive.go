package governor

func positiveGovernorDirectNoLock(gov *ChainGovernor) {
	_ = gov.parseMsgAlreadyLocked(nil)
}

func positiveGovernorDifferentReceiver(gov *ChainGovernor, other *ChainGovernor) {
	other.mutex.Lock()
	defer other.mutex.Unlock()
	_ = gov.loadFromDBAlreadyLocked()
}

func positiveGovernorMismatchedReadLock(gov *ChainGovernor) {
	gov.rw.RLock()
	defer gov.rw.RUnlock()
	_ = gov.parseMsgAlreadyLocked(nil)
}

func positiveGovernorUnlockedBeforeCall(gov *ChainGovernor) {
	gov.mutex.Lock()
	gov.mutex.Unlock()
	_ = gov.loadFromDBAlreadyLocked()
}
