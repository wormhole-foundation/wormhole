// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
  RelayProviderDoesNotSupportTargetChain,
  InsufficientMaxTransactionFee,
  InvalidMsgValue,
  ExceedsMaximumBudget,
  ReceiverValueGreaterThanUint128,
  MaxTransactionFeeGreaterThanUint128,
  VaaKey,
  ExecutionParameters,
  DeliveryInstruction,
  RedeliveryInstruction,
  MaxTransactionFeeGreaterThanUint128,
  IWormholeRelayerSend
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";

import {toWormholeFormat} from "./Utils.sol";
import {CoreRelayerSerde, Send} from "./CoreRelayerSerde.sol";
import {ForwardInstruction, getDefaultRelayProviderState} from "./CoreRelayerStorage.sol";
import {CoreRelayerBase} from "./CoreRelayerBase.sol";
import "../../interfaces/relayer/TypedUnits.sol";

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
    uint32 gasLimit,
    uint128 receiverValue
  ) external payable returns (uint64 sequence) {
    return sendToEvm(
      targetChainId,
      targetAddress,
      payload,
      chainId(),
      msg.sender,
      gasLimit,
      receiverValue,
      0, 
      0, 
      getDefaultRelayProvider(),
      new VaaKey[](0),
      CONSISTENCY_LEVEL_FINALIZED
    );
  }

  function sendToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    uint16 refundChainId,
    address refundAddress,
    uint32 gasLimit,
    uint128 receiverValue
  ) external payable returns (uint64 sequence) {
    return sendToEvm(
      targetChainId,
      targetAddress,
      payload,
      refundChainId,
      refundAddress,
      gasLimit,
      receiverValue,
      0, 
      0, 
      getDefaultRelayProvider(),
      new VaaKey[](0),
      CONSISTENCY_LEVEL_FINALIZED
    );
  }


  function sendToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    uint16 refundChainId,
    address refundAddress,
    uint32 gasLimit,
    uint128 receiverValue,
    uint128 targetChainRefundPerUnitGasUnused,
    uint128 amountToSpendOnForwardPerUnitGasUnused,
    uint128 paymentForExtraReceiverValue,
    address relayProviderAddress,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) public payable returns (uint64 sequence) {
    IRelayProvider provider = IRelayProvider(relayProviderAddress);
    checkTargetChainSupported(provider, targetChainId);
    Wei deliveryPrice = provider.quoteEVMDeliveryPrice(targetChainId, gasLimit, receiverValue, amountToSpendOnForwardPerUnitGasUnused);
    bytes memory encodedExecutionParameters = encodeEVMExecutionParameters(EVMExecutionParameters(gasLimit, refundChainId, refundAddress, targetChainRefundPerUnitGasUnused, amountToSpendOnForwardPerUnitGasUnused, refundRelayProvider));
    sequence = send(deliveryPrice, targetChainId, toWormholeFormat(targetAddress), payload, receiverValue, paymentForExtraReceiverValue, provider, vaaKeys, consistencyLevel, ExecutionEnvironment.EVM, encodedExecutionParameters);
  }

  function send(
    Wei deliveryPrice,
    uint16 targetChainId,
    bytes32 targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Wei paymentForExtraReceiverValue,
    IRelayProvider provider,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel,
    uint8 executionEnvironment,
    bytes memory encodedExecutionParameters
  ) internal returns (uint64 sequence) {
    Wei wormholeMessageFee = wormholeMessageFee();
    if(msgValue() != deliveryPrice + paymentForExtraReceiverValue + wormholeMessageFee) {
      revert InvalidMsgValue(msg.value, deliveryPrice);
    }
    DeliveryInstruction memory instruction = DeliveryInstruction({
      targetChainId: targetChainId,
      targetAddress: targetAddress,
      sourceRelayProvider: toWormholeFormat(address(provider)),
      senderAddress: toWormholeFormat(msg.sender),
      sourcePayment: deliveryPrice + paymentForExtraReceiverValue,
      paymentForExtraReceiverValue: paymentForExtraReceiverValue,
      vaaKeys: vaaKeys,
      receiverValue: receiverValue,
      executionEnvironment: executionEnvironment,
      encodedExecutionParameters: encodedExecutionParameters,
      payload: payload
    });
    sequence = publishAndPay(wormholeMessageFee, paymentForExtraReceiverValue + deliveryPrice, instruction.encode(), consistencyLevel, provider);
  }


  function forwardToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    uint16 refundChainId,
    address refundAddress,
    uint32 gasLimit,
    uint128 receiverValue
  ) external payable {
    forwardToEvm(
      targetChainId,
      targetAddress,
      payload,
      refundChainId,
      refundAddress,
      gasLimit,
      receiverValue,
      0,
      0,
      0,
      new VaaKey[](0),
      CONSISTENCY_LEVEL_FINALIZED
    );
  }

  function forwardToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    uint16 refundChainId,
    address refundAddress,
    uint32 gasLimit,
    uint128 receiverValue,
    uint128 targetChainRefundPerUnitGasUnused,
    uint128 amountToSpendOnForwardPerUnitGasUnused,
    uint128 paymentForExtraReceiverValue,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) public payable {
    forward(constructSend(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      maxTransactionFee,
      receiverValue,
      payload,
      vaaKeys,
      consistencyLevel,
      relayProviderAddress,
      relayParameters
    ));
  }

  function forward(
    Wei deliveryPrice,
    uint16 targetChainId,
    bytes32 targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Wei paymentForExtraReceiverValue,
    IRelayProvider provider,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel,
    uint8 executionEnvironment,
    bytes memory encodedExecutionParameters
  ) internal returns (uint64 sequence) {
    checkMsgSenderInDelivery();

    DeliveryInstruction memory instruction = DeliveryInstruction({
      targetChainId: targetChainId,
      targetAddress: targetAddress,
      sourceRelayProvider: toWormholeFormat(address(provider)),
      senderAddress: toWormholeFormat(msg.sender),
      sourcePayment: deliveryPrice + paymentForExtraReceiverValue,
      paymentForExtraReceiverValue: paymentForExtraReceiverValue,
      vaaKeys: vaaKeys,
      receiverValue: receiverValue,
      executionEnvironment: executionEnvironment,
      encodedExecutionParameters: encodedExecutionParameters,
      payload: payload
    });

    appendForwardInstruction({
      encodedInstruction: instruction.encode(),
      msgValue: Wei.wrap(msg.value),
      totalFee: instruction.sourcePayment + getWormholeMessageFee()
    });
    
  }
  /* 
   * Non overload logic 
   */ 


  function forward(Send memory sendParams) internal {
    checkMsgSenderInDelivery();

    calcParamsAndCheckBudgetConstraints(
      sendParams.targetChainId,
      sendParams.maxTransactionFee,
      sendParams.receiverValue,
      IRelayProvider(sendParams.relayProviderAddress)
    );

    //Temporarily save information about the forward in state, so it can be processed after the
    //  execution of `receiveWormholeMessages`, because we will then know how much of the
    //  `maxTransactionFee` of the current delivery is still available for use in this forward.
    appendForwardInstruction(
      ForwardInstruction({
        encodedSend: sendParams.encode(),
        msgValue: Wei.wrap(msg.value),
        totalFee:
          sendParams.maxTransactionFee + sendParams.receiverValue + getWormholeMessageFee()
      })
    );

    //after this function, this.balance is increased by msg.value
  }

  function resend(
    VaaKey memory key,
    uint256 newMaxTransactionFee,
    uint256 newReceiverValue,
    uint16 targetChainId,
    address relayProviderAddress
  ) external payable returns (uint64 sequence) {
    return resendInternal(
      key,
      Wei.wrap(newMaxTransactionFee),
      Wei.wrap(newReceiverValue),
      targetChainId,
      relayProviderAddress
    );
  }

  function resendInternal(
    VaaKey memory key,
    Wei newMaxTransactionFee,
    Wei newReceiverValue,
    uint16 targetChainId,
    address relayProviderAddress
  ) internal returns (uint64 sequence) {
    Wei wormholeMessageFee = getWormholeMessageFee();
    calcAndCheckFees(newMaxTransactionFee, newReceiverValue, wormholeMessageFee);

    IRelayProvider relayProvider = IRelayProvider(relayProviderAddress);

    (Wei maximumRefundTarget, Wei receiverValueTarget, Gas gasLimit) =
      calcParamsAndCheckBudgetConstraints(
        targetChainId, newMaxTransactionFee, newReceiverValue, relayProvider
      );

    RedeliveryInstruction memory instruction = RedeliveryInstruction({
      key: key,
      newMaximumRefundTarget: maximumRefundTarget,
      newReceiverValueTarget: receiverValueTarget,
      sourceRelayProvider: toWormholeFormat(relayProviderAddress),
      targetChainId: targetChainId,
      executionParameters: ExecutionParameters({gasLimit: gasLimit})
    });

    sequence = publishAndPay(
      wormholeMessageFee,
      newMaxTransactionFee,
      newReceiverValue,
      instruction.encode(),
      CONSISTENCY_LEVEL_INSTANT,
      relayProvider
    );
  }



 
  function getDefaultRelayProvider() public view returns (address relayProvider) {
    relayProvider = getDefaultRelayProviderState().defaultRelayProvider;
  }


  // ------------------------------------------- PRIVATE -------------------------------------------

