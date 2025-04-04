package supervisor

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
)

// The processor maintains runnable goroutines - ie., when requested will start one, and then once it exists it will
// record the result and act accordingly. It is also responsible for detecting and acting upon supervision subtrees that
// need to be restarted after death (via a 'GC' process)

// processorRequest is a request for the processor. Only one of the fields can be set.
type processorRequest struct {
	schedule    *processorRequestSchedule
	died        *processorRequestDied
	waitSettled *processorRequestWaitSettled
}

// processorRequestSchedule requests that a given node's runnable be started.
type processorRequestSchedule struct {
	dn string
}

// processorRequestDied is a signal from a runnable goroutine that the runnable has died.
type processorRequestDied struct {
	dn  string
	err error
}

type processorRequestWaitSettled struct {
	waiter chan struct{}
}

// processor is the main processing loop.
func (s *supervisor) processor(ctx context.Context) {
	s.ilogger.Info("supervisor processor started")

	// Waiters waiting for the GC to be settled.
	var waiters []chan struct{}

	// The GC will run every millisecond if needed. Any time the processor requests a change in the supervision tree
	// (ie a death or a new runnable) it will mark the state as dirty and run the GC on the next millisecond cycle.
	gc := time.NewTicker(1 * time.Millisecond)
	defer gc.Stop()
	clean := true

	// How long has the GC been clean. This is used to notify 'settled' waiters.
	cleanCycles := 0

	markDirty := func() {
		clean = false
		cleanCycles = 0
	}

	for {
		select {
		case <-ctx.Done():
			s.ilogger.Info("supervisor processor exiting...", zap.Error(ctx.Err()))
			s.processKill()
			s.ilogger.Info("supervisor exited")
			return
		case <-gc.C:
			if !clean {
				s.processGC()
			}
			clean = true
			cleanCycles += 1

			// This threshold is somewhat arbitrary. It's a balance between test speed and test reliability.
			if cleanCycles > 50 {
				for _, w := range waiters {
					close(w)
				}
				waiters = nil
			}
		case r := <-s.pReq:
			switch {
			case r.schedule != nil:
				s.processSchedule(r.schedule)
				markDirty()
			case r.died != nil:
				s.processDied(r.died)
				markDirty()
			case r.waitSettled != nil:
				waiters = append(waiters, r.waitSettled.waiter)
			default:
				panic(fmt.Errorf("unhandled request %+v", r))
			}
		}
	}
}

// processKill cancels all nodes in the supervision tree. This is only called right before exiting the processor, so
// they do not get automatically restarted.
func (s *supervisor) processKill() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Gather all context cancel functions.
	var cancels []func()
	queue := []*node{s.root}
	for {
		if len(queue) == 0 {
			break
		}

		cur := queue[0]
		queue = queue[1:]

		cancels = append(cancels, cur.ctxC)
		for _, c := range cur.children {
			queue = append(queue, c)
		}
	}

	// Call all context cancels.
	for _, c := range cancels {
		c()
	}
}

// processSchedule starts a node's runnable in a goroutine and records its output once it's done.
func (s *supervisor) processSchedule(r *processorRequestSchedule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := s.nodeByDN(r.dn)
	go func() {
		if !s.propagatePanic {
			defer func() {
				if rec := recover(); rec != nil {
					s.pReq <- &processorRequest{
						died: &processorRequestDied{
							dn:  r.dn,
							err: fmt.Errorf("panic: %v, stacktrace: %s", rec, string(debug.Stack())),
						},
					}
				}
			}()
		}

		res := n.runnable(n.ctx)

		s.pReq <- &processorRequest{
			died: &processorRequestDied{
				dn:  r.dn,
				err: res,
			},
		}
	}()
}

// processDied records the result from a runnable goroutine, and updates its node state accordingly. If the result
// is a death and not an expected exit, related nodes (ie. children and group siblings) are canceled accordingly.
func (s *supervisor) processDied(r *processorRequestDied) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Okay, so a Runnable has quit. What now?
	n := s.nodeByDN(r.dn)
	ctx := n.ctx

	// Simple case: it was marked as Done and quit with no error.
	if n.state == nodeStateDone && r.err == nil {
		// Do nothing. This was supposed to happen. Keep the process as DONE.
		return
	}

	// Find innermost error to check if it's a context canceled error.
	perr := r.err
	for {
		if inner := errors.Unwrap(perr); inner != nil {
			perr = inner
			continue
		}
		break
	}

	// Simple case: the context was canceled and the returned error is the context error.
	//nolint:errorlint // Unwrapping of error handled above
	if err := ctx.Err(); err != nil && perr == err {
		// Mark the node as canceled successfully.
		n.state = nodeStateCanceled
		return
	}

	// Otherwise, the Runnable should not have died or quit. Handle accordingly.
	err := r.err
	// A lack of returned error is also an error.
	if err == nil {
		err = fmt.Errorf("returned when %s", n.state)
	} else {
		err = fmt.Errorf("returned error when %s: %w", n.state, err)
	}

	s.ilogger.Error("Runnable died", zap.String("dn", n.dn()), zap.Error(err))
	// Mark as dead.
	n.state = nodeStateDead

	// Cancel that node's context, just in case something still depends on it.
	n.ctxC()

	// Cancel all siblings.
	if n.parent != nil {
		for name := range n.parent.groupSiblings(n.name) {
			if name == n.name {
				continue
			}
			sibling := n.parent.children[name]
			// TODO(q3k): does this need to run in a goroutine, ie. can a context cancel block?
			sibling.ctxC()
		}
	}
}

