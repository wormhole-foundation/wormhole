// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../interfaces/IWormholeRelayer.sol";

interface IWormholeRelayerInternalStructs {
    struct DeliveryInstructionsContainer {
        uint8 payloadId; //1
        bytes32 senderAddress;
        IWormholeRelayer.MessageInfo[] messageInfos;
        DeliveryInstruction[] instructions;
    }

    struct DeliveryInstruction {
        uint16 targetChain;
        bytes32 targetAddress;
        bytes32 refundAddress;
        uint16 refundChain;
        uint256 maximumRefundTarget;
        uint256 receiverValueTarget;
        bytes32 targetRelayProvider;
        ExecutionParameters executionParameters;
        bytes payload;
    }

    struct ExecutionParameters {
        uint8 version;
        uint32 gasLimit;
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
        bytes[] encodedVMs;
        address payable relayerRefundAddress;
        DeliveryInstructionsContainer deliveryContainer;
        DeliveryInstruction internalInstruction;
    }
}
