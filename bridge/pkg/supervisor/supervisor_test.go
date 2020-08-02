package supervisor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
)

func runnableBecomesHealthy(healthy, done chan struct{}) Runnable {
	return func(ctx context.Context) error {
		Signal(ctx, SignalHealthy)

		go func() {
			if healthy != nil {
				healthy <- struct{}{}
			}
		}()

		<-ctx.Done()

		go func() {
			if done != nil {
				done <- struct{}{}
			}
		}()

		return ctx.Err()
	}
}

func runnableSpawnsMore(healthy, done chan struct{}, levels int) Runnable {
	return func(ctx context.Context) error {
		if levels > 0 {
			err := RunGroup(ctx, map[string]Runnable{
				"a": runnableSpawnsMore(nil, nil, levels-1),
				"b": runnableSpawnsMore(nil, nil, levels-1),
			})
			if err != nil {
				return err
			}
		}

		Signal(ctx, SignalHealthy)

		go func() {
			if healthy != nil {
				healthy <- struct{}{}
			}
		}()

		<-ctx.Done()

		go func() {
			if done != nil {
				done <- struct{}{}
			}
		}()
		return ctx.Err()
	}
}

// rc is a Remote Controlled runnable. It is a generic runnable used for testing the supervisor.
type rc struct {
	req chan rcRunnableRequest
}

type rcRunnableRequest struct {
	cmd    rcRunnableCommand
	stateC chan rcRunnableState
}

type rcRunnableCommand int

const (
	rcRunnableCommandBecomeHealthy rcRunnableCommand = iota
	rcRunnableCommandBecomeDone
	rcRunnableCommandDie
	rcRunnableCommandPanic
	rcRunnableCommandState
)

type rcRunnableState int

const (
	rcRunnableStateNew rcRunnableState = iota
	rcRunnableStateHealthy
	rcRunnableStateDone
)

func (r *rc) becomeHealthy() {
	r.req <- rcRunnableRequest{cmd: rcRunnableCommandBecomeHealthy}
}

func (r *rc) becomeDone() {
	r.req <- rcRunnableRequest{cmd: rcRunnableCommandBecomeDone}
}
func (r *rc) die() {
	r.req <- rcRunnableRequest{cmd: rcRunnableCommandDie}
}

func (r *rc) panic() {
	r.req <- rcRunnableRequest{cmd: rcRunnableCommandPanic}
}

func (r *rc) state() rcRunnableState {
	c := make(chan rcRunnableState)
	r.req <- rcRunnableRequest{
		cmd:    rcRunnableCommandState,
		stateC: c,
	}
	return <-c
}

