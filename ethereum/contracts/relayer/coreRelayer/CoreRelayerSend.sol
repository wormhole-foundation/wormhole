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
    uint16 refundChainId,
    address refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload
  ) external payable returns (uint64 sequence) {
    return send(
      targetChainId,
      toWormholeFormat(targetAddress),
      refundChainId,
      toWormholeFormat(refundAddress),
      maxTransactionFee,
      receiverValue,
      payload
    );
  }

  function sendToEvm(
    uint16 targetChainId,
    address targetAddress,
    uint16 refundChainId,
    address refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) external payable returns (uint64 sequence) {
    return send(
      targetChainId,
      toWormholeFormat(targetAddress),
      refundChainId,
      toWormholeFormat(refundAddress),
      maxTransactionFee,
      receiverValue,
      payload,
      vaaKeys,
      consistencyLevel
    );
  }


  function send(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload
  ) public payable returns (uint64 sequence) {
    sequence = send(constructSend(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      maxTransactionFee,
      receiverValue,
      payload,
      new VaaKey[](0),
      CONSISTENCY_LEVEL_FINALIZED,
      getDefaultRelayProvider(),
      getDefaultRelayParams()
    ));
  }

  function send(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) public payable returns (uint64 sequence) {
    sequence = send(constructSend(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      maxTransactionFee,
      receiverValue,
      payload,
      vaaKeys,
      consistencyLevel,
      getDefaultRelayProvider(),
      getDefaultRelayParams()
    ));
  }

  function send(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel,
    address relayProviderAddress,
    bytes memory relayParameters
  ) public payable returns (uint64 sequence) {
    sequence = send(constructSend(
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


  function forwardToEvm(
    uint16 targetChainId,
    address targetAddress,
    uint16 refundChainId,
    address refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) external payable {
    forward(constructSend(
      targetChainId,
      toWormholeFormat(targetAddress),
      refundChainId,
      toWormholeFormat(refundAddress),
      maxTransactionFee,
      receiverValue,
      payload,
      vaaKeys,
      consistencyLevel,
      getDefaultRelayProvider(),
      getDefaultRelayParams()
    ));
  }

  function forward(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel,
    address relayProviderAddress,
    bytes memory relayParameters
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
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) external payable {
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
      getDefaultRelayProvider(),
      getDefaultRelayParams()
    ));
  }

  /* 
   * Non overload logic 
   */ 

  function constructSend(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel,
    address relayProviderAddress,
    bytes memory relayParameters
  ) internal pure returns (Send memory) {
    if (maxTransactionFee > type(uint128).max )
      revert MaxTransactionFeeGreaterThanUint128();
    if ( receiverValue > type(uint128).max)
      revert ReceiverValueGreaterThanUint128();

    return Send(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      Wei.wrap(maxTransactionFee), 
      Wei.wrap(receiverValue),
      payload,
      vaaKeys,
      consistencyLevel,
      relayProviderAddress,
      relayParameters
    );
  }

  function send(Send memory sendParams) internal returns (uint64 sequence) {
    Wei wormholeMessageFee = getWormholeMessageFee();
    calcAndCheckFees(sendParams.maxTransactionFee, sendParams.receiverValue, wormholeMessageFee);

    (DeliveryInstruction memory instruction, IRelayProvider relayProvider) =
      convertSendToDeliveryInstruction(sendParams);

    checkBudgetConstraints(
      instruction.targetChainId,
      instruction.maximumRefundTarget,
      instruction.receiverValueTarget,
      instruction.executionParameters.gasLimit,
      relayProvider
    );

    sequence = publishAndPay(
      wormholeMessageFee,
      sendParams.maxTransactionFee,
      sendParams.receiverValue,
      instruction.encode(),
      sendParams.consistencyLevel,
      relayProvider
    );
  }

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

  function quoteGas(
    uint16 targetChainId,
    uint32 _gasLimit,
    address relayProvider
  ) public view returns (uint256 maxTransactionFee) {
    IRelayProvider provider = IRelayProvider(relayProvider);
    Gas gasLimit = Gas.wrap(_gasLimit);

    Wei overhead = provider.quoteDeliveryOverhead(targetChainId);
    Wei weiForGas = gasLimit.toWei(provider.quoteGasPrice(targetChainId));

    maxTransactionFee = (overhead + weiForGas).unwrap();

    //maxTransactionFee is a linear function of the amount of gas desired
    if (maxTransactionFee > type(uint128).max)
      revert MaxTransactionFeeGreaterThanUint128();
  }

  function quoteReceiverValue(
    uint16 targetChainId,
    uint256 targetAmount,
    address relayProvider
  ) public view returns (uint256 receiverValue) {
    IRelayProvider provider = IRelayProvider(relayProvider);
    if (!provider.isChainSupported(targetChainId))
      revert RelayProviderDoesNotSupportTargetChain(address(provider), targetChainId);

    (WeiPrice sourcePrice, WeiPrice targetPrice) =
      getAssetPricesWithBuffer(getChainId(), targetChainId, provider);

    //we have to round up her since we are going from target to source and we are truncating (i.e.
    // rounding down) when going the other direction
    receiverValue = Wei.unwrap(convertAmount(Wei.wrap(targetAmount), targetPrice, sourcePrice, true));
  }

  function getDefaultRelayProvider() public view returns (address relayProvider) {
    relayProvider = getDefaultRelayProviderState().defaultRelayProvider;
  }

  //this function is `view` in the interface but `pure` here, for now
  function getDefaultRelayParams() public pure returns (bytes memory relayParams) {
    return new bytes(0);
  }

  // ------------------------------------------- PRIVATE -------------------------------------------

  
  function calcAndCheckFees(
    Wei maxTransactionFee,
    Wei receiverValue,
    Wei wormholeMsgFee
  ) private view {
    Wei totalFee = maxTransactionFee + receiverValue + wormholeMsgFee;
    if (msgValue() != totalFee)
      revert InvalidMsgValue(msg.value, totalFee.unwrap());
  }

  //Check that the total amount of value the relay provider needs to use for this send is <= the
  //  relayProvider's maximum budget for `targetChainId` and check that the calculated gas is > 0
  function calcParamsAndCheckBudgetConstraints(
    uint16 targetChainId,
    Wei maxTransactionFee,
    Wei receiverValue,
    IRelayProvider relayProvider
  ) private view returns (
    Wei maximumRefundTarget,
    Wei receiverValueTarget,
    Gas gasLimit
  ) {
    (maximumRefundTarget, receiverValueTarget, gasLimit) =
      calculateTargetParams(targetChainId, maxTransactionFee, receiverValue, relayProvider);

    checkBudgetConstraints(
      targetChainId, maximumRefundTarget, receiverValueTarget, gasLimit, relayProvider
    );
  }

  //Check that the total amount of value the relay provider needs to use for this send is <= the
  //  relayProvider's maximum budget for 'targetChainId' and check that the calculated gas is > 0
  function checkBudgetConstraints(
    uint16 targetChainId,
    Wei maximumRefundTarget,
    Wei receiverValueTarget,
    Gas gasLimit,
    IRelayProvider relayProvider
  ) private view {
    if (Gas.unwrap(gasLimit) == 0)
      revert InsufficientMaxTransactionFee();

    Wei maxBudget = relayProvider.quoteMaximumBudget(targetChainId);
    Wei requestedBudget = maximumRefundTarget + receiverValueTarget;
    if (requestedBudget > maxBudget)
      revert ExceedsMaximumBudget(
        Wei.unwrap(requestedBudget), Wei.unwrap(maxBudget), address(relayProvider), targetChainId
      );
  }
}
