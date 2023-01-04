package common

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ScissorsErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scissor_errors_caught",
			Help: "Total number of unhandled errors caught",
		}, []string{"scissors", "name"})
)

// Start a go routine with recovering from any panic by sending an error to a error channel
func RunWithScissors(ctx context.Context, errC chan error, name string, runnable supervisor.Runnable) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				switch x := r.(type) {
				case error:
					errC <- fmt.Errorf("%s: %w", name, x)
				default:
					errC <- fmt.Errorf("%s: %v", name, x)
				}
				ScissorsErrors.WithLabelValues("scissors",  name).Inc()

			}
		}()
		err := runnable(ctx)
		if err != nil {
			errC <- err
		}
	}()
}

func WrapWithScissors(runnable supervisor.Runnable, name string) supervisor.Runnable {
	return func(ctx context.Context) (result error) {
		defer func() {
			if r := recover(); r != nil {
				switch x := r.(type) {
				case error:
					result = fmt.Errorf("%s: %w", name, x)
				default:
					result = fmt.Errorf("%s: %v", name, x)
				}
				ScissorsErrors.WithLabelValues("scissors",  name).Inc()
			}
		}()

		return runnable(ctx)
	}
}
