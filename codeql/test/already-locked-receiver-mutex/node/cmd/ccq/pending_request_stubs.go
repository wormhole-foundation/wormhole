package ccq

import "sync"

type PendingResponses struct {
	mu               sync.Mutex
	pendingResponses map[string]int
}

func (p *PendingResponses) updateMetricsAlreadyLocked() {}
