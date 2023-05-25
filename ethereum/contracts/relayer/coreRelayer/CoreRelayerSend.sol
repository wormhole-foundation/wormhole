// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
  RelayProviderDoesNotSupportTargetChain,
  InvalidMsgValue,
  VaaKey,
  IWormholeRelayerSend
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";

import {toWormholeFormat} from "../../libraries/relayer/Utils.sol";
import {DeliveryInstruction, RedeliveryInstruction} from "../../libraries/relayer/RelayerInternalStructs.sol";
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
  
  function sendToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Gas gasLimit
  ) external payable returns (uint64 sequence) {
    return sendToEvm(
      targetChainId,
      targetAddress,
      payload,
      receiverValue,
      Wei.wrap(0),
      gasLimit,
      getWormhole().chainId(),
      msg.sender,
      getDefaultRelayProvider(),
      new VaaKey[](0),
      CONSISTENCY_LEVEL_FINALIZED
    );
  }

  function sendToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Gas gasLimit,
    uint16 refundChainId,
    address refundAddress
  ) external payable returns (uint64 sequence) {
    return sendToEvm(
      targetChainId,
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

  function sendToEvm(
    uint16 targetChainId,
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
      targetChainId,
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


  function forwardToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Gas gasLimit,
    uint16 refundChainId,
    address refundAddress
  ) external payable {
    forwardToEvm(
      targetChainId,
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

  function forwardToEvm(
    uint16 targetChainId,
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
    forward(
      targetChainId,
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
    uint16 targetChainId,
    Wei newReceiverValue,
    Gas newGasLimit,
    address newRelayProviderAddress
  ) public payable returns (uint64 sequence) {
    sequence = resend(
      deliveryVaaKey,
      targetChainId,
      newReceiverValue,
      encodeEvmExecutionParamsV1(EvmExecutionParamsV1(newGasLimit)),
      newRelayProviderAddress
    );
  }

  function send(
    uint16 targetChainId,
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
    sequence = send(Send(
      targetChainId,
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
    ));
  }

  function forward(
    uint16 targetChainId,
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
    forward(Send(
      targetChainId,
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
    ));
  }

  
  /* 
   * Non overload logic 
   */ 

  struct Send {
    uint16 targetChainId;
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
    if(!provider.isChainSupported(sendParams.targetChainId)) {
      revert RelayProviderDoesNotSupportTargetChain(sendParams.relayProviderAddress, sendParams.targetChainId);
    }
    (Wei deliveryPrice, bytes memory encodedExecutionInfo) = provider.quoteDeliveryPrice(sendParams.targetChainId, sendParams.receiverValue, sendParams.encodedExecutionParameters);

    Wei wormholeMessageFee = getWormholeMessageFee();
    checkMsgValue(wormholeMessageFee, deliveryPrice, sendParams.paymentForExtraReceiverValue);

    bytes memory encodedInstruction = DeliveryInstruction({
        targetChainId: sendParams.targetChainId,
        targetAddress: sendParams.targetAddress,
        payload: sendParams.payload,
        requestedReceiverValue: sendParams.receiverValue,
        extraReceiverValue: provider.quoteAssetConversion(sendParams.targetChainId, sendParams.paymentForExtraReceiverValue),
        encodedExecutionInfo: encodedExecutionInfo,
        refundChainId: sendParams.refundChainId,
        refundAddress: sendParams.refundAddress,
        refundRelayProvider: provider.getTargetChainAddress(sendParams.targetChainId),
        sourceRelayProvider: toWormholeFormat(sendParams.relayProviderAddress),
        senderAddress: toWormholeFormat(msg.sender),
        vaaKeys: sendParams.vaaKeys
    }).encode();

    sequence = publishAndPay(wormholeMessageFee, deliveryPrice, sendParams.paymentForExtraReceiverValue, encodedInstruction, sendParams.consistencyLevel, provider);
  }

  function forward(Send memory sendParams) internal {
    IRelayProvider provider = IRelayProvider(sendParams.relayProviderAddress);
    if(!provider.isChainSupported(sendParams.targetChainId)) {
      revert RelayProviderDoesNotSupportTargetChain(sendParams.relayProviderAddress, sendParams.targetChainId);
    }
    (Wei deliveryPrice, bytes memory encodedExecutionInfo) = provider.quoteDeliveryPrice(sendParams.targetChainId, sendParams.receiverValue, sendParams.encodedExecutionParameters);

    bytes memory encodedInstruction = DeliveryInstruction({
        targetChainId: sendParams.targetChainId,
        targetAddress: sendParams.targetAddress,
        payload: sendParams.payload,
        requestedReceiverValue: sendParams.receiverValue,
        extraReceiverValue: provider.quoteAssetConversion(sendParams.targetChainId, sendParams.paymentForExtraReceiverValue),
        encodedExecutionInfo: encodedExecutionInfo,
        refundChainId: sendParams.refundChainId,
        refundAddress: sendParams.refundAddress,
        refundRelayProvider: provider.getTargetChainAddress(sendParams.targetChainId),
        sourceRelayProvider: toWormholeFormat(sendParams.relayProviderAddress),
        senderAddress: toWormholeFormat(msg.sender),
        vaaKeys: sendParams.vaaKeys
    }).encode();

    appendForwardInstruction(ForwardInstruction({
        encodedInstruction: encodedInstruction,
        msgValue: Wei.wrap(msg.value),
        deliveryPrice: deliveryPrice,
        paymentForExtraReceiverValue: sendParams.paymentForExtraReceiverValue,
        consistencyLevel: sendParams.consistencyLevel
    }));
  }

  function resend(
    VaaKey memory deliveryVaaKey,
    uint16 targetChainId,
    Wei newReceiverValue,
    bytes memory newEncodedExecutionParameters,
    address newRelayProviderAddress
  ) public payable returns (uint64 sequence) {
    IRelayProvider provider = IRelayProvider(newRelayProviderAddress);
    if(!provider.isChainSupported(targetChainId)) {
      revert RelayProviderDoesNotSupportTargetChain(newRelayProviderAddress, targetChainId);
    }
    (Wei deliveryPrice, bytes memory encodedExecutionInfo) = provider.quoteDeliveryPrice(targetChainId, newReceiverValue, newEncodedExecutionParameters);

    Wei wormholeMessageFee = getWormholeMessageFee();
    checkMsgValue(wormholeMessageFee, deliveryPrice, Wei.wrap(0));

    bytes memory encodedInstruction = RedeliveryInstruction({
        deliveryVaaKey: deliveryVaaKey,
        targetChainId: targetChainId,
        newRequestedReceiverValue:newReceiverValue,
        newEncodedExecutionInfo: encodedExecutionInfo,
        newSourceRelayProvider: toWormholeFormat(newRelayProviderAddress),
        newSenderAddress: toWormholeFormat(msg.sender)
    }).encode();

    sequence = publishAndPay(wormholeMessageFee, deliveryPrice, Wei.wrap(0), encodedInstruction, CONSISTENCY_LEVEL_INSTANT, provider);
  }
  
  function getDefaultRelayProvider() public view returns (address relayProvider) {
    relayProvider = getDefaultRelayProviderState().defaultRelayProvider;
  }

  function quoteEVMDeliveryPrice(uint16 targetChainId, uint128 receiverValue, uint32 gasLimit, address relayProviderAddress) public view returns (uint256 nativePriceQuote, uint256 targetChainRefundPerGasUnused) {
    (uint256 quote, bytes memory encodedExecutionInfo) = quoteDeliveryPrice(targetChainId, receiverValue, encodeEvmExecutionParamsV1(EvmExecutionParamsV1(Gas.wrap(gasLimit))), relayProviderAddress);
    nativePriceQuote = quote;
    targetChainRefundPerGasUnused = GasPrice.unwrap(decodeEvmExecutionInfoV1(encodedExecutionInfo).targetChainRefundPerGasUnused);
  }

  function quoteEVMDeliveryPrice(uint16 targetChainId, uint128 receiverValue, uint32 gasLimit) public view returns (uint256 nativePriceQuote, uint256 targetChainRefundPerGasUnused) {
    return quoteEVMDeliveryPrice(targetChainId, receiverValue, gasLimit, getDefaultRelayProvider());
  }

  function quoteDeliveryPrice(uint16 targetChainId, uint128 receiverValue, bytes memory encodedExecutionParameters, address relayProviderAddress) public view returns (uint256 nativePriceQuote, bytes memory encodedExecutionInfo) {
    IRelayProvider provider = IRelayProvider(relayProviderAddress);
    (Wei deliveryPrice, bytes memory _encodedExecutionInfo) = provider.quoteDeliveryPrice(targetChainId, Wei.wrap(receiverValue), encodedExecutionParameters);
    encodedExecutionInfo = _encodedExecutionInfo;
    nativePriceQuote = deliveryPrice.unwrap();
  }

  function quoteAssetConversion(
    uint16 targetChainId,
    uint128 currentChainAmount,
    address relayProviderAddress
  ) public view returns (uint256 targetChainAmount) {
    IRelayProvider provider = IRelayProvider(relayProviderAddress);
    return provider.quoteAssetConversion(targetChainId, Wei.wrap(currentChainAmount)).unwrap();
  }
}