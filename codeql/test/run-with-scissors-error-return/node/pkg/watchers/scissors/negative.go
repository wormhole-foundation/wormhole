package scissors

import (
	"context"
	"errors"
	"fmt"

	"github.com/wormhole-foundation/wormhole/node/pkg/common"
)

func returnedErrorOnly(ctx context.Context, errC chan error) {
	common.RunWithScissors(ctx, errC, "return", func() error {
		return fmt.Errorf("return failed: %w", errors.New("boom"))
	})
}

func siblingReadOnly(ctx context.Context, errC chan error) {
	common.RunWithScissors(ctx, errC, "read", func() error {
		select {
		case err := <-errC:
			return fmt.Errorf("sibling failed: %w", err)
		default:
			return nil
		}
	})
}

func unrelatedLocalChannel(ctx context.Context, errC chan error) {
	otherErrC := make(chan error, 1)
	common.RunWithScissors(ctx, errC, "unrelated-local", func() error {
		otherErrC <- errors.New("not wrapper channel")
		return nil
	})
}

func unrelatedReceiverField(ctx context.Context, w *watcher) {
	common.RunWithScissors(ctx, w.errC, "unrelated-field", func() error {
		w.metricsErrC <- errors.New("metrics only")
		return nil
	})
}

func exactReceiverOnly(ctx context.Context, owner *watcher, other *watcher) {
	common.RunWithScissors(ctx, owner.errC, "exact-receiver", other.run)
}

func unrelatedHelperArgument(ctx context.Context, errC chan error, otherErrC chan error) {
	common.RunWithScissors(ctx, errC, "unrelated-helper", func() error {
		return reportOnChannel(otherErrC, errors.New("other helper channel"))
	})
}

func wrapperOwnedForwarding(errC chan error, runnable func() error) {
	if err := runnable(); err != nil {
		select {
		case errC <- err:
		default:
		}
	}
}

func thinWrapperNoSend(ctx context.Context, errC chan error, name string, runnable func() error) {
	common.RunWithScissors(ctx, errC, name, runnable)
}

func callerThroughUnsupportedThinWrapper(ctx context.Context, errC chan error) {
	thinWrapperNoSend(ctx, errC, "unsupported-wrapper", func() error {
		errC <- errors.New("currently unsupported wrapper boundary")
		return nil
	})
}

func localFunctionValueAssignedAfterRunCall(ctx context.Context, errC chan error) {
	var run func() error
	common.RunWithScissors(ctx, errC, "assigned-after-call", run)
	run = func() error {
		errC <- errors.New("assignment happens after RunWithScissors")
		return nil
	}
}

func localFunctionValueOverwrittenBeforeRunCall(ctx context.Context, errC chan error) {
	run := func() error {
		errC <- errors.New("overwritten before RunWithScissors")
		return nil
	}
	run = func() error {
		return errors.New("current runnable returns through RunWithScissors")
	}
	common.RunWithScissors(ctx, errC, "overwritten-before-call", run)
}

func goLaunchedHelperOutOfScope(ctx context.Context, errC chan error) {
	common.RunWithScissors(ctx, errC, "go-helper", func() error {
		go reportOnChannel(errC, errors.New("async helper is out of scope"))
		return nil
	})
}
