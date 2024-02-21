package accountant

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// nttEnabled returns true if NTT is enabled, false if not.
func (acct *Accountant) nttEnabled() bool {
	return acct.nttContract != ""
}

// nttStart initializes the NTT accountant and starts the NTT specific worker and watcher runnables.
func (acct *Accountant) nttStart(ctx context.Context) error {
	acct.logger.Debug("entering nttStart", zap.Bool("enforceFlag", acct.enforceFlag))

	var err error
	acct.nttDirectEmitters, acct.nttArEmitters, err = nttGetEmitters(acct.env)
	if err != nil {
		return fmt.Errorf("failed to set up NTT emitters: %w", err)
	}

	for emitter := range acct.nttDirectEmitters {
		acct.logger.Info("will monitor NTT emitter", zap.Stringer("emitterChainId", emitter.emitterChainId), zap.Stringer("emitterAddr", emitter.emitterAddr))
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
	if len(payload) < 84 {
		return false
	}

	if !bytes.Equal(payload[0:4], WH_PREFIX) {
		return false
	}

	if !bytes.Equal(payload[80:84], NTT_PREFIX) {
		return false
	}

	return true
}

// isMsgDirectNTT determines if a message publication is for a Native Token Transfer directly from an NTT endpoint.
func nttIsMsgDirectNTT(msg *common.MessagePublication, emitters validEmitters) bool {
	// Look up the emitter in the NTT map, return false if it's not there.
	if _, exists := emitters[emitterKey{emitterChainId: msg.EmitterChain, emitterAddr: msg.EmitterAddress}]; !exists {
		return false
	}
	return nttIsPayloadNTT(msg.Payload)
}

// nttIsMsgArNTT determines if a message publication is for a Native Token Transfer forwarded from an automatic relayer.
// It first checks if the emitter is a configured relayer. If so, it parses the AR payload to see get the sender and
// checks to see if the emitter chain / sender address are for a Native Token Transfer emitter.
func nttIsMsgArNTT(msg *common.MessagePublication, arEmitters validEmitters, nttEmitters validEmitters) bool {
	// Look up the emitter in the AR map, return false if it's not there.
	if _, exists := arEmitters[emitterKey{emitterChainId: msg.EmitterChain, emitterAddr: msg.EmitterAddress}]; !exists {
		return false
	}

	return nttIsArPayloadNTT(msg.EmitterChain, msg.Payload, nttEmitters)
}

// nttIsArPayloadNTT extracts the sender from the AR payload and determines if it is a native token transfer. This is based on the following implementation:
// https://github.com/wormhole-foundation/wormhole/blob/main/ethereum/contracts/relayer/wormholeRelayer/WormholeRelayerSerde.sol#L70-L97
func nttIsArPayloadNTT(emitterChain vaa.ChainID, msgPayload []byte, nttEmitters validEmitters) bool {
	reader := bytes.NewReader(msgPayload[:])

	var deliveryInstruction uint8
	if err := binary.Read(reader, binary.BigEndian, &deliveryInstruction); err != nil {
		return false
	}

	if deliveryInstruction != 1 { // PAYLOAD_ID_DELIVERY_INSTRUCTION
		return false
	}

	var targetChain uint16
	if err := binary.Read(reader, binary.BigEndian, &targetChain); err != nil {
		return false
	}

	var targetAddress [32]byte
	if n, err := reader.Read(targetAddress[:]); err != nil || n != 32 {
		return false
	}

	var payloadLen uint32
	if err := binary.Read(reader, binary.BigEndian, &payloadLen); err != nil {
		return false
	}

	payload := make([]byte, payloadLen)
	if n, err := reader.Read(payload[:]); err != nil || n != int(payloadLen) {
		return false
	}

	var requestedReceiverValue [32]byte
	if n, err := reader.Read(requestedReceiverValue[:]); err != nil || n != 32 {
		return false
	}

	var extraReceiverValue [32]byte
	if n, err := reader.Read(extraReceiverValue[:]); err != nil || n != 32 {
		return false
	}

	var encodedExecutionInfoLen uint32
	if err := binary.Read(reader, binary.BigEndian, &encodedExecutionInfoLen); err != nil {
		return false
	}

	encodedExecutionInfo := make([]byte, encodedExecutionInfoLen)
	if n, err := reader.Read(encodedExecutionInfo[:]); err != nil || n != int(encodedExecutionInfoLen) {
		return false
	}

	var refundChain uint16
	if err := binary.Read(reader, binary.BigEndian, &refundChain); err != nil {
		return false
	}

	var refundAddress [32]byte
	if n, err := reader.Read(refundAddress[:]); err != nil || n != 32 {
		return false
	}

	var refundDeliveryProvider [32]byte
	if n, err := reader.Read(refundDeliveryProvider[:]); err != nil || n != 32 {
		return false
	}

	var sourceDeliveryProvider [32]byte
	if n, err := reader.Read(sourceDeliveryProvider[:]); err != nil || n != 32 {
		return false
	}

	var senderAddress [32]byte
	if n, err := reader.Read(senderAddress[:]); err != nil || n != 32 {
		return false
	}

	// See if msg.EmitterChain / ar.Sender is in nttEmitters.
	if _, exists := nttEmitters[emitterKey{emitterChainId: emitterChain, emitterAddr: senderAddress}]; !exists {
		return false
	}

	return true
}
