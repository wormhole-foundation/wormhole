// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./IWormholeRelayer.sol";

interface IWormholeRelayerInstructionParser {
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

    struct RedeliveryInstruction {
        IWormholeRelayer.VaaKey key;
        uint256 newMaximumRefundTarget;
        uint256 newReceiverValueTarget;
        bytes32 sourceRelayProvider;
        ExecutionParameters executionParameters;
    }

    struct ExecutionParameters {
        uint8 version;
        uint32 gasLimit;
    }

    struct DeliveryOverride {
        uint32 gasLimit;
        uint256 maximumRefund;
        uint256 receiverValue;
        bytes32 redeliveryHash;
    }

    function decodeDeliveryInstruction(bytes memory encoded)
        external
        pure
        returns (DeliveryInstruction memory);

    function decodeRedeliveryInstruction(bytes memory encoded) 
        external
        pure
        returns (RedeliveryInstruction memory);

    function encodeDeliveryOverride(DeliveryOverride memory request) external pure returns (bytes memory encoded);

    function decodeDeliveryOverride(bytes memory encoded) external pure returns (DeliveryOverride memory output);

    function toWormholeFormat(address addr) external pure returns (bytes32 whFormat);

    function fromWormholeFormat(bytes32 whFormatAddress) external pure returns (address addr);
}
