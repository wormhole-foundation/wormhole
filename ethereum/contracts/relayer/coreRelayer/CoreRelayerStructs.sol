// contracts/Structs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/relayer/IWormholeRelayer.sol";

abstract contract CoreRelayerStructs {
    struct DeliveryInstructionsContainer {
        uint8 payloadId; //1
        bool sufficientlyFunded;
        IWormholeRelayer.MessageInfo[] messages;
        DeliveryInstruction[] instructions;
    }

    struct DeliveryInstruction {
        uint16 targetChain;
        bytes32 targetAddress;
        bytes32 refundAddress;
        uint256 maximumRefundTarget;
        uint256 receiverValueTarget;
        ExecutionParameters executionParameters;
    }

    struct ExecutionParameters {
        uint8 version;
        uint32 gasLimit;
        bytes32 providerDeliveryAddress;
    }

    struct RedeliveryByTxHashInstruction {
        uint8 payloadId; //2
        uint16 sourceChain;
        bytes32 sourceTxHash;
        uint64 deliveryVAASequence;
        uint16 targetChain;
        uint8 multisendIndex;
        uint256 newMaximumRefundTarget;
        uint256 newReceiverValueTarget;
        ExecutionParameters executionParameters;
    }

    struct ForwardInstruction {
        bytes container;
        address sender;
        uint256 msgValue;
        uint256 totalFee;
        address relayProvider;
        bool isValid;
    }

    struct DeliveryVAAInfo {
        uint16 sourceChain;
        uint64 sourceSequence;
        bytes32 deliveryVaaHash;
    }
}
