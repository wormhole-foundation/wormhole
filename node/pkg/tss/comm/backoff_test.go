package comm

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestBackoffRepeats(t *testing.T) {
	waiters := newBackoffHeap()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	waiters.Enqueue("test1")

	for range 7 {
		dialTo := ""

		select {
		case <-waiters.WaitOnTimer(): // waiting on blocker
		case <-ctx.Done():
			t.FailNow()

			return
		}
		dialTo = waiters.Dequeue()

		if dialTo == "" {
			continue // skip (nothing to dial to)
		}

		fmt.Println("Oh no! failed to dial to peer!", dialTo)
		waiters.Enqueue(dialTo) // ensuring a retry.
	}

}
