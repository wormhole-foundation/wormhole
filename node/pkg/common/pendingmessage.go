package common

import (
	"bytes"
	"cmp"
	"container/heap"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const (
	// marshaledPendingMsgLenMin is the minimum length of a marshaled pending message.
	// It includes 8 bytes for the timestamp.
	marshaledPendingMsgLenMin = 8 + marshaledMsgLenMin
)

// PendingMessage is a wrapper type around a [MessagePublication] that includes the time for which it
// should be released.
type PendingMessage struct {
	ReleaseTime time.Time
	Msg         MessagePublication
}

func (p PendingMessage) Compare(other PendingMessage) int {
	return cmp.Compare(p.ReleaseTime.Unix(), other.ReleaseTime.Unix())
}

// MarshalBinary implements BinaryMarshaler for [PendingMessage].
func (p *PendingMessage) MarshalBinary() ([]byte, error) {
	msgPubBz, err := p.Msg.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal pending message: %w", err)
	}

	buf := new(bytes.Buffer)
	// Compare with [PendingTransfer.Marshal].
	// #nosec G115  -- int64 and uint64 have the same number of bytes, and Unix time won't be negative.
	vaa.MustWrite(buf, binary.BigEndian, uint64(p.ReleaseTime.Unix()))
	buf.Write(msgPubBz)
	return buf.Bytes(), nil
}

// UnmarshalBinary implements BinaryUnmarshaler for [PendingMessage].
func (p *PendingMessage) UnmarshalBinary(data []byte) error {

	if len(data) < marshaledPendingMsgLenMin {
		return ErrInputSize{Msg: "pending message too short", Want: marshaledPendingMsgLenMin, Got: len(data)}
	}

	// Compare with [UnmarshalPendingTransfer].
	p.ReleaseTime = time.Unix(
		// #nosec G115  -- int64 and uint64 have the same number of bytes, and Unix time won't be negative.
		int64(binary.BigEndian.Uint64(data[0:8])),
		0,
	)

	err := p.Msg.UnmarshalBinary(data[8:])

	if err != nil {
		return fmt.Errorf("unmarshal pending message: %w", err)
	}

	return nil
}

// A pendingMessageHeap is a min-heap of [PendingMessage] and uses the heap interface
// by implementing the methods below.
// As a result:
// - The heap is ordered by ReleaseTime. Only the root is guaranteed to be in position; the rest
// of the slice is not sorted.
// - The earliest (smallest) ReleaseTime is always at index 0.
// If the root's ReleaseTime is after whatever time threshold we specify, we know that
// there are no other messages that need to be released because their
// release times must be later. Checking the root takes constant time and removing it
// takes logarithmic time.
//
// See: https://pkg.go.dev/container/heap#Interface
type pendingMessageHeap []*PendingMessage

