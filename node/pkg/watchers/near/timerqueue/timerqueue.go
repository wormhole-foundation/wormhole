// Package timerqueue implements a priority queue for objects scheduled at a
// particular time.
package timerqueue

import (
	"container/heap"
	"errors"
	"sync"
	"time"
)

type Timer interface{}

// Timerqueue is a time-sorted collection of Timer objects.
type Timerqueue struct {
	heap  timerHeap
	table map[Timer]*timerData
	mu    sync.Mutex
}

type timerData struct {
	timer Timer
	time  time.Time
	index int
}

// New creates a new timer priority queue.
func New() *Timerqueue {
	return &Timerqueue{
		table: make(map[Timer]*timerData),
	}
}

// Len returns the current number of timer objects in the queue.
func (q *Timerqueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.heap)
}

// Schedule schedules a timer for exectuion at time tm. If the
// timer was already scheduled, it is rescheduled.
func (q *Timerqueue) Schedule(t Timer, tm time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if data, ok := q.table[t]; !ok {
		data = &timerData{t, tm, 0}
		heap.Push(&q.heap, data)
		q.table[t] = data
	} else {
		data.time = tm
		heap.Fix(&q.heap, data.index)
	}
}

// Unschedule unschedules a timer's execution.
func (q *Timerqueue) Unschedule(t Timer) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if data, ok := q.table[t]; ok {
		heap.Remove(&q.heap, data.index)
		delete(q.table, t)
	}
}

// GetTime returns the time at which the timer is scheduled.
// If the timer isn't currently scheduled, an error is returned.
func (q *Timerqueue) GetTime(t Timer) (tm time.Time, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if data, ok := q.table[t]; ok {
		return data.time, nil
	}
	return time.Time{}, errors.New("timerqueue: timer not scheduled")
}

// IsScheduled returns true if the timer is currently scheduled.
func (q *Timerqueue) IsScheduled(t Timer) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.table[t]
	return ok
}

// Clear unschedules all currently scheduled timers.
func (q *Timerqueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.heap, q.table = nil, make(map[Timer]*timerData)
}

// PopFirst removes and returns the next timer to be scheduled and
// the time at which it is scheduled to run.
func (q *Timerqueue) PopFirst() (t Timer, tm time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.heap) > 0 {
		data := heap.Pop(&q.heap).(*timerData)
		delete(q.table, data.timer)
		return data.timer, data.time
	}
	return nil, time.Time{}
}

// PopFirstIfReady removes and returns the next timer *if* it is ready
// the time at which it is scheduled to run.
func (q *Timerqueue) PopFirstIfReady() (Timer, time.Time, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.heap) > 0 {
		if q.heap[0].time.After(time.Now()) {
			// first job is ready. Pop it.
			data := heap.Pop(&q.heap).(*timerData)
			delete(q.table, data.timer)
			return data.timer, data.time, nil
		}
	}
	return nil, time.Time{}, errors.New("no job ready")
}

// PeekFirst returns the next timer to be scheduled and the time
// at which it is scheduled to run. It does not modify the contents
// of the timer queue.
func (q *Timerqueue) PeekFirst() (t Timer, tm time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.heap) > 0 {
		return q.heap[0].timer, q.heap[0].time
	}
	return nil, time.Time{}
}

/*
 * timerHeap
 */

type timerHeap []*timerData

func (h timerHeap) Len() int {
	return len(h)
}

func (h timerHeap) Less(i, j int) bool {
	return h[i].time.Before(h[j].time)
}

func (h timerHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}

func (h *timerHeap) Push(x interface{}) {
	data := x.(*timerData)
	*h = append(*h, data)
	data.index = len(*h) - 1
}

func (h *timerHeap) Pop() interface{} {
	n := len(*h)
	data := (*h)[n-1]
	*h = (*h)[:n-1]
	data.index = -1
	return data
}