func (r *rc) waitState(s rcRunnableState) {
	// This is poll based. Making it non-poll based would make the RC runnable logic a bit more complex for little gain.
	for {
		got := r.state()
		if got == s {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func newRC() *rc {
	return &rc{
		req: make(chan rcRunnableRequest),
	}
}

// Remote Controlled Runnable
func (r *rc) runnable() Runnable {
	return func(ctx context.Context) error {
		state := rcRunnableStateNew

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case r := <-r.req:
				switch r.cmd {
				case rcRunnableCommandBecomeHealthy:
					Signal(ctx, SignalHealthy)
					state = rcRunnableStateHealthy
				case rcRunnableCommandBecomeDone:
					Signal(ctx, SignalDone)
					state = rcRunnableStateDone
				case rcRunnableCommandDie:
					return fmt.Errorf("died on request")
				case rcRunnableCommandPanic:
					panic("at the disco")
				case rcRunnableCommandState:
					r.stateC <- state
				}
			}
		}
	}
}

func TestSimple(t *testing.T) {
	h1 := make(chan struct{})
	d1 := make(chan struct{})
	h2 := make(chan struct{})
	d2 := make(chan struct{})

	log, _ := zap.NewDevelopment()
	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()
	s := New(ctx, log, func(ctx context.Context) error {
		err := RunGroup(ctx, map[string]Runnable{
			"one": runnableBecomesHealthy(h1, d1),
			"two": runnableBecomesHealthy(h2, d2),
		})
		if err != nil {
			return err
		}
		Signal(ctx, SignalHealthy)
		Signal(ctx, SignalDone)
		return nil
	}, WithPropagatePanic)

	// Expect both to start running.
	s.waitSettleError(ctx, t)
	select {
	case <-h1:
	default:
		t.Fatalf("runnable 'one' didn't start")
	}
	select {
	case <-h2:
	default:
		t.Fatalf("runnable 'one' didn't start")
	}
}

func TestSimpleFailure(t *testing.T) {
	h1 := make(chan struct{})
	d1 := make(chan struct{})
	two := newRC()

	log, _ := zap.NewDevelopment()
	ctx, ctxC := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxC()
	s := New(ctx, log, func(ctx context.Context) error {
		err := RunGroup(ctx, map[string]Runnable{
			"one": runnableBecomesHealthy(h1, d1),
			"two": two.runnable(),
		})
		if err != nil {
			return err
		}
		Signal(ctx, SignalHealthy)
		Signal(ctx, SignalDone)
		return nil
	}, WithPropagatePanic)
	s.waitSettleError(ctx, t)

	two.becomeHealthy()
	s.waitSettleError(ctx, t)
	// Expect one to start running.
	select {
	case <-h1:
	default:
		t.Fatalf("runnable 'one' didn't start")
	}

	// Kill off two, one should restart.
	two.die()
	s.waitSettleError(ctx, t)
	select {
	case <-d1:
	default:
		t.Fatalf("runnable 'one' didn't acknowledge cancel")
	}

	// And one should start running again.
	s.waitSettleError(ctx, t)
	select {
	case <-h1:
	default:
		t.Fatalf("runnable 'one' didn't restart")
	}
}

func TestDeepFailure(t *testing.T) {
	h1 := make(chan struct{})
	d1 := make(chan struct{})
	two := newRC()

	log, _ := zap.NewDevelopment()

	ctx, ctxC := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxC()
	s := New(ctx, log, func(ctx context.Context) error {
		err := RunGroup(ctx, map[string]Runnable{
			"one": runnableSpawnsMore(h1, d1, 5),
			"two": two.runnable(),
		})
		if err != nil {
			return err
		}
		Signal(ctx, SignalHealthy)
		Signal(ctx, SignalDone)
		return nil
	}, WithPropagatePanic)

	two.becomeHealthy()
	s.waitSettleError(ctx, t)
	// Expect one to start running.
	select {
	case <-h1:
	default:
		t.Fatalf("runnable 'one' didn't start")
	}

	// Kill off two, one should restart.
	two.die()
	s.waitSettleError(ctx, t)
	select {
	case <-d1:
	default:
		t.Fatalf("runnable 'one' didn't acknowledge cancel")
	}

	// And one should start running again.
	s.waitSettleError(ctx, t)
	select {
	case <-h1:
	default:
		t.Fatalf("runnable 'one' didn't restart")
	}
}

func TestPanic(t *testing.T) {
	h1 := make(chan struct{})
	d1 := make(chan struct{})
	two := newRC()

	log, _ := zap.NewDevelopment()
	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()
	s := New(ctx, log, func(ctx context.Context) error {
		err := RunGroup(ctx, map[string]Runnable{
			"one": runnableBecomesHealthy(h1, d1),
			"two": two.runnable(),
		})
		if err != nil {
			return err
		}
		Signal(ctx, SignalHealthy)
		Signal(ctx, SignalDone)
		return nil
	})

	two.becomeHealthy()
	s.waitSettleError(ctx, t)
	// Expect one to start running.
	select {
	case <-h1:
	default:
		t.Fatalf("runnable 'one' didn't start")
	}

	// Kill off two, one should restart.
	two.panic()
	s.waitSettleError(ctx, t)
	select {
	case <-d1:
	default:
		t.Fatalf("runnable 'one' didn't acknowledge cancel")
	}

	// And one should start running again.
	s.waitSettleError(ctx, t)
	select {
	case <-h1:
	default:
		t.Fatalf("runnable 'one' didn't restart")
	}
}

func TestMultipleLevelFailure(t *testing.T) {
	log, _ := zap.NewDevelopment()
	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()
	New(ctx, log, func(ctx context.Context) error {
		err := RunGroup(ctx, map[string]Runnable{
			"one": runnableSpawnsMore(nil, nil, 4),
			"two": runnableSpawnsMore(nil, nil, 4),
		})
		if err != nil {
			return err
		}
		Signal(ctx, SignalHealthy)
		Signal(ctx, SignalDone)
		return nil
	}, WithPropagatePanic)
}

func TestBackoff(t *testing.T) {
	one := newRC()

	log, _ := zap.NewDevelopment()
	ctx, ctxC := context.WithTimeout(context.Background(), 20*time.Second)
	defer ctxC()

	s := New(ctx, log, func(ctx context.Context) error {
		if err := Run(ctx, "one", one.runnable()); err != nil {
			return err
		}
		Signal(ctx, SignalHealthy)
		Signal(ctx, SignalDone)
		return nil
	}, WithPropagatePanic)

	one.becomeHealthy()
	// Die a bunch of times in a row, this brings up the next exponential backoff to over a second.
	for i := 0; i < 4; i += 1 {
		one.die()
		one.waitState(rcRunnableStateNew)
	}
	// Measure how long it takes for the runnable to respawn after a number of failures
	start := time.Now()
	one.die()
	one.becomeHealthy()
	one.waitState(rcRunnableStateHealthy)
	taken := time.Since(start)
	if taken < 1*time.Second {
		t.Errorf("Runnable took %v to restart, wanted at least a second from backoff", taken)
	}

	s.waitSettleError(ctx, t)
	// Now that we've become healthy, die again. Becoming healthy resets the backoff.
	start = time.Now()
	one.die()
	one.becomeHealthy()
	one.waitState(rcRunnableStateHealthy)
	taken = time.Since(start)
	if taken > 1*time.Second || taken < 100*time.Millisecond {
		t.Errorf("Runnable took %v to restart, wanted at least 100ms from backoff and at most 1s from backoff reset", taken)
	}
}

// TestResilience throws some curveballs at the supervisor - either programming errors or high load. It then ensures
// that another runnable is running, and that it restarts on its sibling failure.
func TestResilience(t *testing.T) {
	// request/response channel for testing liveness of the 'one' runnable
	req := make(chan chan struct{})

	// A runnable that responds on the 'req' channel.
	one := func(ctx context.Context) error {
		Signal(ctx, SignalHealthy)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case r := <-req:
				r <- struct{}{}
			}
		}
	}
	oneSibling := newRC()

	oneTest := func() {
		timeout := time.NewTicker(1000 * time.Millisecond)
		ping := make(chan struct{})
		req <- ping
		select {
		case <-ping:
		case <-timeout.C:
			t.Fatalf("one ping response timeout")
		}
		timeout.Stop()
	}

	// A nasty runnable that calls Signal with the wrong context (this is a programming error)
	two := func(ctx context.Context) error {
		Signal(context.TODO(), SignalHealthy)
		return nil
	}

	// A nasty runnable that calls Signal wrong (this is a programming error).
	three := func(ctx context.Context) error {
		Signal(ctx, SignalDone)
		return nil
	}

	// A nasty runnable that runs in a busy loop (this is a programming error).
	four := func(ctx context.Context) error {
		for {
			time.Sleep(0)
		}
	}

	// A nasty runnable that keeps creating more runnables.
	five := func(ctx context.Context) error {
		i := 1
		for {
			err := Run(ctx, fmt.Sprintf("r%d", i), runnableSpawnsMore(nil, nil, 2))
			if err != nil {
				return err
			}

			time.Sleep(100 * time.Millisecond)
			i += 1
		}
	}

	log, _ := zap.NewDevelopment()
	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()
	New(ctx, log, func(ctx context.Context) error {
		RunGroup(ctx, map[string]Runnable{
			"one":        one,
			"oneSibling": oneSibling.runnable(),
		})
		rs := map[string]Runnable{
			"two": two, "three": three, "four": four, "five": five,
		}
		for k, v := range rs {
			if err := Run(ctx, k, v); err != nil {
				return err
			}
		}
		Signal(ctx, SignalHealthy)
		Signal(ctx, SignalDone)
		return nil
	})

	// Five rounds of letting one run, then restarting it.
	for i := 0; i < 5; i += 1 {
		oneSibling.becomeHealthy()
		oneSibling.waitState(rcRunnableStateHealthy)

		// 'one' should work for at least a second.
		deadline := time.Now().Add(1 * time.Second)
		for {
			if time.Now().After(deadline) {
				break
			}

			oneTest()
		}

		// Killing 'oneSibling' should restart one.
		oneSibling.panic()
	}
	// Make sure 'one' is still okay.
	oneTest()
}

func ExampleNew() {
	// Minimal runnable that is immediately done.
	childC := make(chan struct{})
	child := func(ctx context.Context) error {
		Signal(ctx, SignalHealthy)
		close(childC)
		Signal(ctx, SignalDone)
		return nil
	}

	log, _ := zap.NewDevelopment()

	// Start a supervision tree with a root runnable.
	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()
	New(ctx, log, func(ctx context.Context) error {
		err := Run(ctx, "child", child)
		if err != nil {
			return fmt.Errorf("could not run 'child': %w", err)
		}
		Signal(ctx, SignalHealthy)

		t := time.NewTicker(time.Second)
		defer t.Stop()

		// Do something in the background, and exit on context cancel.
		for {
			select {
			case <-t.C:
				fmt.Printf("tick!")
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	// root.child will close this channel.
	<-childC
}
