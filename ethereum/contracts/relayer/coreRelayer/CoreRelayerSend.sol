// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
  InsufficientMaxTransactionFee,
  InvalidMsgValue,
  ExceedsMaximumBudget,
  RelayProviderDoesNotSupportTargetChain,
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

  function forward(Send memory sendParams) public payable {
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
    uint256 wormholeMessageFee =
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
      + (gasLimit * provider.quoteGasPrice(targetChainId));
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

  //TODO AMO: solc suggested changing view to pure - is this fine given that IWormholeRelayerSend
  //            has declared it as view?
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
