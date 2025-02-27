package supervisor

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

// node is a supervision tree node. It represents the state of a Runnable within this tree, its relation to other tree
// elements, and contains supporting data needed to actually supervise it.
type node struct {
	// The name of this node. Opaque string. It's used to make up the 'dn' (distinguished name) of a node within
	// the tree. When starting a runnable inside a tree, this is where that name gets used.
	name     string
	runnable Runnable

	// The supervisor managing this tree.
	sup *supervisor
	// The parent, within the tree, of this node. If this is the root node of the tree, this is nil.
	parent *node
	// Children of this tree. This is represented by a map keyed from child node names, for easy access.
	children map[string]*node
	// Supervision groups. Each group is a set of names of children. Sets, and as such groups, don't overlap between
	// each other. A supervision group indicates that if any child within that group fails, all others should be
	// canceled and restarted together.
	groups []map[string]bool

	// The current state of the runnable in this node.
	state nodeState

	// Backoff used to keep runnables from being restarted too fast.
	bo *backoff.ExponentialBackOff

	// Context passed to the runnable, and its cancel function.
	ctx  context.Context
	ctxC context.CancelFunc
}

// nodeState is the state of a runnable within a node, and in a way the node itself.
// This follows the state diagram from go/supervision.
type nodeState int

const (
	// A node that has just been created, and whose runnable has been started already but hasn't signaled anything yet.
	nodeStateNew nodeState = iota
	// A node whose runnable has signaled being healthy - this means it's ready to serve/act.
	nodeStateHealthy
	// A node that has unexpectedly returned or panicked.
	nodeStateDead
	// A node that has declared that its done with its work and should not be restarted, unless a supervision tree
	// failure requires that.
	nodeStateDone
	// A node that has returned after being requested to cancel.
	nodeStateCanceled
)

func (s nodeState) String() string {
	switch s {
	case nodeStateNew:
		return "NODE_STATE_NEW"
	case nodeStateHealthy:
		return "NODE_STATE_HEALTHY"
	case nodeStateDead:
		return "NODE_STATE_DEAD"
	case nodeStateDone:
		return "NODE_STATE_DONE"
	case nodeStateCanceled:
		return "NODE_STATE_CANCELED"
	}
	return "UNKNOWN"
}

func (n *node) String() string {
	return fmt.Sprintf("%s (%s)", n.dn(), n.state.String())
}

// contextKey is a type used to keep data within context values.
type contextKey string

var (
	supervisorKey = contextKey("supervisor")
	dnKey         = contextKey("dn")
)

// fromContext retrieves a tree node from a runnable context. It takes a lock on the tree and returns an unlock
// function. This unlock function needs to be called once mutations on the tree/supervisor/node are done.
func fromContext(ctx context.Context) (*node, func()) {
	sup, ok := ctx.Value(supervisorKey).(*supervisor)
	if !ok {
		panic("supervisor function called from non-runnable context")
	}

	sup.mu.Lock()

	dnParent, ok := ctx.Value(dnKey).(string)
	if !ok {
		sup.mu.Unlock()
		panic("supervisor function called from non-runnable context")
	}

	return sup.nodeByDN(dnParent), sup.mu.Unlock
}

// All the following 'internal' supervisor functions must only be called with the supervisor lock taken. Getting a lock
// via fromContext is enough.

// dn returns the distinguished name of a node. This distinguished name is a period-separated, inverse-DNS-like name.
// For instance, the runnable 'foo' within the runnable 'bar' will be called 'root.bar.foo'. The root of the tree is
// always named, and has the dn, 'root'.
func (n *node) dn() string {
	if n.parent != nil {
		return fmt.Sprintf("%s.%s", n.parent.dn(), n.name)
	}
	return n.name
}

// groupSiblings is a helper function to get all runnable group siblings of a given runnable name within this node.
// All children are always in a group, even if that group is unary.
func (n *node) groupSiblings(name string) map[string]bool {
	for _, m := range n.groups {
		if _, ok := m[name]; ok {
			return m
		}
	}
	return nil
}

