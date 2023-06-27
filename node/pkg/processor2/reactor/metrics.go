package reactor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	observationsReceivedTotalVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_observations_received_total_2",
			Help: "Total number of raw observations received from gossip",
		}, []string{"reactor_group"})
	observationsReceivedByGuardianAddressTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_observations_signed_by_guardian_total_2",
			Help: "Total number of signed and verified observations grouped by guardian address",
		}, []string{"reactor_group", "addr"})
	observationsFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_observations_verification_failures_total_2",
			Help: "Total number of observations verification failure, grouped by failure reason",
		}, []string{"reactor_group", "cause"})
	observationsBroadcastTotalVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_observations_broadcast_total_2",
			Help: "Total number of signed observations queued for broadcast",
		}, []string{"reactor_group"})

	messagesObservedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_message_observations_total_2",
			Help: "Total number of messages observed",
		}, []string{"reactor_group"})

	messagesSignedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_message_observations_signed_total_2",
			Help: "Total number of message observations that were successfully signed",
		}, []string{"reactor_group"})

	reactorNum = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_consensus_num_reactors",
			Help: "Current number of consensus reactors",
		}, []string{"reactor_group"})
	reactorTimedOut = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_consensus_num_reactors_timed_out_total",
			Help: "Total number of timed out reactors",
		}, []string{"reactor_group", "time_out_state"})
	reactorObservedLate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_consensus_late_observations_total",
			Help: "Total number of late observations (cluster achieved consensus without us)",
		}, []string{"reactor_group"})
	reactorResubmission = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_consensus_retransmission_total",
			Help: "Total number of signed observation retransmissions",
		}, []string{"reactor_group"})
	reactorQuorum = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_consensus_quorum_total",
			Help: "Total number of reactors that reached quorum",
		}, []string{"reactor_group", "type"})
	reactorFinalized = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_consensus_finalized_total",
			Help: "Total number of reactors that were finalized, counted after waiting a fixed amount of time",
		}, []string{"reactor_group"})
)
