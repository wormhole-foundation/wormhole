// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";

import "./CoreRelayerGetters.sol";
import "../interfaces/IWormholeRelayerInternalStructs.sol";
import "../interfaces/IWormholeRelayer.sol";

contract CoreRelayerMessages is CoreRelayerGetters {
    using BytesLib for bytes;

    error InvalidPayloadId(uint8 payloadId);
    error InvalidDeliveryInstructionsPayload(uint256 length);

    /**
     * @notice This function calculates the total fee to execute all of the Send requests in this MultichainSend container
     * @param sendContainer A MultichainSend struct describing all of the Send requests
     * @return totalFee
     */
    function getTotalFeeMultichainSend(IWormholeRelayer.MultichainSend memory sendContainer, uint256 wormholeMessageFee)
        internal
        pure
        returns (uint256 totalFee)
    {
        totalFee = wormholeMessageFee;
        uint256 length = sendContainer.requests.length;
        for (uint256 i = 0; i < length; i++) {
            IWormholeRelayer.Send memory request = sendContainer.requests[i];
            totalFee += request.maxTransactionFee + request.receiverValue;
        }
    }

    /**
     * @notice This function converts a MultichainSend struct into a DeliveryInstructionsContainer struct that
     * describes to the relayer exactly how to relay for each of the Send requests.
     * Specifically, each Send is converted to a DeliveryInstruction, which is a struct that contains six fields:
     * 1) targetChain, 2) targetAddress, 3) refundAddress (all which are part of the Send struct),
     * 4) maximumRefundTarget: The maximum amount that can be refunded to 'refundAddress' (e.g. if the call to 'receiveWormholeMessages' takes 0 gas),
     * 5) receiverValueTarget: The amount that will be passed into 'receiveWormholeMessages' as value, in target chain currency
     * 6) executionParameters: a struct with information about execution, specifically:
     *    executionParameters.gasLimit: The maximum amount of gas 'receiveWormholeMessages' is allowed to use
     *    executionParameters.providerDeliveryAddress: The address of the relayer that will execute this Send request
     * The latter 3 fields are calculated using the relayProvider's getters
     * @param sendContainer A MultichainSend struct describing all of the Send requests
     * @return instructionsContainer A DeliveryInstructionsContainer struct
     */
    function convertMultichainSendToDeliveryInstructionsContainer(IWormholeRelayer.MultichainSend memory sendContainer)
        internal
        view
        returns (IWormholeRelayerInternalStructs.DeliveryInstructionsContainer memory instructionsContainer)
    {
        instructionsContainer.payloadId = 1;
        instructionsContainer.senderAddress = toWormholeFormat(msg.sender);
        IRelayProvider relayProvider = IRelayProvider(sendContainer.relayProviderAddress);
        instructionsContainer.messageInfos = sendContainer.messageInfos;

        uint256 length = sendContainer.requests.length;
        instructionsContainer.instructions = new IWormholeRelayerInternalStructs.DeliveryInstruction[](length);
        for (uint256 i = 0; i < length; i++) {
            instructionsContainer.instructions[i] =
                convertSendToDeliveryInstruction(sendContainer.requests[i], relayProvider);
        }
    }

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
     * @param relayProvider The relay provider chosen for this Send
     * @return instruction A DeliveryInstruction
     */
    function convertSendToDeliveryInstruction(IWormholeRelayer.Send memory send, IRelayProvider relayProvider)
        internal
        view
        returns (IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction)
    {
        instruction.targetChain = send.targetChain;
        instruction.targetAddress = send.targetAddress;
        instruction.refundChain = send.refundChain;
        instruction.refundAddress = send.refundAddress;

        instruction.maximumRefundTarget =
            calculateTargetDeliveryMaximumRefund(send.targetChain, send.maxTransactionFee, relayProvider);

        instruction.receiverValueTarget =
            convertReceiverValueAmount(send.receiverValue, send.targetChain, relayProvider);

        instruction.targetRelayProvider = relayProvider.getTargetChainAddress(send.targetChain);

        instruction.payload = send.payload;

        instruction.executionParameters = IWormholeRelayerInternalStructs.ExecutionParameters({
            version: 1,
            gasLimit: calculateTargetGasDeliveryAmount(send.targetChain, send.maxTransactionFee, relayProvider)
        });
    }

    /**
     * @notice Check if for each instruction in the DeliveryInstructionContainer,
     * - the total amount of target chain currency needed for execution of the instruction is within the maximum budget,
     *   i.e. (maximumRefundTarget + receiverValueTarget) <= (the relayProvider's maximum budget for the target chain)
     * - the gasLimit is greater than 0
     * @param container A DeliveryInstructionsContainer
     * @param relayProvider The relayProvider whos maximum budget we are checking against
     */
    function checkInstructions(
        IWormholeRelayerInternalStructs.DeliveryInstructionsContainer memory container,
        IRelayProvider relayProvider
    ) internal view {
        uint256 length = container.instructions.length;
        for (uint8 i = 0; i < length; i++) {
            IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction = container.instructions[i];
            if (instruction.executionParameters.gasLimit == 0) {
                revert IWormholeRelayer.MaxTransactionFeeNotEnough(i);
            }
            if (
                instruction.maximumRefundTarget + instruction.receiverValueTarget
                    > relayProvider.quoteMaximumBudget(instruction.targetChain)
            ) {
                revert IWormholeRelayer.FundsTooMuch(i);
            }
        }
    }

    // encode a 'IWormholeRelayerInternalStructs.DeliveryInstructionsContainer' into bytes
    function encodeDeliveryInstructionsContainer(
        IWormholeRelayerInternalStructs.DeliveryInstructionsContainer memory container
    ) public pure returns (bytes memory encoded) {
        encoded = abi.encodePacked(
            container.payloadId,
            container.senderAddress,
            uint8(container.messageInfos.length),
            uint8(container.instructions.length)
        );

        for (uint256 i = 0; i < container.messageInfos.length; i++) {
            encoded = abi.encodePacked(encoded, encodeMessageInfo(container.messageInfos[i]));
        }

        for (uint256 i = 0; i < container.instructions.length; i++) {
            encoded = abi.encodePacked(encoded, encodeDeliveryInstruction(container.instructions[i]));
        }
    }

    // encode a 'MessageInfo' into bytes
    function encodeMessageInfo(IWormholeRelayer.MessageInfo memory messageInfo)
        internal
        pure
        returns (bytes memory encoded)
    {
        encoded = abi.encodePacked(uint8(1), uint8(messageInfo.infoType));
        if (messageInfo.infoType == IWormholeRelayer.MessageInfoType.EMITTER_SEQUENCE) {
            encoded = abi.encodePacked(encoded, messageInfo.emitterAddress, messageInfo.sequence);
        } else if (messageInfo.infoType == IWormholeRelayer.MessageInfoType.VAAHASH) {
            encoded = abi.encodePacked(encoded, messageInfo.vaaHash);
        }
    }

    // encode a 'DeliveryInstruction' into bytes
    function encodeDeliveryInstruction(IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction)
        public
        pure
        returns (bytes memory encoded)
    {
        encoded = abi.encodePacked(
            instruction.targetChain,
            instruction.targetAddress,
            instruction.refundChain,
            instruction.refundAddress,
            instruction.maximumRefundTarget,
            instruction.receiverValueTarget,
            instruction.targetRelayProvider,
            instruction.executionParameters.version,
            instruction.executionParameters.gasLimit,
            uint32(instruction.payload.length),
            instruction.payload
        );
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
     * @param maxTransactionFee uint256
     * @param provider IRelayProvider
     * @return gasAmount
     */
    function calculateTargetGasDeliveryAmount(uint16 targetChain, uint256 maxTransactionFee, IRelayProvider provider)
        internal
        view
        returns (uint32 gasAmount)
    {
        gasAmount = calculateTargetGasDeliveryAmountHelper(
            targetChain, maxTransactionFee, provider.quoteDeliveryOverhead(targetChain), provider
        );
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
        maximumRefund = calculateTargetDeliveryMaximumRefundHelper(
            targetChain, maxTransactionFee, provider.quoteDeliveryOverhead(targetChain), provider
        );
    }

    /**
     * Performs the calculation (maxTransactionFee - overhead)/(price of 1 unit of target chain gas, in source chain currency)
     * and bounds the result between 0 and 2^32-1, inclusive
     *
     * @param targetChain uint16
     * @param maxTransactionFee uint256
     * @param overhead uint256
     * @param provider IRelayProvider
     */
    function calculateTargetGasDeliveryAmountHelper(
        uint16 targetChain,
        uint256 maxTransactionFee,
        uint256 overhead,
        IRelayProvider provider
    ) internal view returns (uint32 gasAmount) {
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
     * Converts (maxTransactionFee - overhead) from source to target chain currency, using the provider's prices.
     *
     * It also applies the assetConversionBuffer, similar to the receiverValue calculation.
     * @param targetChain uint16
     * @param maxTransactionFee uint256
     * @param overhead uint256
     * @param provider IRelayProvider
     */
    function calculateTargetDeliveryMaximumRefundHelper(
        uint16 targetChain,
        uint256 maxTransactionFee,
        uint256 overhead,
        IRelayProvider provider
    ) internal view returns (uint256 maximumRefund) {
        if (maxTransactionFee >= overhead) {
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
        if (!provider.isChainSupported(targetChain)) {
            revert IWormholeRelayer.RelayProviderDoesNotSupportTargetChain();
        }
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
    function convertReceiverValueAmount(uint256 sourceAmount, uint16 targetChain, IRelayProvider provider)
        internal
        view
        returns (uint256 targetAmount)
    {
        (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChain);

        targetAmount = assetConversionHelper(
            chainId(), sourceAmount, targetChain, denominator, uint256(0) + denominator + buffer, false, provider
        );
    }

    // decode a 'DeliveryInstruction' from bytes
    function decodeDeliveryInstruction(bytes memory encoded, uint256 index)
        public
        pure
        returns (IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction, uint256 newIndex)
    {
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

        instruction.targetRelayProvider = encoded.toBytes32(index);
        index += 32;

        instruction.executionParameters.version = encoded.toUint8(index);
        index += 1;

        instruction.executionParameters.gasLimit = encoded.toUint32(index);
        index += 4;

        uint32 payloadLength = encoded.toUint32(index);
        index += 4;

        instruction.payload = encoded.slice(index, payloadLength);
        index += payloadLength;

        newIndex = index;
    }

    // decode a 'MessageInfo' from bytes
    function decodeMessageInfo(bytes memory encoded, uint256 index)
        public
        pure
        returns (IWormholeRelayer.MessageInfo memory messageInfo, uint256 newIndex)
    {
        uint8 payloadId = encoded.toUint8(index);
        index += 1;

        if (payloadId != 1) {
            revert InvalidPayloadId(payloadId);
        }

        IWormholeRelayer.MessageInfoType infoType = IWormholeRelayer.MessageInfoType(encoded.toUint8(index));
        index += 1;

        if (infoType == IWormholeRelayer.MessageInfoType.EMITTER_SEQUENCE) {
            messageInfo.emitterAddress = encoded.toBytes32(index);
            index += 32;

            messageInfo.sequence = encoded.toUint64(index);
            index += 8;
        } else if (infoType == IWormholeRelayer.MessageInfoType.VAAHASH) {
            messageInfo.vaaHash = encoded.toBytes32(index);
            index += 32;
        }
        newIndex = index;
    }

    // decode a 'DeliveryInstructionsContainer' from bytes
    function decodeDeliveryInstructionsContainer(bytes memory encoded)
        public
        pure
        returns (IWormholeRelayerInternalStructs.DeliveryInstructionsContainer memory)
    {
        uint256 index = 0;

        uint8 payloadId = encoded.toUint8(index);
        if (payloadId != 1) {
            revert InvalidPayloadId(payloadId);
        }
        index += 1;

        bytes32 senderAddress = encoded.toBytes32(index);
        index += 32;

        uint8 messagesArrayLen = encoded.toUint8(index);
        index += 1;

        uint8 instructionsArrayLen = encoded.toUint8(index);
        index += 1;

        IWormholeRelayer.MessageInfo[] memory messageInfos = new IWormholeRelayer.MessageInfo[](messagesArrayLen);
        for (uint8 i = 0; i < messagesArrayLen; i++) {
            (messageInfos[i], index) = decodeMessageInfo(encoded, index);
        }

        IWormholeRelayerInternalStructs.DeliveryInstruction[] memory instructionArray =
            new IWormholeRelayerInternalStructs.DeliveryInstruction[](instructionsArrayLen);
        for (uint8 i = 0; i < instructionsArrayLen; i++) {
            (instructionArray[i], index) = decodeDeliveryInstruction(encoded, index);
        }

        if (index != encoded.length) {
            revert InvalidDeliveryInstructionsPayload(encoded.length);
        }

        return IWormholeRelayerInternalStructs.DeliveryInstructionsContainer({
            payloadId: payloadId,
            senderAddress: senderAddress,
            messageInfos: messageInfos,
            instructions: instructionArray
        });
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

    // Helper to put one Send struct into a MultichainSend struct
    function multichainSendContainer(
        IWormholeRelayer.Send memory request,
        address relayProvider,
        IWormholeRelayer.MessageInfo[] memory messageInfos
    ) internal pure returns (IWormholeRelayer.MultichainSend memory container) {
        IWormholeRelayer.Send[] memory requests = new IWormholeRelayer.Send[](1);
        requests[0] = request;
        container = IWormholeRelayer.MultichainSend({
            relayProviderAddress: relayProvider,
            requests: requests,
            messageInfos: messageInfos
        });
    }
}
