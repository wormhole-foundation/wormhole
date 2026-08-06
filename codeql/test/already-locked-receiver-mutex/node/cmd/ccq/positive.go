package ccq

func positiveCCQDirectNoLock(p *PendingResponses) {
	p.updateMetricsAlreadyLocked()
}

func positiveCCQDifferentReceiver(p *PendingResponses, other *PendingResponses) {
	other.mu.Lock()
	defer other.mu.Unlock()
	p.updateMetricsAlreadyLocked()
}
