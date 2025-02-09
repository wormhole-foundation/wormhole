package comm

import (
	"time"

	"github.com/certusone/wormhole/node/pkg/tss/internal"
)

const (
	maxBackoffTimeModifier = 10 // max backoff attempts before time doesn't increase.
	minBackoffTime         = time.Millisecond * 100
)

// NOT THREAD SAFE! do NOT share between two different goroutines.
//
// This struct is a conjunction of min-heap and timer to create a list of tasks with deadlines cheaply.
// Mainly similar to multiple cases of `time.After(func(){})“, but without using goroutines under the hood for each
// invocation of Enqueue.
type backoffHeap struct {
	internal.Ttlheap[*dialWithBackoff]

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

func (d *dialWithBackoff) GetEndTime() time.Time {
	return d.nextRedialTime
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

	elem := &dialWithBackoff{
		hostname: hostname,
		attempt:  d.attemptsPerPeer[hostname],
	}

	elem.setBackoff()

	d.alreadyInHeap[hostname] = true

	d.Ttlheap.Enqueue(elem)
}

func (d *backoffHeap) Dequeue() string {
	elem := d.Ttlheap.Dequeue()
	if elem == nil {
		return ""
	}

	d.alreadyInHeap[elem.hostname] = false

	return elem.hostname
}

func (d *backoffHeap) ResetAttempts(hostname string) {
	delete(d.attemptsPerPeer, hostname)
}

func newBackoffHeap() backoffHeap {
	b := backoffHeap{
		Ttlheap: internal.NewTtlHeap[*dialWithBackoff](),

		alreadyInHeap:   map[string]bool{},
		attemptsPerPeer: map[string]uint64{},
	}

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
