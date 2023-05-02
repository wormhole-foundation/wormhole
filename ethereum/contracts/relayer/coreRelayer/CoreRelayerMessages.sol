// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../libraries/external/BytesLib.sol";

import "./CoreRelayerGetters.sol";
import "../../interfaces/relayer/IWormholeRelayerInternalStructs.sol";
import "../../interfaces/relayer/IWormholeRelayer.sol";
import "../../interfaces/relayer/IDelivery.sol";

abstract contract CoreRelayerMessages is CoreRelayerGetters {
    using BytesLib for bytes;

    error InvalidPayloadId(uint8 payloadId);
    error InvalidDeliveryInstructionsPayload(uint256 length);

    /**
     * @notice This function converts a Send struct into a DeliveryInstruction struct that
     * describes to the relayer exactly how to relay for the Send.
     * Specifically, the DeliveryInstruction struct that contains six fields:
     * 1) targetChain, 2) targetAddress, 3) refundAddress (all which are part of the Send struct),
     * 4) maximumRefundTarget: The maximum amount that can be refunded to 'refundAddress' (e.g. if the call to 'receiveWormholeMessages' takes 0 gas),
     * 5) receiverValueTarget: The amount that will be passed into 'receiveWormholeMessages' as value, in target chain currency
     * 6) executionParameters: a struct with information about execution, specifically:
     *    executionParameters.gasLimit: The maximum amount of gas 'receiveWormholeMessages' is allowed to use
     *    executionParameters.providerDeliveryAddress: The address of the relayer that will execute this Send request
     * The latter 3 fields are calculated using the relayProvider's getters
     * @param send A Send struct
     * @return instruction A DeliveryInstruction
     */
    function convertSendToDeliveryInstruction(IWormholeRelayer.Send memory send)
        internal
        view
        returns (IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction)
    {
        instruction.targetChain = send.targetChain;
        instruction.targetAddress = send.targetAddress;
        instruction.refundChain = send.refundChain;
        instruction.refundAddress = send.refundAddress;

        IRelayProvider relayProvider = IRelayProvider(send.relayProviderAddress);

        instruction.maximumRefundTarget =
            calculateTargetDeliveryMaximumRefund(send.targetChain, send.maxTransactionFee, relayProvider);

        instruction.receiverValueTarget =
            convertReceiverValueAmountToTarget(send.receiverValue, send.targetChain, relayProvider);

        instruction.senderAddress = toWormholeFormat(msg.sender);
        instruction.sourceRelayProvider = toWormholeFormat(address(relayProvider));
        instruction.targetRelayProvider = relayProvider.getTargetChainAddress(send.targetChain);

        instruction.vaaKeys = send.vaaKeys;
        instruction.consistencyLevel = send.consistencyLevel;
        instruction.payload = send.payload;

        instruction.executionParameters = IWormholeRelayerInternalStructs.ExecutionParameters({
            version: 1,
            gasLimit: calculateTargetGasDeliveryAmount(send.targetChain, send.maxTransactionFee, relayProvider)
        });
    }

    /**
     * @notice Check if for each instruction in the DeliveryInstructionContainer,
     * - the total amount of target chain currency needed is within the maximum budget,
     *   i.e. (maximumRefundTarget + receiverValueTarget) <= (the relayProvider's maximum budget for the target chain)
     * - the gasLimit is greater than 0
     * @param instruction A DeliveryInstruction
     * @param relayProvider The relayProvider whos maximum budget we are checking against
     */
    function checkInstruction(
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction,
        IRelayProvider relayProvider
    ) internal view {
        if (instruction.executionParameters.gasLimit == 0) {
            revert IWormholeRelayer.MaxTransactionFeeNotEnough();
        }
        if (
            instruction.maximumRefundTarget + instruction.receiverValueTarget
                > relayProvider.quoteMaximumBudget(instruction.targetChain)
        ) {
            revert IWormholeRelayer.MsgValueTooMuch();
        }
    }

    // encode a 'VaaKey' into bytes
    function encodeVaaKey(IWormholeRelayer.VaaKey memory vaaKey) internal pure returns (bytes memory encoded) {
        encoded = abi.encodePacked(uint8(1), uint8(vaaKey.infoType));
        if (vaaKey.infoType == IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE) {
            encoded = abi.encodePacked(encoded, vaaKey.chainId, vaaKey.emitterAddress, vaaKey.sequence);
        } else if (vaaKey.infoType == IWormholeRelayer.VaaKeyType.VAAHASH) {
            encoded = abi.encodePacked(encoded, vaaKey.vaaHash);
        }
    }

    // encode a 'DeliveryInstruction' into bytes
    function encodeDeliveryInstruction(IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction)
        public
        pure
        returns (bytes memory encoded)
    {
        uint8 length = uint8(instruction.vaaKeys.length);
        bytes memory encodedVaaKeys = abi.encodePacked(length);
        for (uint8 i = 0; i < length; i++) {
            encodedVaaKeys = abi.encodePacked(encodedVaaKeys, encodeVaaKey(instruction.vaaKeys[i]));
        }
        encoded = abi.encodePacked(
            uint8(1),
            instruction.targetChain,
            instruction.targetAddress,
            instruction.refundChain,
            instruction.refundAddress,
            instruction.maximumRefundTarget,
            instruction.receiverValueTarget
        );
        encoded = abi.encodePacked(
            encoded,
            instruction.sourceRelayProvider,
            instruction.targetRelayProvider,
            instruction.senderAddress,
            encodedVaaKeys,
            instruction.consistencyLevel
        );
        encoded = abi.encodePacked(
            encoded,
            instruction.executionParameters.version,
            instruction.executionParameters.gasLimit,
            uint32(instruction.payload.length),
            instruction.payload
        );
    }

    // encode a 'Send' into bytes
    function encodeSend(IWormholeRelayer.Send memory sendParams) public pure returns (bytes memory encoded) {
        uint8 length = uint8(sendParams.vaaKeys.length);
        bytes memory encodedVaaKeys = abi.encodePacked(length);
        for (uint8 i = 0; i < length; i++) {
            encodedVaaKeys = abi.encodePacked(encodedVaaKeys, encodeVaaKey(sendParams.vaaKeys[i]));
        }
        encoded = abi.encodePacked(
            sendParams.targetChain,
            sendParams.targetAddress,
            sendParams.refundChain,
            sendParams.refundAddress,
            sendParams.maxTransactionFee,
            sendParams.receiverValue
        );
        encoded =
            abi.encodePacked(encoded, sendParams.relayProviderAddress, encodedVaaKeys, sendParams.consistencyLevel);
        encoded = abi.encodePacked(
            encoded,
            uint32(sendParams.payload.length),
            sendParams.payload,
            uint32(sendParams.relayParameters.length),
            sendParams.relayParameters
        );
    }

    // decode a 'Send' from bytes
    function decodeSend(bytes memory encoded) public pure returns (IWormholeRelayer.Send memory sendParams) {
        uint256 index = 0;

        // target chain
        sendParams.targetChain = encoded.toUint16(index);
        index += 2;

        // target contract address
        sendParams.targetAddress = encoded.toBytes32(index);
        index += 32;

        sendParams.refundChain = encoded.toUint16(index);
        index += 2;
        // address to send the refund to
        sendParams.refundAddress = encoded.toBytes32(index);
        index += 32;

        sendParams.maxTransactionFee = encoded.toUint256(index);
        index += 32;

        sendParams.receiverValue = encoded.toUint256(index);
        index += 32;

        sendParams.relayProviderAddress = encoded.toAddress(index);
        index += 20;

        uint8 length = encoded.toUint8(index);
        index += 1;

        sendParams.vaaKeys = new IWormholeRelayer.VaaKey[](length);
        for (uint8 i = 0; i < length; i++) {
            (sendParams.vaaKeys[i], index) = decodeVaaKey(encoded, index);
        }

        sendParams.consistencyLevel = encoded.toUint8(index);
        index += 1;

        uint32 payloadLength = encoded.toUint32(index);
        index += 4;

        sendParams.payload = encoded.slice(index, payloadLength);
        index += payloadLength;

        uint32 relayParametersLength = encoded.toUint32(index);
        index += 4;

        sendParams.payload = encoded.slice(index, relayParametersLength);
        index += relayParametersLength;
    }

    /**
     * Given a targetChain, maxTransactionFee, and a relay provider, this function calculates what the gas limit of the delivery transaction
     * should be
     *
     * It does this by calculating (maxTransactionFee - deliveryOverhead)/gasPrice
     * where 'deliveryOverhead' is the relayProvider's base fee for delivering to targetChain (in units of source chain currency)
     * and 'gasPrice' is the relayProvider's fee per unit of target chain gas (in units of source chain currency)
     *
     * @param targetChain target chain
     * @param maxTransactionFee uint256 in source chain currency
     * @param provider IRelayProvider
     * @return gasAmount
     */
    function calculateTargetGasDeliveryAmount(uint16 targetChain, uint256 maxTransactionFee, IRelayProvider provider)
        internal
        view
        returns (uint32 gasAmount)
    {
        uint256 overhead = provider.quoteDeliveryOverhead(targetChain);
        if (maxTransactionFee <= overhead) {
            gasAmount = 0;
        } else {
            uint256 gas = (maxTransactionFee - overhead) / provider.quoteGasPrice(targetChain);
            if (gas > type(uint32).max) {
                gasAmount = type(uint32).max;
            } else {
                gasAmount = uint32(gas);
            }
        }
    }

    /**
     * Given a targetChain, maxTransactionFee, and a relay provider, this function calculates what the maximum refund of the delivery transaction
     * should be, in terms of target chain currency
     *
     * The maximum refund is the amount that would be refunded to refundAddress if the call to 'receiveWormholeMessages' takes 0 gas
     *
     * It does this by calculating (maxTransactionFee - deliveryOverhead) and converting (using the relay provider's prices) to target chain currency
     * (where 'deliveryOverhead' is the relayProvider's base fee for delivering to targetChain [in units of source chain currency])
     *
     * @param targetChain target chain
     * @param maxTransactionFee uint256
     * @param provider IRelayProvider
     * @return maximumRefund uint256
     */
    function calculateTargetDeliveryMaximumRefund(
        uint16 targetChain,
        uint256 maxTransactionFee,
        IRelayProvider provider
    ) internal view returns (uint256 maximumRefund) {
        uint256 overhead = provider.quoteDeliveryOverhead(targetChain);
        if (maxTransactionFee > overhead) {
            (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChain);
            uint256 remainder = maxTransactionFee - overhead;
            maximumRefund = assetConversionHelper(
                chainId(), remainder, targetChain, denominator, uint256(0) + denominator + buffer, false, provider
            );
        } else {
            maximumRefund = 0;
        }
    }

    /**
     * Converts 'sourceAmount' of source chain currency to units of target chain currency
     * using the prices of 'provider'
     * and also multiplying by a specified fraction 'multiplier/multiplierDenominator',
     * rounding up or down specified by 'roundUp', and without performing intermediate rounding,
     * i.e. the result should be as if float arithmetic was done and the rounding performed at the end
     *
     * @param sourceChain source chain
     * @param sourceAmount amount of source chain currency to be converted
     * @param targetChain target chain
     * @param multiplier numerator of a fraction to multiply by
     * @param multiplierDenominator denominator of a fraction to multiply by
     * @param roundUp whether or not to round up
     * @param provider relay provider
     * @return targetAmount amount of target chain currency
     */
    function assetConversionHelper(
        uint16 sourceChain,
        uint256 sourceAmount,
        uint16 targetChain,
        uint256 multiplier,
        uint256 multiplierDenominator,
        bool roundUp,
        IRelayProvider provider
    ) internal view returns (uint256 targetAmount) {
        uint256 srcNativeCurrencyPrice = provider.quoteAssetPrice(sourceChain);
        if (srcNativeCurrencyPrice == 0) {
            revert IWormholeRelayer.RelayProviderDoesNotSupportTargetChain();
        }

        uint256 dstNativeCurrencyPrice = provider.quoteAssetPrice(targetChain);
        if (dstNativeCurrencyPrice == 0) {
            revert IWormholeRelayer.RelayProviderDoesNotSupportTargetChain();
        }
        uint256 numerator = sourceAmount * srcNativeCurrencyPrice * multiplier;
        uint256 denominator = dstNativeCurrencyPrice * multiplierDenominator;
        if (roundUp) {
            targetAmount = (numerator + denominator - 1) / denominator;
        } else {
            targetAmount = numerator / denominator;
        }
    }

    /**
     * If the user specifies (for 'receiverValue) 'sourceAmount' of source chain currency, with relay provider 'provider',
     * then this function calculates how much the relayer will pass into receiveWormholeMessages on the target chain (in target chain currency)
     *
     * The calculation simply converts this amount to target chain currency, but also applies a multiplier of 'denominator/(denominator + buffer)'
     * where these values are also specified by the relay provider 'provider'
     *
     * @param sourceAmount amount of source chain currency
     * @param targetChain target chain
     * @param provider relay provider
     * @return targetAmount amount of target chain currency
     */
    function convertReceiverValueAmountToTarget(uint256 sourceAmount, uint16 targetChain, IRelayProvider provider)
        internal
        view
        returns (uint256 targetAmount)
    {
        (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChain);

        // todo: akshaj why is `uint256(0) + ...` present?
        targetAmount = assetConversionHelper(
            chainId(), sourceAmount, targetChain, denominator, uint256(0) + denominator + buffer, false, provider
        );
    }

    // decode a 'DeliveryInstruction' from bytes
    function decodeDeliveryInstruction(bytes memory encoded)
        public
        pure
        returns (IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction)
    {
        uint256 index = 0;

        uint8 payloadId = encoded.toUint8(index);
        index += 1;
        if (payloadId != 1) {
            revert InvalidPayloadId(payloadId);
        }
        // target chain of the delivery instruction
        instruction.targetChain = encoded.toUint16(index);
        index += 2;

        // target contract address
        instruction.targetAddress = encoded.toBytes32(index);
        index += 32;

        instruction.refundChain = encoded.toUint16(index);
        index += 2;
        // address to send the refund to
        instruction.refundAddress = encoded.toBytes32(index);
        index += 32;

        instruction.maximumRefundTarget = encoded.toUint256(index);
        index += 32;

        instruction.receiverValueTarget = encoded.toUint256(index);
        index += 32;

        instruction.sourceRelayProvider = encoded.toBytes32(index);
        index += 32;

        instruction.targetRelayProvider = encoded.toBytes32(index);
        index += 32;

        instruction.senderAddress = encoded.toBytes32(index);
        index += 32;

        uint8 length = encoded.toUint8(index);
        index += 1;

        instruction.vaaKeys = new IWormholeRelayer.VaaKey[](length);
        for (uint8 i = 0; i < length; i++) {
            (instruction.vaaKeys[i], index) = decodeVaaKey(encoded, index);
        }

        instruction.consistencyLevel = encoded.toUint8(index);
        index += 1;

        instruction.executionParameters.version = encoded.toUint8(index);
        index += 1;

        instruction.executionParameters.gasLimit = encoded.toUint32(index);
        index += 4;

        uint32 payloadLength = encoded.toUint32(index);
        index += 4;

        instruction.payload = encoded.slice(index, payloadLength);
        index += payloadLength;
    }

    // decode a 'VaaKey' from bytes
    function decodeVaaKey(bytes memory encoded, uint256 index)
        public
        pure
        returns (IWormholeRelayer.VaaKey memory vaaKey, uint256 newIndex)
    {
        uint8 payloadId = encoded.toUint8(index);
        index += 1;

        if (payloadId != 1) {
            revert InvalidPayloadId(payloadId);
        }

        vaaKey.infoType = IWormholeRelayer.VaaKeyType(encoded.toUint8(index));
        index += 1;

        if (vaaKey.infoType == IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE) {
            vaaKey.chainId = encoded.toUint16(index);
            index += 2;

            vaaKey.emitterAddress = encoded.toBytes32(index);
            index += 32;

            vaaKey.sequence = encoded.toUint64(index);
            index += 8;
        } else if (vaaKey.infoType == IWormholeRelayer.VaaKeyType.VAAHASH) {
            vaaKey.vaaHash = encoded.toBytes32(index);
            index += 32;
        }
        newIndex = index;
    }

    function encodeDeliveryOverride(IDelivery.DeliveryOverride memory request)
        public
        pure
        returns (bytes memory encoded)
    {
        encoded = abi.encodePacked(
            uint8(1), request.gasLimit, request.maximumRefund, request.receiverValue, request.redeliveryHash
        );
    }

    function decodeDeliveryOverride(bytes memory encoded)
        public
        pure
        returns (IDelivery.DeliveryOverride memory output)
    {
        uint256 index = 0;

        uint8 payloadId = encoded.toUint8(index);
        if (payloadId != 1) {
            revert InvalidPayloadId(payloadId);
        }

        index += 1;

        output.gasLimit = encoded.toUint32(index);
        index += 4;

        output.maximumRefund = encoded.toUint256(index);
        index += 32;

        output.receiverValue = encoded.toUint256(index);
        index += 32;

        output.redeliveryHash = encoded.toBytes32(index);
    }

    function encodeRedeliveryInstruction(IWormholeRelayerInternalStructs.RedeliveryInstruction memory ins)
        public
        pure
        returns (bytes memory encoded)
    {
        bytes memory vaaKey = encodeVaaKey(ins.key);
        encoded = abi.encodePacked(
            uint8(2),
            vaaKey,
            ins.newMaximumRefundTarget,
            ins.newReceiverValueTarget,
            ins.sourceRelayProvider,
            ins.targetChain,
            ins.executionParameters.version,
            ins.executionParameters.gasLimit
        );
    }

    function decodeRedeliveryInstruction(bytes memory encoded)
        public
        pure
        returns (IWormholeRelayerInternalStructs.RedeliveryInstruction memory output)
    {
        uint256 index = 0;

        uint8 payloadId = encoded.toUint8(index);
        if (payloadId != 2) {
            revert InvalidPayloadId(payloadId);
        }
        index += 1;

        (output.key, index) = decodeVaaKey(encoded, index);

        output.newMaximumRefundTarget = encoded.toUint256(index);
        index += 32;

        output.newReceiverValueTarget = encoded.toUint256(index);
        index += 32;

        output.sourceRelayProvider = encoded.toBytes32(index);
        index += 32;

        output.targetChain = encoded.toUint16(index);
        index += 2;

        output.executionParameters.version = 1;
        index += 1;

        output.executionParameters.gasLimit = encoded.toUint32(index);
        index += 4;
    }

    /**
     * @notice Helper function that converts an EVM address to wormhole format
     * @param addr (EVM 20-byte address)
     * @return whFormat (32-byte address in Wormhole format)
     */
    function toWormholeFormat(address addr) public pure returns (bytes32 whFormat) {
        return bytes32(uint256(uint160(addr)));
    }

    /**
     * @notice Helper function that converts an Wormhole format (32-byte) address to the EVM 'address' 20-byte format
     * @param whFormatAddress (32-byte address in Wormhole format)
     * @return addr (EVM 20-byte address)
     */
    function fromWormholeFormat(bytes32 whFormatAddress) public pure returns (address addr) {
        return address(uint160(uint256(whFormatAddress)));
    }
}
