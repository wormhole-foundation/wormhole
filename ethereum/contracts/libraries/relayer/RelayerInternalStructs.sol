// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/relayer/TypedUnits.sol";
import "../../interfaces/relayer/IWormholeRelayer.sol";

struct DeliveryInstruction {
    uint16 targetChain;
    bytes32 targetAddress;
    bytes payload;
    Wei requestedReceiverValue;
    Wei extraReceiverValue;
    bytes encodedExecutionInfo;
    uint16 refundChain;
    bytes32 refundAddress;
    bytes32 refundRelayProvider;
    bytes32 sourceRelayProvider;
    bytes32 senderAddress;
    VaaKey[] vaaKeys;
}

struct RedeliveryInstruction {
    VaaKey deliveryVaaKey;
    uint16 targetChain;
    Wei newRequestedReceiverValue;
    bytes newEncodedExecutionInfo;
    bytes32 newSourceRelayProvider;
    bytes32 newSenderAddress;
}

/**
 * @notice When a user requests a `resend()`, a `RedeliveryInstruction` is emitted by the
 *     WormholeRelayer and in turn converted by the relay provider into an encoded (=serialized)
 *     `DeliveryOverride` struct which is then passed to `delivery()` to override the parameters of
 *     a previously failed delivery attempt.
 *
 * @custom:member gasLimit - must be >= than the `gasLimit` specified in the `executionParameters`
 *     of the original `DeliveryInstruction`
 * @custom:member maximumRefund - must >= than the `maximumRefund` specified in the original
 *     `DeliveryInstruction`
 * @custom:member receiverValue - must >= than the `receiverValue` specified in the original
 *     `DeliveryInstruction`
 * @custom:member redeliveryHash - the hash of the redelivery which is being performed
 */
struct DeliveryOverride {
    Wei newReceiverValue;
    bytes newExecutionInfo;
    bytes32 redeliveryHash;
}
