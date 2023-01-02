package common

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ScissorsErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "scissor_errors_caught",
			Help: "Total number of unhandled errors caught",
		})
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
				ScissorsErrors.Inc()
			}
		}()
		err := runnable(ctx)
		if err != nil {
			errC <- err
		}
	}()
}

type (
	Scissors struct {
		runnable supervisor.Runnable
	}
)

func WrapWithScissors(runnable supervisor.Runnable) supervisor.Runnable {
	s := Scissors{runnable: runnable}
	return s.Run
}

func (e *Scissors) Run(ctx context.Context) (result error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case error:
				result = x
			default:
				result = fmt.Errorf("%v", x)
			}
			ScissorsErrors.Inc()
		}
	}()

	return e.runnable(ctx)
}
