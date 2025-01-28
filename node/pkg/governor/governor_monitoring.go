// This file contains the code to monitor the chain governor. This includes the following functions:
// - Admin commands
// - REST APIs
// - Prometheus metrics

// The chain governor supports the following admin client commands:
//   - governor-status - displays the status of the chain governor to the log file.
//   - governor-reload - reloads the state of the chain governor from the database.
//   - governor-drop-pending-vaa [VAA_ID] - removes the specified transfer from the pending list and discards it.
//   - governor-release-pending-vaa [VAA_ID] - removes the specified transfer from the pending list and publishes it, without regard to the threshold.
//   - governor-reset-release-timer - resets the release timer for the specified VAA to the configured maximum.
//
// The VAA_ID is of the form "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/3", which is "emitter chain / emitter address / sequence number".

// The chain governor also supports the following REST queries:
//
// Query: http://localhost:7071/v1/governor/available_notional_by_chain
//
// Returns:
// {"entries":[
//	{"chainId":1,"remainingAvailableNotional":"96217","notionalLimit":"100000","bigTransactionSize":"10000"},
//  {"chainId":2,"remainingAvailableNotional":"100000","notionalLimit":"100000","bigTransactionSize":"10000"},
//  {"chainId":5,"remainingAvailableNotional":"275000","notionalLimit":"275000","bigTransactionSize":"20000"}
// ]}
//
// Query: http://localhost:7071/v1/governor/enqueued_vaas
//
// Returns:
// {"entries":[
// 	{"emitterChain":1,"emitterAddress":"c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f","sequence":"3","releaseTime":1662057609,"notionalValue":"69","txHash":"0xccdb6891688b551c1a182292f93e5a9e9e9671bc902116162f044041cafbdcaf"},
// 	{"emitterChain":1,"emitterAddress":"c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f","sequence":"4","releaseTime":1662057673,"notionalValue":"34","txHash":"0x95641b9b3f9cfdd82ca97f251b9249183d838a096ae3feea60032a22726d6f42"}
// ]}
//
// Query: http://localhost:7071/v1/governor/is_vaa_enqueued/1/c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f/3
//
// Returns:
// {"isEnqueued":true}
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

// The chain governor also publishes the following messages to the gossip network
//
// SignedChainGovernorConfig
// - Published once every five minutes.
// - Contains a list of configured chains, along with the daily limit, big transaction size and current price.
//
// - SignedChainGovernorStatus
//   - Published once a minute.
//   - Contains a list of configured chains along with their remaining available notional value, the number of enqueued VAAs
//     and information on zero or more enqueued VAAs.
//   - Only the first 20 enqueued VAAs are include, to constrain the message size.

package governor

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"google.golang.org/protobuf/proto"
)

