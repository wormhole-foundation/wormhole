package accountant

func TestExcludedTestFile(acct *Accountant, pe *pendingEntry) {
	acct.publishTransferAlreadyLocked(pe)
}
