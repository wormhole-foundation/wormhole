package tvm

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var currentHeight = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "wormhole_ton_current_height",
		Help: "Current TON block height",
	})

var messagesConfirmed = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "wormhole_ton_observations_confirmed_total",
		Help: "Total number of verified TON observations found",
	})
