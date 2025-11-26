package notary

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Prometheus metrics for notary events.
// These are only registered when the Notary is actually enabled and Run() is called.
var (
	notaryReleasedMessagesCounter prometheus.Counter
	notaryDelayedMessagesGauge    prometheus.Gauge
	notaryBlackholedMessagesGauge prometheus.Gauge
	notaryErrors                  *prometheus.CounterVec
	notaryTokenTransferNonApprove *prometheus.CounterVec
)

// initMetrics registers all notary metrics with Prometheus.
// This is called once when the Notary's Run() method is invoked.
// Safe to call multiple times - will only register once.
func initMetrics(logger *zap.Logger) {
	// Only initialize once
	if notaryReleasedMessagesCounter != nil {
		logger.Info("notary metrics already initialized, skipping")
		return
	}

	notaryReleasedMessagesCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_notary_messages_released_total",
			Help: "Total number of delayed messages released by the notary",
		})

	notaryDelayedMessagesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_notary_delayed_messages",
			Help: "Current number of messages in the delayed queue",
		})

	notaryBlackholedMessagesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_notary_blackholed_messages",
			Help: "Current number of blackholed messages",
		})

	notaryErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_notary_errors_total",
			Help: "Total number of notary errors",
		}, []string{"error_type"})

	notaryTokenTransferNonApprove = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_notary_token_transfer_non_approve_total",
			Help: "Total number of token transfers that received a non-Approve verdict from the notary",
		}, []string{"verdict"})

	// Register all metrics with the default Prometheus registry
	prometheus.MustRegister(
		notaryReleasedMessagesCounter,
		notaryDelayedMessagesGauge,
		notaryBlackholedMessagesGauge,
		notaryErrors,
		notaryTokenTransferNonApprove,
	)
}
