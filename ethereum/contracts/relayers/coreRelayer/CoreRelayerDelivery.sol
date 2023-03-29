// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../interfaces/IWormholeReceiver.sol";
import "../interfaces/IDelivery.sol";
import "./CoreRelayerGovernance.sol";
import "./CoreRelayerStructs.sol";

contract CoreRelayerDelivery is CoreRelayerGovernance {
    enum DeliveryStatus {
        SUCCESS,
        RECEIVER_FAILURE,
        FORWARD_REQUEST_FAILURE,
        FORWARD_REQUEST_SUCCESS,
        INVALID_REDELIVERY
    }

    event Delivery(
        address indexed recipientContract,
        uint16 indexed sourceChain,
        uint64 indexed sequence,
        bytes32 deliveryVaaHash,
        DeliveryStatus status
    );

    /**
     * - Checks if enough funds were passed into a forward
     * - Increases the maxTransactionFee of the first forward in the MultichainSend container
     *   in order to use all of the funds
     * - Publishes the DeliveryInstruction, with a 'sufficientlyFunded' flag indicating whether the forward had enough funds
     * - If the forward was funded, pay the relayer's reward address to deliver the forward
     *
     * @param transactionFeeRefundAmount amount of maxTransactionFee that was unused
     * @param forwardInstruction A struct containing information about the user's forward/multichainForward request
     *
     * @return forwardIsFunded whether or not the funds for the forward were enough
     */
    function emitForward(uint256 transactionFeeRefundAmount, ForwardInstruction memory forwardInstruction)
        internal
        returns (bool forwardIsFunded)
    {
        DeliveryInstructionsContainer memory container =
            decodeDeliveryInstructionsContainer(forwardInstruction.container);

        // Add any additional funds which were passed in to the forward as msg.value
        transactionFeeRefundAmount = transactionFeeRefundAmount + forwardInstruction.msgValue;

        // Checks if enough funds were passed into the forward
        forwardIsFunded = (transactionFeeRefundAmount >= forwardInstruction.totalFee);

        IRelayProvider relayProvider = IRelayProvider(forwardInstruction.relayProvider);
        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();

        // Increases the maxTransactionFee of the first forward in the MultichainSend container
        // in order to use all of the funds
        if (forwardIsFunded) {
            uint256 amountUnderMaximum = relayProvider.quoteMaximumBudget(container.instructions[0].targetChain)
                - (
                    wormholeMessageFee + container.instructions[0].maximumRefundTarget
                        + container.instructions[0].receiverValueTarget
                );
            uint256 convertedExtraAmount = calculateTargetDeliveryMaximumRefund(
                container.instructions[0].targetChain,
                transactionFeeRefundAmount - forwardInstruction.totalFee,
                relayProvider
            );
            container.instructions[0].maximumRefundTarget +=
                (amountUnderMaximum > convertedExtraAmount) ? convertedExtraAmount : amountUnderMaximum;
        }

        // Publishes the DeliveryInstruction, with a 'sufficientlyFunded' flag indicating whether the forward had enough funds
        container.sufficientlyFunded = forwardIsFunded;
        wormhole.publishMessage{value: wormholeMessageFee}(
            0, encodeDeliveryInstructionsContainer(container), relayProvider.getConsistencyLevel()
        );

        // if funded, pay out reward to provider. Otherwise, the delivery code will handle sending a refund.
        if (forwardIsFunded) {
            pay(relayProvider.getRewardAddress(), transactionFeeRefundAmount);
        }

        //clear forwarding request from storage
        clearForwardInstruction();
    }

    /**
     * Performs the following actions:
     * - Calls the 'receiveWormholeMessages' endpoint on the contract 'internalInstruction.targetAddress'
     * (with the gas limit and value specified in internalInstruction, and 'encodedVMs' as the input)
     *
     * - Calculates how much of 'maxTransactionFee' is left
     * - If the call succeeded and during execution of 'receiveWormholeMessages' there was a forward/multichainForward, then:
     *      if there is enough 'maxTransactionFee' left to execute the forward, then execute the forward
     *      else emit the forward instruction but with a flag (sufficientlyFunded = false) indicating that it wasn't paid for
     * - else:
     *      refund any of the 'maxTransactionFee' not used to internalInstruction.refundAddress
     *      if the call reverted, refund the 'receiverValue' to internalInstruction.refundAddress
     * - refund anything leftover to the relayer
     *
     * @param internalInstruction instruction to execute
     * @param encodedVMs list of signed wormhole messages (VAAs)
     * @param relayerRefundAddress address to send the relayer's refund to
     * @param vaaInfo struct specifying:
     *      - sourceChain chain id that the delivery originated from
     *      - sourceSequence sequence number of the delivery VAA on the source chain
     *      - deliveryVaaHash hash of delivery VAA
     */
    function _executeDelivery(
        DeliveryInstruction memory internalInstruction,
        bytes[] memory encodedVMs,
        address payable relayerRefundAddress,
        DeliveryVAAInfo memory vaaInfo
    ) internal {
        // lock the contract to prevent reentrancy
        if (isContractLocked()) {
            revert IDelivery.ReentrantCall();
        }

        setContractLock(true);
        setLockedTargetAddress(fromWormholeFormat(internalInstruction.targetAddress));

        uint256 preGas = gasleft();

        // Calls the 'receiveWormholeMessages' endpoint on the contract 'internalInstruction.targetAddress'
        // (with the gas limit and value specified in internalInstruction, and 'encodedVMs' as the input)
        (bool callToTargetContractSucceeded,) = fromWormholeFormat(internalInstruction.targetAddress).call{
            gas: internalInstruction.executionParameters.gasLimit,
            value: internalInstruction.receiverValueTarget
        }(abi.encodeCall(IWormholeReceiver.receiveWormholeMessages, (encodedVMs, new bytes[](0))));

        uint256 postGas = gasleft();

        // Calculate the amount of gas used in the call (upperbounding at the gas limit, which shouldn't have been exceeded)
        uint256 gasUsed = (preGas - postGas) > internalInstruction.executionParameters.gasLimit
            ? internalInstruction.executionParameters.gasLimit
            : (preGas - postGas);

        // Calculate the amount of maxTransactionFee to refund (multiply the maximum refund by the fraction of gas unused)
        uint256 transactionFeeRefundAmount = (internalInstruction.executionParameters.gasLimit - gasUsed)
            * internalInstruction.maximumRefundTarget / internalInstruction.executionParameters.gasLimit;

        // unlock the contract
        setContractLock(false);

        // Retrieve the forward instruction created during execution of 'receiveWormholeMessages'
        ForwardInstruction memory forwardingRequest = getForwardInstruction();
        DeliveryStatus status;

        // Represents whether or not (amount user passed into forward as msg.value) + (remaining maxTransactionFee) is enough to fund the forward
        bool forwardIsFunded = false;

        if (forwardingRequest.isValid) {
            // If the user made a forward/multichainForward request, then try to execute it
            forwardIsFunded = emitForward(transactionFeeRefundAmount, forwardingRequest);
            status = forwardIsFunded ? DeliveryStatus.FORWARD_REQUEST_SUCCESS : DeliveryStatus.FORWARD_REQUEST_FAILURE;
        } else {
            status = callToTargetContractSucceeded ? DeliveryStatus.SUCCESS : DeliveryStatus.RECEIVER_FAILURE;
        }

        // Emit a status update that can be read by a SDK
        emit Delivery({
            recipientContract: fromWormholeFormat(internalInstruction.targetAddress),
            sourceChain: vaaInfo.sourceChain,
            sequence: vaaInfo.sourceSequence,
            deliveryVaaHash: vaaInfo.deliveryVaaHash,
            status: status
        });

        payRefunds(
            internalInstruction,
            relayerRefundAddress,
            transactionFeeRefundAmount,
            callToTargetContractSucceeded,
            forwardingRequest.isValid,
            forwardIsFunded
        );
    }

    function payRefunds(
        DeliveryInstruction memory internalInstruction,
        address payable relayerRefundAddress,
        uint256 transactionFeeRefundAmount,
        bool callToTargetContractSucceeded,
        bool forwardingRequestExists,
        bool forwardWasFunded
    ) internal {
        // Amount of receiverValue that is refunded to the user (0 if the call to 'receiveWormholeMessages' did not revert, or the full receiverValue otherwise)
        uint256 receiverValueRefundAmount =
            (callToTargetContractSucceeded ? 0 : internalInstruction.receiverValueTarget);

        // Total refund to the user
        uint256 refundToRefundAddress = receiverValueRefundAmount + (forwardWasFunded ? 0 : transactionFeeRefundAmount);

        // Whether or not the refund succeeded
        bool refundPaidToRefundAddress =
            pay(payable(fromWormholeFormat(internalInstruction.refundAddress)), refundToRefundAddress);

        uint256 wormholeMessageFee = wormhole().messageFee();
        // Funds that the relayer passed as msg.value over what they needed
        uint256 extraRelayerFunds = (
            msg.value - internalInstruction.receiverValueTarget - internalInstruction.maximumRefundTarget
                - wormholeMessageFee
        );

        // Refund the relayer (their extra funds) + (the amount that the relayer spent on gas) + (the wormhole message fee if no forward was sent)
        // + (the users refund if that refund didn't succeed)
        uint256 relayerRefundAmount = extraRelayerFunds
            + (internalInstruction.maximumRefundTarget - transactionFeeRefundAmount)
            + (forwardingRequestExists ? 0 : wormholeMessageFee) + (refundPaidToRefundAddress ? 0 : refundToRefundAddress);
        pay(relayerRefundAddress, relayerRefundAmount);
    }

    function verifyRelayerVM(IWormhole.VM memory vm) internal view returns (bool) {
        return registeredCoreRelayerContract(vm.emitterChainId) == vm.emitterAddress;
    }

    /**
     * @notice The relay provider calls 'redeliverSingle' to relay messages as described by one redelivery instruction
     *
     * The instruction specifies, among other things, the target chain (must be this chain), refund address, new maximum refund (in this chain's currency),
     * new receiverValue (in this chain's currency), new upper bound on gas
     *
     * The relay provider must pass in the signed wormhole message with the new redelivery instructions (the redelivery VAA)
     * as well as the original signed wormhole messages (VAAs) from the source chain
     * as well as the original signed wormhole message with the delivery instructions (the delivery VAA)
     *
     * The messages will be relayed to the target address (with the specified gas limit and receiver value) iff the following checks are met:
     * - the redelivery VAA (targetParams.redeliveryVM) has a valid signature
     * - the redelivery VAA's emitter is one of these CoreRelayer contracts
     * - the original delivery VAA has a valid signature
     * - the original delivery VAA's emitter is one of these CoreRelayer contracts
     * - the original signed VAAs match the descriptions from the original delivery instructions (the VAA hashes match, or the emitter address, sequence number pair matches, depending on the description given)
     * - the new redelivery instruction's upper bound on gas >= the original instruction's upper bound on gas
     * - the new redelivery instruction's 'receiver value' amount >= the original instruction's 'receiver value' amount
     * - the redelivery instruction's target chain = the original instruction's target chain = this chain
     * - for the redelivery instruction, the relay provider passed in at least [(one wormhole message fee) + instruction.newMaximumRefundTarget + instruction.newReceiverValueTarget] of this chain's currency as msg.value
     * - msg.sender is the permissioned address allowed to execute this redelivery instruction
     * - msg.sender is the permissioned address allowed to execute the old instruction
     *
     * @param targetParams struct containing the signed wormhole messages and encoded redelivery instruction (and other information)
     */
    function redeliverSingle(IDelivery.TargetRedeliveryByTxHashParamsSingle memory targetParams) public payable {
        IWormhole wormhole = wormhole();

        (IWormhole.VM memory redeliveryVM, bool valid, string memory reason) =
            wormhole.parseAndVerifyVM(targetParams.redeliveryVM);

        // Check that the redelivery VAA (targetParams.redeliveryVM) has a valid signature
        if (!valid) {
            revert IDelivery.InvalidRedeliveryVM(reason);
        }

        // Check that the redelivery VAA's emitter is one of these CoreRelayer contracts
        if (!verifyRelayerVM(redeliveryVM)) {
            revert IDelivery.InvalidEmitterInRedeliveryVM();
        }

        RedeliveryByTxHashInstruction memory redeliveryInstruction = decodeRedeliveryInstruction(redeliveryVM.payload);

        // Obtain the original delivery VAA
        IWormhole.VM memory originalDeliveryVM;
        (originalDeliveryVM, valid, reason) = wormhole.parseAndVerifyVM(targetParams.originalEncodedDeliveryVAA);

        // Check that the original delivery VAA has a valid signature
        if (!valid) {
            revert IDelivery.InvalidDeliveryVaa(reason);
        }

        // Check that the original delivery VAA's emitter is one of these CoreRelayer contracts
        if (!verifyRelayerVM(originalDeliveryVM)) {
            revert IDelivery.InvalidEmitterInOriginalDeliveryVM();
        }

        DeliveryInstructionsContainer memory originalContainer =
            decodeDeliveryInstructionsContainer(originalDeliveryVM.payload);

        // Obtain the specific old instruction that was originally executed (and is meant to be re-executed with new parameters)
        // specifying the the target chain (must be this chain), target address, refund address, old maximum refund (in this chain's currency),
        // old receiverValue (in this chain's currency), old upper bound on gas, and the permissioned address allowed to execute this instruction
        DeliveryInstruction memory originalInstruction =
            originalContainer.instructions[redeliveryInstruction.multisendIndex];

        checkMessageInfosWithVAAs(originalContainer.messages, targetParams.sourceEncodedVMs);

        // Perform the following checks:
        // - the new redelivery instruction's upper bound on gas >= the original instruction's upper bound on gas
        // - the new redelivery instruction's 'receiver value' amount >= the original instruction's 'receiver value' amount
        // - the redelivery instruction's target chain = this chain
        // - the original instruction's target chain = this chain
        // - for the redelivery instruction, the relay provider passed in at least [(one wormhole message fee) + instruction.newMaximumRefundTarget + instruction.newReceiverValueTarget] of this chain's currency as msg.value
        // - msg.sender is the permissioned address allowed to execute this redelivery instruction
        // - the permissioned address allowed to execute this redelivery instruction is the permissioned address allowed to execute the old instruction
        valid = checkRedeliveryInstructionTarget(redeliveryInstruction, originalInstruction);

        // Emit an 'Invalid Redelivery' event if one of the following four checks failed:
        // - the permissioned address allowed to execute this redelivery instruction is the permissioned address allowed to execute the old instruction
        // - the original instruction's target chain = this chain
        // - the new redelivery instruction's 'receiver value' amount >= the original instruction's 'receiver value' amount
        // - the new redelivery instruction's upper bound on gas >= the original instruction's upper bound on gas
        if (!valid) {
            emit Delivery({
                recipientContract: fromWormholeFormat(originalInstruction.targetAddress),
                sourceChain: originalDeliveryVM.emitterChainId,
                sequence: originalDeliveryVM.sequence,
                deliveryVaaHash: originalDeliveryVM.hash,
                status: DeliveryStatus.INVALID_REDELIVERY
            });
            pay(targetParams.relayerRefundAddress, msg.value);
            return;
        }

        // Replace maximumRefund, receiverValue, and the gasLimit on the original request
        originalInstruction.maximumRefundTarget = redeliveryInstruction.newMaximumRefundTarget;
        originalInstruction.receiverValueTarget = redeliveryInstruction.newReceiverValueTarget;
        originalInstruction.executionParameters = redeliveryInstruction.executionParameters;

        _executeDelivery(
            originalInstruction,
            targetParams.sourceEncodedVMs,
            targetParams.relayerRefundAddress,
            DeliveryVAAInfo({
                sourceChain: originalDeliveryVM.emitterChainId,
                sourceSequence: originalDeliveryVM.sequence,
                deliveryVaaHash: originalDeliveryVM.hash
            })
        );
    }

    /**
     * Check that:
     * - the new redelivery instruction's upper bound on gas >= the original instruction's upper bound on gas
     * - the new redelivery instruction's 'receiver value' amount >= the original instruction's 'receiver value' amount
     * - the redelivery instruction's target chain = this chain
     * - the original instruction's target chain = this chain
     * - for the redelivery instruction, the relay provider passed in at least [(one wormhole message fee) + instruction.newMaximumRefundTarget + instruction.newReceiverValueTarget] of this chain's currency as msg.value
     * - msg.sender is the permissioned address allowed to execute this redelivery instruction
     * - the permissioned address allowed to execute this redelivery instruction is the permissioned address allowed to execute the old instruction
     * @param redeliveryInstruction redelivery instruction
     * @param originalInstruction old instruction
     */
    function checkRedeliveryInstructionTarget(
        RedeliveryByTxHashInstruction memory redeliveryInstruction,
        DeliveryInstruction memory originalInstruction
    ) internal view returns (bool isValid) {
        address providerAddress = fromWormholeFormat(redeliveryInstruction.executionParameters.providerDeliveryAddress);

        // Check that msg.sender is the permissioned address allowed to execute this redelivery instruction
        if (providerAddress != msg.sender) {
            revert IDelivery.UnexpectedRelayer();
        }

        uint16 whChainId = chainId();

        // Check that the redelivery instruction's target chain = this chain
        if (whChainId != redeliveryInstruction.targetChain) {
            revert IDelivery.TargetChainIsNotThisChain(redeliveryInstruction.targetChain);
        }

        uint256 wormholeMessageFee = wormhole().messageFee();

        // Check that for the redelivery instruction, the relay provider passed in at least [(one wormhole message fee) + instruction.newMaximumRefundTarget + instruction.newReceiverValueTarget] of this chain's currency as msg.value
        if (
            msg.value
                < redeliveryInstruction.newMaximumRefundTarget + redeliveryInstruction.newReceiverValueTarget
                    + wormholeMessageFee
        ) {
            revert IDelivery.InsufficientRelayerFunds();
        }

        // Check that the permissioned address allowed to execute this redelivery instruction is the permissioned address allowed to execute the old instruction
        isValid = (
            providerAddress == fromWormholeFormat(originalInstruction.executionParameters.providerDeliveryAddress)
        )
        // Check that the original instruction's target chain = this chain
        && whChainId == originalInstruction.targetChain
        // Check that the new redelivery instruction's 'receiver value' amount >= the original instruction's 'receiver value' amount
        && originalInstruction.receiverValueTarget <= redeliveryInstruction.newReceiverValueTarget
        // Check that the new redelivery instruction's upper bound on gas >= the original instruction's upper bound on gas
        && originalInstruction.executionParameters.gasLimit <= redeliveryInstruction.executionParameters.gasLimit;
    }

    /**
     * @notice The relay provider calls 'deliverSingle' to relay messages as described by one delivery instruction
     *
     * The instruction specifies the target chain (must be this chain), target address, refund address, maximum refund (in this chain's currency),
     * receiver value (in this chain's currency), upper bound on gas, and the permissioned address allowed to execute this instruction
     *
     * The relay provider must pass in the signed wormhole messages (VAAs) from the source chain
     * as well as the signed wormhole message with the delivery instructions (the delivery VAA)
     * as well as identify which of the many instructions in the multichainSend container is meant to be executed
     *
     * The messages will be relayed to the target address (with the specified gas limit and receiver value) iff the following checks are met:
     * - the delivery VAA has a valid signature
     * - the delivery VAA's emitter is one of these CoreRelayer contracts
     * - the delivery instruction container in the delivery VAA was fully funded
     * - msg.sender is the permissioned address allowed to execute this instruction
     * - the relay provider passed in at least [(one wormhole message fee) + instruction.maximumRefundTarget + instruction.receiverValueTarget] of this chain's currency as msg.value
     * - the instruction's target chain is this chain
     * - the relayed signed VAAs match the descriptions in container.messages (the VAA hashes match, or the emitter address, sequence number pair matches, depending on the description given)
     *
     * @param targetParams struct containing the signed wormhole messages and encoded delivery instruction container (and other information)
     */
    function deliverSingle(IDelivery.TargetDeliveryParametersSingle memory targetParams) public payable {
        IWormhole wormhole = wormhole();

        // Obtain the delivery VAA
        (IWormhole.VM memory deliveryVM, bool valid, string memory reason) =
            wormhole.parseAndVerifyVM(targetParams.encodedDeliveryVAA);

        // Check that the delivery VAA has a valid signature
        if (!valid) {
            revert IDelivery.InvalidDeliveryVaa(reason);
        }

        // Check that the delivery VAA's emitter is one of these CoreRelayer contracts
        if (!verifyRelayerVM(deliveryVM)) {
            revert IDelivery.InvalidEmitter();
        }

        DeliveryInstructionsContainer memory container = decodeDeliveryInstructionsContainer(deliveryVM.payload);

        // Check that the delivery instruction container in the delivery VAA was fully funded
        if (!container.sufficientlyFunded) {
            revert IDelivery.SendNotSufficientlyFunded();
        }

        // Obtain the specific instruction that is intended to be executed in this function
        // specifying the the target chain (must be this chain), target address, refund address, maximum refund (in this chain's currency),
        // receiverValue (in this chain's currency), upper bound on gas, and the permissioned address allowed to execute this instruction
        DeliveryInstruction memory deliveryInstruction = container.instructions[targetParams.multisendIndex];

        // Check that msg.sender is the permissioned address allowed to execute this instruction
        if (fromWormholeFormat(deliveryInstruction.executionParameters.providerDeliveryAddress) != msg.sender) {
            revert IDelivery.UnexpectedRelayer();
        }

        uint256 wormholeMessageFee = wormhole.messageFee();

        // Check that the relay provider passed in at least [(one wormhole message fee) + instruction.maximumRefund + instruction.receiverValue] of this chain's currency as msg.value
        if (
            msg.value
                < deliveryInstruction.maximumRefundTarget + deliveryInstruction.receiverValueTarget + wormholeMessageFee
        ) {
            revert IDelivery.InsufficientRelayerFunds();
        }

        // Check that the instruction's target chain is this chain
        if (chainId() != deliveryInstruction.targetChain) {
            revert IDelivery.TargetChainIsNotThisChain(deliveryInstruction.targetChain);
        }

        // Check that the relayed signed VAAs match the descriptions in container.messages (the VAA hashes match, or the emitter address, sequence number pair matches, depending on the description given)
        checkMessageInfosWithVAAs(container.messages, targetParams.encodedVMs);

        _executeDelivery(
            deliveryInstruction,
            targetParams.encodedVMs,
            targetParams.relayerRefundAddress,
            DeliveryVAAInfo({
                sourceChain: deliveryVM.emitterChainId,
                sourceSequence: deliveryVM.sequence,
                deliveryVaaHash: deliveryVM.hash
            })
        );
    }

    /**
     * @notice checkMessageInfosWithVAAs checks that the array of signed VAAs 'signedVaas' matches the descriptions
     * given by the array of MessageInfo structs 'messageInfos'
     *
     * @param messageInfos Array of MessageInfo structs, each describing a wormhole message (VAA)
     * @param signedVaas Array of signed wormhole messages (signed VAAs)
     */
    function checkMessageInfosWithVAAs(IWormholeRelayer.MessageInfo[] memory messageInfos, bytes[] memory signedVaas)
        internal
        view
    {
        if (messageInfos.length != signedVaas.length) {
            revert IDelivery.MessageInfosLengthDoesNotMatchVaasLength();
        }
        for (uint8 i = 0; i < messageInfos.length; i++) {
            if (!messageInfoMatchesVAA(messageInfos[i], signedVaas[i])) {
                revert IDelivery.MessageInfosDoNotMatchVaas(i);
            }
        }
    }

    /**
     * @notice messageInfosWithVAAs checks that signedVaa matches the description given by 'messageInfo'
     * Specifically, if 'messageInfo.infoType' is MessageInfoType.EMITTER_SEQUENCE, then we check
     * if the emitterAddress and sequence match
     * else if 'messageInfo.infoType' is MessageInfoType.EMITTER_SEQUENCE, then we check if the VAA hash matches
     *
     * @param messageInfo MessageInfo struct describing a wormhole message (VAA)
     * @param signedVaa signed wormhole message
     */
    function messageInfoMatchesVAA(IWormholeRelayer.MessageInfo memory messageInfo, bytes memory signedVaa)
        internal
        view
        returns (bool)
    {
        IWormhole.VM memory parsedVaa = wormhole().parseVM(signedVaa);
        if (messageInfo.infoType == IWormholeRelayer.MessageInfoType.EMITTER_SEQUENCE) {
            return
                (messageInfo.emitterAddress == parsedVaa.emitterAddress) && (messageInfo.sequence == parsedVaa.sequence);
        } else if (messageInfo.infoType == IWormholeRelayer.MessageInfoType.VAAHASH) {
            return (messageInfo.vaaHash == parsedVaa.hash);
        } else {
            return false;
        }
    }

    /**
     * @notice Helper function that converts an EVM address to wormhole format
     * @param addr (EVM 20-byte address)
     * @return whFormat (32-byte address in Wormhole format)
     */
    function toWormholeFormat(address addr) public pure returns (bytes32 whFormat) {
        return bytes32(uint256(uint160(addr)));
    }

    /**
     * @notice Helper function that converts an Wormhole format (32-byte) address to the EVM 'address' 20-byte format
     * @param whFormatAddress (32-byte address in Wormhole format)
     * @return addr (EVM 20-byte address)
     */
    function fromWormholeFormat(bytes32 whFormatAddress) public pure returns (address addr) {
        return address(uint160(uint256(whFormatAddress)));
    }

    function pay(address payable receiver, uint256 amount) internal returns (bool success) {
        if (amount > 0) {
            (success,) = receiver.call{value: amount}("");
        } else {
            success = true;
        }
    }
}