// Admin command to display status to the log.
func (gov *ChainGovernor) Status() (resp string) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))

	for _, ce := range gov.chains {
		netValue, _, _, err := sumValue(ce.transfers, startTime)
		if err != nil {
			// We don't want to actually return an error or otherwise stop
			// execution in this case. Instead of propagating the error here, print the contents of the
			// error message.
			return fmt.Sprintf("chain: %v, dailyLimit: OVERFLOW. error: %s", ce.emitterChainId, err)
		}
		s1 := fmt.Sprintf("chain: %v, dailyLimit: %v, total: %v, numPending: %v", ce.emitterChainId, ce.dailyLimit, netValue, len(ce.pending))
		resp += s1 + "\n"
		gov.logger.Info(s1)
		if len(ce.pending) != 0 {
			for idx, pe := range ce.pending {
				value, _ := computeValue(pe.amount, pe.token)
				s1 := fmt.Sprintf("chain: %v, pending[%v], value: %v, vaa: %v, timeStamp: %v, releaseTime: %v", ce.emitterChainId, idx, value,
					pe.dbData.Msg.MessageIDString(), pe.dbData.Msg.Timestamp.String(), pe.dbData.ReleaseTime.String())
				gov.logger.Info(s1)
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
		gov.logger.Error("failed to load from the database", zap.Error(err))
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
			msgId := pe.dbData.Msg.MessageIDString()
			if msgId == vaaId {
				value, _ := computeValue(pe.amount, pe.token)
				gov.logger.Info("dropping pending vaa",
					zap.String("msgId", msgId),
					zap.Uint64("value", value),
					zap.Stringer("timeStamp", pe.dbData.Msg.Timestamp),
				)

				if err := gov.db.DeletePendingMsg(&pe.dbData); err != nil {
					return "", err
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
			msgId := pe.dbData.Msg.MessageIDString()
			if msgId == vaaId {
				value, _ := computeValue(pe.amount, pe.token)
				gov.logger.Info("releasing pending vaa, should be published soon",
					zap.String("msgId", msgId),
					zap.Uint64("value", value),
					zap.Stringer("timeStamp", pe.dbData.Msg.Timestamp),
				)

				gov.msgsToPublish = append(gov.msgsToPublish, &pe.dbData.Msg)

				// We delete the pending message from the database, but we don't add it to the transfers
				// because released messages do not apply to the limit.

				if err := gov.db.DeletePendingMsg(&pe.dbData); err != nil {
					return "", err
				}

				ce.pending = append(ce.pending[:idx], ce.pending[idx+1:]...)
				str := fmt.Sprintf("pending vaa \"%v\" has been released and will be published soon", msgId)
				return str, nil
			}
		}
	}

	return "", fmt.Errorf("vaa not found in the pending list")
}

// Admin command to reset the release timer for a pending VAA, extending it to the configured limit.
func (gov *ChainGovernor) ResetReleaseTimer(vaaId string, numDays uint32) (string, error) {
	return gov.resetReleaseTimerForTime(vaaId, time.Now(), numDays)
}

func (gov *ChainGovernor) resetReleaseTimerForTime(vaaId string, now time.Time, numDays uint32) (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		for _, pe := range ce.pending {
			msgId := pe.dbData.Msg.MessageIDString()
			if msgId == vaaId {
				pe.dbData.ReleaseTime = now.Add(time.Duration(numDays) * time.Hour * 24)
				gov.logger.Info("updating the release time due to admin command",
					zap.String("msgId", msgId),
					zap.Stringer("timeStamp", pe.dbData.Msg.Timestamp),
					zap.Stringer("newReleaseTime", pe.dbData.ReleaseTime),
				)

				if err := gov.db.StorePendingMsg(&pe.dbData); err != nil {
					gov.logger.Error("failed to store updated pending vaa", zap.String("msgID", msgId), zap.Error(err))
					return "", err
				}

				str := fmt.Sprintf("release time on pending vaa \"%v\" has been updated to %v", msgId, pe.dbData.ReleaseTime.String())
				return str, nil
			}
		}
	}

	return "", fmt.Errorf("vaa not found in the pending list")

}

// sumValue sums the value of all `transfers`, returning separate fields for:
// - the net sum of all outgoing small tranasfers minus flow cancel sum
// - the sum of all outgoing small tranasfers
// - the sum of all incoming flow-cancelling transfers
// NOTE these sums exclude "big transfers" as they are always queued for 24h and are never added to the chain entry's 'transfers' field.
// Returns an error if the sum of all transfers would overflow the bounds of Int64. In this case, the function
// returns a value of 0.
func sumValue(transfers []transfer, startTime time.Time) (netNotional int64, smallTxOutgoingNotional uint64, flowCancelNotional uint64, err error) {
	if len(transfers) == 0 {
		return 0, 0, 0, nil
	}

	// Sum of all outgoing small tranasfers minus incoming flow cancel transfers. Big transfers are excluded
	netNotional = int64(0)
	smallTxOutgoingNotional = uint64(0)
	flowCancelNotional = uint64(0)

	for _, t := range transfers {
		if t.dbTransfer.Timestamp.Before(startTime) {
			continue
		}
		netNotional, err = CheckedAddInt64(netNotional, t.value)
		if err != nil {
			// We have to stop and return an error here (rather than saturate, for example). The
			// transfers are not sorted by value so we can't make any guarantee on the final value
			// if we hit the upper or lower bound. We don't expect this to happen in any case.
			return 0, 0, 0, err
		}
		if t.value < 0 {
			// If a transfer is negative then it is an incoming, flow-cancelling transfer.
			// We can use the dbTransfer.Value for calculating the sum because it is the unsigned version
			// of t.Value
			flowCancelNotional += t.dbTransfer.Value
		} else {
			smallTxOutgoingNotional += t.dbTransfer.Value
		}
	}

	return netNotional, smallTxOutgoingNotional, flowCancelNotional, nil
}

// REST query to get the current available notional value per chain. This is defined as the sum of all transfers
// subtracted from the chains's dailyLimit.
// The available notional limit by chain represents the remaining capacity of a chain. As a result, it should not be
// a negative number: we don't want to represent that there is "negative value" available.
func (gov *ChainGovernor) GetAvailableNotionalByChain() (resp []*publicrpcv1.GovernorGetAvailableNotionalByChainResponse_Entry) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))

	// Iterate deterministically by accessing keys from this slice instead of the chainEntry map directly
	for _, chainId := range gov.chainIds {
		ce := gov.chains[chainId]
		netUsage, _, incoming, err := sumValue(ce.transfers, startTime)
		if err != nil {
			// Report 0 available notional if we can't calculate the current usage
			gov.logger.Error("GetAvailableNotionalByChain: failed to compute sum of transfers for chain entry",
				zap.String("chainID", chainId.String()),
				zap.Error(err))
			resp = append(resp, &publicrpcv1.GovernorGetAvailableNotionalByChainResponse_Entry{
				ChainId:                    uint32(ce.emitterChainId),
				RemainingAvailableNotional: 0,
				NotionalLimit:              ce.dailyLimit,
				BigTransactionSize:         ce.bigTransactionSize,
			})
			continue
		}

		remaining := gov.availableNotionalValue(chainId, netUsage)

		if !gov.flowCancelEnabled {
			// When flow cancel is disabled, we expect that both the netUsage and remaining notional should be
			// within the range of [0, dailyLimit]. Flow cancel allows flexibility here. netUsage may be
			// negative if there is a lot of incoming flow; conversely, it may exceed dailyLimit if incoming
			// flow added space, allowed additional transfers through, and then expired after 24h.
			// Note that if flow cancel is enabled and then later disabled, netUsage can exceed dailyLimit
			// for 24h as old transfers will be loaded from the database into the Governor, but the flow
			// cancel transfers will not. The value should return to the normal range after 24h has elapsed
			// since the old transfers were sent.
			if netUsage < 0 || incoming != 0 {
				gov.logger.Warn("GetAvailableNotionalByChain: net value for chain is negative even though flow cancel is disabled",
					zap.String("chainID", chainId.String()),
					zap.Uint64("dailyLimit", ce.dailyLimit),
					zap.Int64("netUsage", netUsage),
					zap.Error(err))
			} else if uint64(netUsage) > ce.dailyLimit {
				gov.logger.Warn("GetAvailableNotionalByChain: net value for chain exceeds daily limit even though flow cancel is disabled",
					zap.String("chainID", chainId.String()),
					zap.Uint64("dailyLimit", ce.dailyLimit),
					zap.Error(err))
			}

		}

		resp = append(resp, &publicrpcv1.GovernorGetAvailableNotionalByChainResponse_Entry{
			ChainId:                    uint32(ce.emitterChainId),
			RemainingAvailableNotional: remaining,
			NotionalLimit:              ce.dailyLimit,
			BigTransactionSize:         ce.bigTransactionSize,
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
			value, err := computeValue(pe.amount, pe.token)
			if err != nil {
				gov.logger.Error("failed to compute value of pending transfer", zap.String("msgID", pe.dbData.Msg.MessageIDString()), zap.Error(err))
				value = 0
			}

			resp = append(resp, &publicrpcv1.GovernorGetEnqueuedVAAsResponse_Entry{
				EmitterChain:   uint32(pe.dbData.Msg.EmitterChain),
				EmitterAddress: pe.dbData.Msg.EmitterAddress.String(),
				Sequence:       pe.dbData.Msg.Sequence,
				ReleaseTime:    uint32(pe.dbData.ReleaseTime.Unix()), // #nosec G115 -- This conversion is safe until year 2106
				NotionalValue:  value,
				TxHash:         pe.dbData.Msg.TxIDString(),
			})
		}
	}

	return resp
}

