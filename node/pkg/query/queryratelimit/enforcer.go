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

	enforcementCount int
}

func NewEnforcer() *Enforcer {
	return &Enforcer{
		secondLimits: narasu.NewMemoryStore(time.Second),
		minuteLimits: narasu.NewMemoryStore(time.Minute),
	}
}

type EnforcementResponse struct {
	Allowed       bool    `json:"allowed"`
	ExceededTypes []uint8 `json:"exceeded_types"`
}

func (e *Enforcer) EnforcePolicy(ctx context.Context, policy *Policy, action *Action) (*EnforcementResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enforcementCount++
	// TODO(elee): we can probably tune these variable better, if memory consumption or cpu usage of the ratelimiter ever becomes an issue
	if e.enforcementCount%1024 == 0 {
		e.enforcementCount = 0
		e.secondLimits.Cleanup(ctx, time.Now(), 1*time.Hour)
		e.minuteLimits.Cleanup(ctx, time.Now(), 1*time.Hour)
	}
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
	return out, nil
}
