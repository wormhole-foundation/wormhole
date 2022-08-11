// This file contains the code to monitor the reobservation monitor. This includes the following functions:
// - Admin commands
// - Prometheus metrics

// The reobservation monitor supports the following admin client commands:
//   - reobserver-status - displays the status of the reobservation monitor.
//   - reobserver-drop-vaa [VAA_ID] - removes the specified VAA from the reobservation list.
//
// The VAA_ID is of the form "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/3", which is "emitter chain / emitter address / sequence number".

// The reobservation monitor also supports the following Prometheus metrics:
//
// guardian_reobserver_pending_vaas 1
// - This metric lists the number of VAAs the reobservation monitor is attempting to reobserver.
//
// guardian_reobserver_successful_reobservations 1
// - This metric lists the number of VAAs the reobservation monitor successfully reobserved.
//
// guardian_reobserver_failed_reobservation_attempts 1
// - This metric lists the number of VAAs the reobservation monitor failed to reobserver.

package reobserver

import (
	"fmt"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Admin command to display status.
func (reob *Reobserver) Status() string {
	reob.mutex.Lock()
	defer reob.mutex.Unlock()

	foundSomething := false
	var resp string
	for _, oe := range reob.observations {
		if !oe.quorumReached {
			foundSomething = true
			s1 := fmt.Sprintf("quorum not reached on message: %v, numRetries: %v, timeStamp: %v", oe.msgId, oe.numRetries, oe.timeStamp.String())
			s2 := fmt.Sprintf("reobserver: %v", s1)
			resp += s1 + "\n"
			reob.logger.Info(s2)
		}
	}

	if !foundSomething {
		return "There are no reobservations in progress."
	}

	return resp
}

// Admin command to remove a VAA from the list.
func (reob *Reobserver) DropVAA(msgId string) (string, error) {
	reob.mutex.Lock()
	defer reob.mutex.Unlock()

	_, exists := reob.observations[msgId]
	if !exists {
		return "", fmt.Errorf("vaa not found in the list")
	}

	delete(reob.observations, msgId)
	str := fmt.Sprintf("vaa \"%v\" has been dropped from the list", msgId)
	return str, nil
}

var (
	metricPendingVAAs = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "guardian_reobserver_pending_vaas",
			Help: "Number of reobservation attempts currently in progress",
		})

	metricSuccessfulReobservations = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "guardian_reobserver_successful_reobservations",
			Help: "Total number of successful reobservation attempts",
		})

	metricFailedReobservationAttempts = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "guardian_reobserver_failed_reobservation_attempts",
			Help: "Total number of failed reobservation attempts",
		})
)

func (reob *Reobserver) CollectMetrics(hb *gossipv1.Heartbeat) {
	reob.mutex.Lock()
	defer reob.mutex.Unlock()

	totalPending := 0
	for _, oe := range reob.observations {
		if !oe.quorumReached {
			totalPending++
		}
	}

	metricPendingVAAs.Set(float64(totalPending))
}
