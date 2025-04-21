package queryratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/query/narasu"
	"github.com/ethereum/go-ethereum/common"
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
	Allowed          bool
	ExceededNetworks []string
}

func (e *Enforcer) EnforcePolicy(ctx context.Context, policy *Policy, action *Action) (*EnforcementResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enforcementCount++
	// TODO(elee): we can probably tune these variable better, if memory consumption or cpu usage of the ratelimiter ever becomes an issue
	if e.enforcementCount > 1024 {
		e.enforcementCount = 0
		e.secondLimits.Cleanup(ctx, time.Now(), 1*time.Hour)
		e.minuteLimits.Cleanup(ctx, time.Now(), 1*time.Hour)
	}
	out := &EnforcementResponse{
		Allowed:          true,
		ExceededNetworks: []string{},
	}
	for network, amount := range action.Networks {
		if amount == 0 {
			continue
		}
		limitForNetwork, ok := policy.Limits.Networks[network]
		if !ok {
			out.Allowed = false
			out.ExceededNetworks = append(out.ExceededNetworks, network)
			continue
		}
		thisSecond, err := e.secondLimits.IncrKey(ctx, action.Key.String(), amount, action.Time)
		if err != nil {
			// on failure to contact the rate limiter, we just error
			return nil, err
		}
		if thisSecond > limitForNetwork.MaxPerSecond {
			out.Allowed = false
			out.ExceededNetworks = append(out.ExceededNetworks, network)
			continue
		}
	}
	return out, nil
}

type Action struct {
	Time     time.Time      `json:"time"`
	Key      common.Address `json:"key"`
	Networks map[string]int `json:"networks"`
}

type Policy struct {
	Limits Limits `json:"limits"`
}

type Limits struct {
	Networks map[string]Rule `json:"networks"`
}

type Rule struct {
	MaxPerSecond int `json:"max_per_second"`
	MaxPerMinute int `json:"max_per_minute"`
}
