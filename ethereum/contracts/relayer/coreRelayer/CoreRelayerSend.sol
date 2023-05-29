// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
    RelayProviderDoesNotSupportTargetChain,
    InvalidMsgValue,
    VaaKey,
    IWormholeRelayerSend
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";

import {toWormholeFormat, fromWormholeFormat} from "../../libraries/relayer/Utils.sol";
import {
    DeliveryInstruction,
    RedeliveryInstruction
} from "../../libraries/relayer/RelayerInternalStructs.sol";
import {CoreRelayerSerde} from "./CoreRelayerSerde.sol";
import {ForwardInstruction, getDefaultRelayProviderState} from "./CoreRelayerStorage.sol";
import {CoreRelayerBase} from "./CoreRelayerBase.sol";
import "../../interfaces/relayer/TypedUnits.sol";
import "../../libraries/relayer/ExecutionParameters.sol";

//TODO:
// Introduce basic sanity checks on sendParams (e.g. all valus below 2^128?) so we can get rid of
//   all the silly checked math and ensure that we can't have overflow Panics either.
// In send() and resend() we already check that maxTransactionFee + receiverValue == msg.value (via
//   calcAndCheckFees(). We could perhaps introduce a similar check of <= this.balance in forward()
//   and presumably a few more in our calculation/conversion functions CoreRelayerBase to ensure
//   sensible numeric ranges everywhere.

