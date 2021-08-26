package supervisor

// The service supervision library allows for writing of reliable, service-style software within SignOS.
// It builds upon the Erlang/OTP supervision tree system, adapted to be more Go-ish.
// For detailed design see go/supervision.

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// A Runnable is a function that will be run in a goroutine, and supervised throughout its lifetime. It can in turn
// start more runnables as its children, and those will form part of a supervision tree.
// The context passed to a runnable is very important and needs to be handled properly. It will be live (non-errored) as
// long as the runnable should be running, and canceled (ctx.Err() will be non-nil) when the supervisor wants it to
// exit. This means this context is also perfectly usable for performing any blocking operations.
type Runnable func(ctx context.Context) error

// RunGroup starts a set of runnables as a group. These runnables will run together, and if any one of them quits
// unexpectedly, the result will be canceled and restarted.
// The context here must be an existing Runnable context, and the spawned runnables will run under the node that this
// context represents.
func RunGroup(ctx context.Context, runnables map[string]Runnable) error {
	node, unlock := fromContext(ctx)
	defer unlock()
	return node.runGroup(runnables)
}

// Run starts a single runnable in its own group.
func Run(ctx context.Context, name string, runnable Runnable) error {
	return RunGroup(ctx, map[string]Runnable{
		name: runnable,
	})
}

// Signal tells the supervisor that the calling runnable has reached a certain state of its lifecycle. All runnables
// should SignalHealthy when they are ready with set up, running other child runnables and are now 'serving'.
func Signal(ctx context.Context, signal SignalType) {
	node, unlock := fromContext(ctx)
	defer unlock()
	node.signal(signal)
}

type SignalType int

const (
	// The runnable is healthy, done with setup, done with spawning more Runnables, and ready to serve in a loop.
	// The runnable needs to check the parent context and ensure that if that context is done, the runnable exits.
	SignalHealthy SignalType = iota
	// The runnable is done - it does not need to run any loop. This is useful for Runnables that only set up other
	// child runnables. This runnable will be restarted if a related failure happens somewhere in the supervision tree.
	SignalDone
)

// Logger returns a Zap logger that will be named after the Distinguished Name of a the runnable (ie its place in the
// supervision tree, dot-separated).
func Logger(ctx context.Context) *zap.Logger {
	node, unlock := fromContext(ctx)
	defer unlock()
	return node.getLogger()
}

// supervisor represents and instance of the supervision system. It keeps track of a supervision tree and a request
// channel to its internal processor goroutine.
type supervisor struct {
	// mu guards the entire state of the supervisor.
	mu sync.RWMutex
	// root is the root node of the supervision tree, named 'root'. It represents the Runnable started with the
	// supervisor.New call.
	root *node
	// logger is the Zap logger used to create loggers available to runnables.
	logger *zap.Logger
	// ilogger is the Zap logger used for internal logging by the supervisor.
	ilogger *zap.Logger

	// pReq is an interface channel to the lifecycle processor of the supervisor.
	pReq chan *processorRequest

	// propagate panics, ie. don't catch them.
	propagatePanic bool
}

// SupervisorOpt are runtime configurable options for the supervisor.
type SupervisorOpt func(s *supervisor)

var (
	// WithPropagatePanic prevents the Supervisor from catching panics in runnables and treating them as failures.
	// This is useful to enable for testing and local debugging.
	WithPropagatePanic = func(s *supervisor) {
		s.propagatePanic = true
	}
)

// New creates a new supervisor with its root running the given root runnable.
// The given context can be used to cancel the entire supervision tree.
func New(ctx context.Context, logger *zap.Logger, rootRunnable Runnable, opts ...SupervisorOpt) *supervisor {
	sup := &supervisor{
		logger:  logger,
		ilogger: logger.Named("supervisor"),
		pReq:    make(chan *processorRequest),
	}

	for _, o := range opts {
		o(sup)
	}

	sup.root = newNode("root", rootRunnable, sup, nil)

	go sup.processor(ctx)

	sup.pReq <- &processorRequest{
		schedule: &processorRequestSchedule{dn: "root"},
	}

	return sup
}
