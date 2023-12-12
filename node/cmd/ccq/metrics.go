package ccq

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	allQueryRequestsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_server_total_query_requests_received",
			Help: "Total number of query requests received, valid and invalid",
		})

	validQueryRequestsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_server_total_valid_query_requests_received",
			Help: "Total number of valid query requests received",
		})

	invalidQueryRequestReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_invalid_query_requests_received_by_reason",
			Help: "Total number of invalid query requests received by reason",
		}, []string{"reason"})

	totalRequestedCallsByChain = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_total_requested_calls_by_chain",
			Help: "Total number of requested calls by chain",
		}, []string{"chain_name"})

	totalRequestsByUser = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_total_requests_by_user",
			Help: "Total number of requests by user name",
		}, []string{"user_name"})

	invalidRequestsByUser = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_invalid_requests_by_user",
			Help: "Total number of invalid requests by user name",
		}, []string{"user_name"})

	queryResponsesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_total_query_responses_received_by_peer_id",
			Help: "Total number of query responses received by peer ID",
		}, []string{"peer_id"})

	inboundP2pError = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_inbound_p2p_errors",
			Help: "Total number of inbound p2p errors",
		}, []string{"reason"})

	totalQueryTime = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ccq_server_total_query_time_in_ms",
			Help:    "Time from request to response published in ms",
			Buckets: []float64{10.0, 100.0, 250.0, 500.0, 1000.0, 5000.0, 10000.0, 30000.0},
		})

	permissionFileReloadsSuccess = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_server_perm_file_reload_success",
			Help: "Total number of times the permissions file was successfully reloaded",
		})

	permissionFileReloadsFailure = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_server_perm_file_reload_failure",
			Help: "Total number of times the permissions file failed to reload",
		})
)