// REST query to see if a VAA is enqueued.
func (gov *ChainGovernor) IsVAAEnqueued(msgId *publicrpcv1.MessageID) (bool, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	if msgId == nil {
		return false, fmt.Errorf("no message ID specified")
	}

	if msgId.GetEmitterChain() > math.MaxUint16 {
		return false, fmt.Errorf("emitter chain id must be no greater than 16 bits")
	}

	emitterChain := vaa.ChainID(msgId.GetEmitterChain()) // #nosec G115 -- This conversion is checked above

	emitterAddress, err := vaa.StringToAddress(msgId.EmitterAddress)
	if err != nil {
		return false, err
	}

	for _, ce := range gov.chains {
		for _, pe := range ce.pending {
			if pe.dbData.Msg.EmitterChain == emitterChain && pe.dbData.Msg.EmitterAddress == emitterAddress && pe.dbData.Msg.Sequence == msgId.Sequence {
				return true, nil
			}
		}
	}

	return false, nil
}

// availableNotionalValue calculates the available notional USD value for a chain entry based on the net value
// of the chain.
func (gov *ChainGovernor) availableNotionalValue(id vaa.ChainID, netUsage int64) uint64 {
	remaining := uint64(0)
	ce := gov.chains[id]

	// Handle negative case here so we can safely cast to uint64 below
	if netUsage < 0 {
		// The full capacity is available for the chain.
		remaining = ce.dailyLimit
	} else if uint64(netUsage) > ce.dailyLimit {
		remaining = 0
	} else {
		remaining = ce.dailyLimit - uint64(netUsage)
	}

	return remaining
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

func (gov *ChainGovernor) CollectMetrics(ctx context.Context, hb *gossipv1.Heartbeat, sendC chan<- []byte, guardianSigner guardiansigner.GuardianSigner, ourAddr ethCommon.Address) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	totalPending := 0
	for _, n := range hb.Networks {
		if n == nil {
			continue
		}

		if n.Id > math.MaxUint16 {
			gov.logger.Error("CollectMetrics: chain id is not a valid uint16", zap.Uint32("chain_id", n.Id))
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
			netUsage, _, _, err := sumValue(ce.transfers, startTime)

			remaining := uint64(0)
			if err != nil {
				// Error can occur if the sum overflows. Return 0 in this case rather than returning an
				// error.
				gov.logger.Error("CollectMetrics: failed to compute sum of transfers for chain entry", zap.String("chain", chain.String()), zap.Error(err))
				remaining = 0
			} else {
				remaining = gov.availableNotionalValue(chain, netUsage)
			}

			pending := len(ce.pending)
			totalNotional = fmt.Sprint(ce.dailyLimit)
			available = float64(remaining)
			numPending = float64(pending)
			totalPending += pending
		}

		//"chain_id", "chain_name", "enabled", "total_notional"
		metricAvailableNotional.WithLabelValues(
			chainId,        // chain_id
			chain.String(), // chain_name
			enabled,        // enabled
			totalNotional,  // total_notional
		).Set(available)

		metricEnqueuedVAAs.WithLabelValues(
			chainId,        // chain_id
			chain.String(), // chain_name
			enabled,        // enabled
		).Set(numPending)
	}

	metricTotalEnqueuedVAAs.Set(float64(totalPending))

	if startTime.After(gov.nextConfigPublishTime) {
		gov.publishConfig(ctx, hb, sendC, guardianSigner, ourAddr)
		gov.nextConfigPublishTime = startTime.Add(time.Minute * time.Duration(5))
	}

	if startTime.After(gov.nextStatusPublishTime) {
		gov.publishStatus(ctx, hb, sendC, startTime, guardianSigner, ourAddr)
		gov.nextStatusPublishTime = startTime.Add(time.Minute)
	}
}

