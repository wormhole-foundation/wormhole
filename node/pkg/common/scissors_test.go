package common

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
