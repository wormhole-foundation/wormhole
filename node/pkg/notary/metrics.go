package notary

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for notary events
var (
	notaryReleasedMessagesCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_notary_messages_released_total",
			Help: "Total number of delayed messages released by the notary",
		})

	notaryDelayedMessagesGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_notary_delayed_messages",
			Help: "Current number of messages in the delayed queue",
		})

	notaryBlackholedMessagesGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_notary_blackholed_messages",
			Help: "Current number of blackholed messages",
		})

	notaryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_notary_errors_total",
			Help: "Total number of notary errors",
		}, []string{"error_type"})

	NotaryTokenTransferNonApprove = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_notary_token_transfer_non_approve_total",
			Help: "Total number of token transfers that received a non-Approve verdict from the notary",
		}, []string{"verdict"})
)
