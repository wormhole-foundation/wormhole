// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
    DeliveryProviderDoesNotSupportTargetChain,
    DeliveryProviderDoesNotSupportMessageKeyType,
    InvalidMsgValue,
    DeliveryProviderCannotReceivePayment,
    MessageKey,
    VaaKey,
    IWormholeRelayerSend
} from "../../interfaces/relayer/IWormholeRelayerTyped.sol";
import {IDeliveryProvider} from "../../interfaces/relayer/IDeliveryProviderTyped.sol";

import {toWormholeFormat, fromWormholeFormat} from "../../relayer/libraries/Utils.sol";
import {
    DeliveryInstruction,
    RedeliveryInstruction
} from "../../relayer/libraries/RelayerInternalStructs.sol";
import {WormholeRelayerSerde} from "./WormholeRelayerSerde.sol";
import {getDefaultDeliveryProviderState} from "./WormholeRelayerStorage.sol";
import {WormholeRelayerBase} from "./WormholeRelayerBase.sol";
import "../../interfaces/relayer/TypedUnits.sol";
import "../../relayer/libraries/ExecutionParameters.sol";

abstract contract WormholeRelayerSend is WormholeRelayerBase, IWormholeRelayerSend {
    using WormholeRelayerSerde for *;
    using WeiLib for Wei;
    using GasLib for Gas;
    using TargetNativeLib for TargetNative;
    using LocalNativeLib for LocalNative;

    /*
    * Public convenience overloads
    */

    function sendPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit
    ) external payable returns (uint64 sequence) {
        return sendToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            LocalNative.wrap(0),
            gasLimit,
            targetChain,
            address(0x0),
            getDefaultDeliveryProvider(),
            new VaaKey[](0),
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function sendPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit,
        uint16 refundChain,
        address refundAddress
    ) external payable returns (uint64 sequence) {
        return sendToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            LocalNative.wrap(0),
            gasLimit,
            refundChain,
            refundAddress,
            getDefaultDeliveryProvider(),
            new VaaKey[](0),
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function sendVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit,
        VaaKey[] memory vaaKeys
    ) external payable returns (uint64 sequence) {
        return sendToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            LocalNative.wrap(0),
            gasLimit,
            targetChain,
            address(0x0),
            getDefaultDeliveryProvider(),
            vaaKeys,
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function sendVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit,
        VaaKey[] memory vaaKeys,
        uint16 refundChain,
        address refundAddress
    ) external payable returns (uint64 sequence) {
        return sendToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            LocalNative.wrap(0),
            gasLimit,
            refundChain,
            refundAddress,
            getDefaultDeliveryProvider(),
            vaaKeys,
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function sendToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        Gas gasLimit,
        uint16 refundChain,
        address refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) public payable returns (uint64 sequence) {
        sequence = send(
            targetChain,
            toWormholeFormat(targetAddress),
            payload,
            receiverValue,
            paymentForExtraReceiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(gasLimit)),
            refundChain,
            toWormholeFormat(refundAddress),
            deliveryProviderAddress,
            vaaKeys,
            consistencyLevel
        );
    }

    function sendToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        Gas gasLimit,
        uint16 refundChain,
        address refundAddress,
        address deliveryProviderAddress,
        MessageKey[] memory messageKeys,
        uint8 consistencyLevel
    ) public payable returns (uint64 sequence) {
        sequence = send(
            targetChain,
            toWormholeFormat(targetAddress),
            payload,
            receiverValue,
            paymentForExtraReceiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(gasLimit)),
            refundChain,
            toWormholeFormat(refundAddress),
            deliveryProviderAddress,
            messageKeys,
            consistencyLevel
        );
    }

    function resendToEvm(
        VaaKey memory deliveryVaaKey,
        uint16 targetChain,
        TargetNative newReceiverValue,
        Gas newGasLimit,
        address newDeliveryProviderAddress
    ) public payable returns (uint64 sequence) {
        sequence = resend(
            deliveryVaaKey,
            targetChain,
            newReceiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(newGasLimit)),
            newDeliveryProviderAddress
        );
    }

    function send(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        bytes memory encodedExecutionParameters,
        uint16 refundChain,
        bytes32 refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) public payable returns (uint64 sequence) {
        sequence = send(
            Send(
                targetChain,
                targetAddress,
                payload,
                receiverValue,
                paymentForExtraReceiverValue,
                encodedExecutionParameters,
                refundChain,
                refundAddress,
                deliveryProviderAddress,
                WormholeRelayerSerde.vaaKeyArrayToMessageKeyArray(vaaKeys),
                consistencyLevel
            )
        );
    }

    function send(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        bytes memory encodedExecutionParameters,
        uint16 refundChain,
        bytes32 refundAddress,
        address deliveryProviderAddress,
        MessageKey[] memory messageKeys,
        uint8 consistencyLevel
    ) public payable returns (uint64 sequence) {
        sequence = send(
            Send(
                targetChain,
                targetAddress,
                payload,
                receiverValue,
                paymentForExtraReceiverValue,
                encodedExecutionParameters,
                refundChain,
                refundAddress,
                deliveryProviderAddress,
                messageKeys,
                consistencyLevel
            )
        );
    }

    /* 
    * Non overload logic 
    */

    struct Send {
        uint16 targetChain;
        bytes32 targetAddress;
        bytes payload;
        TargetNative receiverValue;
        LocalNative paymentForExtraReceiverValue;
        bytes encodedExecutionParameters;
        uint16 refundChain;
        bytes32 refundAddress;
        address deliveryProviderAddress;
        MessageKey[] messageKeys;
        uint8 consistencyLevel;
    }

    function send(Send memory sendParams) internal returns (uint64 sequence) {
        IDeliveryProvider provider = IDeliveryProvider(sendParams.deliveryProviderAddress);

        // Revert if delivery provider does not support the target chain
        if (!provider.isChainSupported(sendParams.targetChain)) {
            revert DeliveryProviderDoesNotSupportTargetChain(
                sendParams.deliveryProviderAddress, sendParams.targetChain
            );
        }

        // Obtain the delivery provider's fee for this delivery, as well as some encoded info (e.g. refund per unit of gas unused)
        (LocalNative deliveryPrice, bytes memory encodedExecutionInfo) = provider.quoteDeliveryPrice(
            sendParams.targetChain, sendParams.receiverValue, sendParams.encodedExecutionParameters
        );

        // Check if user passed in 'one wormhole message fee' + 'delivery provider's fee'
        LocalNative wormholeMessageFee = getWormholeMessageFee();
        checkMsgValue(wormholeMessageFee, deliveryPrice, sendParams.paymentForExtraReceiverValue);

        checkKeyTypesSupported(provider, sendParams.messageKeys);

        // Encode all relevant info the delivery provider needs to perform the delivery as requested
        bytes memory encodedInstruction = DeliveryInstruction({
            targetChain: sendParams.targetChain,
            targetAddress: sendParams.targetAddress,
            payload: sendParams.payload,
            requestedReceiverValue: sendParams.receiverValue,
            extraReceiverValue: provider.quoteAssetConversion(
                sendParams.targetChain, sendParams.paymentForExtraReceiverValue
                ),
            encodedExecutionInfo: encodedExecutionInfo,
            refundChain: sendParams.refundChain,
            refundAddress: sendParams.refundAddress,
            refundDeliveryProvider: provider.getTargetChainAddress(sendParams.targetChain),
            sourceDeliveryProvider: toWormholeFormat(sendParams.deliveryProviderAddress),
            senderAddress: toWormholeFormat(msg.sender),
            messageKeys: sendParams.messageKeys
        }).encode();

        // Publish the encoded delivery instruction as a wormhole message
        // and pay the delivery provider their fee
        bool paymentSucceeded;
        (sequence, paymentSucceeded) = publishAndPay(
            wormholeMessageFee,
            deliveryPrice,
            sendParams.paymentForExtraReceiverValue,
            encodedInstruction,
            sendParams.consistencyLevel,
            provider.getRewardAddress()
        );

        if (!paymentSucceeded) {
            revert DeliveryProviderCannotReceivePayment();
        }
    }

    function checkKeyTypesSupported(
        IDeliveryProvider provider,
        MessageKey[] memory messageKeys
    ) internal view {
        uint256 len = messageKeys.length;
        if (len == 0) {
            return;
        }

        uint256 supportedKeyTypes = provider.getSupportedKeys();
        for (uint256 i = 0; i < len;) {
            uint8 keyType = messageKeys[i].keyType;
            if ((supportedKeyTypes & (1 << keyType)) == 0) {
                revert DeliveryProviderDoesNotSupportMessageKeyType(keyType);
            }
            unchecked {
                ++i;
            }
        }
    }

    function resend(
        VaaKey memory deliveryVaaKey,
        uint16 targetChain,
        TargetNative newReceiverValue,
        bytes memory newEncodedExecutionParameters,
        address newDeliveryProviderAddress
    ) public payable returns (uint64 sequence) {
        IDeliveryProvider provider = IDeliveryProvider(newDeliveryProviderAddress);

        // Revert if delivery provider does not support the target chain
        if (!provider.isChainSupported(targetChain)) {
            revert DeliveryProviderDoesNotSupportTargetChain(
                newDeliveryProviderAddress, targetChain
            );
        }

        // Obtain the delivery provider's fee for this delivery, as well as some encoded info (e.g. refund per unit of gas unused)
        (LocalNative deliveryPrice, bytes memory encodedExecutionInfo) = provider.quoteDeliveryPrice(
            targetChain, newReceiverValue, newEncodedExecutionParameters
        );

        // Check if user passed in 'one wormhole message fee' + 'delivery provider's fee'
        LocalNative wormholeMessageFee = getWormholeMessageFee();
        checkMsgValue(wormholeMessageFee, deliveryPrice, LocalNative.wrap(0));

        // Encode all relevant info the delivery provider needs to perform this redelivery as requested
        bytes memory encodedInstruction = RedeliveryInstruction({
            deliveryVaaKey: deliveryVaaKey,
            targetChain: targetChain,
            newRequestedReceiverValue: newReceiverValue,
            newEncodedExecutionInfo: encodedExecutionInfo,
            newSourceDeliveryProvider: toWormholeFormat(newDeliveryProviderAddress),
            newSenderAddress: toWormholeFormat(msg.sender)
        }).encode();

        // Publish the encoded redelivery instruction as a wormhole message
        // and pay the delivery provider their fee
        bool paymentSucceeded;
        (sequence, paymentSucceeded) = publishAndPay(
            wormholeMessageFee,
            deliveryPrice,
            LocalNative.wrap(0),
            encodedInstruction,
            CONSISTENCY_LEVEL_INSTANT,
            provider.getRewardAddress()
        );
        if (!paymentSucceeded) {
            revert DeliveryProviderCannotReceivePayment();
        }
    }

    function getDefaultDeliveryProvider() public view returns (address deliveryProvider) {
        deliveryProvider = getDefaultDeliveryProviderState().defaultDeliveryProvider;
    }

    function quoteEVMDeliveryPrice(
        uint16 targetChain,
        TargetNative receiverValue,
        Gas gasLimit,
        address deliveryProviderAddress
    ) public view returns (LocalNative nativePriceQuote, GasPrice targetChainRefundPerGasUnused) {
        (LocalNative quote, bytes memory encodedExecutionInfo) = quoteDeliveryPrice(
            targetChain,
            receiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(gasLimit)),
            deliveryProviderAddress
        );
        nativePriceQuote = quote;
        targetChainRefundPerGasUnused =
            decodeEvmExecutionInfoV1(encodedExecutionInfo).targetChainRefundPerGasUnused;
    }

    function quoteEVMDeliveryPrice(
        uint16 targetChain,
        TargetNative receiverValue,
        Gas gasLimit
    ) public view returns (LocalNative nativePriceQuote, GasPrice targetChainRefundPerGasUnused) {
        return quoteEVMDeliveryPrice(
            targetChain, receiverValue, gasLimit, getDefaultDeliveryProvider()
        );
    }

    function quoteDeliveryPrice(
        uint16 targetChain,
        TargetNative receiverValue,
        bytes memory encodedExecutionParameters,
        address deliveryProviderAddress
    ) public view returns (LocalNative nativePriceQuote, bytes memory encodedExecutionInfo) {
        IDeliveryProvider provider = IDeliveryProvider(deliveryProviderAddress);
        (LocalNative deliveryPrice, bytes memory _encodedExecutionInfo) =
            provider.quoteDeliveryPrice(targetChain, receiverValue, encodedExecutionParameters);
        encodedExecutionInfo = _encodedExecutionInfo;
        nativePriceQuote = deliveryPrice + getWormholeMessageFee();
    }

    function quoteNativeForChain(
        uint16 targetChain,
        LocalNative currentChainAmount,
        address deliveryProviderAddress
    ) public view returns (TargetNative targetChainAmount) {
        return IDeliveryProvider(deliveryProviderAddress).quoteAssetConversion(
            targetChain, currentChainAmount
        );
    }

    // Forwards

    function forwardPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit
    ) external payable {
        forward(
            targetChain,
            toWormholeFormat(targetAddress),
            payload,
            receiverValue,
            LocalNative.wrap(0),
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(gasLimit)),
            getCurrentRefundChain(),
            getCurrentRefundAddress(),
            getDefaultDeliveryProvider(),
            new VaaKey[](0),
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function forwardVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit,
        VaaKey[] memory vaaKeys
    ) external payable {
        forward(
            targetChain,
            toWormholeFormat(targetAddress),
            payload,
            receiverValue,
            LocalNative.wrap(0),
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(gasLimit)),
            getCurrentRefundChain(),
            getCurrentRefundAddress(),
            getDefaultDeliveryProvider(),
            vaaKeys,
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function forwardToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        Gas gasLimit,
        uint16 refundChain,
        address refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) public payable {
        forward(
            targetChain,
            toWormholeFormat(targetAddress),
            payload,
            receiverValue,
            paymentForExtraReceiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(gasLimit)),
            refundChain,
            toWormholeFormat(refundAddress),
            deliveryProviderAddress,
            vaaKeys,
            consistencyLevel
        );
    }

    function forward(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative,
        bytes memory encodedExecutionParameters,
        uint16 refundChain,
        bytes32 refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) public payable {
        (LocalNative cost,) = quoteDeliveryPrice(targetChain, receiverValue, encodedExecutionParameters, deliveryProviderAddress);
        send(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            LocalNative.wrap(msg.value) - cost, // include the extra value that is passed in
            encodedExecutionParameters,
            refundChain,
            refundAddress,
            deliveryProviderAddress,
            vaaKeys,
            consistencyLevel
        );
    }
}
