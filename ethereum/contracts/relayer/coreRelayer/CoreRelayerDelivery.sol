// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/relayer/IWormholeReceiver.sol";
import "../../interfaces/relayer/IDelivery.sol";
import "../../interfaces/relayer/IForwardWrapper.sol";
import "./CoreRelayerGovernance.sol";
import "../../interfaces/relayer/IWormholeRelayerInternalStructs.sol";
import "./CoreRelayerMessages.sol";

contract CoreRelayerDelivery is CoreRelayerGovernance {
    enum DeliveryStatus {
        SUCCESS,
        RECEIVER_FAILURE,
        FORWARD_REQUEST_FAILURE,
        FORWARD_REQUEST_SUCCESS
    }

    event Delivery(
        address indexed recipientContract,
        uint16 indexed sourceChain,
        uint64 indexed sequence,
        bytes32 deliveryVaaHash,
        DeliveryStatus status
    );

    event Redelivery(
        bytes32 indexed redeliveryVaaHash,
        uint256 maximumRefund,
        uint256 receiverValue,
        uint32 gasAmount
    );

    /**
     * - Checks if enough funds were passed into a forward
     * - Increases the maxTransactionFee of the first forward in the MultichainSend container
     *   in order to use all of the funds
     * - Publishes the DeliveryInstruction 
     * - Pays the relayer's reward address to deliver the forward
     *
     * @param transactionFeeRefundAmount amount of maxTransactionFee that was unused
     * @param forwardInstructions A struct containing information about the user's forward/multichainForward request
     *
     */
    function emitForward(
        uint256 transactionFeeRefundAmount,
        IWormholeRelayerInternalStructs.ForwardInstruction[] memory forwardInstructions
    ) internal returns (uint256 remainingRefundAmount) {

        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();

        IWormholeRelayer.Send[] memory sendRequests = new IWormholeRelayer.Send[](forwardInstructions.length);
        uint256 totalMsgValue = 0;
        uint256 totalFee = 0;
        for(uint8 i=0; i<forwardInstructions.length; i++) {
            totalMsgValue += forwardInstructions[i].msgValue;
            totalFee += forwardInstructions[i].totalFee;
            sendRequests[i] = decodeSend(forwardInstructions[i].encodedSend);
        }

        // Add any additional funds which were passed in to the forward as msg.value
        uint256 fundsForForward = transactionFeeRefundAmount + totalMsgValue;

        // Checks if enough funds were passed into the forward (should always be true as it was already checked)
        if (fundsForForward < totalFee) {
            revert IDelivery.ForwardNotSufficientlyFunded(fundsForForward, totalFee);
        }

        // Increases the maxTransactionFee of the first forward 
        // in order to use all of the funds

        uint256 increaseAmount = amountToIncreaseMaxTransactionFeeToStayUnderMaximumBudget(sendRequests[0], (fundsForForward - totalFee));
        sendRequests[0].maxTransactionFee += increaseAmount;
        
        // Publishes the DeliveryInstruction
        for(uint8 i=0; i<forwardInstructions.length; i++) {
            wormhole.publishMessage{value: wormholeMessageFee}(
                0, encodeDeliveryInstruction(convertSendToDeliveryInstruction(sendRequests[i])), sendRequests[i].consistencyLevel
            );
            pay(IRelayProvider(sendRequests[i].relayProviderAddress).getRewardAddress(), sendRequests[i].maxTransactionFee + sendRequests[i].receiverValue);
        }

        return (fundsForForward - totalFee) - increaseAmount;
    }

    function amountToIncreaseMaxTransactionFeeToStayUnderMaximumBudget(IWormholeRelayer.Send memory sendParams, uint256 increaseAmount) internal view returns (uint256) {
        IRelayProvider relayProvider = IRelayProvider(sendParams.relayProviderAddress);

        (uint16 buffer, uint16 denominator) = relayProvider.getAssetConversionBuffer(sendParams.targetChain);
        uint256 maxPaymentUnderMaximumBudget = relayProvider.quoteMaximumBudget(sendParams.targetChain) * (denominator + buffer) / denominator - sendParams.maxTransactionFee - sendParams.receiverValue;
        return increaseAmount > maxPaymentUnderMaximumBudget ? maxPaymentUnderMaximumBudget : increaseAmount;
    }

    /**
     * Performs the following actions:
     * - Calls the 'receiveWormholeMessages' endpoint on the contract 'internalInstruction.targetAddress'
     * (with the gas limit and value specified in internalInstruction, and 'encodedVMs' as the input)
     *
     * - Calculates how much of 'maxTransactionFee' is left
     * - If the call succeeded and during execution of 'receiveWormholeMessages' there was a forward/multichainForward, then:
     *      if there is enough 'maxTransactionFee' left to execute the forward, then execute the forward.
     * - else:
     *      revert the delivery to trigger a forwarding failure
     *      refund any of the 'maxTransactionFee' not used to internalInstruction.refundAddress
     *      if the call reverted, refund the 'receiverValue' to internalInstruction.refundAddress
     * - refund anything leftover to the relayer
     *
     * @param vaaInfo struct specifying:
     *      - sourceChain chain id that the delivery originated from
     *      - sourceSequence sequence number of the delivery VAA on the source chain
     *      - deliveryVaaHash hash of delivery VAA
     *      - relayerRefundAddress address that should be paid for relayer refunds
     *      - encodedVMs list of signed wormhole messages (VAAs)
     *      - deliveryContainer the container with all delivery instructions
     *      - internalInstruction the specific instruction which is being executed.
     */
    function _executeDelivery(
        IWormholeRelayerInternalStructs.DeliveryVAAInfo memory vaaInfo
    ) internal {
        if (vaaInfo.internalInstruction.targetAddress == 0x0) {
            payRefunds(
                vaaInfo.internalInstruction, vaaInfo.relayerRefundAddress, vaaInfo.internalInstruction.maximumRefundTarget, false, vaaInfo.internalInstruction.maximumRefundTarget, vaaInfo.internalInstruction.targetRelayProvider
            );
            return;
        }
        if (isContractLocked()) {
            revert IDelivery.ReentrantCall();
        }

        setContractLock(true);
        setLockedTargetAddress(fromWormholeFormat(vaaInfo.internalInstruction.targetAddress));
        clearForwardInstructions();

        uint256 preGas = gasleft();

        (bool callToInstructionExecutorSucceeded, bytes memory data) = getWormholeRelayerCallerAddress().call{
            value: vaaInfo.internalInstruction.receiverValueTarget
        }(abi.encodeWithSelector(IForwardWrapper.executeInstruction.selector, vaaInfo.internalInstruction, IWormholeReceiver.DeliveryData({
            sourceAddress: vaaInfo.internalInstruction.senderAddress,
            sourceChain: vaaInfo.sourceChain,
            maximumRefund: vaaInfo.internalInstruction.maximumRefundTarget,
            deliveryHash: vaaInfo.deliveryVaaHash,
            payload: vaaInfo.internalInstruction.payload
        }), vaaInfo.encodedVMs));

        uint256 postGas = gasleft();

        uint256 transactionFeeRefundAmount;
        bool callToTargetContractSucceeded = true;
        if (callToInstructionExecutorSucceeded) {
            (callToTargetContractSucceeded, transactionFeeRefundAmount) = abi.decode(data, (bool, uint256));
        } else {
            // Calculate the amount of gas used in the call (upperbounding at the gas limit, which shouldn't have been exceeded)
            uint256 gasUsed = (preGas - postGas) > vaaInfo.internalInstruction.executionParameters.gasLimit
                ? vaaInfo.internalInstruction.executionParameters.gasLimit
                : (preGas - postGas);

            // Calculate the amount of maxTransactionFee to refund (multiply the maximum refund by the fraction of gas unused)
            transactionFeeRefundAmount = (vaaInfo.internalInstruction.executionParameters.gasLimit - gasUsed)
                * vaaInfo.internalInstruction.maximumRefundTarget / vaaInfo.internalInstruction.executionParameters.gasLimit;
        }

        // Retrieve the forward instruction created during execution of 'receiveWormholeMessages'
        IWormholeRelayerInternalStructs.ForwardInstruction[] memory forwardInstructions = getForwardInstructions();

        //clear forwarding request from storage
        clearForwardInstructions();

        // unlock the contract
        setContractLock(false);

        DeliveryStatus status;
        uint256 transactionFeeRefundAmountPostForward = transactionFeeRefundAmount;
        if (forwardInstructions.length > 0) {
            // If the user made a forward/multichainForward request, then try to execute it
            transactionFeeRefundAmountPostForward = emitForward(transactionFeeRefundAmount, forwardInstructions);
            status = DeliveryStatus.FORWARD_REQUEST_SUCCESS;
        } else {
            status = callToTargetContractSucceeded
                ? (callToInstructionExecutorSucceeded ? DeliveryStatus.SUCCESS : DeliveryStatus.FORWARD_REQUEST_FAILURE)
                : DeliveryStatus.RECEIVER_FAILURE;
        }

        // Emit a status update that can be read by a SDK
        emit Delivery({
            recipientContract: fromWormholeFormat(vaaInfo.internalInstruction.targetAddress),
            sourceChain: vaaInfo.sourceChain,
            sequence: vaaInfo.sourceSequence,
            deliveryVaaHash: vaaInfo.deliveryVaaHash,
            status: status
        });

        //If the relayer provided a redelivery address, log a redelivery event
        if(vaaInfo.redeliveryHash != 0x0){
            emit Redelivery(vaaInfo.redeliveryHash, vaaInfo.internalInstruction.maximumRefundTarget , vaaInfo.internalInstruction.receiverValueTarget, vaaInfo.internalInstruction.executionParameters.gasLimit);
        }

        payRefunds(
            vaaInfo.internalInstruction,
            vaaInfo.relayerRefundAddress,
            transactionFeeRefundAmount,
            callToInstructionExecutorSucceeded && callToTargetContractSucceeded,
            transactionFeeRefundAmountPostForward,
            vaaInfo.internalInstruction.targetRelayProvider
        );
    }

    function payRefunds(
        IWormholeRelayerInternalStructs.DeliveryInstruction memory internalInstruction,
        address payable relayerRefundAddress,
        uint256 transactionFeeRefundAmount,
        bool receiverValueWasPaid,
        uint256 transactionFeeRefundAmountPostForward,
        bytes32 providerAddress
    ) internal {
        // Amount of receiverValue that is refunded to the user (0 if the call to 'receiveWormholeMessages' did not revert, or the full receiverValue otherwise)
        uint256 receiverValueRefundAmount = (receiverValueWasPaid ? 0 : internalInstruction.receiverValueTarget);

        // Total refund to the user
        uint256 refundToRefundAddress =
            receiverValueRefundAmount + transactionFeeRefundAmountPostForward;

        // Whether or not the refund succeeded
        bool refundPaidToRefundAddress = payRefundToRefundAddress(internalInstruction, refundToRefundAddress, providerAddress);

        // Refund the relayer (their extra funds) + (the amount that the relayer spent on gas)
        // + (the users refund if that refund didn't succeed)
        uint256 relayerRefundAmount = (
            msg.value - internalInstruction.receiverValueTarget - internalInstruction.maximumRefundTarget
        ) + (internalInstruction.maximumRefundTarget - transactionFeeRefundAmount)
            + (refundPaidToRefundAddress ? 0 : refundToRefundAddress);

        pay(relayerRefundAddress, relayerRefundAmount);
    }

    function payRefundToRefundAddress(
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction,
        uint256 refundAmount,
        bytes32 relayerAddress
    ) internal returns (bool refundPaidToRefundAddress) {
        if (instruction.refundChain == chainId()) {
            refundPaidToRefundAddress = pay(payable(fromWormholeFormat(instruction.refundAddress)), refundAmount);
        } else {
            IRelayProvider provider = IRelayProvider(fromWormholeFormat(relayerAddress));

            (bool success, bytes memory data) = getWormholeRelayerCallerAddress().call(
                abi.encodeWithSelector(IForwardWrapper.safeRelayProviderSupportsChain.selector, provider, instruction.refundChain));

            if(!success){
                return false;
            }

            success = abi.decode(data, (bool));

            if(!success){
                return false;
            }
            refundPaidToRefundAddress = payRefundRemote(instruction, refundAmount, provider);   
        }
    }

    function payRefundRemote(
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction,
        uint256 refundAmount,
        IRelayProvider provider
    ) internal returns(bool){
        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();
        uint256 overhead = wormholeMessageFee + provider.quoteDeliveryOverhead(instruction.refundChain);

        if (refundAmount > overhead) {
            IWormholeRelayerInternalStructs.DeliveryInstruction memory refundInstruction = getInstructionForEmptyMessageWithReceiverValue(
                        instruction.refundChain, instruction.refundAddress, refundAmount - overhead, provider
            );
            if(refundInstruction.receiverValueTarget == 0) {
                return false;
            }
            wormhole.publishMessage{value: wormholeMessageFee}(
                0,
                encodeDeliveryInstruction(
                    refundInstruction
                ),
                refundInstruction.consistencyLevel
            );

            pay(provider.getRewardAddress(), refundAmount - wormholeMessageFee);

            return true;
            
        } else {
            return false;
        }
        
    }

    function getInstructionForEmptyMessageWithReceiverValue(
        uint16 targetChain,
        bytes32 targetAddress,
        uint256 receiverValue,
        IRelayProvider provider
    ) internal view returns (IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction) {
        instruction = convertSendToDeliveryInstruction(

                IWormholeRelayer.Send({
                    targetChain: targetChain,
                    targetAddress: bytes32(0x0),
                    refundAddress: targetAddress,
                    refundChain: targetChain,
                    maxTransactionFee: 0,
                    receiverValue: receiverValue,
                    relayProviderAddress: address(provider),
                    vaaKeys: new IWormholeRelayer.VaaKey[](0),
                    consistencyLevel: 200, //send message instantly
                    payload: bytes(""),
                    relayParameters: bytes("")
                })
                
        );

        uint256 maximumBudget = provider.quoteMaximumBudget(targetChain);
        if (instruction.receiverValueTarget > maximumBudget) {
            instruction.receiverValueTarget = maximumBudget;
        }
    }

    function verifyRelayerVM(IWormhole.VM memory vm) internal view returns (bool) {
        return registeredCoreRelayerContract(vm.emitterChainId) == vm.emitterAddress;
    }

    /**
     * @notice The relay provider calls 'deliver' to relay messages as described by one delivery instruction
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
    function deliver(IDelivery.TargetDeliveryParameters memory targetParams) public payable {
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

        IWormholeRelayerInternalStructs.DeliveryInstruction memory deliveryInstruction =
            decodeDeliveryInstruction(deliveryVM.payload);

        bytes32 redeliveryHash = 0x0;
        (deliveryInstruction,redeliveryHash) = processOverrides(deliveryInstruction, targetParams.overrides);

        // Check that the relay provider passed in at least [(one wormhole message fee) + instruction.maximumRefund + instruction.receiverValue] of this chain's currency as msg.value
        if (msg.value < deliveryInstruction.maximumRefundTarget + deliveryInstruction.receiverValueTarget) {
            revert IDelivery.InsufficientRelayerFunds();
        }

        // Check that the instruction's target chain is this chain
        if (chainId() != deliveryInstruction.targetChain) {
            revert IDelivery.TargetChainIsNotThisChain(deliveryInstruction.targetChain);
        }

        // Check that the relayed signed VAAs match the descriptions in container.messages (the VAA hashes match, or the emitter address, sequence number pair matches, depending on the description given)
        checkVaaKeysWithVAAs(deliveryInstruction.vaaKeys, targetParams.encodedVMs);

        _executeDelivery(
            IWormholeRelayerInternalStructs.DeliveryVAAInfo({
                sourceChain: deliveryVM.emitterChainId,
                sourceSequence: deliveryVM.sequence,
                deliveryVaaHash: deliveryVM.hash,
                relayerRefundAddress: targetParams.relayerRefundAddress,
                encodedVMs: targetParams.encodedVMs,
                internalInstruction: deliveryInstruction,
                redeliveryHash: redeliveryHash
            })
        );
    }

    /**
     * @notice checkVaaKeysWithVAAs checks that the array of signed VAAs 'signedVaas' matches the descriptions
     * given by the array of VaaKey structs 'vaaKeys'
     *
     * @param vaaKeys Array of VaaKey structs, each describing a wormhole message (VAA)
     * @param signedVaas Array of signed wormhole messages (signed VAAs)
     */
    function checkVaaKeysWithVAAs(IWormholeRelayer.VaaKey[] memory vaaKeys, bytes[] memory signedVaas)
        internal
        view
    {
        if (vaaKeys.length != signedVaas.length) {
            revert IDelivery.VaaKeysLengthDoesNotMatchVaasLength();
        }
        for (uint8 i = 0; i < vaaKeys.length; i++) {
            if (!vaaKeyMatchesVAA(vaaKeys[i], signedVaas[i])) {
                revert IDelivery.VaaKeysDoNotMatchVaas(i);
            }
        }
    }

    function processOverrides(IWormholeRelayerInternalStructs.DeliveryInstruction memory deliveryInstruction, bytes memory encoded) internal pure returns (IWormholeRelayerInternalStructs.DeliveryInstruction memory withOverrides, bytes32 redelivery){
        if(encoded.length == 0){
            return (deliveryInstruction, 0x0);
        } else {
            IDelivery.DeliveryOverride memory overrides = decodeDeliveryOverride(encoded);
            require(overrides.gasLimit >= deliveryInstruction.executionParameters.gasLimit);
            require(overrides.receiverValue >= deliveryInstruction.receiverValueTarget );
            require(overrides.maximumRefund >= deliveryInstruction.maximumRefundTarget);

            deliveryInstruction.executionParameters.gasLimit = overrides.gasLimit;
            deliveryInstruction.receiverValueTarget = overrides.receiverValue;
            deliveryInstruction.maximumRefundTarget = overrides.maximumRefund;
            return (deliveryInstruction, overrides.redeliveryHash);
        }
    }

    /**
     * @notice vaaKeysWithVAAs checks that signedVaa matches the description given by 'vaaKey'
     * Specifically, if 'vaaKey.infoType' is VaaKeyType.EMITTER_SEQUENCE, then we check
     * if the emitterAddress and sequence match
     * else if 'vaaKey.infoType' is VaaKeyType.EMITTER_SEQUENCE, then we check if the VAA hash matches
     *
     * @param vaaKey VaaKey struct describing a wormhole message (VAA)
     * @param signedVaa signed wormhole message
     */
    function vaaKeyMatchesVAA(IWormholeRelayer.VaaKey memory vaaKey, bytes memory signedVaa)
        internal
        view
        returns (bool)
    {
        IWormhole.VM memory parsedVaa = wormhole().parseVM(signedVaa);
        if (vaaKey.infoType == IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE) {
            return
                (vaaKey.chainId == parsedVaa.emitterChainId) && (vaaKey.emitterAddress == parsedVaa.emitterAddress) && (vaaKey.sequence == parsedVaa.sequence);
        } else if (vaaKey.infoType == IWormholeRelayer.VaaKeyType.VAAHASH) {
            return (vaaKey.vaaHash == parsedVaa.hash);
        } else {
            return false;
        }
    }

    function pay(address payable receiver, uint256 amount) internal returns (bool success) {
        if (amount > 0) {
            (success,) = receiver.call{value: amount}("");
        } else {
            success = true;
        }
    }

    receive() external payable {}
}
