// contracts/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface IDelivery {
    function deliverSingle(TargetDeliveryParametersSingle memory targetParams) external payable;

    function redeliverSingle(TargetRedeliveryByTxHashParamsSingle memory targetParams) external payable;

    struct TargetDeliveryParametersSingle {
        // encoded batchVM to be delivered on the target chain
        bytes[] encodedVMs;
        // Index of the delivery VM in a batch
        uint8 deliveryIndex;
        // Index of the target chain inside the delivery VM
        uint8 multisendIndex;
        //refund address
        address payable relayerRefundAddress;
    }

    struct TargetRedeliveryByTxHashParamsSingle {
        bytes redeliveryVM;
        bytes[] sourceEncodedVMs;
        address payable relayerRefundAddress;
    }

    error InvalidEmitterInOriginalDeliveryVM(uint8 index);
    error InvalidRedeliveryVM(string reason);
    error InvalidEmitterInRedeliveryVM();
    error MismatchingRelayProvidersInRedelivery(); // The same relay provider must be specified when doing a single VAA redeliver
    error UnexpectedRelayer(); // msg.sender must be the provider
    error InvalidVaa(uint8 index);
    error InvalidEmitter();
    error SendNotSufficientlyFunded(); // This delivery request was not sufficiently funded, and must request redelivery
    error InsufficientRelayerFunds(); // The relayer didn't pass sufficient funds (msg.value does not cover the necessary budget fees)
    error TargetChainIsNotThisChain(uint16 targetChainId);
}