var governorMessagePrefixConfig = []byte("governor_config_000000000000000000|")
var governorMessagePrefixStatus = []byte("governor_status_000000000000000000|")

func (gov *ChainGovernor) publishConfig(ctx context.Context, hb *gossipv1.Heartbeat, sendC chan<- []byte, guardianSigner guardiansigner.GuardianSigner, ourAddr ethCommon.Address) {
	chains := make([]*gossipv1.ChainGovernorConfig_Chain, 0)
	// Iterate deterministically by accessing keys from this slice instead of the chainEntry map directly
	for _, cid := range gov.chainIds {
		ce := gov.chains[cid]
		chains = append(chains, &gossipv1.ChainGovernorConfig_Chain{
			ChainId:            uint32(ce.emitterChainId),
			NotionalLimit:      ce.dailyLimit,
			BigTransactionSize: ce.bigTransactionSize,
		})
	}

	tokens := make([]*gossipv1.ChainGovernorConfig_Token, 0)
	for tk, te := range gov.tokens {
		price, _ := te.price.Float32()
		tokens = append(tokens, &gossipv1.ChainGovernorConfig_Token{
			OriginChainId: uint32(tk.chain),
			OriginAddress: "0x" + tk.addr.String(),
			Price:         price,
		})
	}

	gov.configPublishCounter += 1
	payload := &gossipv1.ChainGovernorConfig{
		NodeName:          hb.NodeName,
		Counter:           gov.configPublishCounter,
		Timestamp:         hb.Timestamp,
		Chains:            chains,
		Tokens:            tokens,
		FlowCancelEnabled: gov.flowCancelEnabled,
	}

	b, err := proto.Marshal(payload)
	if err != nil {
		gov.logger.Error("failed to marshal config message", zap.Error(err))
		return
	}

	digest := ethCrypto.Keccak256Hash(append(governorMessagePrefixConfig, b...))

	sig, err := guardianSigner.Sign(ctx, digest.Bytes())
	if err != nil {
		panic(err)
	}

	msg := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedChainGovernorConfig{
		SignedChainGovernorConfig: &gossipv1.SignedChainGovernorConfig{
			Config:       b,
			Signature:    sig,
			GuardianAddr: ourAddr.Bytes(),
		}}}

	b, err = proto.Marshal(&msg)
	if err != nil {
		panic(err)
	}

	sendC <- b
}

