// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {IWormhole} from "../../interfaces/IWormhole.sol";
import {
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
    RequesterNotWormholeRelayer,
    DeliveryProviderCannotReceivePayment,
    VaaKey,
    IWormholeRelayerDelivery,
    IWormholeRelayerSend,
    RETURNDATA_TRUNCATION_THRESHOLD
} from "../../interfaces/relayer/IWormholeRelayerTyped.sol";
import {IWormholeReceiver} from "../../interfaces/relayer/IWormholeReceiver.sol";
import {IDeliveryProvider} from "../../interfaces/relayer/IDeliveryProviderTyped.sol";

import {pay, min, toWormholeFormat, fromWormholeFormat, returnLengthBoundedCall} from "../../libraries/relayer/Utils.sol";
import {
    DeliveryInstruction,
    DeliveryOverride,
    EvmDeliveryInstruction
} from "../../libraries/relayer/RelayerInternalStructs.sol";
import {BytesParsing} from "../../libraries/relayer/BytesParsing.sol";
import {WormholeRelayerSerde} from "./WormholeRelayerSerde.sol";
import {ForwardInstruction} from "./WormholeRelayerStorage.sol";
import {WormholeRelayerBase} from "./WormholeRelayerBase.sol";
import "../../interfaces/relayer/TypedUnits.sol";
import "../../libraries/relayer/ExecutionParameters.sol";

