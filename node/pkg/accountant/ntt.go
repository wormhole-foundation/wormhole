package accountant

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"go.uber.org/zap"
)

// nttEnabled returns true if NTT is enabled, false if not.
func (acct *Accountant) nttEnabled() bool {
	return acct.nttContract != ""
}

// nttStart initializes the NTT accountant and starts the NTT specific worker and watcher runnables.
func (acct *Accountant) nttStart(ctx context.Context) error {
	acct.logger.Debug("entering nttStart")

	var err error
	acct.nttDirectEmitters, acct.nttArEmitters, err = nttGetEmitters(acct.env)
	if err != nil {
		return fmt.Errorf("failed to set up NTT emitters: %w", err)
	}

	for emitter, enforceFlag := range acct.nttDirectEmitters {
		tag := ""
		if !enforceFlag {
			tag = " in log only mode"
		}
		acct.logger.Info(fmt.Sprintf("will monitor%s for NTT emitter", tag), zap.Stringer("emitterChainId", emitter.emitterChainId), zap.Stringer("emitterAddr", emitter.emitterAddr))
	}

	for emitter := range acct.nttArEmitters {
		acct.logger.Info("will monitor AR emitter for NTT", zap.Stringer("emitterChainId", emitter.emitterChainId), zap.Stringer("emitterAddr", emitter.emitterAddr))
	}

	// Start the watcher to listen to transfer events from the smart contract.
	if acct.env == common.AccountantMock {
		// We're not in a runnable context, so we can't use supervisor.
		go func() {
			_ = acct.nttWorker(ctx)
		}()
	} else if acct.env != common.GoTest {
		if err := supervisor.Run(ctx, "nttacctworker", common.WrapWithScissors(acct.nttWorker, "nttacctworker")); err != nil {
			return fmt.Errorf("failed to start NTT submit observation worker: %w", err)
		}

		if err := supervisor.Run(ctx, "nttacctwatcher", common.WrapWithScissors(acct.nttWatcher, "nttacctwatcher")); err != nil {
			return fmt.Errorf("failed to start NTT watcher: %w", err)
		}
	}

	return nil
}

var WH_PREFIX = []byte{0x99, 0x45, 0xFF, 0x10}
var NTT_PREFIX = []byte{0x99, 0x4E, 0x54, 0x54}

const NTT_PREFIX_OFFSET = 136
const NTT_PREFIX_END = NTT_PREFIX_OFFSET + 4

// isNTT determines if the payload bytes are for a Native Token Transfer, according to the following implementation:
// https://github.com/wormhole-foundation/example-native-token-transfers/blob/22bde0c7d8139675582d861dc8245eb1912324fa/evm/test/TransceiverStructs.t.sol#L42
func nttIsPayloadNTT(payload []byte) bool {
	if len(payload) < NTT_PREFIX_END {
		return false
	}

	if !bytes.Equal(payload[0:4], WH_PREFIX) {
		return false
	}

	if !bytes.Equal(payload[NTT_PREFIX_OFFSET:NTT_PREFIX_END], NTT_PREFIX) {
		return false
	}

	return true
}

// isMsgDirectNTT determines if a message publication is for a Native Token Transfer directly from an NTT endpoint.
// It also returns if NTT accounting should be enforced for this emitter.
func nttIsMsgDirectNTT(msg *common.MessagePublication, emitters validEmitters) (bool, bool) {
	enforceFlag, exists := emitters[emitterKey{emitterChainId: msg.EmitterChain, emitterAddr: msg.EmitterAddress}]
	if !exists {
		return false, false
	}
	if !nttIsPayloadNTT(msg.Payload) {
		return false, false
	}
	return true, enforceFlag
}

