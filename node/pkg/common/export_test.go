package common

// Peek returns a copy of the message with the earliest ReleaseTime without removing it.
// Returns nil if the queue is empty.
//
// Peek is defined for tests only. Production code must use [PendingMessageQueue.Pop], which
// checks readiness and removes the message under one lock.
func (q *PendingMessageQueue) Peek() *PendingMessage {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.heap.Len() == 0 {
		return nil
	}

	// container/heap stores the root (earliest ReleaseTime) at index 0.
	root := *q.heap[0]
	return &root
}
