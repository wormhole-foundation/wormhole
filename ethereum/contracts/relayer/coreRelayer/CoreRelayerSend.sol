// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
  InsufficientMaxTransactionFee,
  InvalidMsgValue,
  ExceedsMaximumBudget,
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

abstract contract CoreRelayerSend is CoreRelayerBase, IWormholeRelayerSend {
  using CoreRelayerSerde for *; //somewhat yucky but unclear what's a better alternative

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

  function send(Send memory sendParams) internal returns (uint64 sequence) {
    uint256 wormholeMessageFee =
      calcAndCheckFees(sendParams.maxTransactionFee, sendParams.receiverValue);

    IRelayProvider relayProvider = IRelayProvider(sendParams.relayProviderAddress);
    checkRelayProviderSupportsChain(relayProvider, sendParams.targetChainId);

    DeliveryInstruction memory instruction = convertSendToDeliveryInstruction(sendParams);

    checkBudgetConstraints(
      instruction.maximumRefundTarget,
      instruction.receiverValueTarget,
      instruction.executionParameters.gasLimit,
      relayProvider,
      instruction.targetChainId
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

    //TODO AMO: Introduce basic sanity checks on sendParams (e.g. all valus below 2^128?)
    //          In send() we check that maxTransactionFee + receiverValue < msg.value so we
    //            there we are safe already.
    //          One very easy way to achieve this is by enforcing a max on
    //            relayProvider.quoteMaximumBudget() since that is enforced as an upper limit.

    IRelayProvider relayProvider = IRelayProvider(sendParams.relayProviderAddress);
    checkRelayProviderSupportsChain(relayProvider, sendParams.targetChainId);

    checkBudgetConstraints(
      calculateTargetDeliveryMaximumRefund(
        sendParams.targetChainId, sendParams.maxTransactionFee, relayProvider
      ),
      convertReceiverValueAmountToTarget(
        sendParams.receiverValue, sendParams.targetChainId, relayProvider
      ),
      calculateTargetGasDeliveryAmount(
        sendParams.targetChainId, sendParams.maxTransactionFee, relayProvider
      ),
      relayProvider,
      sendParams.targetChainId
    );

    //Temporarily save information about the forward in state, so it can be processed after the
    //  execution of 'receiveWormholeMessages', because we will then know how much of the
    //  'maxTransactionFee' of the current delivery is still available for use in this forward.
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
    (uint128 newMaxTransactionFee_, uint128 newReceiverValue_) = 
      checkFeesLessThanU128(newMaxTransactionFee, newReceiverValue);
    return resend(
      key, 
      newMaxTransactionFee_,
      newReceiverValue_,
      targetChainId,
      relayProviderAddress
    );
  }

  function resend(
    VaaKey memory key,
    uint128 newMaxTransactionFee,
    uint128 newReceiverValue,
    uint16 targetChainId,
    address relayProviderAddress
  ) public payable returns (uint64 sequence) {
    uint128 wormholeMessageFee =
      calcAndCheckFees(newMaxTransactionFee, newReceiverValue);

    IRelayProvider relayProvider = IRelayProvider(relayProviderAddress);
    checkRelayProviderSupportsChain(relayProvider, targetChainId);

    RedeliveryInstruction memory instruction = RedeliveryInstruction({
      key: key,
      newMaximumRefundTarget: calculateTargetDeliveryMaximumRefund(
        targetChainId, newMaxTransactionFee, relayProvider
      ),
      newReceiverValueTarget: convertReceiverValueAmountToTarget(
        newReceiverValue, targetChainId, relayProvider
      ),
      sourceRelayProvider: toWormholeFormat(relayProviderAddress),
      targetChainId: targetChainId,
      executionParameters: ExecutionParameters({
        gasLimit: calculateTargetGasDeliveryAmount(
          targetChainId, newMaxTransactionFee, relayProvider
        )
      })
    });

    checkBudgetConstraints(
      instruction.newMaximumRefundTarget,
      instruction.newReceiverValueTarget,
      instruction.executionParameters.gasLimit,
      relayProvider,
      targetChainId
    );

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
      + (uint128(gasLimit) * provider.quoteGasPrice(targetChainId));
    if (maxTransactionFee > type(uint128).max)
      revert MaxTransactionFeeGreaterThanUint128();
  }

  function quoteReceiverValue(
    uint16 targetChainId,
    uint256 targetAmount,
    address relayProvider
  ) public view returns (uint256 receiverValue) {
    IRelayProvider provider = IRelayProvider(relayProvider);

    //Converts 'targetAmount' from target chain currency to source chain currency (using
    //  relayProvider's prices) and applies a multiplier of '1 + (buffer / denominator)'
    (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChainId);
    uint256 numerator = uint256(denominator) + buffer;
    receiverValue = assetConversionHelper(
      targetChainId, targetAmount, getChainId(), numerator, denominator, true, provider
    );
  }

  function getDefaultRelayProvider() public view returns (address relayProvider) {
    relayProvider = getDefaultRelayProviderState().defaultRelayProvider;
  }

  function getDefaultRelayParams() public view returns (bytes memory relayParams) {
    return new bytes(0);
  }

  // ------------------------------------------- PRIVATE -------------------------------------------

  
  function calcAndCheckFees(
    uint128 maxTransactionFee,
    uint128 receiverValue
  ) private view returns (uint128 wormholeMessageFee) {
    wormholeMessageFee = uint128(getWormhole().messageFee());
    uint256 totalFee = uint256(maxTransactionFee) + receiverValue + wormholeMessageFee;
    if (msg.value != totalFee)
      revert InvalidMsgValue(msg.value, totalFee);
  }

  //Check that the total amount of value the relay provider needs to use for this send is <= the
  //  relayProvider's maximum budget for 'targetChainId' and check that the calculated gas is > 0
  function checkBudgetConstraints(
    uint256 maximumRefundTarget,
    uint256 receiverValueTarget,
    uint32 gasLimit,
    IRelayProvider relayProvider,
    uint16 targetChainId
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