func (gov *ChainGovernor) publishStatus(ctx context.Context, hb *gossipv1.Heartbeat, sendC chan<- []byte, startTime time.Time, guardianSigner guardiansigner.GuardianSigner, ourAddr ethCommon.Address) {
	chains := make([]*gossipv1.ChainGovernorStatus_Chain, 0)
	numEnqueued := 0
	for chainId, ce := range gov.chains {
		// The capacity for the chain to emit further messages, denoted as USD value.
		remaining := uint64(0)
		netUsage, smallTxNotional, flowCancelNotional, err := sumValue(ce.transfers, startTime)

		if err != nil {
			gov.logger.Error("publishStatus: failed to compute sum of transfers for chain entry", zap.String("chain", chainId.String()), zap.Error(err))
		} else {
			remaining = gov.availableNotionalValue(chainId, netUsage)
		}

		enqueuedVaas := make([]*gossipv1.ChainGovernorStatus_EnqueuedVAA, 0)
		for _, pe := range ce.pending {
			value, err := computeValue(pe.amount, pe.token)
			if err != nil {
				gov.logger.Error("failed to compute value of pending transfer", zap.String("msgID", pe.dbData.Msg.MessageIDString()), zap.Error(err))
				value = 0
			}

			if numEnqueued < 20 {
				numEnqueued = numEnqueued + 1
				enqueuedVaas = append(enqueuedVaas, &gossipv1.ChainGovernorStatus_EnqueuedVAA{
					Sequence:      pe.dbData.Msg.Sequence,
					ReleaseTime:   uint32(pe.dbData.ReleaseTime.Unix()), // #nosec G115 -- This conversion is safe until year 2106
					NotionalValue: value,
					TxHash:        pe.dbData.Msg.TxIDString(),
				})
			}
		}

		emitter := gossipv1.ChainGovernorStatus_Emitter{
			EmitterAddress:    "0x" + ce.emitterAddr.String(),
			TotalEnqueuedVaas: uint64(len(ce.pending)),
			EnqueuedVaas:      enqueuedVaas,
		}

		chains = append(chains, &gossipv1.ChainGovernorStatus_Chain{
			ChainId:                      uint32(ce.emitterChainId),
			RemainingAvailableNotional:   remaining,
			Emitters:                     []*gossipv1.ChainGovernorStatus_Emitter{&emitter},
			SmallTxNetNotionalValue:      netUsage,
			SmallTxOutgoingNotionalValue: smallTxNotional,
			FlowCancelNotionalValue:      flowCancelNotional,
		})
	}

	gov.statusPublishCounter += 1
	payload := &gossipv1.ChainGovernorStatus{
		NodeName:  hb.NodeName,
		Counter:   gov.statusPublishCounter,
		Timestamp: hb.Timestamp,
		Chains:    chains,
	}

	b, err := proto.Marshal(payload)
	if err != nil {
		gov.logger.Error("failed to marshal status message", zap.Error(err))
		return
	}

	digest := ethCrypto.Keccak256Hash(append(governorMessagePrefixStatus, b...))

	sig, err := guardianSigner.Sign(ctx, digest.Bytes())
	if err != nil {
		panic(err)
	}

	msg := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedChainGovernorStatus{
		SignedChainGovernorStatus: &gossipv1.SignedChainGovernorStatus{
			Status:       b,
			Signature:    sig,
			GuardianAddr: ourAddr.Bytes(),
		}}}

	b, err = proto.Marshal(&msg)
	if err != nil {
		panic(err)
	}

	sendC <- b
}
