package common

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func getCounterValue(metric *prometheus.CounterVec, runnableName string) float64 {
	var m = &dto.Metric{}
	if err := metric.WithLabelValues(runnableName).Write(m); err != nil {
		return 0
	}
	return m.Counter.GetValue()
}

func throwNil(ctx context.Context) error {
	var x *int = nil
	*x = 5
	return nil
}

func runTest(t *testing.T, ctx context.Context, testCase int) (result error) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				result = errors.New(x)
			case error:
				result = x
			default:
				result = fmt.Errorf("unknown panic in runTest/%d", testCase)
			}
		}
	}()

	errC := make(chan error)

	switch testCase {
	case 0:
		_ = throwNil(ctx) // fall into defer above
	case 1:
		RunWithScissors(ctx, errC, "test1Thread", throwNil)
	case 2:
		RunWithScissors(ctx, errC, "test2Thread", func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		})
		_ = throwNil(ctx)

	case 3:
		go func() { _ = throwNil(ctx) }() // uncatchable
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

func TestSupervisor(t *testing.T) {
	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			rootCtx := context.Background()
			ctx, fn := context.WithCancel(rootCtx)

			err := runTest(t, ctx, i)

			switch i {
			case 0:
				assert.EqualError(t, err, "runtime error: invalid memory address or nil pointer dereference")
			case 1:
				assert.EqualError(t, err, "test1Thread: runtime error: invalid memory address or nil pointer dereference")
			case 2:
				assert.EqualError(t, err, "runtime error: invalid memory address or nil pointer dereference")
			}
			fn()
		},
		)
	}
}

func TestRunWithScissorsCleanExit(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	RunWithScissors(ctx, errC, "TestRunWithScissorsCleanExit", func(ctx context.Context) error {
		itRan <- true
		return nil
	})

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)

	// Need to wait a bit to make sure the scissors code completes without hanging.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsCleanExit"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsCleanExit"))
}

func TestRunWithScissorsPanicReturned(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	RunWithScissors(ctx, errC, "TestRunWithScissorsPanicReturned", func(ctx context.Context) error {
		itRan <- true
		panic("Some random panic")
	})

	var err error
	select {
	case <-ctx.Done():
		break
	case err = <-errC:
		break
	}

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)
	assert.Error(t, err)
	assert.Equal(t, "TestRunWithScissorsPanicReturned: Some random panic", err.Error())
	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsPanicReturned"))
	assert.Equal(t, 1.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsPanicReturned"))
}

func TestRunWithScissorsPanicDoesNotBlockWhenNoListener(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	RunWithScissors(ctx, errC, "TestRunWithScissorsPanicDoesNotBlockWhenNoListener", func(ctx context.Context) error {
		itRan <- true
		panic("Some random panic")
	})

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)

	// Need to wait a bit to make sure the scissors code completes without hanging.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsPanicDoesNotBlockWhenNoListener"))
	assert.Equal(t, 1.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsPanicDoesNotBlockWhenNoListener"))
}

func TestRunWithScissorsErrorReturned(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	RunWithScissors(ctx, errC, "TestRunWithScissorsErrorReturned", func(ctx context.Context) error {
		itRan <- true
		return fmt.Errorf("Some random error")
	})

	var err error
	select {
	case <-ctx.Done():
		break
	case err = <-errC:
		break
	}

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)
	assert.Error(t, err)
	assert.Equal(t, "Some random error", err.Error())
	assert.Equal(t, 1.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsErrorReturned"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsErrorReturned"))
}

func TestRunWithScissorsErrorDoesNotBlockWhenNoListener(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	RunWithScissors(ctx, errC, "TestRunWithScissorsErrorDoesNotBlockWhenNoListener", func(ctx context.Context) error {
		itRan <- true
		return fmt.Errorf("Some random error")
	})

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)

	// Need to wait a bit to make sure the scissors code completes without hanging.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsErrorDoesNotBlockWhenNoListener"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsErrorDoesNotBlockWhenNoListener"))
}

func TestStartRunnable_CleanExit(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	StartRunnable(ctx, errC, true, "TestStartRunnable_CleanExit", func(ctx context.Context) error {
		itRan <- true
		return nil
	})

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)

	// Need to wait a bit to make sure the scissors code completes without hanging.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestStartRunnable_CleanExit"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestStartRunnable_CleanExit"))
}

func TestStartRunnable_OnError(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	StartRunnable(ctx, errC, true, "TestStartRunnable_OnError", func(ctx context.Context) error {
		itRan <- true
		return fmt.Errorf("Some random error")
	})

	var err error
	select {
	case <-ctx.Done():
		break
	case err = <-errC:
		break
	}

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)
	assert.Error(t, err)
	assert.Equal(t, "Some random error", err.Error())
	assert.Equal(t, 1.0, getCounterValue(ScissorsErrorsCaught, "TestStartRunnable_OnError"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestStartRunnable_OnError"))
}

func TestStartRunnable_DontCatchPanics_OnPanic(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	itPanicked := make(chan bool, 1)

	// We can't use StartRunnable() because we cannot test for a panic in another go routine.
	// This verifies that startRunnable() lets the panic through so it gets caught here, allowing us to test for it.
	func() {
		defer func() {
			if r := recover(); r != nil {
				itPanicked <- true
			}
			itRan <- true
		}()

		startRunnable(ctx, errC, "TestStartRunnable_DontCatchPanics_OnPanic", func(ctx context.Context) error {
			panic("Some random panic")
		})
	}()

	var shouldHaveRun bool
	select {
	case <-ctx.Done():
		break
	case shouldHaveRun = <-itRan:
		break
	}

	require.Equal(t, true, shouldHaveRun)

	require.Equal(t, 1, len(itPanicked))
	shouldHavePanicked := <-itPanicked
	require.Equal(t, true, shouldHavePanicked)

	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestStartRunnable_DontCatchPanics_OnPanic"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestStartRunnable_DontCatchPanics_OnPanic"))
}

func TestStartRunnable_CatchPanics_OnPanic(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	StartRunnable(ctx, errC, true, "TestStartRunnable_CatchPanics_OnPanic", func(ctx context.Context) error {
		itRan <- true
		panic("Some random panic")
	})

	var err error
	select {
	case <-ctx.Done():
		break
	case err = <-errC:
		break
	}

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)
	assert.Error(t, err)
	assert.Equal(t, "TestStartRunnable_CatchPanics_OnPanic: Some random panic", err.Error())
	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestStartRunnable_CatchPanics_OnPanic"))
	assert.Equal(t, 1.0, getCounterValue(ScissorsPanicsCaught, "TestStartRunnable_CatchPanics_OnPanic"))
}

func TestStartRunnable_DoesNotBlockWhenNoListener(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := make(chan bool, 1)
	StartRunnable(ctx, errC, true, "TestStartRunnable_DoesNotBlockWhenNoListener", func(ctx context.Context) error {
		itRan <- true
		panic("Some random panic")
	})

	shouldHaveRun := <-itRan
	require.Equal(t, true, shouldHaveRun)

	// Need to wait a bit to make sure the scissors code completes without hanging.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestStartRunnable_DoesNotBlockWhenNoListener"))
	assert.Equal(t, 1.0, getCounterValue(ScissorsPanicsCaught, "TestStartRunnable_DoesNotBlockWhenNoListener"))
}
