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

// isNTT determines if the payload bytes are for a Native Token Transfer, according to the following implementation:
// https://github.com/wormhole-foundation/example-native-token-transfers/blob/41ac7baae5bb0f60fff2ec87603970af39dced01/test/EndpointStructs.t.sol
func nttIsPayloadNTT(payload []byte) bool {
	if len(payload) < 140 {
		return false
	}

	if !bytes.Equal(payload[0:4], WH_PREFIX) {
		return false
	}

	if !bytes.Equal(payload[136:140], NTT_PREFIX) {
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

	if success, senderAddress := nttParseArPayload(msg.Payload); success {
		// If msg.EmitterChain / ar.Sender is in nttEmitters then this is a Native Token Transfer.
		if enforceFlag, exists := nttEmitters[emitterKey{emitterChainId: msg.EmitterChain, emitterAddr: senderAddress}]; exists {
			return true, enforceFlag
		}
	}

	return false, false
}

// nttParseArPayload extracts the sender address from an AR payload. This is based on the following implementation:
// https://github.com/wormhole-foundation/wormhole/blob/main/ethereum/contracts/relayer/wormholeRelayer/WormholeRelayerSerde.sol#L70-L97
func nttParseArPayload(msgPayload []byte) (bool, [32]byte) {
	var senderAddress [32]byte
	reader := bytes.NewReader(msgPayload[:])

	var deliveryInstruction uint8
	if err := binary.Read(reader, binary.BigEndian, &deliveryInstruction); err != nil {
		return false, senderAddress
	}

	if deliveryInstruction != 1 { // PAYLOAD_ID_DELIVERY_INSTRUCTION
		return false, senderAddress
	}

	var targetChain uint16
	if err := binary.Read(reader, binary.BigEndian, &targetChain); err != nil {
		return false, senderAddress
	}

	var targetAddress [32]byte
	if n, err := reader.Read(targetAddress[:]); err != nil || n != 32 {
		return false, senderAddress
	}

	var payloadLen uint32
	if err := binary.Read(reader, binary.BigEndian, &payloadLen); err != nil {
		return false, senderAddress
	}

	payload := make([]byte, payloadLen)
	if n, err := reader.Read(payload[:]); err != nil || n != int(payloadLen) {
		return false, senderAddress
	}

	var requestedReceiverValue [32]byte
	if n, err := reader.Read(requestedReceiverValue[:]); err != nil || n != 32 {
		return false, senderAddress
	}

	var extraReceiverValue [32]byte
	if n, err := reader.Read(extraReceiverValue[:]); err != nil || n != 32 {
		return false, senderAddress
	}

	var encodedExecutionInfoLen uint32
	if err := binary.Read(reader, binary.BigEndian, &encodedExecutionInfoLen); err != nil {
		return false, senderAddress
	}

	encodedExecutionInfo := make([]byte, encodedExecutionInfoLen)
	if n, err := reader.Read(encodedExecutionInfo[:]); err != nil || n != int(encodedExecutionInfoLen) {
		return false, senderAddress
	}

	var refundChain uint16
	if err := binary.Read(reader, binary.BigEndian, &refundChain); err != nil {
		return false, senderAddress
	}

	var refundAddress [32]byte
	if n, err := reader.Read(refundAddress[:]); err != nil || n != 32 {
		return false, senderAddress
	}

	var refundDeliveryProvider [32]byte
	if n, err := reader.Read(refundDeliveryProvider[:]); err != nil || n != 32 {
		return false, senderAddress
	}

	var sourceDeliveryProvider [32]byte
	if n, err := reader.Read(sourceDeliveryProvider[:]); err != nil || n != 32 {
		return false, senderAddress
	}

	if n, err := reader.Read(senderAddress[:]); err != nil || n != 32 {
		return false, senderAddress
	}

	return true, senderAddress
}
