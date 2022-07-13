// This file contains the code to monitor the chain governor. This includes the following functions:
// - Admin commands
// - REST APIs
// - Prometheus metrics

// The chain governor supports the following admin client commands:
//   - governor-status - displays the status of the chain governor to the log file.
//   - governor-reload - reloads the state of the chain governor from the database.
//   - governor-drop-pending-vaa [VAA_ID] - removes the specified transfer from the pending list and discards it.
//   - governor-release-pending-vaa [VAA_ID] - removes the specified transfer from the pending list and publishes it, without regard to the threshold.
//
// The VAA_ID is of the form "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/3", which is "emitter chain / emitter address / sequence number".

// The chain governor also supports the following REST queries:
//
// Query: http://localhost:7071/v1/governor/available_notional_by_chain
//
// Returns:
// {"entries":[
//	{"chainId":1,"remainingAvailableNotional":"96217","notionalLimit":"100000"},
//  {"chainId":2,"remainingAvailableNotional":"100000","notionalLimit":"100000"},
//  {"chainId":5,"remainingAvailableNotional":"275000","notionalLimit":"275000"}
// ]}
//
// Query: http://localhost:7071/v1/governor/enqueued_vaas
//
// Returns:
// {"entries":[
// 	{"emitterChain":1, "emitterAddress":"c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f", "sequence":"1"},
// 	{"emitterChain":1, "emitterAddress":"c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f", "sequence":"2"}
// ]}
//
// Query: http://localhost:7071/v1/governor/token_list
//
// Returns:
// {"entries":[
//	{"originChainId":3, "originAddress":"0x0000000000000000000000008f5cd460d57ac54e111646fc569179144c7f0c28"},
//	{"originChainId":4, "originAddress":"0x00000000000000000000000086812b970bbdce75b4590243ba2cbff671d0b754"},
//	{"originChainId":1, "originAddress":"0xc6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61"},
//	]}

// The chain governor also supports the following Prometheus metrics:
//
// guardian_governor_available_notional{chain_id="1",chain_name="solana",enabled="1",total_notional="1000"} 304
// - This metric provides the current remaining notional value before a chain starts enqueuing VAAs. There is a metric
//   for all existing chains, where the enabled flag indicates whether the governor is monitoring the chain or not.
//   The total_notional value is the configured limit for the chain, and is zero if the chain is not configured in the governor.
//
// guardian_governor_enqueued_vaas{chain_id="1",chain_name="solana",enabled="1"} 1
// - This metric lists the number of VAAs currently enqueued for that chain because they would exceed the notional limit.
//   There is a metric for all existing chains, where the enabled flag indicates whether the governor is monitoring the chain or not.
//
// guardian_governor_total_enqueued_vaas 1
// - This is a single metric that indicates the total number of enqueued VAAs across all chains. This provides a quick check if
//   anything is currently being limited.

package governor