// nttIsMsgArNTT determines if a message publication is for a Native Token Transfer forwarded from an automatic relayer.
// It first checks if the emitter is a configured relayer. If so, it parses the AR payload to get the sender address and
// checks to see if the emitter chain / sender address are for a Native Token Transfer emitter.
// It also returns if NTT accounting should be enforced for this emitter.
func nttIsMsgArNTT(msg *common.MessagePublication, arEmitters validEmitters, nttEmitters validEmitters) (bool, bool) {
	if _, exists := arEmitters[emitterKey{emitterChainId: msg.EmitterChain, emitterAddr: msg.EmitterAddress}]; !exists {
		return false, false
	}

	if success, senderAddress, nttPayload := nttParseArPayload(msg.Payload); success {
		// If msg.EmitterChain / ar.Sender is in nttEmitters then this is a Native Token Transfer.
		if enforceFlag, exists := nttEmitters[emitterKey{emitterChainId: msg.EmitterChain, emitterAddr: senderAddress}]; exists {
			if nttIsPayloadNTT(nttPayload) {
				return true, enforceFlag
			}
		}
	}

	return false, false
}

const PAYLOAD_ID_DELIVERY_INSTRUCTION = uint8(1)

// nttParseArPayload extracts the sender address from an AR payload. This is based on the following implementation:
// https://github.com/wormhole-foundation/wormhole/blob/main/ethereum/contracts/relayer/wormholeRelayer/WormholeRelayerSerde.sol#L70-L97
func nttParseArPayload(msgPayload []byte) (bool, [32]byte, []byte) {
	var senderAddress [32]byte
	reader := bytes.NewReader(msgPayload[:])

	var deliveryInstruction uint8
	if err := binary.Read(reader, binary.BigEndian, &deliveryInstruction); err != nil {
		return false, senderAddress, nil
	}

	if deliveryInstruction != PAYLOAD_ID_DELIVERY_INSTRUCTION {
		return false, senderAddress, nil
	}

	var targetChain uint16
	if err := binary.Read(reader, binary.BigEndian, &targetChain); err != nil {
		return false, senderAddress, nil
	}

	var targetAddress [32]byte
	if n, err := reader.Read(targetAddress[:]); err != nil || n != 32 {
		return false, senderAddress, nil
	}

	var payloadLen uint32
	if err := binary.Read(reader, binary.BigEndian, &payloadLen); err != nil {
		return false, senderAddress, nil
	}

	payload := make([]byte, payloadLen)
	if n, err := reader.Read(payload[:]); err != nil || n != int(payloadLen) {
		return false, senderAddress, nil
	}

	var requestedReceiverValue [32]byte
	if n, err := reader.Read(requestedReceiverValue[:]); err != nil || n != 32 {
		return false, senderAddress, nil
	}

	var extraReceiverValue [32]byte
	if n, err := reader.Read(extraReceiverValue[:]); err != nil || n != 32 {
		return false, senderAddress, nil
	}

	var encodedExecutionInfoLen uint32
	if err := binary.Read(reader, binary.BigEndian, &encodedExecutionInfoLen); err != nil {
		return false, senderAddress, nil
	}

	encodedExecutionInfo := make([]byte, encodedExecutionInfoLen)
	if n, err := reader.Read(encodedExecutionInfo[:]); err != nil || n != int(encodedExecutionInfoLen) {
		return false, senderAddress, nil
	}

	var refundChain uint16
	if err := binary.Read(reader, binary.BigEndian, &refundChain); err != nil {
		return false, senderAddress, nil
	}

	var refundAddress [32]byte
	if n, err := reader.Read(refundAddress[:]); err != nil || n != 32 {
		return false, senderAddress, nil
	}

	var refundDeliveryProvider [32]byte
	if n, err := reader.Read(refundDeliveryProvider[:]); err != nil || n != 32 {
		return false, senderAddress, nil
	}

	var sourceDeliveryProvider [32]byte
	if n, err := reader.Read(sourceDeliveryProvider[:]); err != nil || n != 32 {
		return false, senderAddress, nil
	}

	if n, err := reader.Read(senderAddress[:]); err != nil || n != 32 {
		return false, senderAddress, nil
	}

	return true, senderAddress, payload
}
