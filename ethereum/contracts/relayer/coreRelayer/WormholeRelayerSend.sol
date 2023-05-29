// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
    DeliveryProviderDoesNotSupportTargetChain,
    InvalidMsgValue,
    VaaKey,
    IWormholeRelayerSend
} from "../../interfaces/relayer/IWormholeRelayerTyped.sol";
import {IDeliveryProvider} from "../../interfaces/relayer/IDeliveryProviderTyped.sol";

import {toWormholeFormat, fromWormholeFormat} from "../../libraries/relayer/Utils.sol";
import {
    DeliveryInstruction,
    RedeliveryInstruction
} from "../../libraries/relayer/RelayerInternalStructs.sol";
import {WormholeRelayerSerde} from "./WormholeRelayerSerde.sol";
import {ForwardInstruction, getDefaultDeliveryProviderState} from "./WormholeRelayerStorage.sol";
import {WormholeRelayerBase} from "./WormholeRelayerBase.sol";
import "../../interfaces/relayer/TypedUnits.sol";
import "../../libraries/relayer/ExecutionParameters.sol";

//TODO:
// Introduce basic sanity checks on sendParams (e.g. all valus below 2^128?) so we can get rid of
//   all the silly checked math and ensure that we can't have overflow Panics either.
// In send() and resend() we already check that maxTransactionFee + receiverValue == msg.value (via
//   calcAndCheckFees(). We could perhaps introduce a similar check of <= this.balance in forward()
//   and presumably a few more in our calculation/conversion functions WormholeRelayerBase to ensure
//   sensible numeric ranges everywhere.

