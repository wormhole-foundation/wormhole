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

    struct ExecutionParameters {
        uint8 version;
        uint32 gasLimit;
    }

    function decodeDeliveryInstruction(bytes memory encoded)
        external
        pure
        returns (DeliveryInstruction memory);

    function toWormholeFormat(address addr) external pure returns (bytes32 whFormat);

    function fromWormholeFormat(bytes32 whFormatAddress) external pure returns (address addr);
}
