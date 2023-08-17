// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/relayer/TypedUnits.sol";
import "../../interfaces/relayer/IWormholeRelayerTyped.sol";

struct DeliveryInstruction {
    uint16 targetChain;
    bytes32 targetAddress;
    bytes payload;
    TargetNative requestedReceiverValue;
    TargetNative extraReceiverValue;
    bytes encodedExecutionInfo;
    uint16 refundChain;
    bytes32 refundAddress;
    bytes32 refundDeliveryProvider;
    bytes32 sourceDeliveryProvider;
    bytes32 senderAddress;
    MessageKey[] messageKeys;
}

// Meant to hold all necessary values for `CoreRelayerDelivery::executeInstruction`
// Nothing more and nothing less.
struct EvmDeliveryInstruction {
  uint16 sourceChain;
  bytes32 targetAddress;
  bytes payload;
  Gas gasLimit;
  TargetNative totalReceiverValue;
  GasPrice targetChainRefundPerGasUnused;
  bytes32 senderAddress;
  bytes32 deliveryHash;
  bytes[] signedVaas;
}

struct RedeliveryInstruction {
    VaaKey deliveryVaaKey;
    uint16 targetChain;
    TargetNative newRequestedReceiverValue;
    bytes newEncodedExecutionInfo;
    bytes32 newSourceDeliveryProvider;
    bytes32 newSenderAddress;
}

/**
 * @notice When a user requests a `resend()`, a `RedeliveryInstruction` is emitted by the
 *     WormholeRelayer and in turn converted by the relay provider into an encoded (=serialized)
 *     `DeliveryOverride` struct which is then passed to `delivery()` to override the parameters of
 *     a previously failed delivery attempt.
 *
 * @custom:member newReceiverValue - must >= than the `receiverValue` specified in the original
 *     `DeliveryInstruction`
 * @custom:member newExecutionInfo - for EVM_V1, must contain a gasLimit and targetChainRefundPerGasUnused
 * such that 
 * - gasLimit is >= the `gasLimit` specified in the `executionParameters`
 *     of the original `DeliveryInstruction`
 * - targetChainRefundPerGasUnused is >=  the `targetChainRefundPerGasUnused` specified in the original
 *     `DeliveryInstruction`
 * @custom:member redeliveryHash - the hash of the redelivery which is being performed
 */
struct DeliveryOverride {
    TargetNative newReceiverValue;
    bytes newExecutionInfo;
    bytes32 redeliveryHash;
}