abstract contract WormholeRelayerSend is WormholeRelayerBase, IWormholeRelayerSend {
    using WormholeRelayerSerde for *; //somewhat yucky but unclear what's a better alternative
    using WeiLib for Wei;
    using GasLib for Gas;
    using TargetNativeLib for TargetNative;

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
            Wei.wrap(0),
            gasLimit,
            targetChain,
            getDefaultDeliveryProviderOnChain(targetChain),
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
            Wei.wrap(0),
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
            Wei.wrap(0),
            gasLimit,
            targetChain,
            getDefaultDeliveryProviderOnChain(targetChain),
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
            Wei.wrap(0),
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
        Wei paymentForExtraReceiverValue,
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

    function forwardPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit
    ) external payable {
        (address deliveryProvider, address deliveryProviderOnTarget) =
            getOriginalOrDefaultDeliveryProvider(targetChain);
        forwardToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            Wei.wrap(0),
            gasLimit,
            targetChain,
            deliveryProviderOnTarget,
            deliveryProvider,
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
        (address deliveryProvider, address deliveryProviderOnTarget) =
            getOriginalOrDefaultDeliveryProvider(targetChain);
        forwardToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            Wei.wrap(0),
            gasLimit,
            targetChain,
            deliveryProviderOnTarget,
            deliveryProvider,
            vaaKeys,
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function forwardToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Wei paymentForExtraReceiverValue,
        Gas gasLimit,
        uint16 refundChain,
        address refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) public payable {
        // provide ability to use original relay provider
        if (deliveryProviderAddress == address(0)) {
            deliveryProviderAddress = getOriginalDeliveryProvider();
        }

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
        Wei paymentForExtraReceiverValue,
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
                vaaKeys,
                consistencyLevel
            )
        );
    }

    function forward(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Wei paymentForExtraReceiverValue,
        bytes memory encodedExecutionParameters,
        uint16 refundChain,
        bytes32 refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) public payable {
        forward(
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
                vaaKeys,
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
        Wei paymentForExtraReceiverValue;
        bytes encodedExecutionParameters;
        uint16 refundChain;
        bytes32 refundAddress;
        address deliveryProviderAddress;
        VaaKey[] vaaKeys;
        uint8 consistencyLevel;
    }

    function send(Send memory sendParams) internal returns (uint64 sequence) {
        IDeliveryProvider provider = IDeliveryProvider(sendParams.deliveryProviderAddress);
        if (!provider.isChainSupported(sendParams.targetChain)) {
            revert DeliveryProviderDoesNotSupportTargetChain(
                sendParams.deliveryProviderAddress, sendParams.targetChain
            );
        }
        (Wei deliveryPrice, bytes memory encodedExecutionInfo) = provider.quoteDeliveryPrice(
            sendParams.targetChain, sendParams.receiverValue, sendParams.encodedExecutionParameters
        );

        Wei wormholeMessageFee = getWormholeMessageFee();
        checkMsgValue(wormholeMessageFee, deliveryPrice, sendParams.paymentForExtraReceiverValue);

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
            vaaKeys: sendParams.vaaKeys
        }).encode();

        sequence = publishAndPay(
            wormholeMessageFee,
            deliveryPrice,
            sendParams.paymentForExtraReceiverValue,
            encodedInstruction,
            sendParams.consistencyLevel,
            provider
        );
    }

    function forward(Send memory sendParams) internal {
        checkMsgSenderInDelivery();
        IDeliveryProvider provider = IDeliveryProvider(sendParams.deliveryProviderAddress);
        if (!provider.isChainSupported(sendParams.targetChain)) {
            revert DeliveryProviderDoesNotSupportTargetChain(
                sendParams.deliveryProviderAddress, sendParams.targetChain
            );
        }
        (Wei deliveryPrice, bytes memory encodedExecutionInfo) = provider.quoteDeliveryPrice(
            sendParams.targetChain, sendParams.receiverValue, sendParams.encodedExecutionParameters
        );

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
            vaaKeys: sendParams.vaaKeys
        }).encode();

        appendForwardInstruction(
            ForwardInstruction({
                encodedInstruction: encodedInstruction,
                msgValue: Wei.wrap(msg.value),
                deliveryPrice: deliveryPrice,
                paymentForExtraReceiverValue: sendParams.paymentForExtraReceiverValue,
                consistencyLevel: sendParams.consistencyLevel
            })
        );
    }

    function resend(
        VaaKey memory deliveryVaaKey,
        uint16 targetChain,
        TargetNative newReceiverValue,
        bytes memory newEncodedExecutionParameters,
        address newDeliveryProviderAddress
    ) public payable returns (uint64 sequence) {
        IDeliveryProvider provider = IDeliveryProvider(newDeliveryProviderAddress);
        if (!provider.isChainSupported(targetChain)) {
            revert DeliveryProviderDoesNotSupportTargetChain(
                newDeliveryProviderAddress, targetChain
            );
        }
        (Wei deliveryPrice, bytes memory encodedExecutionInfo) = provider.quoteDeliveryPrice(
            targetChain, newReceiverValue, newEncodedExecutionParameters
        );

        Wei wormholeMessageFee = getWormholeMessageFee();
        checkMsgValue(wormholeMessageFee, deliveryPrice, Wei.wrap(0));

        bytes memory encodedInstruction = RedeliveryInstruction({
            deliveryVaaKey: deliveryVaaKey,
            targetChain: targetChain,
            newRequestedReceiverValue: newReceiverValue,
            newEncodedExecutionInfo: encodedExecutionInfo,
            newSourceDeliveryProvider: toWormholeFormat(newDeliveryProviderAddress),
            newSenderAddress: toWormholeFormat(msg.sender)
        }).encode();

        sequence = publishAndPay(
            wormholeMessageFee,
            deliveryPrice,
            Wei.wrap(0),
            encodedInstruction,
            CONSISTENCY_LEVEL_INSTANT,
            provider
        );
    }

    function getDefaultDeliveryProvider() public view returns (address deliveryProvider) {
        deliveryProvider = getDefaultDeliveryProviderState().defaultDeliveryProvider;
    }

    function getDefaultDeliveryProviderOnChain(uint16 targetChain)
        public
        view
        returns (address deliveryProvider)
    {
        deliveryProvider = fromWormholeFormat(
            IDeliveryProvider(getDefaultDeliveryProviderState().defaultDeliveryProvider)
                .getTargetChainAddress(targetChain)
        );
    }

    function getOriginalOrDefaultDeliveryProvider(uint16 targetChain)
        public
        view
        returns (address deliveryProvider, address deliveryProviderOnTarget)
    {
        deliveryProvider = getOriginalDeliveryProvider();
        if (
            deliveryProvider == address(0)
                || !IDeliveryProvider(deliveryProvider).isChainSupported(targetChain)
        ) {
            deliveryProvider = getDefaultDeliveryProvider();
        }
        deliveryProviderOnTarget = fromWormholeFormat(
            IDeliveryProvider(deliveryProvider).getTargetChainAddress(targetChain)
        );
    }

    function quoteEVMDeliveryPrice(
        uint16 targetChain,
        uint128 receiverValue,
        uint32 gasLimit,
        address deliveryProviderAddress
    ) public view returns (uint256 nativePriceQuote, uint256 targetChainRefundPerGasUnused) {
        (uint256 quote, bytes memory encodedExecutionInfo) = quoteDeliveryPrice(
            targetChain,
            receiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(Gas.wrap(gasLimit))),
            deliveryProviderAddress
        );
        nativePriceQuote = quote;
        targetChainRefundPerGasUnused = GasPrice.unwrap(
            decodeEvmExecutionInfoV1(encodedExecutionInfo).targetChainRefundPerGasUnused
        );
    }

    function quoteEVMDeliveryPrice(
        uint16 targetChain,
        uint128 receiverValue,
        uint32 gasLimit
    ) public view returns (uint256 nativePriceQuote, uint256 targetChainRefundPerGasUnused) {
        return quoteEVMDeliveryPrice(
            targetChain, receiverValue, gasLimit, getDefaultDeliveryProvider()
        );
    }

    function quoteDeliveryPrice(
        uint16 targetChain,
        uint128 receiverValue,
        bytes memory encodedExecutionParameters,
        address deliveryProviderAddress
    ) public view returns (uint256 nativePriceQuote, bytes memory encodedExecutionInfo) {
        IDeliveryProvider provider = IDeliveryProvider(deliveryProviderAddress);
        (Wei deliveryPrice, bytes memory _encodedExecutionInfo) = provider.quoteDeliveryPrice(
            targetChain, TargetNative.wrap(receiverValue), encodedExecutionParameters
        );
        encodedExecutionInfo = _encodedExecutionInfo;
        nativePriceQuote = deliveryPrice.unwrap();
    }

    function quoteNativeForChain(
        uint16 targetChain,
        uint128 currentChainAmount,
        address deliveryProviderAddress
    ) public view returns (uint256 targetChainAmount) {
        IDeliveryProvider provider = IDeliveryProvider(deliveryProviderAddress);
        return provider.quoteAssetConversion(targetChain, Wei.wrap(currentChainAmount)).unwrap();
    }
}
