// package readiness implements a minimal health-checking mechanism for use as k8s readiness probes. It will always
// return a "ready" state after the conditions have been met for the first time - it's not meant for monitoring.
//
// Uses a global singleton registry (similar to the Prometheus client's default behavior).
package readiness

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
)

var (
	// TODO is a hack to support running multiple guardians in one process;
	// This package should be rewritten to support multiple registries in one process instead of using a global registry
	NoPanic  = false
	mu       = sync.Mutex{}
	registry = map[string]bool{}
)

type Component string

// RegisterComponent registers the given component name such that it is required to be ready for the global check to succeed.
func RegisterComponent(component Component) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := registry[string(component)]; ok {
		if !NoPanic {
			panic("component already registered")
		}
		return
	}
	registry[string(component)] = false
}

// SetReady sets the given global component state.
func SetReady(component Component) {
	mu.Lock()
	defer mu.Unlock()
	if !registry[string(component)] {
		registry[string(component)] = true
	}
}

// Handler returns a net/http handler for the readiness check. It returns 200 OK if all components are ready,
// or 412 Precondition Failed otherwise. For operator convenience, a list of components and their states
// is returned as plain text (not meant for machine consumption!).
func Handler(w http.ResponseWriter, r *http.Request) {
	ready := true

	resp := new(bytes.Buffer)
	_, err := resp.Write([]byte("[not suitable for monitoring - do not parse]\n\n"))
	if err != nil {
		panic(err)
	}
	_, err = resp.Write([]byte("[these values update AT STARTUP ONLY - see https://github.com/wormhole-foundation/wormhole/blob/main/docs/operations.md#readyz]\n\n"))
	if err != nil {
		panic(err)
	}

	mu.Lock()
	defer mu.Unlock()
	for k, v := range registry {
		_, err = fmt.Fprintf(resp, "%s\t%v\n", k, v)
		if err != nil {
			panic(err)
		}

		if !v {
			ready = false
		}
	}

	if !ready {
		w.WriteHeader(http.StatusPreconditionFailed)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	_, _ = resp.WriteTo(w)
}
