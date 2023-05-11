// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {IWormhole} from "../../interfaces/IWormhole.sol";
import {
  RETURNDATA_TRUNCATION_THRESHOLD,
  InvalidDeliveryVaa,
  InvalidEmitter,
  InsufficientRelayerFunds,
  TargetChainIsNotThisChain,
  ForwardNotSufficientlyFunded,
  VaaKeysLengthDoesNotMatchVaasLength,
  VaaKeysDoNotMatchVaas,
  InvalidOverrideGasLimit,
  InvalidOverrideReceiverValue,
  InvalidOverrideMaximumRefund,
  RequesterNotCoreRelayer,
  VaaKey,
  VaaKeyType,
  Send,
  TargetDeliveryParameters,
  DeliveryInstruction,
  DeliveryOverride,
  IWormholeRelayerDelivery
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {DeliveryData, IWormholeReceiver} from "../../interfaces/relayer/IWormholeReceiver.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";

import {pay, min, fromWormholeFormat} from "./Utils.sol";
import {BytesParsing} from "./BytesParsing.sol";
import {CoreRelayerSerde} from "./CoreRelayerSerde.sol";
import {ForwardInstruction} from "./CoreRelayerStorage.sol";
import {CoreRelayerBase} from "./CoreRelayerBase.sol";

abstract contract CoreRelayerDelivery is CoreRelayerBase, IWormholeRelayerDelivery {
  using CoreRelayerSerde for *; //somewhat yucky but unclear what's a better alternative
  using BytesParsing for bytes;

  function deliver(TargetDeliveryParameters memory targetParams) public payable {
    (IWormhole.VM memory vm, bool valid, string memory reason) =
      getWormhole().parseAndVerifyVM(targetParams.encodedDeliveryVAA);

    if (!valid)
      revert InvalidDeliveryVaa(reason);

    bytes32 registeredCoreRelayer = getRegisteredCoreRelayerContract(vm.emitterChainId);
    if (vm.emitterAddress != registeredCoreRelayer)
      revert InvalidEmitter(vm.emitterAddress, registeredCoreRelayer, vm.emitterChainId);

    DeliveryInstruction memory instruction = vm.payload.decodeDeliveryInstruction();

    //"lock" as soon as possible (we could also lock after all checks have completed), but locking
    //  early seems more defensive when it comes to additional code changes and does not change the
    //  cost of the happy path.
    startDelivery(fromWormholeFormat(instruction.targetAddress));

    // If present, apply redelivery overrides to current instruction
    bytes32 redeliveryHash = 0;
    if (targetParams.overrides.length != 0) {
      DeliveryOverride memory overrides = targetParams.overrides.decodeDeliveryOverride();

      if (overrides.gasLimit < instruction.executionParameters.gasLimit)
        revert InvalidOverrideGasLimit();
      
      if (overrides.receiverValue < instruction.receiverValueTarget)
        revert InvalidOverrideReceiverValue();
      
      if (overrides.maximumRefund < instruction.maximumRefundTarget)
        revert InvalidOverrideMaximumRefund();

      instruction.executionParameters.gasLimit = overrides.gasLimit;
      instruction.receiverValueTarget = overrides.receiverValue;
      instruction.maximumRefundTarget = overrides.maximumRefund;

      redeliveryHash = overrides.redeliveryHash;
    }

    uint256 requiredFunds = instruction.maximumRefundTarget + instruction.receiverValueTarget;
    if (msg.value < requiredFunds)
      revert InsufficientRelayerFunds(msg.value, requiredFunds);

    if (getChainId() != instruction.targetChainId)
      revert TargetChainIsNotThisChain(instruction.targetChainId);

    checkVaaKeysWithVAAs(instruction.vaaKeys, targetParams.encodedVMs);

    executeDelivery(
      DeliveryVAAInfo(
        vm.emitterChainId,
        vm.sequence,
        vm.hash,
        targetParams.relayerRefundAddress,
        targetParams.encodedVMs,
        instruction,
        redeliveryHash
      )
    );

    finishDelivery();
  }

  // ------------------------------------------- PRIVATE -------------------------------------------

  error Cancelled(uint32 gasUsed, uint256 available, uint256 required);

  struct DeliveryVAAInfo {
    uint16 sourceChainId;
    uint64 sourceSequence;
    bytes32 deliveryVaaHash;
    address payable relayerRefundAddress;
    bytes[] encodedVMs;
    DeliveryInstruction deliveryInstruction;
    bytes32 redeliveryHash; //optional (0 if not present)
  }

  /**
   * Performs the following actions:
   * - Calls the `receiveWormholeMessages` method on the contract
   *     `deliveryInstruction.targetAddress` (with the gas limit and value specified in
   *     deliveryInstruction, and `encodedVMs` as the input)
   *
   * - Calculates how much of `maxTransactionFee` is left
   * - If the call succeeded and during execution of `receiveWormholeMessages` there was a
   *     forward/multichainForward, then it executes the forward if there is enough
   *     `maxTransactionFee` left
   * - else:
   *     revert the delivery to trigger a forwarding failure
   *     refund any of the `maxTransactionFee` not used to deliveryInstruction.refundAddress
   *     if the call reverted, refund the `receiverValue` to deliveryInstruction.refundAddress
   * - refund anything leftover to the relayer
   *
   * @param vaaInfo struct specifying:
   *    - sourceChainId chain id that the delivery originated from
   *    - sourceSequence sequence number of the delivery VAA on the source chain
   *    - deliveryVaaHash hash of delivery VAA
   *    - relayerRefundAddress address that should be paid for relayer refunds
   *    - encodedVMs list of signed wormhole messages (VAAs)
   *    - deliveryInstruction the specific instruction which is being executed.
   *    - (optional) redeliveryHash hash of redelivery Vaa
   */
  function executeDelivery(DeliveryVAAInfo memory vaaInfo) private {
    if (vaaInfo.deliveryInstruction.targetAddress == 0x0) {
      payRefunds(
        vaaInfo.deliveryInstruction,
        vaaInfo.relayerRefundAddress,
        vaaInfo.deliveryInstruction.maximumRefundTarget,
        DeliveryStatus.RECEIVER_FAILURE
      );
      return;
    }

    uint32 gasUsed;
    DeliveryStatus status;
    bytes memory additionalStatusInfo;
    
    try //force external call!
      this.executeInstruction(
        vaaInfo.deliveryInstruction,
        DeliveryData({
          sourceAddress: vaaInfo.deliveryInstruction.senderAddress,
          sourceChainId: vaaInfo.sourceChainId,
          maximumRefund: vaaInfo.deliveryInstruction.maximumRefundTarget,
          deliveryHash:  vaaInfo.deliveryVaaHash,
          payload:       vaaInfo.deliveryInstruction.payload
        }),
        vaaInfo.encodedVMs
      )
    returns (uint8 _status, uint32 _gasUsed, bytes memory targetRevertDataTruncated) {
      gasUsed = _gasUsed;
      status = DeliveryStatus(_status);
      //will carry the correct value regardless of outcome (empty if successful, error otherwise)
      additionalStatusInfo = targetRevertDataTruncated; 
    }
    //executeInstruction should only revert with a Cancelled error though for now it can
    //  theoretically also revert with a Panic
    //  revert for any other reason (though it might for overflows atm)
    catch (bytes memory revertData) {
      //decode returned Cancelled error
      uint256 available;
      uint256 required;
      (gasUsed, available, required) = decodeCancelled(revertData);
      //Calculate the amount of maxTransactionFee to refund (multiply the maximum refund by the
      //  fraction of gas unused)
      status = DeliveryStatus.FORWARD_REQUEST_FAILURE;
      additionalStatusInfo =
        abi.encodeWithSelector(ForwardNotSufficientlyFunded.selector, available, required);
    }
    
    emit Delivery(
      fromWormholeFormat(vaaInfo.deliveryInstruction.targetAddress),
      vaaInfo.sourceChainId,
      vaaInfo.sourceSequence,
      vaaInfo.deliveryVaaHash,
      status,
      gasUsed,
      payRefunds(
        vaaInfo.deliveryInstruction,
        vaaInfo.relayerRefundAddress,
        calculateTransactionFeeRefundAmount(vaaInfo.deliveryInstruction, gasUsed),
        status
      ),
      additionalStatusInfo,
      (vaaInfo.redeliveryHash != 0)
        ? DeliveryOverride(
            vaaInfo.deliveryInstruction.executionParameters.gasLimit,
            vaaInfo.deliveryInstruction.maximumRefundTarget,
            vaaInfo.deliveryInstruction.receiverValueTarget,
            vaaInfo.redeliveryHash
          ).encode()
        : new bytes(0)
    );
  }

  function calculateTransactionFeeRefundAmount(
    DeliveryInstruction memory instruction,
    uint32 gasUsed
  ) private pure returns (uint256 transactionFeeRefundAmount) {
    unchecked {transactionFeeRefundAmount = instruction.executionParameters.gasLimit - gasUsed;}
    transactionFeeRefundAmount *= instruction.maximumRefundTarget;
    transactionFeeRefundAmount /= instruction.executionParameters.gasLimit;
  }

  function executeInstruction(
    DeliveryInstruction calldata instruction,
    DeliveryData calldata data,
    bytes[] memory signedVaas
  ) external returns (
    uint8 status,
    uint32 gasUsed,
    bytes memory targetRevertDataTruncated
  ) {
    //despite being external, we only allow ourselves to call this function (via CALL opcode)
    //  used as a means to retroactively revert the call to the delivery target if the forwards
    //  can't be funded
    if (msg.sender != address(this))
      revert RequesterNotCoreRelayer();

    uint256 preGas = gasleft();

    // Calls the `receiveWormholeMessages` endpoint on the contract `instruction.targetAddress`
    // (with the gas limit and value specified in instruction, and `encodedVMs` as the input)
    IWormholeReceiver deliveryTarget =
      IWormholeReceiver(fromWormholeFormat(instruction.targetAddress));
    try deliveryTarget.receiveWormholeMessages{
          gas:   instruction.executionParameters.gasLimit,
          value: instruction.receiverValueTarget
        } (data, signedVaas) {
      targetRevertDataTruncated = new bytes(0);
      status = uint8(DeliveryStatus.SUCCESS);
    }
    catch (bytes memory revertData) {
      if (revertData.length > RETURNDATA_TRUNCATION_THRESHOLD)
        (targetRevertDataTruncated,) =
          revertData.sliceUnchecked(0, RETURNDATA_TRUNCATION_THRESHOLD);
      else
        targetRevertDataTruncated = revertData;
      status = uint8(DeliveryStatus.RECEIVER_FAILURE);
    }

    uint256 postGas = gasleft();
    
    unchecked{gasUsed = uint32(min(preGas - postGas, instruction.executionParameters.gasLimit));}

    ForwardInstruction[] storage forwardInstructions = getForwardInstructions();
    if (forwardInstructions.length > 0) {
      //Calculate the amount of maxTransactionFee to refund (multiply the maximum refund by the
      //  fraction of gas unused)
      uint256 transactionFeeRefundAmount =
        calculateTransactionFeeRefundAmount(instruction, gasUsed);

      uint256 totalMsgValue = 0;
      uint256 totalFee = 0;
      for (uint i = 0; i < forwardInstructions.length;) {
        unchecked{totalMsgValue += forwardInstructions[i].msgValue;}
        totalFee += forwardInstructions[i].totalFee;
        unchecked{++i;}
      }

      //If we don't have enough funds to pay for the forward, then we retroactively revert the call
      //  to the delivery target too.
      //This does not revert our entire transaction because we invoked executeInstruction via CALL
      //  rather than through a normal, internal function call.
      uint256 feeForForward = transactionFeeRefundAmount + totalMsgValue;
      if (feeForForward < totalFee)
        revert Cancelled(gasUsed, feeForForward, totalFee);

      emitForward(transactionFeeRefundAmount, forwardInstructions);
      status = uint8(DeliveryStatus.FORWARD_REQUEST_SUCCESS);
    }
  }

  /**
   * - Checks if enough funds were passed into a forward
   * - Increases the maxTransactionFee of the first forward in order to use all of the funds
   * - Publishes the DeliveryInstruction
   * - Pays the relayer's reward address to deliver the forward
   *
   * @param transactionFeeRefundAmount amount of maxTransactionFee that was unused
   * @param forwardInstructions An array of structs containing information about the user's forward
   *     request(s)
   */
  function emitForward(
    uint256 transactionFeeRefundAmount,
    ForwardInstruction[] storage forwardInstructions
  ) private {
    uint256 wormholeMessageFee = getWormhole().messageFee();

    //Decode send requests and aggregate fee and payment
    Send[] memory sendRequests = new Send[](forwardInstructions.length);
    uint256 totalMsgValue = 0;
    uint256 totalFee = 0;
    for (uint i = 0; i < forwardInstructions.length;) {
      unchecked{totalMsgValue += forwardInstructions[i].msgValue;}
      totalFee += forwardInstructions[i].totalFee;
      sendRequests[i] = forwardInstructions[i].encodedSend.decodeSend();
      unchecked{++i;}
    }

    //Combine refund amount with any additional funds which were passed in to the forward as
    //  msg.value and check that enough funds were passed into the forward (should always be true
    //  as it was already checked)
    uint256 fundsForForward;
    unchecked{fundsForForward = transactionFeeRefundAmount + totalMsgValue;}
    if (fundsForForward < totalFee)
      revert ForwardNotSufficientlyFunded(fundsForForward, totalFee);

    //Increases the maxTransactionFee of the first forward in order to use all of the funds
    unchecked{
      sendRequests[0].maxTransactionFee += fundsForForward - totalFee;
    }

    DeliveryInstruction memory firstDeliveryInstruction =
      convertSendToDeliveryInstruction(sendRequests[0]);

    firstDeliveryInstruction.maximumRefundTarget = min(
      firstDeliveryInstruction.maximumRefundTarget,
      IRelayProvider(sendRequests[0].relayProviderAddress)
        .quoteMaximumBudget(sendRequests[0].targetChainId)
        - firstDeliveryInstruction.receiverValueTarget
    );

    //Publishes the DeliveryInstruction and pays the associated relayProvider
    for (uint i = 0; i < forwardInstructions.length;) {
      publishAndPay(
        wormholeMessageFee,
        sendRequests[i].maxTransactionFee,
        sendRequests[i].receiverValue,
        ( i == 0
          ? firstDeliveryInstruction
          : convertSendToDeliveryInstruction(sendRequests[i])
        ).encode(),
        sendRequests[i].consistencyLevel,
        IRelayProvider(sendRequests[i].relayProviderAddress)
      );
      unchecked{++i;}
    }
  }

  function payRefunds(
    DeliveryInstruction memory deliveryInstruction,
    address payable relayerRefundAddress,
    uint256 transactionFeeRefundAmount,
    DeliveryStatus status
  ) private returns (RefundStatus refundStatus) {
    //Amount of receiverValue that is refunded to the user (0 if the call to
    //  `receiveWormholeMessages` did not revert, or the full receiverValue otherwise)
    uint256 receiverValueRefundAmount =
     (status == DeliveryStatus.FORWARD_REQUEST_SUCCESS || status == DeliveryStatus.SUCCESS)
     ? 0
     : deliveryInstruction.receiverValueTarget;

    //Total refund to the user
    uint256 refundToRefundAddress = receiverValueRefundAmount
      + (status == DeliveryStatus.FORWARD_REQUEST_SUCCESS ? 0 : transactionFeeRefundAmount);
    
    //Refund the user
    try this.payRefundToRefundAddress(
      deliveryInstruction.refundChainId,
      deliveryInstruction.refundAddress,
      refundToRefundAddress,
      deliveryInstruction.targetRelayProvider
    )
    returns (RefundStatus _refundStatus) {
      refundStatus = _refundStatus;
    } 
    catch (bytes memory) {
      refundStatus = RefundStatus.CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED;
    }

    //Refund the relayer (their extra funds) + (the amount that the relayer spent on gas)
    //  + (the users refund if that refund didn't succeed)
    uint256 relayerRefundAmount = (
      msg.value - deliveryInstruction.receiverValueTarget - deliveryInstruction.maximumRefundTarget
    ) + (deliveryInstruction.maximumRefundTarget - transactionFeeRefundAmount)
    //TODO AMO: Isn't this a bug? We add the same amount regardless of whether we hit the max or not
      + ((refundStatus == RefundStatus.REFUND_SENT ||
          refundStatus == RefundStatus.CROSS_CHAIN_REFUND_SENT ||
          refundStatus == RefundStatus.CROSS_CHAIN_REFUND_SENT_MAXIMUM_BUDGET
         ) ? 0 : refundToRefundAddress);

    //TODO AMO: what if pay fails? (i.e. returns false)
    //Refund the relay provider
    pay(relayerRefundAddress, relayerRefundAmount);
  }

  function payRefundToRefundAddress(
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 refundAmount,
    bytes32 relayerAddress
  ) external returns (RefundStatus) {
    //despite being external, we only allow ourselves to call this function (via CALL opcode)
    //  used as a means to catch reverts when we external call the relay provider in this function
    if (msg.sender != address(this))
      revert RequesterNotCoreRelayer();

    //same chain refund
    if (refundChainId == getChainId())
      return pay(payable(fromWormholeFormat(refundAddress)), refundAmount)
        ? RefundStatus.REFUND_SENT
        : RefundStatus.REFUND_FAIL;
    
    //cross-chain refund
    IRelayProvider relayProvider = IRelayProvider(fromWormholeFormat(relayerAddress));
    uint256 wormholeMessageFee = getWormhole().messageFee();
    uint256 overhead = relayProvider.quoteDeliveryOverhead(refundChainId);
    if (refundAmount <= wormholeMessageFee + overhead)
      return RefundStatus.CROSS_CHAIN_REFUND_FAIL_NOT_ENOUGH;

    if (!relayProvider.isChainSupported(refundChainId))
      return RefundStatus.CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED;

    uint256 refundSubMessageFee;
    unchecked{refundSubMessageFee = refundAmount - wormholeMessageFee;}

    DeliveryInstruction memory crossChainRefundInstruction =
      convertSendToDeliveryInstruction(Send({
        targetChainId: refundChainId,
        targetAddress: bytes32(0x0),
        refundChainId: refundChainId,
        refundAddress: refundAddress,
        maxTransactionFee: overhead,
        receiverValue: refundSubMessageFee - overhead,
        relayProviderAddress: fromWormholeFormat(relayerAddress),
        vaaKeys: new VaaKey[](0),
        consistencyLevel: CONSISTENCY_LEVEL_INSTANT,
        payload: new bytes(0),
        relayParameters: new bytes(0)
      }));
    
    //If refundAmount is not enough to pay for one wei of receiver value, then do not perform the
    //  cross-chain refund (i.e. if (delivery overhead) + (wormhole message fee) + (cost of one wei
    //  of receiver value) is larger than the remaining refund)
    if (crossChainRefundInstruction.receiverValueTarget == 0)
      return RefundStatus.CROSS_CHAIN_REFUND_FAIL_NOT_ENOUGH;

    uint256 maxBudget = relayProvider.quoteMaximumBudget(refundChainId);
    bool exceedsMaxBudget = false;
    if (crossChainRefundInstruction.receiverValueTarget > maxBudget) {
      crossChainRefundInstruction.receiverValueTarget = maxBudget;
      exceedsMaxBudget = true;
    }

    publishAndPay(
      wormholeMessageFee,
      0,
      refundSubMessageFee,
      crossChainRefundInstruction.encode(),
      CONSISTENCY_LEVEL_INSTANT,
      relayProvider
    );

    return exceedsMaxBudget
      ? RefundStatus.CROSS_CHAIN_REFUND_SENT_MAXIMUM_BUDGET
      : RefundStatus.CROSS_CHAIN_REFUND_SENT;
  }

  function decodeCancelled(
    bytes memory revertData
  ) private pure returns (uint32 gasUsed, uint256 available, uint256 required) {
    uint offset = 0;
    bytes4 selector;
    (selector, offset) = revertData.asBytes4Unchecked(offset);
    offset += 28;
    (gasUsed, offset) = revertData.asUint32Unchecked(offset);
    (available, offset) = revertData.asUint256Unchecked(offset);
    (required, offset) = revertData.asUint256Unchecked(offset);
    assert(offset == revertData.length && selector == Cancelled.selector);
  }

  function checkVaaKeysWithVAAs(
    VaaKey[] memory vaaKeys,
    bytes[] memory signedVaas
  ) private view {
    if (vaaKeys.length != signedVaas.length)
      revert VaaKeysLengthDoesNotMatchVaasLength(vaaKeys.length, signedVaas.length);

    for (uint i = 0; i < vaaKeys.length;) {
      IWormhole.VM memory parsedVaa = getWormhole().parseVM(signedVaas[i]);
      VaaKey memory vaaKey = vaaKeys[i];

      //this if is exhaustive, i.e vaaKey.infoType only has the two variants
      if (( vaaKey.infoType == VaaKeyType.EMITTER_SEQUENCE &&
            ( vaaKey.chainId != parsedVaa.emitterChainId ||
              vaaKey.emitterAddress != parsedVaa.emitterAddress ||
              vaaKey.sequence != parsedVaa.sequence
            )
          ) ||
          ( vaaKey.infoType == VaaKeyType.VAAHASH &&
            vaaKey.vaaHash != parsedVaa.hash
          ))
        revert VaaKeysDoNotMatchVaas(uint8(i));
      
      unchecked{++i;}
    }
  }
}
