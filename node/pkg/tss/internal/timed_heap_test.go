package internal

import (
	"testing"
	"time"
)

type TestStruct struct {
	endTime time.Time
}

func (t TestStruct) GetEndTime() time.Time {
	return t.endTime
}

func TestFireEvenWithPassedTimes(t *testing.T) {
	// test that if we enqueue a bunch of elements, and wait to long to treat them, then it still fires for each one of them.

	heap := NewTtlHeap[TestStruct]()

	heap.Enqueue(TestStruct{endTime: time.Now().Add(time.Microsecond * 50)})
	heap.Enqueue(TestStruct{endTime: time.Now().Add(time.Millisecond * 100)})
	heap.Enqueue(TestStruct{endTime: time.Now().Add(time.Millisecond * 150)})

	time.Sleep(time.Millisecond * 200)

	failure := time.After(time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-heap.WaitOnTimer():
		case <-failure:
			t.FailNow()
		}

		heap.Dequeue()
	}
}
