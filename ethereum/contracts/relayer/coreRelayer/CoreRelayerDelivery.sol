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
  InvalidOverrideRefundPerGasUnused,
  RequesterNotCoreRelayer,
  VaaKey,
  IWormholeRelayerDelivery,
  IWormholeRelayerSend
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {DeliveryData, IWormholeReceiver} from "../../interfaces/relayer/IWormholeReceiver.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";

import {pay, min, toWormholeFormat, fromWormholeFormat} from "../../libraries/relayer/Utils.sol";
import {DeliveryInstruction, DeliveryOverride} from "../../libraries/relayer/RelayerInternalStructs.sol";
import {BytesParsing} from "../../libraries/relayer/BytesParsing.sol";
import {CoreRelayerSerde} from "./CoreRelayerSerde.sol";
import {ForwardInstruction} from "./CoreRelayerStorage.sol";
import {CoreRelayerBase} from "./CoreRelayerBase.sol";
import "../../interfaces/relayer/TypedUnits.sol";
import "../../libraries/relayer/ExecutionParameters.sol";

abstract contract CoreRelayerDelivery is CoreRelayerBase, IWormholeRelayerDelivery {
  using CoreRelayerSerde for *; //somewhat yucky but unclear what's a better alternative
  using BytesParsing for bytes;
  using WeiLib for Wei;
  using GasLib for Gas;
  using GasPriceLib for GasPrice;

  function deliver(
    bytes[] memory encodedVMs,
    bytes memory encodedDeliveryVAA,
    address payable relayerRefundAddress,
    bytes memory deliveryOverrides
  ) public payable {
    (IWormhole.VM memory vm, bool valid, string memory reason) =
      getWormhole().parseAndVerifyVM(encodedDeliveryVAA);

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
    
    DeliveryVAAInfo memory deliveryVaaInfo = DeliveryVAAInfo({
      sourceChainId: vm.emitterChainId,
      sourceSequence: vm.sequence,
      deliveryVaaHash: vm.hash,
      relayerRefundAddress: relayerRefundAddress,
      encodedVMs: encodedVMs,
      deliveryInstruction: instruction,
      gasLimit: Gas.wrap(0),
      targetChainRefundPerGasUnused: GasPrice.wrap(0),
      totalReceiverValue: Wei.wrap(0),
      encodedOverrides: deliveryOverrides,
      redeliveryHash: bytes32(0)
    });
    
    (deliveryVaaInfo.gasLimit, deliveryVaaInfo.targetChainRefundPerGasUnused, deliveryVaaInfo.totalReceiverValue, deliveryVaaInfo.redeliveryHash) = getDeliveryParametersEvmV1(instruction, deliveryOverrides);

    Wei requiredFunds = deliveryVaaInfo.gasLimit.toWei(deliveryVaaInfo.targetChainRefundPerGasUnused) + deliveryVaaInfo.totalReceiverValue;
    if (msgValue() < requiredFunds)
      revert InsufficientRelayerFunds(msgValue(), requiredFunds);

    if (getChainId() != instruction.targetChainId)
      revert TargetChainIsNotThisChain(instruction.targetChainId);

    checkVaaKeysWithVAAs(instruction.vaaKeys, encodedVMs);

    executeDelivery(deliveryVaaInfo);

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
    Gas gasLimit;
    GasPrice targetChainRefundPerGasUnused;
    Wei totalReceiverValue;
    bytes encodedOverrides;
    bytes32 redeliveryHash; //optional (0 if not present)
  }

  function getDeliveryParametersEvmV1(DeliveryInstruction memory instruction, bytes memory encodedOverrides) internal pure returns (Gas gasLimit, GasPrice targetChainRefundPerGasUnused, Wei totalReceiverValue, bytes32 redeliveryHash) {
    
    ExecutionInfoVersion instructionExecutionInfoVersion = decodeExecutionInfoVersion(instruction.encodedExecutionInfo);
    if(instructionExecutionInfoVersion != ExecutionInfoVersion.EVM_V1) {
      revert UnexpectedExecutionInfoVersion(uint8(instructionExecutionInfoVersion), uint8(ExecutionInfoVersion.EVM_V1));
    }

    EvmExecutionInfoV1 memory executionInfo = decodeEvmExecutionInfoV1(instruction.encodedExecutionInfo);

    // If present, apply redelivery deliveryOverrides to current instruction
    if (encodedOverrides.length != 0) {
      DeliveryOverride memory deliveryOverrides = encodedOverrides.decodeDeliveryOverride();
      
      // Check to see if gasLimit >= original gas limit, receiver value >= original receiver value, and refund >= original refund
      // If so, replace the corresponding variables with the overriden variables
      (instruction.requestedReceiverValue, executionInfo) = decodeAndCheckOverridesEvmV1(instruction.requestedReceiverValue, executionInfo, deliveryOverrides);
      instruction.extraReceiverValue = Wei.wrap(0);
      redeliveryHash = deliveryOverrides.redeliveryHash;
    } 

    gasLimit = executionInfo.gasLimit;
    targetChainRefundPerGasUnused = executionInfo.targetChainRefundPerGasUnused;
    totalReceiverValue = (instruction.requestedReceiverValue + instruction.extraReceiverValue);
    
  }

  function decodeAndCheckOverridesEvmV1(Wei receiverValue, EvmExecutionInfoV1 memory executionInfo, DeliveryOverride memory deliveryOverrides) internal pure returns (Wei deliveryOverridesReceiverValue, EvmExecutionInfoV1 memory deliveryOverridesExecutionInfo) {
    
    if (deliveryOverrides.newReceiverValue < receiverValue) {
        revert InvalidOverrideReceiverValue();
    } 
 
    ExecutionInfoVersion deliveryOverridesExecutionInfoVersion = decodeExecutionInfoVersion(deliveryOverrides.newExecutionInfo);
    if(ExecutionInfoVersion.EVM_V1 != deliveryOverridesExecutionInfoVersion) {
      revert VersionMismatchOverride(uint8(ExecutionInfoVersion.EVM_V1), uint8(deliveryOverridesExecutionInfoVersion));
    }

    deliveryOverridesExecutionInfo = decodeEvmExecutionInfoV1(deliveryOverrides.newExecutionInfo);
    deliveryOverridesReceiverValue = deliveryOverrides.newReceiverValue;

    if(deliveryOverridesExecutionInfo.targetChainRefundPerGasUnused.unwrap() < executionInfo.targetChainRefundPerGasUnused.unwrap()) {
      revert InvalidOverrideRefundPerGasUnused();
    }
    if(deliveryOverridesExecutionInfo.gasLimit < executionInfo.gasLimit) {
      revert InvalidOverrideGasLimit();
    }

  }

  struct DeliveryResults {
    Gas gasUsed;
    DeliveryStatus status;
    bytes additionalStatusInfo;
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
    
    if (checkIfCrossChainRefund(vaaInfo)) {
      return;
    }
  
    DeliveryResults memory results;
    
    try //force external call!
      this.executeInstruction(
        vaaInfo.deliveryInstruction.targetAddress,
        DeliveryData({
          sourceAddress: vaaInfo.deliveryInstruction.senderAddress,
          sourceChainId: vaaInfo.sourceChainId,
          targetChainRefundPerGasUnused: vaaInfo.targetChainRefundPerGasUnused.unwrap(),
          deliveryHash:  vaaInfo.deliveryVaaHash,
          payload:       vaaInfo.deliveryInstruction.payload
        }),
        vaaInfo.gasLimit,
        vaaInfo.totalReceiverValue,
        vaaInfo.targetChainRefundPerGasUnused,
        vaaInfo.encodedVMs
      )
    returns (uint8 _status, Gas _gasUsed, bytes memory targetRevertDataTruncated) {
      results = DeliveryResults(
        _gasUsed,
        DeliveryStatus(_status),
        //will carry the correct value regardless of outcome (empty if successful, error otherwise)
        targetRevertDataTruncated
      );
    }
    //executeInstruction should only revert with a Cancelled error though for now it can
    //  theoretically also revert with a Panic
    //  revert for any other reason (though it might for overflows atm)
    catch (bytes memory revertData) {
      //decode returned Cancelled error
      
      uint256 available;
      uint256 required;
      uint32 gasUsed_;
      (gasUsed_, available, required) = decodeCancelled(revertData);
      results = DeliveryResults(
        Gas.wrap(gasUsed_),
        DeliveryStatus.FORWARD_REQUEST_FAILURE,
        abi.encodeWithSelector(ForwardNotSufficientlyFunded.selector, available, required)
      );

    }
    
    emit Delivery(
      fromWormholeFormat(vaaInfo.deliveryInstruction.targetAddress),
      vaaInfo.sourceChainId,
      vaaInfo.sourceSequence,
      vaaInfo.deliveryVaaHash,
      results.status,
      uint32(results.gasUsed.unwrap()),
      payRefunds(
        vaaInfo.deliveryInstruction,
        vaaInfo.relayerRefundAddress,
        (vaaInfo.gasLimit - results.gasUsed).toWei(vaaInfo.targetChainRefundPerGasUnused),
        results.status
      ),
      results.additionalStatusInfo,
      (vaaInfo.redeliveryHash != 0)
        ? vaaInfo.encodedOverrides
        : new bytes(0)
    );
  }

  function checkIfCrossChainRefund(DeliveryVAAInfo memory vaaInfo) internal returns (bool isCrossChainRefund) {
    if (vaaInfo.deliveryInstruction.targetAddress == 0x0) {
      emit Delivery(
        fromWormholeFormat(vaaInfo.deliveryInstruction.targetAddress),
        vaaInfo.sourceChainId,
        vaaInfo.sourceSequence,
        vaaInfo.deliveryVaaHash,
        DeliveryStatus.SUCCESS,
        0,
        payRefunds(
        vaaInfo.deliveryInstruction,
        vaaInfo.relayerRefundAddress,
        Wei.wrap(0),
        DeliveryStatus.RECEIVER_FAILURE
        ),
        bytes(""),
        (vaaInfo.redeliveryHash != 0)
        ? vaaInfo.encodedOverrides
        : new bytes(0) 
      );
      isCrossChainRefund = true;
    }
  }



  function executeInstruction(
    bytes32 targetAddress,
    DeliveryData calldata data,
    Gas gasLimit,
    Wei totalReceiverValue,
    GasPrice targetChainRefundPerGasUnused,
    bytes[] memory signedVaas
  ) external returns (
    uint8 status,
    Gas gasUsed,
    bytes memory targetRevertDataTruncated
  ) {
   
    //despite being external, we only allow ourselves to call this function (via CALL opcode)
    //  used as a means to retroactively revert the call to the delivery target if the forwards
    //  can't be funded
    if (msg.sender != address(this))
      revert RequesterNotCoreRelayer();

    Gas preGas = Gas.wrap(gasleft());

    // Calls the `receiveWormholeMessages` endpoint on the contract `instruction.targetAddress`
    // (with the gas limit and value specified in instruction, and `encodedVMs` as the input)
    IWormholeReceiver deliveryTarget =
      IWormholeReceiver(fromWormholeFormat(targetAddress));
    try deliveryTarget.receiveWormholeMessages{
          gas:   gasLimit.unwrap(),
          value: totalReceiverValue.unwrap()
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

    Gas postGas = Gas.wrap(gasleft());
    
    unchecked{gasUsed = (preGas - postGas).min(gasLimit);}
    
    ForwardInstruction[] storage forwardInstructions = getForwardInstructions();
    
    if (forwardInstructions.length > 0) {
      //Calculate the amount of maxTransactionFee to refund (multiply the maximum refund by the
      //  fraction of gas unused)
      Wei transactionFeeRefundAmount = (gasLimit - gasUsed).toWei(targetChainRefundPerGasUnused);
      emitForward(gasUsed, transactionFeeRefundAmount, forwardInstructions);
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
    Gas gasUsed,
    Wei transactionFeeRefundAmount,
    ForwardInstruction[] storage forwardInstructions
  ) private {
    Wei wormholeMessageFee = getWormholeMessageFee();

    //Decode send requests and aggregate fee and payment
    DeliveryInstruction[] memory instructions = new DeliveryInstruction[](forwardInstructions.length); 

    Wei totalMsgValue = Wei.wrap(0);
    Wei totalFee = Wei.wrap(0);
    for (uint i = 0; i < forwardInstructions.length;) {
      unchecked{totalMsgValue = totalMsgValue + forwardInstructions[i].msgValue;}
      instructions[i] = (forwardInstructions[i].encodedInstruction).decodeDeliveryInstruction();
      totalFee = totalFee + forwardInstructions[i].deliveryPrice + forwardInstructions[i].paymentForExtraReceiverValue + wormholeMessageFee;
      unchecked{++i;}
    }

    //Combine refund amount with any additional funds which were passed in to the forward as
    //  msg.value and check that enough funds were passed into the forward (should always be true
    //  as it was already checked)
    Wei fundsForForward;
    unchecked{fundsForForward = transactionFeeRefundAmount + totalMsgValue;}
    if (fundsForForward < totalFee) {
        revert Cancelled(uint32(gasUsed.unwrap()), fundsForForward.unwrap(), totalFee.unwrap());
    }

    Wei extraReceiverValue = IRelayProvider(fromWormholeFormat(instructions[0].sourceRelayProvider)).quoteAssetConversion(instructions[0].targetChainId, fundsForForward - totalFee);
    //Increases the maxTransactionFee of the first forward in order to use all of the funds
    unchecked{
      instructions[0].extraReceiverValue = instructions[0].extraReceiverValue + extraReceiverValue;
    }

    //Publishes the DeliveryInstruction and pays the associated relayProvider
    for (uint i = 0; i < forwardInstructions.length;) {
      publishAndPay(
        wormholeMessageFee,
        forwardInstructions[i].deliveryPrice,
        forwardInstructions[i].paymentForExtraReceiverValue + ((i == 0) ? (fundsForForward - totalFee) : Wei.wrap(0)),
        i == 0 ? instructions[0].encode() : forwardInstructions[i].encodedInstruction,
        forwardInstructions[i].consistencyLevel,
        IRelayProvider(fromWormholeFormat(instructions[i].sourceRelayProvider))
      );
      unchecked{++i;}
    }
  }

  function payRefunds(
    DeliveryInstruction memory deliveryInstruction,
    address payable relayerRefundAddress,
    Wei transactionFeeRefundAmount,
    DeliveryStatus status
  ) private returns (RefundStatus refundStatus) {
    //Amount of receiverValue that is refunded to the user (0 if the call to
    //  'receiveWormholeMessages' did not revert, or the full receiverValue otherwise)
    Wei receiverValueRefundAmount = Wei.wrap(0);

    if(status == DeliveryStatus.FORWARD_REQUEST_FAILURE || status == DeliveryStatus.RECEIVER_FAILURE) {
      receiverValueRefundAmount = (deliveryInstruction.requestedReceiverValue + deliveryInstruction.extraReceiverValue);
    }

    //Total refund to the user
    Wei refundToRefundAddress =
      receiverValueRefundAmount + (status == DeliveryStatus.FORWARD_REQUEST_SUCCESS ? Wei.wrap(0) : transactionFeeRefundAmount);
    
    //Refund the user
    try this.payRefundToRefundAddress(
      deliveryInstruction.refundChainId,
      deliveryInstruction.refundAddress,
      refundToRefundAddress,
      deliveryInstruction.refundRelayProvider
    )
    returns (RefundStatus _refundStatus) {
      refundStatus = _refundStatus;
    } 
    catch (bytes memory) {
      refundStatus = RefundStatus.CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED;
    }

    //Refund the relayer (their extra funds) + (the amount that the relayer spent on gas)
    //  + (the users refund if that refund didn't succeed)
    Wei leftoverUserRefund = refundToRefundAddress;
    if(refundStatus == RefundStatus.REFUND_SENT || refundStatus == RefundStatus.CROSS_CHAIN_REFUND_SENT) {
      leftoverUserRefund = Wei.wrap(0);
    }

    Wei relayerRefundAmount = 
      msgValue() - (deliveryInstruction.requestedReceiverValue + deliveryInstruction.extraReceiverValue) - transactionFeeRefundAmount + leftoverUserRefund;

    //TODO AMO: what if pay fails? (i.e. returns false)
    //Refund the relay provider
    pay(relayerRefundAddress, relayerRefundAmount);
  }

  function payRefundToRefundAddress(
    uint16 refundChainId,
    bytes32 refundAddress,
    Wei refundAmount,
    bytes32 relayerAddress
  ) external returns (RefundStatus) {
    //despite being external, we only allow ourselves to call this function (via CALL opcode)
    //  used as a means to catch reverts when we external call the relay provider in this function
    if (msg.sender != address(this))
      revert RequesterNotCoreRelayer();

    //same chain refund
    if (refundChainId == getChainId()) {
      return pay(payable(fromWormholeFormat(refundAddress)), refundAmount)
        ? RefundStatus.REFUND_SENT
        : RefundStatus.REFUND_FAIL;
    }

    //cross-chain refund
    IRelayProvider relayProvider = IRelayProvider(fromWormholeFormat(relayerAddress));
    if (!relayProvider.isChainSupported(refundChainId))
      return RefundStatus.CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED;

    // assuming refund chain is an EVM chain
    // must modify this when we extend system to non-EVM
    (Wei quote,) = relayProvider.quoteDeliveryPrice(refundChainId, Wei.wrap(0), encodeEvmExecutionParamsV1(getEmptyEvmExecutionParamsV1()));
    if (refundAmount <= getWormholeMessageFee() + quote)
      return RefundStatus.CROSS_CHAIN_REFUND_FAIL_NOT_ENOUGH;
    try IWormholeRelayerSend(address(this)).send{
      value: refundAmount.unwrap()
    }(
      refundChainId,
      bytes32(0),
      bytes(""),
      Wei.wrap(0),
      refundAmount - getWormholeMessageFee() - quote,
      encodeEvmExecutionParamsV1(getEmptyEvmExecutionParamsV1()),
      refundChainId,
      refundAddress,
      fromWormholeFormat(relayerAddress),
      new VaaKey[](0),
      CONSISTENCY_LEVEL_INSTANT) 
    returns (uint64) {
      return RefundStatus.CROSS_CHAIN_REFUND_SENT;
    }
    catch (bytes memory) {
      return RefundStatus.CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED;
    }
  }

  function decodeCancelled(
    bytes memory revertData
  ) private view returns (uint32 gasUsed, uint256 available, uint256 required) {
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
      if (vaaKey.chainId != parsedVaa.emitterChainId 
          || vaaKey.emitterAddress != parsedVaa.emitterAddress 
          || vaaKey.sequence != parsedVaa.sequence)
        revert VaaKeysDoNotMatchVaas(uint8(i));

      unchecked{++i;}
    }
  }
}
