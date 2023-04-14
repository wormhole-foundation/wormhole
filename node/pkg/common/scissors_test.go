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

	itRan := false
	RunWithScissors(ctx, errC, "TestRunWithScissorsCleanExit", func(ctx context.Context) error {
		itRan = true
		return nil
	})

	time.Sleep(100 * time.Millisecond)
	require.Equal(t, true, itRan)
	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsCleanExit"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsCleanExit"))
}

func TestRunWithScissorsPanicReturned(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := false
	RunWithScissors(ctx, errC, "TestRunWithScissorsPanicReturned", func(ctx context.Context) error {
		itRan = true
		panic("Some random panic")
	})

	var err error
	select {
	case <-ctx.Done():
		break
	case err = <-errC:
		break
	}

	require.Equal(t, true, itRan)
	assert.Error(t, err)
	assert.Equal(t, "TestRunWithScissorsPanicReturned: Some random panic", err.Error())
	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsPanicReturned"))
	assert.Equal(t, 1.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsPanicReturned"))
}

func TestRunWithScissorsPanicDoesNotBlockWhenNoListener(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := false
	RunWithScissors(ctx, errC, "TestRunWithScissorsPanicDoesNotBlockWhenNoListener", func(ctx context.Context) error {
		itRan = true
		panic("Some random panic")
	})

	time.Sleep(100 * time.Millisecond)
	require.Equal(t, true, itRan)
	assert.Equal(t, 0.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsPanicDoesNotBlockWhenNoListener"))
	assert.Equal(t, 1.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsPanicDoesNotBlockWhenNoListener"))
}

func TestRunWithScissorsErrorReturned(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := false
	RunWithScissors(ctx, errC, "TestRunWithScissorsErrorReturned", func(ctx context.Context) error {
		itRan = true
		return fmt.Errorf("Some random error")
	})

	var err error
	select {
	case <-ctx.Done():
		break
	case err = <-errC:
		break
	}

	require.Equal(t, true, itRan)
	assert.Error(t, err)
	assert.Equal(t, "Some random error", err.Error())
	assert.Equal(t, 1.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsErrorReturned"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsErrorReturned"))
}

func TestRunWithScissorsErrorDoesNotBlockWhenNoListener(t *testing.T) {
	ctx := context.Background()
	errC := make(chan error)

	itRan := false
	RunWithScissors(ctx, errC, "TestRunWithScissorsErrorDoesNotBlockWhenNoListener", func(ctx context.Context) error {
		itRan = true
		return fmt.Errorf("Some random error")
	})

	time.Sleep(100 * time.Millisecond)
	require.Equal(t, true, itRan)
	assert.Equal(t, 1.0, getCounterValue(ScissorsErrorsCaught, "TestRunWithScissorsErrorDoesNotBlockWhenNoListener"))
	assert.Equal(t, 0.0, getCounterValue(ScissorsPanicsCaught, "TestRunWithScissorsErrorDoesNotBlockWhenNoListener"))
}
