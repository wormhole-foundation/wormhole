package supervisor

import (
	"context"
	"testing"
)

// waitSettle waits until the supervisor reaches a 'settled' state - ie., one
// where no actions have been performed for a number of GC cycles.
// This is used in tests only.
func (s *supervisor) waitSettle(ctx context.Context) error {
	waiter := make(chan struct{})
	s.pReq <- &processorRequest{
		waitSettled: &processorRequestWaitSettled{
			waiter: waiter,
		},
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waiter:
		return nil
	}
}

// waitSettleError wraps waitSettle to fail a test if an error occurs, eg. the
// context is canceled.
func (s *supervisor) waitSettleError(ctx context.Context, t *testing.T) {
	err := s.waitSettle(ctx)
	if err != nil {
		t.Fatalf("waitSettle: %v", err)
	}
}
