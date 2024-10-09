package comm

import (
	"container/heap"
	"time"
)

const (
	maxBackoffTimeModifier = 10 // max backoff attempts before time doesn't increase.
	minBackoffTime         = time.Millisecond * 100
)

// NOT THREAD SAFE! do NOT share between two different goroutines.
//
// This struct is a a conjunction of min-heap and timer to create a list of tasks with deadlines cheaply.
// Mainly similar to multiple cases of `time.After(func(){})“, but without using goroutines under the hood for each
// invocation of Enqueue.
type backoffHeap struct {
	heap            []dialWithBackoff
	timer           *time.Timer
	alreadyInHeap   map[string]bool
	attemptsPerPeer map[string]uint64 // on successful dial, reset to 0.
}

type dialWithBackoff struct {
	hostname string
	attempt  uint64

	/*
		We use this redialTime to calculate when to set timers for redial requests.

		IMPORTANT:
		Go 1.19 introduced to `time.Time` type monotonic clocks to ensure local changes to
		the system clock does not affect time comparisons methods
		like t.After(u), t.Before(u), t.Equal(u), t.Compare(u), and t.Sub(u)
		(see time package docs for further information).

		Therefore, we can safely use `time.Time` in this struct to set timers for correctly redialing.
	*/
	nextRedialTime time.Time
}

// Enqueue adds a hostname to the heap, with a new backoff time.
func (d *backoffHeap) Enqueue(hostname string) {
	if d.alreadyInHeap[hostname] {
		return
	}

	if v, ok := d.attemptsPerPeer[hostname]; ok {
		if v+1 <= maxBackoffTimeModifier {
			v += 1
		}

		d.attemptsPerPeer[hostname] = v
	} else {
		d.attemptsPerPeer[hostname] = 0
	}

	elem := dialWithBackoff{
		hostname: hostname,
		attempt:  d.attemptsPerPeer[hostname],
	}

	elem.setBackoff()
	heap.Push(d, elem)
	d.alreadyInHeap[hostname] = true

	d.setTopAsTimer()
}

func (d *backoffHeap) Dequeue() string {
	if len(d.heap) == 0 {
		return ""
	}

	elem, ok := heap.Pop(d).(dialWithBackoff)
	if !ok {
		return "" // shouldn't happen.
	}

	d.alreadyInHeap[elem.hostname] = false

	d.setTopAsTimer()

	return elem.hostname
}

func (d *backoffHeap) ResetAttempts(hostname string) {
	delete(d.attemptsPerPeer, hostname)
}

func (d *backoffHeap) setTopAsTimer() {
	if len(d.heap) == 0 {
		d.stopAndDrainTimer() // no elements: stop the timer.

		return
	}

	endTime := d.peek().nextRedialTime // we have at least one element.

	d.stopAndDrainTimer()
	// endtime.Sub(time.Now()) uses monotonic clock.
	d.timer.Reset(time.Until(endTime))
}

func (d *backoffHeap) stopAndDrainTimer() {
	// stopping the timer, if its channel is not drained: drain it.
	if !d.timer.Stop() && len(d.timer.C) > 0 {
		select {
		case <-d.timer.C:
		default:
		}
	}
}

func newBackoffHeap() backoffHeap {
	b := backoffHeap{
		heap:            []dialWithBackoff{},
		timer:           time.NewTimer(time.Second),
		alreadyInHeap:   map[string]bool{},
		attemptsPerPeer: map[string]uint64{},
	}
	heap.Init(&b)

	b.stopAndDrainTimer() // ensuring it doesn't fire when empty.

	return b
}

func (d *dialWithBackoff) _durationBasedOnNumberOfAttempts() time.Duration {
	return minBackoffTime * (1 << uint(d.attempt))
}

func (d *dialWithBackoff) setBackoff() {
	if d.attempt >= maxBackoffTimeModifier {
		d.attempt = maxBackoffTimeModifier
	}

	duration := d._durationBasedOnNumberOfAttempts()
	if duration < minBackoffTime {
		duration = minBackoffTime // ensuring overflow doesn't write minus duration.
	}

	d.nextRedialTime = time.Now().Add(duration)
}

// =====================================================
// implemntation for heap interface, don't use directly.
// =====================================================

func (d *backoffHeap) Len() int {
	return len(d.heap)
}

func (d *backoffHeap) Swap(i int, j int) {
	d.heap[i], d.heap[j] = d.heap[j], d.heap[i]
}

func (d *backoffHeap) Push(x any) {
	if v, ok := x.(dialWithBackoff); ok {
		d.heap = append(d.heap, v)
	}
}

func (d *backoffHeap) peek() *dialWithBackoff {
	if len(d.heap) == 0 {
		return nil
	}

	return &d.heap[0]
}

func (d *backoffHeap) Less(i int, j int) bool {
	return d.heap[i].nextRedialTime.Before(d.heap[j].nextRedialTime)
}

func (d *backoffHeap) Pop() any {
	elem := d.heap[len(d.heap)-1]
	d.heap = d.heap[:len(d.heap)-1]

	return elem
}
