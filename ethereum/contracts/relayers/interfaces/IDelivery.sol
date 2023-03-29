// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface IDelivery {
    /**
     * @notice TargetDeliveryParametersSingle is the struct that the relay provider passes into 'deliverSingle'
     * containing an array of the signed wormhole messages that are to be relayed
     *
     * @custom:member encodedVMs An array of signed wormhole messages (all from the same source chain transaction)
     * @custom:member encodedDeliveryVAA signed wormhole message from the source chain's CoreRelayer contract with payload being the encoded delivery instruction container
     * @custom:member multisendIndex The delivery instruction container in encodedDeliveryVAA contains many delivery instructions, each specifying a different destination address
     * This 'multisendIndex' indicates which of those delivery instructions should be executed (specifically, the instruction deliveryInstructionsContainer.instructions[multisendIndex])
     * @custom:member relayerRefundAddress The address to which any refunds to the relay provider should be sent
     */
    struct TargetDeliveryParametersSingle {
        bytes[] encodedVMs;
        bytes encodedDeliveryVAA;
        uint8 multisendIndex;
        address payable relayerRefundAddress;
    }

    /**
     * @notice The relay provider calls 'deliverSingle' to relay messages as described by one delivery instruction
     *
     * The instruction specifies the target chain (must be this chain), target address, refund address, maximum refund (in this chain's currency),
     * receiver value (in this chain's currency), upper bound on gas, and the permissioned address allowed to execute this instruction
     *
     * The relay provider must pass in the signed wormhole messages (VAAs) from the source chain
     * as well as the signed wormhole message with the delivery instructions (the delivery VAA)
     * as well as identify which of the many instructions in the multichainSend container is meant to be executed
     *
     * The messages will be relayed to the target address (with the specified gas limit and receiver value) iff the following checks are met:
     * - the delivery VAA has a valid signature
     * - the delivery VAA's emitter is one of these CoreRelayer contracts
     * - the delivery instruction container in the delivery VAA was fully funded
     * - msg.sender is the permissioned address allowed to execute this instruction
     * - the relay provider passed in at least [(one wormhole message fee) + instruction.maximumRefundTarget + instruction.receiverValueTarget] of this chain's currency as msg.value
     * - the instruction's target chain is this chain
     * - the relayed signed VAAs match the descriptions in container.messages (the VAA hashes match, or the emitter address, sequence number pair matches, depending on the description given)
     *
     * @param targetParams struct containing the signed wormhole messages and encoded delivery instruction container (and other information)
     */
    function deliverSingle(TargetDeliveryParametersSingle memory targetParams) external payable;

    /**
     * @notice TargetRedeliveryByTxHashParamsSingle is the struct that the relay provider passes into 'redeliverSingle'
     * containing an array of the signed wormhole messages that are to be relayed
     *
     * @custom:member redeliveryVM The signed wormhole message from the source chain's CoreRelayer contract with payload being the encoded redelivery instruction
     * @custom:member sourceEncodedVMs An array of signed wormhole messages (all from the same source chain transaction), which are the original messages that are meant to be redelivered
     * @custom:member originalEncodedDeliveryVAA original signed wormhole message from the source chain's CoreRelayer contract with payload being the original encoded delivery instruction container
     * @custom:member relayerRefundAddress The address to which any refunds to the relay provider should be sent
     */
    struct TargetRedeliveryByTxHashParamsSingle {
        bytes redeliveryVM;
        bytes[] sourceEncodedVMs;
        bytes originalEncodedDeliveryVAA;
        address payable relayerRefundAddress;
    }

    /**
     * @notice The relay provider calls 'redeliverSingle' to relay messages as described by one redelivery instruction
     *
     * The instruction specifies, among other things, the target chain (must be this chain), refund address, new maximum refund (in this chain's currency),
     * new receiverValue (in this chain's currency), new upper bound on gas
     *
     * The relay provider must pass in the signed wormhole message with the new redelivery instructions (the redelivery VAA)
     * as well as the original signed wormhole messages (VAAs) from the source chain
     * as well as the original signed wormhole message with the delivery instructions (the delivery VAA)
     *
     * The messages will be relayed to the target address (with the specified gas limit and receiver value) iff the following checks are met:
     * - the redelivery VAA (targetParams.redeliveryVM) has a valid signature
     * - the redelivery VAA's emitter is one of these CoreRelayer contracts
     * - the original delivery VAA has a valid signature
     * - the original delivery VAA's emitter is one of these CoreRelayer contracts
     * - the original signed VAAs match the descriptions from the original delivery instructions (the VAA hashes match, or the emitter address, sequence number pair matches, depending on the description given)
     * - the new redelivery instruction's upper bound on gas >= the original instruction's upper bound on gas
     * - the new redelivery instruction's 'receiver value' amount >= the original instruction's 'receiver value' amount
     * - the redelivery instruction's target chain = the original instruction's target chain = this chain
     * - for the redelivery instruction, the relay provider passed in at least [(one wormhole message fee) + instruction.newMaximumRefundTarget + instruction.newReceiverValueTarget] of this chain's currency as msg.value
     * - msg.sender is the permissioned address allowed to execute this redelivery instruction
     * - msg.sender is the permissioned address allowed to execute the old instruction
     *
     * @param targetParams struct containing the signed wormhole messages and encoded redelivery instruction (and other information)
     */
    function redeliverSingle(TargetRedeliveryByTxHashParamsSingle memory targetParams) external payable;

    error InvalidEmitterInRedeliveryVM(); // The redelivery VAA (signed wormhole message with redelivery instructions) has an invalid sender
    error InvalidEmitterInOriginalDeliveryVM(); // The original delivery VAA (original signed wormhole message with delivery instructions) has an invalid sender
    error InvalidRedeliveryVM(string reason); // The redelivery VAA is not valid
    error InvalidVaa(uint8 index, string reason); // The VAA is not valid
    error InvalidDeliveryVaa(string reason); // The Delivery VAA is not valid
    error MismatchingRelayProvidersInRedelivery(); // The relay provider specified for the redelivery is different from the relay provider specified for the original delivery
    error UnexpectedRelayer(); // msg.sender must be the delivery address of the specified relay provider
    error InvalidEmitter(); // The delivery VAA (signed wormhole message with delivery instructions) has an invalid sender
    error SendNotSufficientlyFunded(); // The container of delivery instructions (for which this current delivery was in) was not fully funded on the source chain
    error InsufficientRelayerFunds(); // The relay provider didn't pass in sufficient funds (msg.value does not cover the necessary budget fees)
    error TargetChainIsNotThisChain(uint16 targetChainId); // The specified target chain is not the current chain
    error ReentrantCall(); // A delivery cannot occur during another delivery
    error MessageInfosDoNotMatchVaas(uint8 index); // The VAA at index 'index' does not match the 'index'-th description given on the source chain in the 'messages' field
    error MessageInfosLengthDoesNotMatchVaasLength(); // The VAA array has a different length than the original array of MessageInfo descriptions from the source chain
}
