package queryratelimit

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/query/narasu"
)

type withExpiry[T any] struct {
	v         T
	expiresAt time.Time
}

type Enforcer struct {
	secondLimits narasu.Store
	minuteLimits narasu.Store
	mu           sync.Mutex

	cancel context.CancelFunc
}

func NewEnforcer() *Enforcer {
	e := &Enforcer{
		secondLimits: narasu.NewMemoryStore(time.Second),
		minuteLimits: narasu.NewMemoryStore(time.Minute),
	}

	// Start background cleanup goroutine
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	go e.cleanupLoop(ctx)

	return e
}

// Stop stops the background cleanup goroutine
func (e *Enforcer) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

// cleanupLoop runs periodic cleanup in the background based on elapsed time, not traffic
func (e *Enforcer) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Run cleanup every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Cleanup entries older than 20 seconds for per-second limits
			// (previously 5 minutes - excessive for second-granularity data)
			_ = e.secondLimits.Cleanup(ctx, time.Now(), 20*time.Second)

			// Cleanup entries older than 5 minutes for per-minute limits
			// (previously 1 hour - excessive for minute-granularity data)
			_ = e.minuteLimits.Cleanup(ctx, time.Now(), 5*time.Minute)
		}
	}
}

type EnforcementResponse struct {
	Allowed       bool    `json:"allowed"`
	ExceededTypes []uint8 `json:"exceeded_types"`
}

func (e *Enforcer) EnforcePolicy(ctx context.Context, policy *Policy, action *Action) (*EnforcementResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := &EnforcementResponse{
		Allowed:       true,
		ExceededTypes: []uint8{},
	}
	for queryType, amount := range action.Types {
		if amount == 0 {
			continue
		}
		limitForQueryType, ok := policy.Limits.Types[queryType]
		if !ok {
			out.Allowed = false
			out.ExceededTypes = append(out.ExceededTypes, queryType)
			continue
		}
		fullKey := strconv.Itoa(int(queryType)) + ":" + action.Key.String()
		// Check per-second limit (skip if MaxPerSecond is 0, meaning no QPS limit)
		if limitForQueryType.MaxPerSecond > 0 {
			thisSecond, err := e.secondLimits.IncrKey(ctx, fullKey, amount, action.Time)
			if err != nil {
				// on failure to contact the rate limiter, we just error
				return nil, err
			}
			if thisSecond > limitForQueryType.MaxPerSecond {
				out.Allowed = false
				out.ExceededTypes = append(out.ExceededTypes, queryType)
				continue
			}
		}
		// Check per-minute limit
		thisMinute, err := e.minuteLimits.IncrKey(ctx, fullKey, amount, action.Time)
		if err != nil {
			// on failure to contact the rate limiter, we just error
			return nil, err
		}
		if thisMinute > limitForQueryType.MaxPerMinute {
			out.Allowed = false
			out.ExceededTypes = append(out.ExceededTypes, queryType)
		}
	}
	return out, nil
}
