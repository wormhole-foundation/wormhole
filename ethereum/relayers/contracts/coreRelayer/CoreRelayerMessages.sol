// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";

import "./CoreRelayerGetters.sol";
import "./CoreRelayerStructs.sol";

contract CoreRelayerMessages is CoreRelayerStructs, CoreRelayerGetters {
    using BytesLib for bytes;

    error InvalidPayloadId(uint8 payloadId);
    error InvalidDeliveryInstructionsPayload(uint256 length);
    error InvalidSendsPayload(uint256 length);

    function decodeRedeliveryByTxHashInstruction(bytes memory encoded)
        internal
        pure
        returns (RedeliveryByTxHashInstruction memory instruction)
    {
        uint256 index = 0;

        instruction.payloadId = encoded.toUint8(index);
        index += 1;

        instruction.sourceChain = encoded.toUint16(index);
        index += 2;

        instruction.sourceTxHash = encoded.toBytes32(index);
        index += 32;

        instruction.sourceNonce = encoded.toUint32(index);
        index += 4;

        instruction.targetChain = encoded.toUint16(index);
        index += 2;

        instruction.deliveryIndex = encoded.toUint8(index);
        index += 1;

        instruction.multisendIndex = encoded.toUint8(index);
        index += 1;

        instruction.newMaximumRefundTarget = encoded.toUint256(index);
        index += 32;

        instruction.newReceiverValueTarget = encoded.toUint256(index);
        index += 32;

        instruction.executionParameters.version = encoded.toUint8(index);
        index += 1;

        instruction.executionParameters.gasLimit = encoded.toUint32(index);
        index += 4;

        instruction.executionParameters.providerDeliveryAddress = encoded.toBytes32(index);
        index += 32;
    }

    function decodeDeliveryInstructionsContainer(bytes memory encoded)
        internal
        pure
        returns (DeliveryInstructionsContainer memory)
    {
        uint256 index = 0;

        uint8 payloadId = encoded.toUint8(index);
        if (payloadId != 1) {
            revert InvalidPayloadId(payloadId);
        }
        index += 1;
        bool sufficientlyFunded = encoded.toUint8(index) == 1;
        index += 1;
        uint8 arrayLen = encoded.toUint8(index);
        index += 1;

        DeliveryInstruction[] memory instructionArray = new DeliveryInstruction[](arrayLen);

        for (uint8 i = 0; i < arrayLen; i++) {
            DeliveryInstruction memory instruction;

            // target chain of the delivery instruction
            instruction.targetChain = encoded.toUint16(index);
            index += 2;

            // target contract address
            instruction.targetAddress = encoded.toBytes32(index);
            index += 32;

            // address to send the refund to
            instruction.refundAddress = encoded.toBytes32(index);
            index += 32;

            instruction.maximumRefundTarget = encoded.toUint256(index);
            index += 32;

            instruction.receiverValueTarget = encoded.toUint256(index);
            index += 32;

            instruction.executionParameters.version = encoded.toUint8(index);
            index += 1;

            instruction.executionParameters.gasLimit = encoded.toUint32(index);
            index += 4;

            instruction.executionParameters.providerDeliveryAddress = encoded.toBytes32(index);
            index += 32;

            instructionArray[i] = instruction;
        }

        if (index != encoded.length) {
            revert InvalidDeliveryInstructionsPayload(encoded.length);
        }

        return DeliveryInstructionsContainer({
            payloadId: payloadId,
            sufficientlyFunded: sufficientlyFunded,
            instructions: instructionArray
        });
    }

    function encodeMultichainSend(MultichainSend memory container) internal pure returns (bytes memory encoded) {
        encoded = abi.encodePacked(
            uint8(1), //version payload number
            address(container.relayProviderAddress),
            uint8(container.requests.length) //number of requests in the array
        );

        //Append all the messages to the array.
        for (uint256 i = 0; i < container.requests.length; i++) {
            Send memory request = container.requests[i];

            encoded = abi.encodePacked(
                encoded,
                request.targetChain,
                request.targetAddress,
                request.refundAddress,
                request.maxTransactionFee,
                request.receiverValue,
                uint8(request.relayParameters.length),
                request.relayParameters
            );
        }
    }

    function decodeMultichainSend(bytes memory encoded) internal pure returns (MultichainSend memory) {
        uint256 index = 0;

        uint8 payloadId = encoded.toUint8(index);
        if (payloadId != 1) {
            revert InvalidPayloadId(payloadId);
        }
        index += 1;
        address relayProviderAddress = encoded.toAddress(index);
        index += 20;
        uint8 arrayLen = encoded.toUint8(index);
        index += 1;

        Send[] memory requestArray = new Send[](arrayLen);

        for (uint8 i = 0; i < arrayLen; i++) {
            Send memory request;

            // target chain of the delivery request
            request.targetChain = encoded.toUint16(index);
            index += 2;

            // target contract address
            request.targetAddress = encoded.toBytes32(index);
            index += 32;

            // address to send the refund to
            request.refundAddress = encoded.toBytes32(index);
            index += 32;

            request.maxTransactionFee = encoded.toUint256(index);
            index += 32;

            request.receiverValue = encoded.toUint256(index);
            index += 32;

            uint8 relayParametersLength = encoded.toUint8(index);

            index += 1;

            request.relayParameters = encoded.slice(index, relayParametersLength);

            index += relayParametersLength;

            requestArray[i] = request;
        }

        if (index != encoded.length) {
            revert InvalidSendsPayload(encoded.length);
        }

        return MultichainSend({relayProviderAddress: relayProviderAddress, requests: requestArray});
    }
}
