package governor

func negativeGovernorLocked(gov *ChainGovernor) {
	gov.mutex.Lock()
	_ = gov.parseMsgAlreadyLocked(nil)
	gov.mutex.Unlock()
}

func negativeGovernorDeferUnlock(gov *ChainGovernor) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()
	_ = gov.loadFromDBAlreadyLocked()
}

func negativeGovernorBothHelpers(gov *ChainGovernor) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()
	_ = gov.parseMsgAlreadyLocked(nil)
	_ = gov.loadFromDBAlreadyLocked()
}
