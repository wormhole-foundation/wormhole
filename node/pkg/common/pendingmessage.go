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
// - The heap is always sorted by timestamp.
// - the oldest (smallest) timestamp is always the last element.
// This allows us to pop from the heap in order to get the oldest timestamp. If
// that value greater than whatever time threshold we specify, we know that
// there are no other messages that need to be released because their
// timestamps must be greater. This should allow for constant-time lookups when
// looking for messages to release.
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

// Pops dangerously pops a value from the heap.
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

// PendingMessageQueue is a thread-safe min-heap that sorts [PendingMessage] in descending order of Timestamp.
// It also prevents duplicate [MessagePublication]s from being added to the queue.
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
	if pMsg == nil {
		return
	}
	if len(pMsg.Msg.MessageID()) < MinMsgIdLen {
		return
	}
	// FetchMessagePublication acquires and releases a read lock, so we don't need to write lock
	// until we're inside the if statement.
	if q.FetchMessagePublication(pMsg.Msg.MessageID()) == nil {
		q.mu.Lock()
		heap.Push(&q.heap, pMsg)
		defer q.mu.Unlock()
	}

}

// Pop removes the last element from the heap and returns its value.
// Returns nil if the heap is empty or if the value is not a *[PendingMessage].
func (q *PendingMessageQueue) Pop() *PendingMessage {
	if q.heap.Len() == 0 {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	last, ok := heap.Pop(&q.heap).(*PendingMessage)
	if !ok {
		return nil
	}

	return last
}

func (q *PendingMessageQueue) Len() int {
	return q.heap.Len()
}

// Peek returns the element at the top of the heap without removing it.
func (q *PendingMessageQueue) Peek() *PendingMessage {
	if q.heap.Len() == 0 {
		return nil
	}
	// container/heap stores the "next" element at the first offset.
	last := *q.heap[0]
	return &last
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
