// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
  RelayProviderDoesNotSupportTargetChain,
  InsufficientMaxTransactionFee,
  InvalidMsgValue,
  ExceedsMaximumBudget,
  VaaKey,
  ExecutionParameters,
  Send,
  DeliveryInstruction,
  RedeliveryInstruction,
  IWormholeRelayerSend
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";

import {toWormholeFormat} from "./Utils.sol";
import {CoreRelayerSerde} from "./CoreRelayerSerde.sol";
import {ForwardInstruction, getDefaultRelayProviderState} from "./CoreRelayerStorage.sol";
import {CoreRelayerBase} from "./CoreRelayerBase.sol";

//TODO:
// Introduce basic sanity checks on sendParams (e.g. all valus below 2^128?) so we can get rid of
//   all the silly checked math and ensure that we can't have overflow Panics either.
// In send() and resend() we already check that maxTransactionFee + receiverValue == msg.value (via
//   calcAndCheckFees(). We could perhaps introduce a similar check of <= this.balance in forward()
//   and presumably a few more in our calculation/conversion functions CoreRelayerBase to ensure
//   sensible numeric ranges everywhere.

abstract contract CoreRelayerSend is CoreRelayerBase, IWormholeRelayerSend {
  using CoreRelayerSerde for *; //somewhat yucky but unclear what's a better alternative

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
  ) external payable returns (uint64 sequence) {
    sequence = send(Send(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      maxTransactionFee,
      receiverValue,
      getDefaultRelayProvider(),
      vaaKeys,
      consistencyLevel,
      payload,
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
    bytes memory payload
  ) external payable returns (uint64 sequence) {
    sequence = send(Send(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      maxTransactionFee,
      receiverValue,
      getDefaultRelayProvider(),
      new VaaKey[](0),
      CONSISTENCY_LEVEL_FINALIZED,
      payload,
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
    uint8 consistencyLevel
  ) external payable {
    forward(Send(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      maxTransactionFee,
      receiverValue,
      getDefaultRelayProvider(),
      vaaKeys,
      consistencyLevel,
      payload,
      getDefaultRelayParams()
    ));
  }

  function send(Send memory sendParams) public payable returns (uint64 sequence) {
    uint256 wormholeMessageFee =
      calcAndCheckFees(sendParams.maxTransactionFee, sendParams.receiverValue);

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

  function forward(Send memory sendParams) public payable {
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
        msgValue: msg.value,
        totalFee:
          sendParams.maxTransactionFee + sendParams.receiverValue + getWormhole().messageFee()
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
    uint256 wormholeMessageFee = calcAndCheckFees(newMaxTransactionFee, newReceiverValue);

    IRelayProvider relayProvider = IRelayProvider(relayProviderAddress);

    (uint256 maximumRefundTarget, uint256 receiverValueTarget, uint32 gasLimit) =
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
    uint32 gasLimit,
    address relayProvider
  ) public view returns (uint256 maxTransactionFee) {
    IRelayProvider provider = IRelayProvider(relayProvider);

    //maxTransactionFee is a linear function of the amount of gas desired
    maxTransactionFee = provider.quoteDeliveryOverhead(targetChainId)
      + (gasLimit * provider.quoteGasPrice(targetChainId));
  }

  function quoteReceiverValue(
    uint16 targetChainId,
    uint256 targetAmount,
    address relayProvider
  ) public view returns (uint256 receiverValue) {
    IRelayProvider provider = IRelayProvider(relayProvider);
    if (!provider.isChainSupported(targetChainId))
      revert RelayProviderDoesNotSupportTargetChain(address(provider), targetChainId);

    (uint256 sourcePrice, uint256 targetPrice) =
      getAssetPricesWithBuffer(getChainId(), targetChainId, provider);

    //we have to round up her since we are going from target to source and we are truncating (i.e.
    // rounding down) when going the other direction
    receiverValue = convertAmount(targetAmount, targetPrice, sourcePrice, true);
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
    uint256 maxTransactionFee,
    uint256 receiverValue
  ) private view returns (uint256 wormholeMessageFee) {
    wormholeMessageFee = getWormhole().messageFee();
    uint256 totalFee = maxTransactionFee + receiverValue + wormholeMessageFee;
    if (msg.value != totalFee)
      revert InvalidMsgValue(msg.value, totalFee);
  }

  //Check that the total amount of value the relay provider needs to use for this send is <= the
  //  relayProvider's maximum budget for `targetChainId` and check that the calculated gas is > 0
  function calcParamsAndCheckBudgetConstraints(
    uint16 targetChainId,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    IRelayProvider relayProvider
  ) private view returns (
    uint256 maximumRefundTarget,
    uint256 receiverValueTarget,
    uint32 gasLimit
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
    uint256 maximumRefundTarget,
    uint256 receiverValueTarget,
    uint32 gasLimit,
    IRelayProvider relayProvider
  ) private view {
    if (gasLimit == 0)
      revert InsufficientMaxTransactionFee();

    uint256 maxBudget = relayProvider.quoteMaximumBudget(targetChainId);
    uint256 requestedBudget = maximumRefundTarget + receiverValueTarget;
    if (requestedBudget > maxBudget)
      revert ExceedsMaximumBudget(
        requestedBudget, maxBudget, address(relayProvider), targetChainId
      );
  }
}
