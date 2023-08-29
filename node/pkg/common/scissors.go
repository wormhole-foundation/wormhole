package common

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ScissorsErrorsCaught = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scissor_errors_caught",
			Help: "Total number of unhandled errors caught",
		}, []string{"name"})
	ScissorsPanicsCaught = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scissor_panics_caught",
			Help: "Total number of panics caught",
		}, []string{"name"})
)

// Start a go routine with recovering from any panic by sending an error to a error channel
func RunWithScissors(ctx context.Context, errC chan error, name string, runnable supervisor.Runnable) {
	StartRunnable(ctx, errC, true, name, runnable)
}

func WrapWithScissors(runnable supervisor.Runnable, name string) supervisor.Runnable {
	ScissorsErrorsCaught.WithLabelValues(name).Add(0)
	ScissorsPanicsCaught.WithLabelValues(name).Add(0)
	return func(ctx context.Context) (result error) {
		defer func() {
			if r := recover(); r != nil {
				switch x := r.(type) {
				case error:
					result = fmt.Errorf("%s: %w", name, x)
				default:
					result = fmt.Errorf("%s: %v", name, x)
				}
				ScissorsPanicsCaught.WithLabelValues(name).Inc()
			}
		}()

		return runnable(ctx)
	}
}

// StartRunnable starts a go routine with the ability to recover from errors by publishing them to an error channel. If catchPanics is true,
// it will also catch panics and publish the panic message to the error channel. If catchPanics is false, the panic will be propagated upward.
func StartRunnable(ctx context.Context, errC chan error, catchPanics bool, name string, runnable supervisor.Runnable) {
	ScissorsErrorsCaught.WithLabelValues(name).Add(0)
	if catchPanics {
		ScissorsPanicsCaught.WithLabelValues(name).Add(0)
	}
	go func() {
		if catchPanics {
			defer func() {
				if r := recover(); r != nil {
					var err error
					switch x := r.(type) {
					case error:
						err = fmt.Errorf("%s: %w", name, x)
					default:
						err = fmt.Errorf("%s: %v", name, x)
					}
					// We don't want this to hang if the listener has already gone away.
					select {
					case errC <- err:
					default:
					}
					ScissorsPanicsCaught.WithLabelValues(name).Inc()

				}
			}()
		}
		startRunnable(ctx, errC, name, runnable)
	}()
}

// startRunnable is used by StartRunnable. It is a separate function so we can call it directly from tests.
func startRunnable(ctx context.Context, errC chan error, name string, runnable supervisor.Runnable) {
	err := runnable(ctx)
	if err != nil {
		// We don't want this to hang if the listener has already gone away.
		select {
		case errC <- err:
		default:
		}
		ScissorsErrorsCaught.WithLabelValues(name).Inc()
	}
}
