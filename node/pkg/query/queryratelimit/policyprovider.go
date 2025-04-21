package queryratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// TODO(elee): this should really be an interface where the seprate parts are split out, ala, one for fetching, one for ttl cache.
type PolicyProvider struct {
	fetcher       func(ctx context.Context, signerAddr, stakerAddr common.Address) (*Policy, error)
	fetchTimeout  time.Duration
	cacheDuration time.Duration
	optimistic    bool
	parentContext context.Context
	logger        *zap.Logger

	cache *lru.Cache

	sf singleflight.Group
}

type PolicyProviderOption func(*PolicyProvider)

func WithPolicyProviderLogger(logger *zap.Logger) PolicyProviderOption {
	return func(p *PolicyProvider) {
		p.logger = logger
	}
}

func WithPolicyProviderFetcher(fetcher func(ctx context.Context, signerAddr, stakerAddr common.Address) (*Policy, error)) PolicyProviderOption {
	return func(p *PolicyProvider) {
		p.fetcher = fetcher
	}
}

func WithPolicyProviderCache(cache *lru.Cache) PolicyProviderOption {
	return func(p *PolicyProvider) {
		p.cache = cache
	}
}

func WithPolicyProviderOptimistic(optimistic bool) PolicyProviderOption {
	return func(p *PolicyProvider) {
		p.optimistic = optimistic
	}
}

func WithPolicyProviderCacheDuration(cacheDuration time.Duration) PolicyProviderOption {
	return func(p *PolicyProvider) {
		p.cacheDuration = cacheDuration
	}
}

func WithPolicyProviderParentContext(ctx context.Context) PolicyProviderOption {
	return func(p *PolicyProvider) {
		p.parentContext = ctx
	}
}
func WithPolicyProviderFetchTimeout(timeout time.Duration) PolicyProviderOption {
	return func(p *PolicyProvider) {
		p.fetchTimeout = timeout
	}
}

var ErrNewPolicyProvider = fmt.Errorf("new rate limit policy provider")

func NewPolicyProvider(ops ...PolicyProviderOption) (*PolicyProvider, error) {
	o := &PolicyProvider{
		cacheDuration: time.Minute * 5,
		fetchTimeout:  time.Second * 5,
		parentContext: context.Background(),
	}
	for _, op := range ops {
		if op == nil {
			continue
		}
		op(o)
	}
	if o.cache == nil {
		lru, err := lru.New(1024)
		if err != nil {
			return nil, err
		}
		o.cache = lru
	}
	if o.fetcher == nil {
		return nil, fmt.Errorf("%w: fetcher required", ErrNewPolicyProvider)
	}

	return o, nil
}

func (r *PolicyProvider) GetPolicy(ctx context.Context, signerAddr, stakerAddr common.Address) (*Policy, error) {
	cacheKey := signerAddr.Hex() + ":" + stakerAddr.Hex()
	ival, hit := r.cache.Get(cacheKey)
	if hit {
		val, ok := ival.(withExpiry[*Policy])
		if !ok {
			// Cache corruption - remove and treat as miss
			r.cache.Remove(cacheKey)
			StakingPolicyCacheResults.WithLabelValues("miss_invalid").Inc()
			return r.fetchAndFill(ctx, cacheKey, signerAddr, stakerAddr)
		}
		// Check expiry atomically - if expired, treat as cache miss
		isExpired := time.Now().After(val.expiresAt)
		if isExpired {
			StakingPolicyCacheResults.WithLabelValues("miss_expired").Inc()
			// Remove expired entry to prevent serving stale data
			r.cache.Remove(cacheKey)
			// Fall through to fetch fresh data
		} else {
			StakingPolicyCacheResults.WithLabelValues("hit").Inc()
			// Cache hit with valid (non-expired) data
			if r.optimistic {
				// Trigger background refresh while returning cached value
				go func() { //nolint:contextcheck // Background refresh uses parentContext to continue even if request context is cancelled
					bgCtx, cn := context.WithTimeout(r.parentContext, r.fetchTimeout)
					defer cn()
					if _, err := r.fetchAndFill(bgCtx, cacheKey, signerAddr, stakerAddr); err != nil {
						if r.logger != nil {
							r.logger.Error("failed to fetch rate limit policy in background", zap.Error(err))
						}
					}
				}()
			}
			return val.v, nil
		}
	}
	// Cache miss or expired - fetch fresh data
	if !hit {
		StakingPolicyCacheResults.WithLabelValues("miss").Inc()
	}
	return r.fetchAndFill(ctx, cacheKey, signerAddr, stakerAddr)
}

func (r *PolicyProvider) fetchAndFill(ctx context.Context, cacheKey string, signerAddr, stakerAddr common.Address) (*Policy, error) {
	res, err, _ := r.sf.Do(cacheKey, func() (any, error) {
		start := time.Now()
		policy, err := r.fetcher(ctx, signerAddr, stakerAddr)
		StakingPolicyFetchDuration.Observe(time.Since(start).Seconds())
		if err != nil {
			return nil, err
		}
		r.cache.Add(cacheKey, withExpiry[*Policy]{
			v:         policy,
			expiresAt: time.Now().Add(r.cacheDuration),
		})
		return policy, nil
	})
	if err != nil {
		return nil, err
	}
	policy, ok := res.(*Policy)
	if !ok {
		return nil, fmt.Errorf("singleflight returned unexpected type")
	}
	return policy, nil
}
