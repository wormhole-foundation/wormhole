package query

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	allQueryRequestsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_guardian_total_query_requests_received",
			Help: "Total number of query requests received, valid and invalid",
		})

	validQueryRequestsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_guardian_total_valid_query_requests_received",
			Help: "Total number of valid query requests received",
		})

	invalidQueryRequestReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_guardian_invalid_query_requests_received_by_reason",
			Help: "Total number of invalid query requests received by reason",
		}, []string{"reason"})

	totalRequestsByChain = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_guardian_total_requests_by_chain",
			Help: "Total number of requests by chain",
		}, []string{"chain_name"})

	successfulQueryResponsesReceivedByChain = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_guardian_total_successful_query_responses_received_by_chain",
			Help: "Total number of successful query responses received by chain",
		}, []string{"chain_name"})

	retryNeededQueryResponsesReceivedByChain = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_guardian_total_retry_needed_query_responses_received_by_chain",
			Help: "Total number of retry needed query responses received by chain",
		}, []string{"chain_name"})

	fatalQueryResponsesReceivedByChain = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_guardian_total_fatal_query_responses_received_by_chain",
			Help: "Total number of fatal query responses received by chain",
		}, []string{"chain_name"})

	queryResponsesPublished = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_guardian_total_query_responses_published",
			Help: "Total number of query responses published",
		})

	queryRequestsTimedOut = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_guardian_total_query_requests_timed_out",
			Help: "Total number of query requests that timed out",
		})

	TotalWatcherTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ccq_guardian_total_watcher_query_time_in_ms",
			Help:    "Time from time spent in the watcher per query in ms by chain",
			Buckets: []float64{1.0, 5.0, 10.0, 100.0, 250.0, 500.0, 1000.0, 5000.0, 10000.0, 30000.0},
		}, []string{"chain_name"})
)