func (h pendingMessageHeap) Len() int {
	return len(h)
}
func (h pendingMessageHeap) Less(i, j int) bool {
	return h[i].ReleaseTime.Before(h[j].ReleaseTime)
}
func (h pendingMessageHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push dangerously pushes a value to the heap.
func (h *pendingMessageHeap) Push(x any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	item, ok := x.(*PendingMessage)
	if !ok {
		panic("PendingMessageHeap: cannot push non-*PendingMessage")
	}

	// Null check
	if item == nil {
		panic("PendingMessageHeap: cannot push nil *PendingMessage")
	}

	*h = append(*h, item)
}

// Pop dangerously pops a value from the heap. It removes the last element of the slice, which
// [heap.Pop] has swapped with the root before calling this method.
func (h *pendingMessageHeap) Pop() any {
	old := *h
	n := len(old)
	if n == 0 {
		panic("PendingMessageHeap: cannot Pop from empty heap")
	}
	last := old[n-1]
	*h = old[0 : n-1]
	return last
}

// PendingMessageQueue is a min-heap of [PendingMessage] ordered by ReleaseTime, with the
// earliest ReleaseTime at the top.
// It also prevents duplicate [MessagePublication]s from being added to the queue.
//
// PendingMessageQueue is safe for concurrent use by multiple goroutines. Each method is atomic,
// but a sequence of calls is not: another goroutine can change the queue between two calls. Use
// [PendingMessageQueue.Pop] to check readiness and remove a message in one step. Callers that must
// keep the queue consistent with other state need their own lock around the sequence.
type PendingMessageQueue struct {
	// pendingMessageHeap exposes dangerous APIs as a necessary consequence of implementing [heap.Interface].
	// Wrap it and expose only a safe API.
	heap pendingMessageHeap
	mu   sync.RWMutex
}

func NewPendingMessageQueue() *PendingMessageQueue {
	q := &PendingMessageQueue{heap: pendingMessageHeap{}}
	heap.Init(&q.heap)
	return q
}

// Push adds an element to the heap. If the pending message's message ID is invalid, or if it already exists in the queue, nothing is added.
func (q *PendingMessageQueue) Push(pMsg *PendingMessage) {
	// noop if the message is nil or already in the queue.
	if pMsg == nil || len(pMsg.Msg.MessageID()) < MinMsgIdLen {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Check for duplicates under the write lock
	for _, existing := range q.heap {
		if bytes.Equal(existing.Msg.MessageID(), pMsg.Msg.MessageID()) {
			return
		}
	}
	heap.Push(&q.heap, pMsg)
}

// Pop removes and returns the message with the earliest ReleaseTime if that ReleaseTime is before
// or equal to when. Returns nil if the queue is empty or if no message is ready.
//
// The readiness check and the removal happen under the same lock. Callers must not check
// readiness separately and then remove, because another goroutine can change the queue between
// the two calls. To remove a specific message regardless of its ReleaseTime, use [PendingMessageQueue.RemoveItem].
func (q *PendingMessageQueue) Pop(when time.Time) *PendingMessage {
	if q == nil {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.heap.Len() == 0 {
		return nil
	}

	// The root has the earliest ReleaseTime. If it isn't ready, no other message is.
	ready := q.heap[0]
	if ready.ReleaseTime.After(when) {
		return nil
	}

	heap.Pop(&q.heap)

	return ready
}

// Len returns the number of elements in the queue. Returns 0 if the queue is nil.
func (q *PendingMessageQueue) Len() int {
	if q == nil {
		return 0
	}
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.heap.Len()
}

// Peek returns a copy of the message with the earliest ReleaseTime without removing it.
// Returns nil if the queue is empty.
//
// The result is a snapshot. Another goroutine can change the queue after Peek returns, so do not
// use the result to decide what to remove. Use [PendingMessageQueue.Pop] to remove a ready message, or
// [PendingMessageQueue.RemoveItem] to remove a specific message.
func (q *PendingMessageQueue) Peek() *PendingMessage {
	if q == nil {
		return nil
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.heap.Len() == 0 {
		return nil
	}

	// container/heap stores the root (earliest ReleaseTime) at index 0.
	root := *q.heap[0]
	return &root
}

// RemoveItem removes target MessagePublication with the message ID from the heap. Returns the element that was removed or nil
// if the item was not found. No error is returned if the item was not found.
func (q *PendingMessageQueue) RemoveItem(msgID []byte) (*PendingMessage, error) {
	if msgID == nil {
		return nil, errors.New("pendingmessage: nil argument for RemoveItem")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	var removed *PendingMessage
	for i, item := range q.heap {
		// Assumption: MsgIDs are unique across MessagePublications.
		if bytes.Equal(item.Msg.MessageID(), msgID) {
			pMsg, ok := heap.Remove(&q.heap, i).(*PendingMessage)
			if !ok {
				return nil, errors.New("pendingmessage: item removed from heap is not PendingMessage")
			}
			removed = pMsg
			break
		}
	}

	return removed, nil
}

// Contains determines whether the queue contains a [PendingMessage].
func (q *PendingMessageQueue) Contains(pMsg *PendingMessage) bool {
	if pMsg == nil {
		return false
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	return slices.Contains(q.heap, pMsg)
}

// FetchMessagePublication returns a [MessagePublication] with the given ID if it exists in the queue, and nil
// otherwise.
func (q *PendingMessageQueue) FetchMessagePublication(msgID []byte) (msgPub *MessagePublication) {
	if len(msgID) == 0 {
		return nil
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	// Relies on MessageIDString to be unique.
	for _, pMsg := range q.heap {
		if bytes.Equal(pMsg.Msg.MessageID(), msgID) {
			return &pMsg.Msg
		}
	}

	return nil
}
