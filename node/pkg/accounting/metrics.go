package accounting

import (
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	transfersOutstanding = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_accounting_transfer_vaas_outstanding",
			Help: "Current number of accounting transfers vaas in the pending state",
		})
	transfersSubmitted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_accounting_transfer_vaas_submitted",
			Help: "Total number of accounting transfer vaas submitted",
		})
	transfersApproved = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_accounting_transfer_vaas_submitted_and_approved",
			Help: "Total number of accounting transfer vaas that were submitted and approved",
		})
	submitFailures = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_accounting_submit_failures",
			Help: "Total number of accounting transfer vaas submit failures",
		})
	eventsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_accounting_events_received",
			Help: "Total number of accounting events received",
		})
	connectionErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_accounting_connection_errors_total",
			Help: "Total number of connection errors on accounting",
		})
)