// newNode creates a new node with a given parent. It does not register it with the parent (as that depends on group
// placement).
func newNode(name string, runnable Runnable, sup *supervisor, parent *node) *node {
	// We use exponential backoff for failed runnables, but at some point we cap at a given backoff time.
	// To achieve this, we set MaxElapsedTime to 0, which will cap the backoff at MaxInterval.
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0

	n := &node{
		name:     name,
		runnable: runnable,

		bo: bo,

		sup:    sup,
		parent: parent,
	}
	n.reset()
	return n
}

// resetNode sets up all the dynamic fields of the node, in preparation of starting a runnable. It clears the node's
// children, groups and resets its context.
func (n *node) reset() {
	// Make new context. First, acquire parent context. For the root node that's Background, otherwise it's the
	// parent's context.
	var pCtx context.Context
	if n.parent == nil {
		pCtx = context.Background()
	} else {
		pCtx = n.parent.ctx
	}
	// Mark DN and supervisor in context.
	ctx := context.WithValue(pCtx, dnKey, n.dn())
	ctx = context.WithValue(ctx, supervisorKey, n.sup)
	ctx, ctxC := context.WithCancel(ctx)
	// Set context
	n.ctx = ctx
	n.ctxC = ctxC

	// Clear children and state
	n.state = nodeStateNew
	n.children = make(map[string]*node)
	n.groups = nil

	// The node is now ready to be scheduled.
}

// nodeByDN returns a node by given DN from the supervisor.
func (s *supervisor) nodeByDN(dn string) *node {
	parts := strings.Split(dn, ".")
	if parts[0] != "root" {
		panic("DN does not start with root.")
	}
	parts = parts[1:]
	cur := s.root
	for {
		if len(parts) == 0 {
			return cur
		}

		next, ok := cur.children[parts[0]]
		if !ok {
			panic(fmt.Errorf("could not find %v (%s) in %s", parts, dn, cur))
		}
		cur = next
		parts = parts[1:]
	}
}

// reNodeName validates a node name against constraints.
var reNodeName = regexp.MustCompile(`[a-z0-9_]{1,64}`)

// runGroup schedules a new group of runnables to run on a node.
func (n *node) runGroup(runnables map[string]Runnable) error {
	// Check that the parent node is in the right state.
	if n.state != nodeStateNew {
		return fmt.Errorf("cannot run new runnable on non-NEW node")
	}

	// Check the requested runnable names.
	for name := range runnables {
		if !reNodeName.MatchString(name) {
			return fmt.Errorf("runnable name %q is invalid", name)
		}
		if _, ok := n.children[name]; ok {
			return fmt.Errorf("runnable %q already exists", name)
		}
	}

	// Create child nodes.
	dns := make(map[string]string)
	group := make(map[string]bool)
	for name, runnable := range runnables {
		if g := n.groupSiblings(name); g != nil {
			return fmt.Errorf("duplicate child name %q", name)
		}
		node := newNode(name, runnable, n.sup, n)
		n.children[name] = node

		dns[name] = node.dn()
		group[name] = true
	}
	// Add group.
	n.groups = append(n.groups, group)

	// Schedule execution of group members.
	go func() {
		for name := range runnables {
			n.sup.pReq <- &processorRequest{
				schedule: &processorRequestSchedule{
					dn: dns[name],
				},
			}
		}
	}()
	return nil
}

// signal sequences state changes by signals received from runnables and updates a node's status accordingly.
func (n *node) signal(signal SignalType) {
	switch signal {
	case SignalHealthy:
		if n.state != nodeStateNew {
			panic(fmt.Errorf("node %s signaled healthy", n))
		}
		n.state = nodeStateHealthy
		n.bo.Reset()
	case SignalDone:
		if n.state != nodeStateHealthy {
			panic(fmt.Errorf("node %s signaled done", n))
		}
		n.state = nodeStateDone
		n.bo.Reset()
	}
}

// getLogger creates a new logger for a given supervisor node, to be used by its runnable.
func (n *node) getLogger() *zap.Logger {
	return n.sup.logger.Named(n.dn())
}
