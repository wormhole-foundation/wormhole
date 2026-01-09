package queryratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// StakingPolicyFetchDuration measures the time to fetch staking policy from provider
	StakingPolicyFetchDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ccq_server_staking_policy_fetch_duration_seconds",
			Help:    "Time to fetch staking policy from provider",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		})

	// StakingPolicyRejections counts requests rejected due to staking checks by reason
	StakingPolicyRejections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_staking_rejections_total",
			Help: "Total requests rejected due to staking checks by reason",
		}, []string{"reason"}) // reason: insufficient_stake, rate_limit_exceeded, failed_to_fetch_policy, etc.

	// StakingPolicyCacheResults counts staking policy cache hits and misses
	StakingPolicyCacheResults = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_staking_policy_cache_total",
			Help: "Staking policy cache hits and misses",
		}, []string{"result"}) // result: hit, miss, miss_expired
)