// processGC runs the GC process. It's not really Garbage Collection, as in, it doesn't remove unnecessary tree nodes -
// but it does find nodes that need to be restarted, find the subset that can and then schedules them for running.
// As such, it's less of a Garbage Collector and more of a Necromancer. However, GC is a friendlier name.
func (s *supervisor) processGC() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// The 'GC' serves is the main business logic of the supervision tree. It traverses a locked tree and tries to
	// find subtrees that must be restarted (because of a DEAD/CANCELED runnable). It then finds which of these
	// subtrees that should be restarted can be restarted, ie. which ones are fully recursively DEAD/CANCELED. It
	// also finds the smallest set of largest subtrees that can be restarted, ie. if there's multiple DEAD runnables
	// that can be restarted at once, it will do so.

	// Phase one: Find all leaves.
	// This is a simple DFS that finds all the leaves of the tree, ie all nodes that do not have children nodes.
	leaves := make(map[string]bool)
	queue := []*node{s.root}
	for {
		if len(queue) == 0 {
			break
		}
		cur := queue[0]
		queue = queue[1:]

		for _, c := range cur.children {
			queue = append([]*node{c}, queue...)
		}

		if len(cur.children) == 0 {
			leaves[cur.dn()] = true
		}
	}

	// Phase two: traverse tree from node to root and make note of all subtrees that can be restarted.
	// A subtree is restartable/ready iff every node in that subtree is either CANCELED, DEAD or DONE.
	// Such a 'ready' subtree can be restarted by the supervisor if needed.

	// DNs that we already visited.
	visited := make(map[string]bool)
	// DNs whose subtrees are ready to be restarted.
	// These are all subtrees recursively - ie., root.a.a and root.a will both be marked here.
	ready := make(map[string]bool)

	// We build a queue of nodes to visit, starting from the leaves.
	queue = []*node{}
	for l := range leaves {
		queue = append(queue, s.nodeByDN(l))
	}

	for {
		if len(queue) == 0 {
			break
		}

		cur := queue[0]
		curDn := cur.dn()

		queue = queue[1:]

		// Do we have a decision about our children?
		allVisited := true
		for _, c := range cur.children {
			if !visited[c.dn()] {
				allVisited = false
				break
			}
		}

		// If no decision about children is available, it means we ended up in this subtree through some shorter path
		// of a shorter/lower-order leaf. There is a path to a leaf that's longer than the one that caused this node
		// to be enqueued. Easy solution: just push back the current element and retry later.
		if !allVisited {
			// Push back to queue and wait for a decision later.
			queue = append(queue, cur)
			continue
		}

		// All children have been visited and we have an idea about whether they're ready/restartable. All of the node's
		// children must be restartable in order for this node to be restartable.
		childrenReady := true
		for _, c := range cur.children {
			if !ready[c.dn()] {
				childrenReady = false
				break
			}
		}

		// In addition to children, the node itself must be restartable (ie. DONE, DEAD or CANCELED).
		curReady := false
		switch cur.state {
		case nodeStateDone:
			curReady = true
		case nodeStateCanceled:
			curReady = true
		case nodeStateDead:
			curReady = true
		case nodeStateHealthy:
			curReady = false
		case nodeStateNew:
			curReady = false
		}

		// Note down that we have an opinion on this node, and note that opinion down.
		visited[curDn] = true
		ready[curDn] = childrenReady && curReady

		// Now we can also enqueue the parent of this node for processing.
		if cur.parent != nil && !visited[cur.parent.dn()] {
			queue = append(queue, cur.parent)
		}
	}

	// Phase 3: traverse tree from root to find largest subtrees that need to be restarted and are ready to be
	// restarted.

	// All DNs that need to be restarted by the GC process.
	want := make(map[string]bool)
	// All DNs that need to be restarted and can be restarted by the GC process - a subset of 'want' DNs.
	can := make(map[string]bool)
	// The set difference between 'want' and 'can' are all nodes that should be restarted but can't yet (ie. because
	// a child is still in the process of being canceled).

	// DFS from root.
	queue = []*node{s.root}
	for {
		if len(queue) == 0 {
			break
		}

		cur := queue[0]
		queue = queue[1:]

		// If this node is DEAD or CANCELED it should be restarted.
		if cur.state == nodeStateDead || cur.state == nodeStateCanceled {
			want[cur.dn()] = true
		}

		// If it should be restarted and is ready to be restarted...
		if want[cur.dn()] && ready[cur.dn()] {
			// And its parent context is valid (ie hasn't been canceled), mark it as restartable.
			if cur.parent == nil || cur.parent.ctx.Err() == nil {
				can[cur.dn()] = true
				continue
			}
		}

		// Otherwise, traverse further down the tree to see if something else needs to be done.
		for _, c := range cur.children {
			queue = append(queue, c)
		}
	}

	// Reinitialize and reschedule all subtrees
	for dn := range can {
		n := s.nodeByDN(dn)

		// Only back off when the node unexpectedly died - not when it got canceled.
		bo := time.Duration(0)
		if n.state == nodeStateDead {
			bo = n.bo.NextBackOff()
		}

		// Prepare node for rescheduling - remove its children, reset its state to new.
		n.reset()
		s.ilogger.Info("rescheduling supervised node", zap.String("dn", dn), zap.Duration("backoff", bo))

		// Reschedule node runnable to run after backoff.
		go func(n *node, bo time.Duration) {
			time.Sleep(bo)
			s.pReq <- &processorRequest{
				schedule: &processorRequestSchedule{dn: n.dn()},
			}
		}(n, bo)
	}
}