abstract contract CoreRelayerSend is CoreRelayerBase, IWormholeRelayerSend {
    using CoreRelayerSerde for *; //somewhat yucky but unclear what's a better alternative
    using WeiLib for Wei;
    using GasLib for Gas;

    /*
    * Public convenience overloads
    */

    function sendPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        Wei receiverValue,
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
            getDefaultRelayProviderOnChain(targetChain),
            getDefaultRelayProvider(),
            new VaaKey[](0),
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function sendPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        Wei receiverValue,
        Gas gasLimit,
        uint16 refundChainId,
        address refundAddress
    ) external payable returns (uint64 sequence) {
        return sendToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            Wei.wrap(0),
            gasLimit,
            refundChainId,
            refundAddress,
            getDefaultRelayProvider(),
            new VaaKey[](0),
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function sendVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        Wei receiverValue,
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
            getDefaultRelayProviderOnChain(targetChain),
            getDefaultRelayProvider(),
            vaaKeys,
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function sendVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        Wei receiverValue,
        Gas gasLimit,
        VaaKey[] memory vaaKeys,
        uint16 refundChainId,
        address refundAddress
    ) external payable returns (uint64 sequence) {
        return sendToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            Wei.wrap(0),
            gasLimit,
            refundChainId,
            refundAddress,
            getDefaultRelayProvider(),
            vaaKeys,
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function sendToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        Wei receiverValue,
        Wei paymentForExtraReceiverValue,
        Gas gasLimit,
        uint16 refundChainId,
        address refundAddress,
        address relayProviderAddress,
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
            refundChainId,
            toWormholeFormat(refundAddress),
            relayProviderAddress,
            vaaKeys,
            consistencyLevel
        );
    }

    function forwardPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        Wei receiverValue,
        Gas gasLimit
    ) external payable {
        (address relayProvider, address relayProviderOnTarget) =
            getOriginalOrDefaultRelayProvider(targetChain);
        forwardToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            Wei.wrap(0),
            gasLimit,
            targetChain,
            relayProviderOnTarget,
            relayProvider,
            new VaaKey[](0),
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function forwardVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        Wei receiverValue,
        Gas gasLimit,
        VaaKey[] memory vaaKeys
    ) external payable {
        (address relayProvider, address relayProviderOnTarget) =
            getOriginalOrDefaultRelayProvider(targetChain);
        forwardToEvm(
            targetChain,
            targetAddress,
            payload,
            receiverValue,
            Wei.wrap(0),
            gasLimit,
            targetChain,
            relayProviderOnTarget,
            relayProvider,
            vaaKeys,
            CONSISTENCY_LEVEL_FINALIZED
        );
    }

    function forwardToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        Wei receiverValue,
        Wei paymentForExtraReceiverValue,
        Gas gasLimit,
        uint16 refundChainId,
        address refundAddress,
        address relayProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) public payable {
        // provide ability to use original relay provider
        if (relayProviderAddress == address(0)) {
            relayProviderAddress = getOriginalRelayProvider();
        }

        forward(
            targetChain,
            toWormholeFormat(targetAddress),
            payload,
            receiverValue,
            paymentForExtraReceiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(gasLimit)),
            refundChainId,
            toWormholeFormat(refundAddress),
            relayProviderAddress,
            vaaKeys,
            consistencyLevel
        );
    }

    function resendToEvm(
        VaaKey memory deliveryVaaKey,
        uint16 targetChain,
        Wei newReceiverValue,
        Gas newGasLimit,
        address newRelayProviderAddress
    ) public payable returns (uint64 sequence) {
        sequence = resend(
            deliveryVaaKey,
            targetChain,
            newReceiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(newGasLimit)),
            newRelayProviderAddress
        );
    }

    function send(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes memory payload,
        Wei receiverValue,
        Wei paymentForExtraReceiverValue,
        bytes memory encodedExecutionParameters,
        uint16 refundChainId,
        bytes32 refundAddress,
        address relayProviderAddress,
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
                refundChainId,
                refundAddress,
                relayProviderAddress,
                vaaKeys,
                consistencyLevel
            )
        );
    }

    function forward(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes memory payload,
        Wei receiverValue,
        Wei paymentForExtraReceiverValue,
        bytes memory encodedExecutionParameters,
        uint16 refundChainId,
        bytes32 refundAddress,
        address relayProviderAddress,
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
                refundChainId,
                refundAddress,
                relayProviderAddress,
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
        Wei receiverValue;
        Wei paymentForExtraReceiverValue;
        bytes encodedExecutionParameters;
        uint16 refundChainId;
        bytes32 refundAddress;
        address relayProviderAddress;
        VaaKey[] vaaKeys;
        uint8 consistencyLevel;
    }

    function send(Send memory sendParams) internal returns (uint64 sequence) {
        IRelayProvider provider = IRelayProvider(sendParams.relayProviderAddress);
        if (!provider.isChainSupported(sendParams.targetChain)) {
            revert RelayProviderDoesNotSupportTargetChain(
                sendParams.relayProviderAddress, sendParams.targetChain
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
            refundChainId: sendParams.refundChainId,
            refundAddress: sendParams.refundAddress,
            refundRelayProvider: provider.getTargetChainAddress(sendParams.targetChain),
            sourceRelayProvider: toWormholeFormat(sendParams.relayProviderAddress),
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
        IRelayProvider provider = IRelayProvider(sendParams.relayProviderAddress);
        if (!provider.isChainSupported(sendParams.targetChain)) {
            revert RelayProviderDoesNotSupportTargetChain(
                sendParams.relayProviderAddress, sendParams.targetChain
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
            refundChainId: sendParams.refundChainId,
            refundAddress: sendParams.refundAddress,
            refundRelayProvider: provider.getTargetChainAddress(sendParams.targetChain),
            sourceRelayProvider: toWormholeFormat(sendParams.relayProviderAddress),
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
        Wei newReceiverValue,
        bytes memory newEncodedExecutionParameters,
        address newRelayProviderAddress
    ) public payable returns (uint64 sequence) {
        IRelayProvider provider = IRelayProvider(newRelayProviderAddress);
        if (!provider.isChainSupported(targetChain)) {
            revert RelayProviderDoesNotSupportTargetChain(newRelayProviderAddress, targetChain);
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
            newSourceRelayProvider: toWormholeFormat(newRelayProviderAddress),
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

    function getDefaultRelayProvider() public view returns (address relayProvider) {
        relayProvider = getDefaultRelayProviderState().defaultRelayProvider;
    }

    function getDefaultRelayProviderOnChain(uint16 targetChain)
        public
        view
        returns (address relayProvider)
    {
        relayProvider = fromWormholeFormat(
            IRelayProvider(getDefaultRelayProviderState().defaultRelayProvider)
                .getTargetChainAddress(targetChain)
        );
    }

    function getOriginalOrDefaultRelayProvider(uint16 targetChain)
        public
        view
        returns (address relayProvider, address relayProviderOnTarget)
    {
        relayProvider = getOriginalRelayProvider();
        if (
            relayProvider == address(0)
                || !IRelayProvider(relayProvider).isChainSupported(targetChain)
        ) {
            relayProvider = getDefaultRelayProvider();
        }
        relayProviderOnTarget =
            fromWormholeFormat(IRelayProvider(relayProvider).getTargetChainAddress(targetChain));
    }

    function quoteEVMDeliveryPrice(
        uint16 targetChain,
        uint128 receiverValue,
        uint32 gasLimit,
        address relayProviderAddress
    ) public view returns (uint256 nativePriceQuote, uint256 targetChainRefundPerGasUnused) {
        (uint256 quote, bytes memory encodedExecutionInfo) = quoteDeliveryPrice(
            targetChain,
            receiverValue,
            encodeEvmExecutionParamsV1(EvmExecutionParamsV1(Gas.wrap(gasLimit))),
            relayProviderAddress
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
        return
            quoteEVMDeliveryPrice(targetChain, receiverValue, gasLimit, getDefaultRelayProvider());
    }

    function quoteDeliveryPrice(
        uint16 targetChain,
        uint128 receiverValue,
        bytes memory encodedExecutionParameters,
        address relayProviderAddress
    ) public view returns (uint256 nativePriceQuote, bytes memory encodedExecutionInfo) {
        IRelayProvider provider = IRelayProvider(relayProviderAddress);
        (Wei deliveryPrice, bytes memory _encodedExecutionInfo) = provider.quoteDeliveryPrice(
            targetChain, Wei.wrap(receiverValue), encodedExecutionParameters
        );
        encodedExecutionInfo = _encodedExecutionInfo;
        nativePriceQuote = deliveryPrice.unwrap();
    }

    function quoteNativeForChain(
        uint16 targetChain,
        uint128 currentChainAmount,
        address relayProviderAddress
    ) public view returns (uint256 targetChainAmount) {
        IRelayProvider provider = IRelayProvider(relayProviderAddress);
        return provider.quoteAssetConversion(targetChain, Wei.wrap(currentChainAmount)).unwrap();
    }
}