import (
	"fmt"
	"sort"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Admin command to display status to the log.
func (gov *ChainGovernor) Status() string {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	var resp string
	for _, ce := range gov.chains {
		valueTrans := sumValue(ce.transfers, startTime)
		s1 := fmt.Sprintf("chain: %v, dailyLimit: %v, total: %v, numPending: %v", ce.emitterChainId, ce.dailyLimit, valueTrans, len(ce.pending))
		s2 := fmt.Sprintf("cgov: %v", s1)
		resp += s1 + "\n"
		gov.logger.Info(s2)
		if len(ce.pending) != 0 {
			for idx, pe := range ce.pending {
				value, _ := computeValue(pe.amount, pe.token)
				s1 := fmt.Sprintf("chain: %v, pending[%v], value: %v, vaa: %v, time: %v", ce.emitterChainId, idx, value,
					pe.msg.MessageIDString(), pe.timeStamp.String())
				s2 := fmt.Sprintf("cgov: %v", s1)
				gov.logger.Info(s2)
				resp += "   " + s1 + "\n"
			}
		}
	}

	return resp
}

// Admin command to reload the governor state from the database.
func (gov *ChainGovernor) Reload() (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	if gov.db == nil {
		return "", fmt.Errorf("unable to reload because the database is not initialized")
	}

	for _, ce := range gov.chains {
		ce.transfers = nil
		ce.pending = nil
	}

	if err := gov.loadFromDBAlreadyLocked(); err != nil {
		gov.logger.Error("cgov: failed to load from the database", zap.Error(err))
		return "", err
	}

	return "chain governor has been reset and reloaded", nil
}

// Admin command to remove a VAA from the pending list and discard it.
func (gov *ChainGovernor) DropPendingVAA(vaaId string) (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		for idx, pe := range ce.pending {
			msgId := pe.msg.MessageIDString()
			if msgId == vaaId {
				value, _ := computeValue(pe.amount, pe.token)
				gov.logger.Info("cgov: dropping pending vaa",
					zap.String("msgId", msgId),
					zap.Uint64("value", value),
					zap.Stringer("timeStamp", pe.timeStamp),
				)

				if gov.db != nil {
					if err := gov.db.DeletePendingMsg(pe.msg); err != nil {
						return "", err
					}
				}

				ce.pending = append(ce.pending[:idx], ce.pending[idx+1:]...)
				str := fmt.Sprintf("vaa \"%v\" has been dropped from the pending list", msgId)
				return str, nil
			}
		}
	}

	return "", fmt.Errorf("vaa not found in the pending list")
}

// Admin command to remove a VAA from the pending list and publish it without regard to (or impact on) the daily limit.
func (gov *ChainGovernor) ReleasePendingVAA(vaaId string) (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		for idx, pe := range ce.pending {
			msgId := pe.msg.MessageIDString()
			if msgId == vaaId {
				value, _ := computeValue(pe.amount, pe.token)
				gov.logger.Info("cgov: releasing pending vaa, should be published soon",
					zap.String("msgId", msgId),
					zap.Uint64("value", value),
					zap.Stringer("timeStamp", pe.timeStamp),
				)

				gov.msgsToPublish = append(gov.msgsToPublish, pe.msg)

				if gov.db != nil {
					// We delete the pending message from the database, but we don't add it to the transfers
					// because released messages do not apply to the limit.

					if err := gov.db.DeletePendingMsg(pe.msg); err != nil {
						return "", err
					}
				}

				ce.pending = append(ce.pending[:idx], ce.pending[idx+1:]...)
				str := fmt.Sprintf("pending vaa \"%v\" has been released and will be published soon", msgId)
				return str, nil
			}
		}
	}

	return "", fmt.Errorf("vaa not found in the pending list")
}

func sumValue(transfers []db.Transfer, startTime time.Time) uint64 {
	if len(transfers) == 0 {
		return 0
	}

	var sum uint64

	for _, t := range transfers {
		if !t.Timestamp.Before(startTime) {
			sum += t.Value
		}
	}

	return sum
}

// REST query to get the current available notional value per chain.
func (gov *ChainGovernor) GetAvailableNotionalByChain() []*publicrpcv1.GovernorGetAvailableNotionalByChainResponse_Entry {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	resp := make([]*publicrpcv1.GovernorGetAvailableNotionalByChainResponse_Entry, 0)

	startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	for _, ce := range gov.chains {
		value := sumValue(ce.transfers, startTime)
		if value >= ce.dailyLimit {
			value = 0
		} else {
			value = ce.dailyLimit - value
		}

		resp = append(resp, &publicrpcv1.GovernorGetAvailableNotionalByChainResponse_Entry{
			ChainId:                    uint32(ce.emitterChainId),
			RemainingAvailableNotional: value,
			NotionalLimit:              ce.dailyLimit,
		})
	}

	sort.SliceStable(resp, func(i, j int) bool {
		return (resp[i].ChainId < resp[j].ChainId)
	})

	return resp
}