abstract contract WormholeRelayerDelivery is WormholeRelayerBase, IWormholeRelayerDelivery {
    using WormholeRelayerSerde for *; 
    using BytesParsing for bytes;
    using WeiLib for Wei;
    using GasLib for Gas;
    using GasPriceLib for GasPrice;
    using TargetNativeLib for TargetNative;
    using LocalNativeLib for LocalNative;

    function deliver(
        bytes[] memory encodedVMs,
        bytes memory encodedDeliveryVAA,
        address payable relayerRefundAddress,
        bytes memory deliveryOverrides
    ) public payable {

        // Parse and verify VAA containing delivery instructions, revert if invalid
        (IWormhole.VM memory vm, bool valid, string memory reason) =
            getWormhole().parseAndVerifyVM(encodedDeliveryVAA);
        if (!valid) {
            revert InvalidDeliveryVaa(reason);
        }

        // Revert if the emitter of the VAA is not a Wormhole Relayer contract 
        bytes32 registeredWormholeRelayer = getRegisteredWormholeRelayerContract(vm.emitterChainId);
        if (vm.emitterAddress != registeredWormholeRelayer) {
            revert InvalidEmitter(vm.emitterAddress, registeredWormholeRelayer, vm.emitterChainId);
        }
    
        DeliveryInstruction memory instruction = vm.payload.decodeDeliveryInstruction();

        // Lock the contract (and store some information about the delivery in temporary storage)
        startDelivery(
            fromWormholeFormat(instruction.targetAddress),
            fromWormholeFormat(instruction.refundDeliveryProvider),
            instruction.refundChain,
            instruction.refundAddress
        );

        DeliveryVAAInfo memory deliveryVaaInfo = DeliveryVAAInfo({
            sourceChain: vm.emitterChainId,
            sourceSequence: vm.sequence,
            deliveryVaaHash: vm.hash,
            relayerRefundAddress: relayerRefundAddress,
            encodedVMs: encodedVMs,
            deliveryInstruction: instruction,
            gasLimit: Gas.wrap(0),
            targetChainRefundPerGasUnused: GasPrice.wrap(0),
            totalReceiverValue: TargetNative.wrap(0),
            encodedOverrides: deliveryOverrides,
            redeliveryHash: bytes32(0)
        });

        // Decode information from the execution parameters
        // (overriding them if there was an override requested)
        // Assumes execution parameters and info are of version EVM_V1
        (
            deliveryVaaInfo.gasLimit,
            deliveryVaaInfo.targetChainRefundPerGasUnused,
            deliveryVaaInfo.totalReceiverValue,
            deliveryVaaInfo.redeliveryHash
        ) = getDeliveryParametersEvmV1(instruction, deliveryOverrides);

        // Revert if msg.value is not enough to fund both the receiver value
        // as well as the maximum possible refund 
        // Note: instruction's TargetNative is delivery's LocalNative
        LocalNative requiredFunds = (deliveryVaaInfo.gasLimit.toWei(
            deliveryVaaInfo.targetChainRefundPerGasUnused
        ) + deliveryVaaInfo.totalReceiverValue.asNative()).asLocalNative();
        if (msgValue() < requiredFunds) {
            revert InsufficientRelayerFunds(msgValue(), requiredFunds);
        }

        // Revert if the instruction's target chain is not this chain
        if (getChainId() != instruction.targetChain) {
            revert TargetChainIsNotThisChain(instruction.targetChain);
        }

        // Revert if the VAAs delivered do not match the descriptions specified in the instruction
        checkVaaKeysWithVAAs(instruction.vaaKeys, encodedVMs);

        executeDelivery(deliveryVaaInfo);

        // Unlock contract
        finishDelivery();
    }

    // ------------------------------------------- PRIVATE -------------------------------------------

    error Cancelled(Gas gasUsed, LocalNative available, LocalNative required);
    error DeliveryProviderReverted(Gas gasUsed);
    error DeliveryProviderPaymentFailed(Gas gasUsed);

    struct DeliveryVAAInfo {
        uint16 sourceChain;
        uint64 sourceSequence;
        bytes32 deliveryVaaHash;
        address payable relayerRefundAddress;
        bytes[] encodedVMs;
        DeliveryInstruction deliveryInstruction;
        Gas gasLimit;
        GasPrice targetChainRefundPerGasUnused;
        TargetNative totalReceiverValue;
        bytes encodedOverrides;
        bytes32 redeliveryHash; //optional (0 if not present)
    }

    function getDeliveryParametersEvmV1(
        DeliveryInstruction memory instruction,
        bytes memory encodedOverrides
    )
        internal
        pure
        returns (
            Gas gasLimit,
            GasPrice targetChainRefundPerGasUnused,
            TargetNative totalReceiverValue,
            bytes32 redeliveryHash
        )
    {
        ExecutionInfoVersion instructionExecutionInfoVersion =
            decodeExecutionInfoVersion(instruction.encodedExecutionInfo);
        if (instructionExecutionInfoVersion != ExecutionInfoVersion.EVM_V1) {
            revert UnexpectedExecutionInfoVersion(
                uint8(instructionExecutionInfoVersion), uint8(ExecutionInfoVersion.EVM_V1)
            );
        }

        EvmExecutionInfoV1 memory executionInfo =
            decodeEvmExecutionInfoV1(instruction.encodedExecutionInfo);

        // If present, apply redelivery deliveryOverrides to current instruction
        if (encodedOverrides.length != 0) {
            DeliveryOverride memory deliveryOverrides = encodedOverrides.decodeDeliveryOverride();

            // Check to see if gasLimit >= original gas limit, receiver value >= original receiver value, and refund >= original refund
            // If so, replace the corresponding variables with the overriden variables
            // If not, revert
            (instruction.requestedReceiverValue, executionInfo) = decodeAndCheckOverridesEvmV1(
                instruction.requestedReceiverValue, executionInfo, deliveryOverrides
            );
            instruction.extraReceiverValue = TargetNative.wrap(0);
            redeliveryHash = deliveryOverrides.redeliveryHash;
        }

        gasLimit = executionInfo.gasLimit;
        targetChainRefundPerGasUnused = executionInfo.targetChainRefundPerGasUnused;
        totalReceiverValue = instruction.requestedReceiverValue + instruction.extraReceiverValue;
    }

    function decodeAndCheckOverridesEvmV1(
        TargetNative receiverValue,
        EvmExecutionInfoV1 memory executionInfo,
        DeliveryOverride memory deliveryOverrides
    )
        internal
        pure
        returns (
            TargetNative deliveryOverridesReceiverValue,
            EvmExecutionInfoV1 memory deliveryOverridesExecutionInfo
        )
    {
        if (deliveryOverrides.newReceiverValue.unwrap() < receiverValue.unwrap()) {
            revert InvalidOverrideReceiverValue();
        }

        ExecutionInfoVersion deliveryOverridesExecutionInfoVersion =
            decodeExecutionInfoVersion(deliveryOverrides.newExecutionInfo);
        if (ExecutionInfoVersion.EVM_V1 != deliveryOverridesExecutionInfoVersion) {
            revert VersionMismatchOverride(
                uint8(ExecutionInfoVersion.EVM_V1), uint8(deliveryOverridesExecutionInfoVersion)
            );
        }

        deliveryOverridesExecutionInfo =
            decodeEvmExecutionInfoV1(deliveryOverrides.newExecutionInfo);
        deliveryOverridesReceiverValue = deliveryOverrides.newReceiverValue;

        if (
            deliveryOverridesExecutionInfo.targetChainRefundPerGasUnused.unwrap()
                < executionInfo.targetChainRefundPerGasUnused.unwrap()
        ) {
            revert InvalidOverrideRefundPerGasUnused();
        }
        if (deliveryOverridesExecutionInfo.gasLimit < executionInfo.gasLimit) {
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
     *     `vaaInfo.deliveryInstruction.targetAddress` (with the gas limit and value specified in
     *     vaaInfo.gasLimit and vaaInfo.totalReceiverValue, and `encodedVMs` as the input)
     *
     * - Calculates how much gas from `vaaInfo.gasLimit` is left
     * - If the call succeeded and during execution of `receiveWormholeMessages` there were
     *     forward(s), and (gas left from vaaInfo.gasLimit) * (vaaInfo.targetChainRefundPerGasUnused) is enough extra funds
     *     to execute the forward(s), then the forward(s) are executed
     * - else:
     *     revert the delivery to trigger a receiver failure (or forward request failure if 
     *     there were forward(s))
     *     refund 'vaaInfo.targetChainRefundPerGasUnused'*(amount of vaaInfo.gasLimit unused) to vaaInfo.deliveryInstruction.refundAddress
     *     if the call reverted, refund `vaaInfo.totalReceiverValue` to vaaInfo.deliveryInstruction.refundAddress
     * - refund anything leftover to the relayer
     *
     * @param vaaInfo struct specifying:
     *    - sourceChain chain id that the delivery originated from
     *    - sourceSequence sequence number of the delivery VAA on the source chain
     *    - deliveryVaaHash hash of delivery VAA
     *    - relayerRefundAddress address that should be paid for relayer refunds
     *    - encodedVMs list of signed wormhole messages (VAAs)
     *    - deliveryInstruction the specific instruction which is being executed
     *    - gasLimit the gas limit to call targetAddress with
     *    - targetChainRefundPerGasUnused the amount of (this chain) wei to refund to refundAddress
     *      per unit of gas unused (from gasLimit)
     *    - totalReceiverValue the msg.value to call targetAddress with
     *    - encodedOverrides any (encoded) overrides that were applied
     *    - (optional) redeliveryHash hash of redelivery Vaa
     */

    function executeDelivery(DeliveryVAAInfo memory vaaInfo) private {

        // If the targetAddress is the 0 address
        // Then emit event and return
        // (This is used for cross-chain refunds)
        if (checkIfCrossChainRefund(vaaInfo)) {
            return;
        }

        DeliveryResults memory results;

        // Forces external call
        // In order to catch reverts
        // (If the user's contract requests a forward
        // and there ends up not being enough funds,
        // then we will revert this call)
        try 
        this.executeInstruction(
            EvmDeliveryInstruction({
                sourceChain: vaaInfo.sourceChain,
                targetAddress: vaaInfo.deliveryInstruction.targetAddress,
                payload: vaaInfo.deliveryInstruction.payload,
                gasLimit: vaaInfo.gasLimit,
                totalReceiverValue: vaaInfo.totalReceiverValue,
                targetChainRefundPerGasUnused: vaaInfo.targetChainRefundPerGasUnused,
                senderAddress: vaaInfo.deliveryInstruction.senderAddress,
                deliveryHash: vaaInfo.deliveryVaaHash,
                signedVaas: vaaInfo.encodedVMs
            }
        )) returns (
            uint8 _status, Gas _gasUsed, bytes memory targetRevertDataTruncated
        ) {
            results = DeliveryResults(
                _gasUsed,
                DeliveryStatus(_status),
                targetRevertDataTruncated
            );
        } 
        catch (bytes memory revertData) {
            // Should only revert if 
            // 1) forward(s) were requested but not enough funds were available from the refund
            // 2) forward(s) were requested, but the delivery provider requested for the forward reverted during execution
            // 3) forward(s) were requested, but the payment of the delivery provider for the forward failed

            // Decode returned error (into one of these three known types)
            // obtaining the gas usage of targetAddress
            bool knownError;
            Gas gasUsed_;
            (gasUsed_, knownError) = tryDecodeExecuteInstructionError(revertData);
            results = DeliveryResults(
                knownError? gasUsed_ : vaaInfo.gasLimit,
                DeliveryStatus.FORWARD_REQUEST_FAILURE,
                revertData
            );
        }

        emit Delivery(
            fromWormholeFormat(vaaInfo.deliveryInstruction.targetAddress),
            vaaInfo.sourceChain,
            vaaInfo.sourceSequence,
            vaaInfo.deliveryVaaHash,
            results.status,
            results.gasUsed,
            payRefunds(
                vaaInfo.deliveryInstruction,
                vaaInfo.relayerRefundAddress,
                (vaaInfo.gasLimit - results.gasUsed).toWei(vaaInfo.targetChainRefundPerGasUnused).asLocalNative(),
                results.status
            ),
            results.additionalStatusInfo,
            (vaaInfo.redeliveryHash != 0) ? vaaInfo.encodedOverrides : new bytes(0)
        );
    }

    function checkIfCrossChainRefund(DeliveryVAAInfo memory vaaInfo)
        internal
        returns (bool isCrossChainRefund)
    {
        if (vaaInfo.deliveryInstruction.targetAddress == 0x0) {
            emit Delivery(
                fromWormholeFormat(vaaInfo.deliveryInstruction.targetAddress),
                vaaInfo.sourceChain,
                vaaInfo.sourceSequence,
                vaaInfo.deliveryVaaHash,
                DeliveryStatus.SUCCESS,
                Gas.wrap(0),
                payRefunds(
                    vaaInfo.deliveryInstruction,
                    vaaInfo.relayerRefundAddress,
                    LocalNative.wrap(0),
                    DeliveryStatus.RECEIVER_FAILURE
                ),
                bytes(""),
                (vaaInfo.redeliveryHash != 0) ? vaaInfo.encodedOverrides : new bytes(0)
            );
            isCrossChainRefund = true;
        }
    }

    function executeInstruction(EvmDeliveryInstruction memory evmInstruction)
        external
        returns (uint8 status, Gas gasUsed, bytes memory targetRevertDataTruncated)
    {
        //  despite being external, we only allow ourselves to call this function (via CALL opcode)
        //  used as a means to retroactively revert the call to the delivery target if the forwards
        //  can't be funded
        if (msg.sender != address(this)) {
            revert RequesterNotWormholeRelayer();
        }

        Gas gasLimit = evmInstruction.gasLimit;
        bool success;
        {
            address payable deliveryTarget = payable(fromWormholeFormat(evmInstruction.targetAddress));
            bytes memory callData = abi.encodeCall(IWormholeReceiver.receiveWormholeMessages, (
                evmInstruction.payload,
                evmInstruction.signedVaas,
                evmInstruction.senderAddress,
                evmInstruction.sourceChain,
                evmInstruction.deliveryHash
            ));

            // Measure gas usage of call
            Gas preGas = Gas.wrap(gasleft());

            // Calls the `receiveWormholeMessages` endpoint on the contract `evmInstruction.targetAddress`
            // (with the gas limit and value specified in instruction, and `encodedVMs` as the input)
            // If it reverts, returns the first 132 bytes of the revert message
            (success, targetRevertDataTruncated) = returnLengthBoundedCall(
                deliveryTarget,
                callData,
                gasLimit.unwrap(),
                evmInstruction.totalReceiverValue.unwrap(),
                RETURNDATA_TRUNCATION_THRESHOLD
            );

            Gas postGas = Gas.wrap(gasleft());

            unchecked {
                gasUsed = (preGas - postGas).min(gasLimit);
            }
        }

        if (success) {
            targetRevertDataTruncated = new bytes(0);
            status = uint8(DeliveryStatus.SUCCESS);

            ForwardInstruction[] storage forwardInstructions = getForwardInstructions();

            if (forwardInstructions.length > 0) {
                // forward(s) were requested during execution of the call to targetAddress above

                // Calculate the amount to refund the user for unused gas
                LocalNative transactionFeeRefundAmount = (gasLimit - gasUsed).toWei(
                    evmInstruction.targetChainRefundPerGasUnused
                ).asLocalNative();

                // Check if refund amount is enough, and if so, emit forwards  
                emitForward(gasUsed, transactionFeeRefundAmount, forwardInstructions);

                // If we reach here (i.e. emitForward didn't revert) then the forward succeeded
                status = uint8(DeliveryStatus.FORWARD_REQUEST_SUCCESS);
            }
        } else {
            // Call to 'receiveWormholeMessages' on targetAddress reverted
            status = uint8(DeliveryStatus.RECEIVER_FAILURE);
        }
    }

    /**
     * - Checks if enough funds were passed into a forward (and reverts if not)
     * - Increases the 'extraReceiverValue' of the first forward in order to use all of the funds
     * - Publishes the DeliveryInstruction
     * - Pays the relayer's reward address to deliver the forward
     *
     * @param transactionFeeRefundAmount amount of maxTransactionFee that was unused
     * @param forwardInstructions An array of structs containing information about the user's forward
     *     request(s)
     */
    function emitForward(
        Gas gasUsed,
        LocalNative transactionFeeRefundAmount,
        ForwardInstruction[] storage forwardInstructions
    ) private {
        LocalNative wormholeMessageFee = getWormholeMessageFee();

        // Decode delivery instructions from each 'forward' request
        DeliveryInstruction[] memory instructions =
            new DeliveryInstruction[](forwardInstructions.length);

        // Calculate total msg.value passed into all 'forward' requests
        LocalNative totalMsgValue;
        // Calculate total fee for all 'forward' requests
        LocalNative totalFee;

        for (uint256 i = 0; i < forwardInstructions.length;) {
            unchecked {
                totalMsgValue = totalMsgValue + forwardInstructions[i].msgValue;
            }
            instructions[i] =
                (forwardInstructions[i].encodedInstruction).decodeDeliveryInstruction();
            totalFee = totalFee + forwardInstructions[i].deliveryPrice
                + forwardInstructions[i].paymentForExtraReceiverValue + wormholeMessageFee;
            unchecked {
                ++i;
            }
        }

        //  Combine refund amount with any additional funds which were passed in to the forward as
        //  msg.value and check that this value is enough to fund all the forwards, reverting otherwise
        LocalNative fundsForForward;
        unchecked {
            fundsForForward = transactionFeeRefundAmount + totalMsgValue;
        }
        if (fundsForForward.unwrap() < totalFee.unwrap()) {
            revert Cancelled(gasUsed, fundsForForward, totalFee);
        }
        
        // Simulates if the user had increased the 'paymentForExtraReceiverValue' field of their first forward
        // to the maximum value such that the forward(s) would still have enough funds

        TargetNative extraReceiverValue;
        try IDeliveryProvider(
            fromWormholeFormat(instructions[0].sourceDeliveryProvider)
        ).quoteAssetConversion(instructions[0].targetChain, fundsForForward - totalFee)
        returns (TargetNative _extraReceiverValue) {
            extraReceiverValue = _extraReceiverValue;
        } catch {
            revert DeliveryProviderReverted(gasUsed);
        }
        
        unchecked {
            instructions[0].extraReceiverValue =
                instructions[0].extraReceiverValue + extraReceiverValue;
        }

        //Publishes the DeliveryInstruction(s) for each forward request and pays the associated deliveryProvider
        for (uint256 i = 0; i < forwardInstructions.length;) {
            (, bool paymentSucceeded) = publishAndPay(
                wormholeMessageFee,
                forwardInstructions[i].deliveryPrice,
                // We had increased the 'paymentForExtraReceiverValue' of the first forward
                forwardInstructions[i].paymentForExtraReceiverValue
                    + ((i == 0) ? (fundsForForward - totalFee) : LocalNative.wrap(0)),
                i == 0 ? instructions[0].encode() : forwardInstructions[i].encodedInstruction,
                forwardInstructions[i].consistencyLevel,
                forwardInstructions[i].rewardAddress
            );
            if (!paymentSucceeded) {
                revert DeliveryProviderPaymentFailed(gasUsed);
            }
            unchecked {
                ++i;
            }
        }
    }

    function payRefunds(
        DeliveryInstruction memory deliveryInstruction,
        address payable relayerRefundAddress,
        LocalNative transactionFeeRefundAmount,
        DeliveryStatus status
    ) private returns (RefundStatus refundStatus) {
        //Amount of receiverValue that is refunded to the user (0 if the call to
        //  'receiveWormholeMessages' did not revert, or the full receiverValue otherwise)
        LocalNative receiverValueRefundAmount = LocalNative.wrap(0);

        if (
            status == DeliveryStatus.FORWARD_REQUEST_FAILURE
                || status == DeliveryStatus.RECEIVER_FAILURE
        ) {
            receiverValueRefundAmount = (
                deliveryInstruction.requestedReceiverValue + deliveryInstruction.extraReceiverValue
            ).asNative().asLocalNative(); // NOTE: instruction's target is delivery's local
        }

        // Total refund to the user
        // (If the forward succeeded, the 'transactionFeeRefundAmount' was used there already)
        LocalNative refundToRefundAddress = receiverValueRefundAmount
            + (
                status == DeliveryStatus.FORWARD_REQUEST_SUCCESS
                    ? LocalNative.wrap(0)
                    : transactionFeeRefundAmount
            );

        //Refund the user
        refundStatus = payRefundToRefundAddress(
            deliveryInstruction.refundChain,
            deliveryInstruction.refundAddress,
            refundToRefundAddress,
            deliveryInstruction.refundDeliveryProvider
        );

        //If sending the user's refund failed, this gets added to the relayer's refund
        LocalNative leftoverUserRefund = refundToRefundAddress;
        if (
            refundStatus == RefundStatus.REFUND_SENT
                || refundStatus == RefundStatus.CROSS_CHAIN_REFUND_SENT
        ) {
            leftoverUserRefund = LocalNative.wrap(0);
        }

        // Refund the relayer all remaining funds
        LocalNative relayerRefundAmount = calcRelayerRefundAmount(deliveryInstruction, transactionFeeRefundAmount, leftoverUserRefund);

        bool paymentSucceeded = pay(relayerRefundAddress, relayerRefundAmount);
        if(!paymentSucceeded) {
            revert DeliveryProviderCannotReceivePayment();
        }
    }

    function calcRelayerRefundAmount(
        DeliveryInstruction memory deliveryInstruction,
        LocalNative transactionFeeRefundAmount,
        LocalNative leftoverUserRefund
    ) private view returns (LocalNative) {
        return msgValue()
            // Note: instruction's target is delivery's local
            - (deliveryInstruction.requestedReceiverValue + deliveryInstruction.extraReceiverValue).asNative().asLocalNative() 
            - transactionFeeRefundAmount + leftoverUserRefund;
    }

    function payRefundToRefundAddress(
        uint16 refundChain,
        bytes32 refundAddress,
        LocalNative refundAmount,
        bytes32 relayerAddress
    ) private returns (RefundStatus) {
        // User requested refund on this chain
        if (refundChain == getChainId()) {
            return pay(payable(fromWormholeFormat(refundAddress)), refundAmount)
                ? RefundStatus.REFUND_SENT
                : RefundStatus.REFUND_FAIL;
        }

        // User requested refund on a different chain
        
        IDeliveryProvider deliveryProvider = IDeliveryProvider(fromWormholeFormat(relayerAddress));
        
        // Determine price of an 'empty' delivery
        // (Note: assumes refund chain is an EVM chain)
        LocalNative baseDeliveryPrice;
      
        try deliveryProvider.quoteDeliveryPrice(
            refundChain,
            TargetNative.wrap(0),
            encodeEvmExecutionParamsV1(getEmptyEvmExecutionParamsV1())
        ) returns (LocalNative quote, bytes memory) {
            baseDeliveryPrice = quote;
        } catch (bytes memory) {
            return RefundStatus.CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED;
        }

        // If the refundAmount is not greater than the 'empty delivery price', the refund does not go through
        if (refundAmount <= getWormholeMessageFee() + baseDeliveryPrice) {
            return RefundStatus.CROSS_CHAIN_REFUND_FAIL_NOT_ENOUGH;
        }
        
        // Request a 'send' with 'paymentForExtraReceiverValue' equal to the refund minus the 'empty delivery price'
        try IWormholeRelayerSend(address(this)).send{value: refundAmount.unwrap()}(
            refundChain,
            bytes32(0),
            bytes(""),
            TargetNative.wrap(0),
            refundAmount - getWormholeMessageFee() - baseDeliveryPrice,
            encodeEvmExecutionParamsV1(getEmptyEvmExecutionParamsV1()),
            refundChain,
            refundAddress,
            fromWormholeFormat(relayerAddress),
            new VaaKey[](0),
            CONSISTENCY_LEVEL_INSTANT
        ) returns (uint64) {
            return RefundStatus.CROSS_CHAIN_REFUND_SENT;
        } catch (bytes memory) {
            return RefundStatus.CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED;
        }
    }

    function tryDecodeExecuteInstructionError(
        bytes memory revertData
    ) private pure returns (Gas gasUsed, bool knownError) {
        uint offset = 0;
        bytes4 selector;
        // Check to see if the following decode can be performed
        if(revertData.length < 36) {
            return (Gas.wrap(0), false);
        }
        (selector, offset) = revertData.asBytes4Unchecked(offset);
        if((selector == Cancelled.selector) || (selector == DeliveryProviderReverted.selector) || (selector == DeliveryProviderPaymentFailed.selector)) {
            knownError = true;
            uint256 _gasUsed;
            (_gasUsed, offset) = revertData.asUint256Unchecked(offset);
            gasUsed = Gas.wrap(_gasUsed);
        }
    }

    function checkVaaKeysWithVAAs(
        VaaKey[] memory vaaKeys,
        bytes[] memory signedVaas
    ) private view {
        if (vaaKeys.length != signedVaas.length) {
            revert VaaKeysLengthDoesNotMatchVaasLength(vaaKeys.length, signedVaas.length);
        }

        for (uint256 i = 0; i < vaaKeys.length;) {
            IWormhole.VM memory parsedVaa = getWormhole().parseVM(signedVaas[i]);
            VaaKey memory vaaKey = vaaKeys[i];
            
            if (
                vaaKey.chainId != parsedVaa.emitterChainId
                    || vaaKey.emitterAddress != parsedVaa.emitterAddress
                    || vaaKey.sequence != parsedVaa.sequence
            ) {
                revert VaaKeysDoNotMatchVaas(uint8(i));
            }

            unchecked {
                ++i;
            }
        }
    }
}
