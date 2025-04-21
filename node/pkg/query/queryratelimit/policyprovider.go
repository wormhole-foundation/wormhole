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
	fetcher       func(ctx context.Context, key common.Address) (*Policy, error)
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

func WithPolicyProviderFetcher(fetcher func(ctx context.Context, key common.Address) (*Policy, error)) PolicyProviderOption {
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

func (r *PolicyProvider) GetPolicy(ctx context.Context, key common.Address) (*Policy, error) {
	ival, hit := r.cache.Get(key)
	if hit {
		val := ival.(withExpiry[*Policy])
		if time.Now().After(val.expiresAt) {
			r.cache.Remove(key)
		}
		if r.optimistic {
			go func() {
				ctx, cn := context.WithTimeout(r.parentContext, r.fetchTimeout)
				defer cn()
				if _, err := r.fetchAndFill(ctx, key); err != nil {
					if r.logger != nil {
						r.logger.Error("failed to fetch rate limit policy in background", zap.Error(err))
					}
				}
			}()
			return val.v, nil
		}
	}
	return r.fetchAndFill(ctx, key)
}

func (r *PolicyProvider) fetchAndFill(ctx context.Context, key common.Address) (*Policy, error) {
	res, err, _ := r.sf.Do(key.String(), func() (any, error) {
		policy, err := r.fetcher(ctx, key)
		if err != nil {
			return nil, err
		}
		r.cache.Add(key, withExpiry[*Policy]{
			v:         policy,
			expiresAt: time.Now().Add(r.cacheDuration),
		})
		return policy, nil
	})
	if err != nil {
		return nil, err
	}
	return res.(*Policy), err
}
