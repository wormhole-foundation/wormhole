package ccq

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
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

	successfulQueriesByUser = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_successful_queries_by_user",
			Help: "Total number of successful queries by user name",
		}, []string{"user_name"})

	failedQueriesByUser = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_failed_queries_by_user",
			Help: "Total number of failed queries by user name",
		}, []string{"user_name"})

	queryTimeoutsByUser = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_query_timeouts_by_user",
			Help: "Total number of query timeouts by user name",
		}, []string{"user_name"})

	quorumNotMetByUser = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_quorum_not_met_by_user",
			Help: "Total number of query failures due to quorum not met by user name",
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

	queryResponsesReceivedByChainAndPeerID = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_server_total_query_responses_received_by_chain_and_peer_id",
			Help: "Total number of query responses received by chain and peer ID",
		}, []string{"chain_name", "peer_id"})

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

	successfulReconnects = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccq_server_total_number_of_successful_reconnects",
			Help: "Total number of successful reconnects to bootstrap peers",
		})

	currentNumConcurrentQueriesByChain = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ccq_server_current_num_concurrent_queries_by_chain",
			Help: "Gauge showing the current number of concurrent query requests by chain",
		}, []string{"chain_name"})

	maxConcurrentQueriesByChain = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ccq_server_max_concurrent_queries_by_chain",
			Help: "Gauge showing the maximum concurrent query requests by chain",
		}, []string{"chain_name"})
)

// getGaugeValue returns the current value of a metric.
func getGaugeValue(gauge prometheus.Gauge) (float64, error) {
	metric := &dto.Metric{}
	if err := gauge.Write(metric); err != nil {
		return 0, fmt.Errorf("failed to read metric value: %w", err)
	}
	return metric.GetGauge().GetValue(), nil
}
