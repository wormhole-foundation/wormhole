package common

import "context"

func RunWithScissors(ctx context.Context, errC chan error, name string, runnable func() error) {
	if err := runnable(); err != nil {
		select {
		case errC <- err:
		default:
		}
	}
	_ = ctx
	_ = name
}

func StartRunnable(errC chan error, runnable func() error) {
	if err := runnable(); err != nil {
		select {
		case errC <- err:
		default:
		}
	}
}
