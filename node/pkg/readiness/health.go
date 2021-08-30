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
	mu       = sync.Mutex{}
	registry = map[string]bool{}
)

type Component string

// RegisterComponent registers the given component name such that it is required to be ready for the global check to succeed.
func RegisterComponent(component Component) {
	mu.Lock()
	if _, ok := registry[string(component)]; ok {
		panic("component already registered")
	}
	registry[string(component)] = false
	mu.Unlock()
}

// SetReady sets the given global component state.
func SetReady(component Component) {
	mu.Lock()
	if !registry[string(component)] {
		registry[string(component)] = true
	}
	mu.Unlock()
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
	_, err = resp.Write([]byte("[these values update AT STARTUP ONLY - see https://github.com/certusone/wormhole/blob/dev.v2/docs/operations.md#readyz]\n\n"))
	if err != nil {
		panic(err)
	}

	mu.Lock()
	for k, v := range registry {
		_, err = fmt.Fprintf(resp, "%s\t%v\n", k, v)
		if err != nil {
			panic(err)
		}

		if !v {
			ready = false
		}
	}
	mu.Unlock()

	if !ready {
		w.WriteHeader(http.StatusPreconditionFailed)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	_, _ = resp.WriteTo(w)
}
