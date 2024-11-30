package tss

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	deliveredMsgCntr = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_tss_msg_delivered",
			Help: "total number of tss messages fed to the cryptography module",
		},
	)

	sentMsgCntr = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_tss_sent",
			Help: "total number of tss messages sent (counting broadcasts as 1)",
		},
	)

	receivedMsgCntr = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_tss_received",
			Help: "total number of tss messages received (including echos)",
		},
	)

	inProgressSigs = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_tss_sig_start",
			Help: "total number of tss signing requests",
		},
	)

	sigProducedCntr = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_tss_signature_produced",
			Help: "total number of tss signatures produced",
		},
	)

	tooManySimulSigsErrCntr = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_tss_too_many_signers",
			Help: "total number of tss signing requests that were rejected due to too many simultaneous signature requests",
		},
	)
)
