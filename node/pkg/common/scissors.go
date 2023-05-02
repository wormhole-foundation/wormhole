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
	ScissorsErrorsCaught.WithLabelValues(name).Add(0)
	ScissorsPanicsCaught.WithLabelValues(name).Add(0)
	go func() {
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
		err := runnable(ctx)
		if err != nil {
			// We don't want this to hang if the listener has already gone away.
			select {
			case errC <- err:
			default:
			}
			ScissorsErrorsCaught.WithLabelValues(name).Inc()
		}
	}()
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