// REST query to get the list of enqueued VAAs.
func (gov *ChainGovernor) GetEnqueuedVAAs() []*publicrpcv1.GovernorGetEnqueuedVAAsResponse_Entry {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	resp := make([]*publicrpcv1.GovernorGetEnqueuedVAAsResponse_Entry, 0)

	for _, ce := range gov.chains {
		for _, pe := range ce.pending {
			resp = append(resp, &publicrpcv1.GovernorGetEnqueuedVAAsResponse_Entry{
				EmitterChain:   uint32(pe.msg.EmitterChain),
				EmitterAddress: pe.msg.EmitterAddress.String(),
				Sequence:       pe.msg.Sequence,
			})
		}
	}

	return resp
}

// REST query to get the list of tokens being monitored by the governor.
func (gov *ChainGovernor) GetTokenList() []*publicrpcv1.GovernorGetTokenListResponse_Entry {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	resp := make([]*publicrpcv1.GovernorGetTokenListResponse_Entry, 0)

	for tk, te := range gov.tokens {
		price, _ := te.price.Float32()
		resp = append(resp, &publicrpcv1.GovernorGetTokenListResponse_Entry{
			OriginChainId: uint32(tk.chain),
			OriginAddress: "0x" + tk.addr.String(),
			Price:         price,
		})
	}

	sort.SliceStable(resp, func(i, j int) bool {
		if resp[i].OriginChainId < resp[j].OriginChainId {
			return true
		}
		if resp[i].OriginChainId > resp[j].OriginChainId {
			return false
		}

		return (resp[i].OriginAddress < resp[j].OriginAddress)
	})

	return resp
}

var (
	// guardian_governor_available_notional{chain_id=1, chain_name="solana", enabled=1, total_notional=10000} 100
	metricAvailableNotional = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "guardian_governor_available_notional",
			Help: "Chain governor remaining available notional value per chain",
		}, []string{"chain_id", "chain_name", "enabled", "total_notional"})

	// guardian_governor_enqueued_vaas{chain_id="5",chain_name="polygon",enabled="1"} 0
	metricEnqueuedVAAs = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "guardian_governor_enqueued_vaas",
			Help: "Chain governor number of VAAs enqueued due to limiting per chain",
		}, []string{"chain_id", "chain_name", "enabled"})

	// guardian_governor_total_enqueued_vaas 0
	metricTotalEnqueuedVAAs = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "guardian_governor_total_enqueued_vaas",
			Help: "Chain governor total number of VAAs enqueued due to limiting across all chains",
		})
)

func (gov *ChainGovernor) CollectMetrics(hb *gossipv1.Heartbeat) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	totalPending := 0
	for _, n := range hb.Networks {
		if n == nil {
			continue
		}

		chain := vaa.ChainID(n.Id)

		chainId := fmt.Sprint(n.Id)
		enabled := "0"
		totalNotional := "0"
		available := 0.0
		numPending := 0.0

		ce, exists := gov.chains[chain]

		if exists {
			enabled = "1"
			value := sumValue(ce.transfers, startTime)
			if value >= ce.dailyLimit {
				value = 0
			} else {
				value = ce.dailyLimit - value
			}

			pending := len(ce.pending)
			totalNotional = fmt.Sprint(ce.dailyLimit)
			available = float64(value)
			numPending = float64(pending)
			totalPending += pending
		}

		//"chain_id", "chain_name", "enabled", "total_notional"
		metricAvailableNotional.WithLabelValues(
			chainId,        // chain_id
			chain.String(), // chain_name
			enabled,        // enabled
			totalNotional,  // total_notional
		).Set(float64(available))

		metricEnqueuedVAAs.WithLabelValues(
			chainId,        // chain_id
			chain.String(), // chain_name
			enabled,        // enabled
		).Set(float64(numPending))
	}

	metricTotalEnqueuedVAAs.Set(float64(totalPending))
}
