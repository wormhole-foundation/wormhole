package internal

import (
	"container/heap"
	"time"
)

type HasTTL interface {
	GetEndTime() time.Time
}

type Ttlheap[T HasTTL] interface {
	Enqueue(T)
	Dequeue() T
	// If the heap is not empty, this will fire once per element in the queue according to their time.
	WaitOnTimer() <-chan time.Time

	Peek() T
}

type ttlHeap[T HasTTL] struct {
	theap timedHeap[T]
	timer *time.Timer
}

// Dequeue implements Ttlheap.
func (t *ttlHeap[T]) Dequeue() T {
	var res T
	if t.theap.Len() == 0 {
		return res
	}

	elem := heap.Pop(&t.theap).(T)
	t.setTopAsTimer()
	return elem
}

// Enqueue implements Ttlheap.
func (t *ttlHeap[T]) Enqueue(elem T) {
	heap.Push(&t.theap, elem)
	t.setTopAsTimer()
}

func (t *ttlHeap[T]) setTopAsTimer() {
	if t.theap.Len() == 0 {
		t.stopAndDrainTimer() // no elements: stop the timer.

		return
	}

	endTime := t.theap.peek().GetEndTime() // we have at least one element.

	t.stopAndDrainTimer()
	// endtime.Sub(time.Now()) uses monotonic clock.

	// Notice that if timer.Reset is called with time.Until(endTime) which is negative, it will immediately fire (which is what we want).
	t.timer.Reset(time.Until(endTime))
}

func (t *ttlHeap[T]) stopAndDrainTimer() {
	// stopping the timer, if its channel is not drained: drain it.
	if !t.timer.Stop() && len(t.timer.C) > 0 {
		select {
		case <-t.timer.C:
		default:
		}
	}
}

// WaitOnTimer implements Ttlheap.
func (t *ttlHeap[T]) WaitOnTimer() <-chan time.Time {
	return t.timer.C
}

func NewTtlHeap[T HasTTL]() Ttlheap[T] {
	t := &ttlHeap[T]{
		theap: timedHeap[T]{},
		timer: time.NewTimer(time.Second), // safe creation of timer. we'll drain it soon.
	}

	heap.Init(&t.theap)
	t.stopAndDrainTimer()
	return t
}
func (d *ttlHeap[T]) Peek() T {
	return d.theap.peek()
}

type timedHeap[T HasTTL] struct {
	heap []T
}

func (d *timedHeap[T]) Len() int {
	return len(d.heap)
}

func (d *timedHeap[T]) Swap(i int, j int) {
	d.heap[i], d.heap[j] = d.heap[j], d.heap[i]
}

func (d *timedHeap[T]) Push(x any) {
	if v, ok := x.(T); ok {
		d.heap = append(d.heap, v)
	}
}

func (d *timedHeap[T]) peek() T {
	var res T
	if len(d.heap) == 0 {
		return res
	}
	res = d.heap[0]
	return res
}

func (d *timedHeap[T]) Less(i int, j int) bool {
	return d.heap[i].GetEndTime().Before(d.heap[j].GetEndTime())
}

func (d *timedHeap[T]) Pop() any {
	elem := d.heap[len(d.heap)-1]
	d.heap = d.heap[:len(d.heap)-1]

	return elem
}
