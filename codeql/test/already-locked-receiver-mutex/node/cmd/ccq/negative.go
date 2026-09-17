package ccq

func negativeCCQLocked(p *PendingResponses) {
	p.mu.Lock()
	p.updateMetricsAlreadyLocked()
	p.mu.Unlock()
}

func negativeCCQDeferUnlock(p *PendingResponses) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.updateMetricsAlreadyLocked()
}
