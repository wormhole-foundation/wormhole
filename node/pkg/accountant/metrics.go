package accountant

import (
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	transfersOutstanding = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "global_accountant_transfer_vaas_outstanding",
			Help: "Current number of accountant transfers vaas in the pending state",
		})
	transfersSubmitted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_transfer_vaas_submitted",
			Help: "Total number of accountant transfer vaas submitted",
		})
	transfersApproved = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_transfer_vaas_submitted_and_approved",
			Help: "Total number of accountant transfer vaas that were submitted and approved",
		})
	eventsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_events_received",
			Help: "Total number of accountant events received from the smart contract",
		})
	errorEventsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_error_events_received",
			Help: "Total number of accountant error events received from the smart contract",
		})
	submitFailures = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_submit_failures",
			Help: "Total number of accountant transfer vaas submit failures",
		})
	balanceErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_total_balance_errors",
			Help: "Total number of balance errors detected by accountant",
		})
	digestMismatches = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_total_digest_mismatches",
			Help: "Total number of digest mismatches on accountant",
		})
	connectionErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_connection_errors_total",
			Help: "Total number of connection errors on accountant",
		})
	auditErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "global_accountant_audit_errors_total",
			Help: "Total number of audit errors detected by accountant",
		})
)
