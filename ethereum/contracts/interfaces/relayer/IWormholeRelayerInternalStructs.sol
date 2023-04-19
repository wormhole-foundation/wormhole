// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./IWormholeRelayer.sol";

interface IWormholeRelayerInternalStructs {

    struct DeliveryInstruction {
        uint16 targetChain;
        bytes32 targetAddress;
        bytes32 refundAddress;
        uint16 refundChain;
        uint256 maximumRefundTarget;
        uint256 receiverValueTarget;
        bytes32 sourceRelayProvider;
        bytes32 targetRelayProvider;
        bytes32 senderAddress;
        IWormholeRelayer.VaaKey[] vaaKeys;
        uint8 consistencyLevel;
        ExecutionParameters executionParameters;
        bytes payload;
    }

    struct ExecutionParameters {
        uint8 version;
        uint32 gasLimit;
    }

    struct ForwardInstruction {
        bytes encodedInstruction;
        uint256 msgValue;
        uint256 totalFee;
    }

    struct FirstForwardInfo {
        uint256 maxTransactionFee;
        uint256 receiverValue;
        address relayProviderAddress;
        bytes relayParameters;
    }

    struct DeliveryVAAInfo {
        uint16 sourceChain;
        uint64 sourceSequence;
        bytes32 deliveryVaaHash;
        bytes[] encodedVMs;
        address payable relayerRefundAddress;
        DeliveryInstruction internalInstruction;
    }
}
