// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
  RelayProviderDoesNotSupportTargetChain,
  InvalidMsgValue,
  VaaKey,
  DeliveryInstruction,
  RedeliveryInstruction,
  IWormholeRelayerSend
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";

import {toWormholeFormat} from "./Utils.sol";
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
    uint128 receiverValue,
    uint32 gasLimit
  ) external payable returns (uint64 sequence) {
    return sendToEvm(
      targetChainId,
      targetAddress,
      payload,
      receiverValue,
      0,
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
    uint128 receiverValue,
    uint32 gasLimit,
    uint16 refundChainId,
    address refundAddress
  ) external payable returns (uint64 sequence) {
    return sendToEvm(
      targetChainId,
      targetAddress,
      payload,
      receiverValue,
      0, 
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
    uint128 receiverValue,
    uint128 paymentForExtraReceiverValue,
    uint32 gasLimit,
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
      encodeEvmExecutionParamsV1(EvmExecutionParamsV1(Gas.wrap(gasLimit))),
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
    uint128 receiverValue,
    uint32 gasLimit,
    uint16 refundChainId,
    address refundAddress
  ) external payable {
    forwardToEvm(
      targetChainId,
      targetAddress,
      payload,
      receiverValue,
      0,
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
    uint128 receiverValue,
    uint128 paymentForExtraReceiverValue,
    uint32 gasLimit,
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
      encodeEvmExecutionParamsV1(EvmExecutionParamsV1(Gas.wrap(gasLimit))),
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
    uint128 newReceiverValue,
    uint32 newGasLimit,
    address newRelayProviderAddress
  ) public payable returns (uint64 sequence) {
    sequence = resend(
      deliveryVaaKey,
      targetChainId,
      newReceiverValue,
      encodeEvmExecutionParamsV1(EvmExecutionParamsV1(Gas.wrap(newGasLimit))),
      newRelayProviderAddress
    );
  }

  function send(
    uint16 targetChainId,
    bytes32 targetAddress,
    bytes memory payload,
    uint128 receiverValue,
    uint128 paymentForExtraReceiverValue,
    bytes memory encodedExecutionParameters,
    uint16 refundChainId,
    bytes32 refundAddress,
    address relayProviderAddress,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) public payable returns (uint64 sequence) {
    sequence = sendForwardResend(
      Action.Send,
      targetChainId,
      targetAddress,
      payload,
      Wei.wrap(receiverValue),
      Wei.wrap(paymentForExtraReceiverValue),
      refundChainId,
      refundAddress, 
      encodedExecutionParameters,
      relayProviderAddress,
      vaaKeys,
      consistencyLevel
    );
  }

  function forward(
    uint16 targetChainId,
    bytes32 targetAddress,
    bytes memory payload,
    uint128 receiverValue,
    uint128 paymentForExtraReceiverValue,
    bytes memory encodedExecutionParameters,
    uint16 refundChainId,
    bytes32 refundAddress,
    address relayProviderAddress,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) public payable {
    sendForwardResend(
      Action.Forward,
      targetChainId,
      targetAddress,
      payload,
      Wei.wrap(receiverValue),
      Wei.wrap(paymentForExtraReceiverValue),
      refundChainId,
      refundAddress, 
      encodedExecutionParameters,
      relayProviderAddress,
      vaaKeys,
      consistencyLevel
    );
  }

  function resend(
    VaaKey memory deliveryVaaKey,
    uint16 targetChainId,
    uint128 newReceiverValue,
    bytes memory newEncodedExecutionParameters,
    address newRelayProviderAddress
  ) public payable returns (uint64 sequence) {
    VaaKey[] memory deliveryVaaKeyArray = new VaaKey[](1);
    deliveryVaaKeyArray[0] = deliveryVaaKey;
    sequence = sendForwardResend(
      Action.Resend,
      targetChainId,
      bytes32(0),
      bytes(""),
      Wei.wrap(newReceiverValue),
      Wei.wrap(0),
      0,
      bytes32(0), 
      newEncodedExecutionParameters,
      newRelayProviderAddress,
      deliveryVaaKeyArray,
      CONSISTENCY_LEVEL_INSTANT
    );
  }

  
  /* 
   * Non overload logic 
   */ 

  enum Action {Send, Forward, Resend}

  function sendForwardResend(
    Action action,
    uint16 targetChainId,
    bytes32 targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Wei paymentForExtraReceiverValue,
    uint16 refundChainId,
    bytes32 refundAddress,
    bytes memory encodedExecutionParameters,
    address relayProviderAddress,
    VaaKey[] memory vaaKeys, // is an array of length 1 for resends
    uint8 consistencyLevel
  ) internal returns (uint64 sequence) {
    IRelayProvider provider = IRelayProvider(relayProviderAddress);
     if(!provider.isChainSupported(targetChainId)) {
      revert RelayProviderDoesNotSupportTargetChain(address(provider), targetChainId);
    }
    (Wei deliveryPrice, bytes memory encodedQuoteParameters) = provider.quoteDeliveryPrice(targetChainId, receiverValue, encodedExecutionParameters);
    Wei wormholeMessageFee = getWormholeMessageFee();
    if(msgValue() != deliveryPrice + paymentForExtraReceiverValue + wormholeMessageFee) {
      revert InvalidMsgValue(msg.value, (deliveryPrice + paymentForExtraReceiverValue + wormholeMessageFee).unwrap());
    }
    bytes memory encodedInstruction;
    if(action == Action.Send || action == Action.Forward) {
      DeliveryInstruction memory instruction = DeliveryInstruction({
        targetChainId: targetChainId,
        targetAddress: targetAddress,
        payload: payload,
        requestedReceiverValue: receiverValue,
        extraReceiverValue: provider.quoteAssetConversion(targetChainId, paymentForExtraReceiverValue),
        encodedQuoteParameters: encodedQuoteParameters,
        encodedExecutionParameters: encodedExecutionParameters,
        refundChainId: refundChainId,
        refundAddress: refundAddress,
        refundRelayProvider: provider.getTargetChainAddress(targetChainId),
        sourceRelayProvider: toWormholeFormat(relayProviderAddress),
        senderAddress: toWormholeFormat(msg.sender),
        vaaKeys: vaaKeys
      });
      encodedInstruction = instruction.encode();
    } else if(action == Action.Resend) {
      RedeliveryInstruction memory instruction = RedeliveryInstruction({
        deliveryVaaKey: vaaKeys[0],
        targetChainId: targetChainId,
        newRequestedReceiverValue: receiverValue,
        newEncodedQuoteParameters: encodedQuoteParameters,
        newEncodedExecutionParameters: encodedExecutionParameters,
        newSourceRelayProvider: toWormholeFormat(relayProviderAddress),
        newSenderAddress: toWormholeFormat(msg.sender)
      });
      encodedInstruction = instruction.encode();
    }

    if(action == Action.Send || action == Action.Resend) {
      sequence = publishAndPay(wormholeMessageFee, deliveryPrice, paymentForExtraReceiverValue, encodedInstruction, consistencyLevel, provider);
    } else if(action == Action.Forward) {
      appendForwardInstruction(ForwardInstruction({
        encodedInstruction: encodedInstruction,
        msgValue: Wei.wrap(msg.value),
        deliveryPrice: deliveryPrice,
        paymentForExtraReceiverValue: paymentForExtraReceiverValue,
        consistencyLevel: consistencyLevel
      }));
    }

  }

  

 
  function getDefaultRelayProvider() public view returns (address relayProvider) {
    relayProvider = getDefaultRelayProviderState().defaultRelayProvider;
  }

  function quoteEVMDeliveryPrice(uint16 targetChainId, Wei receiverValue, Gas gasLimit, address relayProviderAddress) public view returns (Wei nativePriceQuote, Wei targetChainRefundPerGasUnused) {
    (Wei quote, bytes memory encodedQuoteParams) = quoteDeliveryPrice(targetChainId, receiverValue, encodeEvmExecutionParamsV1(EvmExecutionParamsV1(gasLimit)), relayProviderAddress);
    nativePriceQuote = quote;
    targetChainRefundPerGasUnused = decodeEvmQuoteParamsV1(encodedQuoteParams).targetChainRefundPerGasUnused;
  }

  function quoteEVMDeliveryPrice(uint16 targetChainId, Wei receiverValue, Gas gasLimit) public view returns (Wei nativePriceQuote, Wei refundAmountPerUnitGasUnused) {
    return quoteEVMDeliveryPrice(targetChainId, receiverValue, gasLimit, getDefaultRelayProvider());
  }

  function quoteDeliveryPrice(uint16 targetChainId, Wei receiverValue, bytes memory encodedExecutionParameters, address relayProviderAddress) public view returns (Wei nativePriceQuote, bytes memory encodedQuoteParams) {
    IRelayProvider provider = IRelayProvider(relayProviderAddress);
    return provider.quoteDeliveryPrice(targetChainId, receiverValue, encodedExecutionParameters);
  }

  function quoteAssetConversion(
    uint16 targetChainId,
    Wei currentChainAmount,
    address relayProviderAddress
  ) public view returns (Wei targetChainAmount) {
    IRelayProvider provider = IRelayProvider(relayProviderAddress);
    return provider.quoteAssetConversion(targetChainId, currentChainAmount);
  }


}


  // ------------------------------------------- PRIVATE -------------------------------------------

