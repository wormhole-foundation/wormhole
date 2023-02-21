// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormholeReceiver.sol";

import "./CoreRelayerGovernance.sol";
import "./CoreRelayerStructs.sol";

contract CoreRelayer is CoreRelayerGovernance {
    using BytesLib for bytes;

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

    error FundsTooMuch();
    error MaxTransactionFeeNotEnough();
    error MsgValueTooLow(); // msg.value must cover the budget specified
    error NonceIsZero();
    error ForwardRequestFromWrongAddress();
    error NoDeliveryInProcess();
    error CantRequestMultipleForwards();
    error RelayProviderDoesNotSupportTargetChain();
    error RolloverChainNotIncluded(); // Rollover chain was not included in the forwarding request
    error ChainNotFoundInSends(uint16 chainId); // Required chain not found in the delivery requests
    error ReentrantCall();
    error InvalidEmitterInOriginalDeliveryVM(uint8 index);
    error InvalidRedeliveryVM(string reason);
    error InvalidEmitterInRedeliveryVM();
    error MismatchingRelayProvidersInRedelivery(); // The same relay provider must be specified when doing a single VAA redeliver
    error InvalidVaa(uint8 index);
    error InvalidEmitter();
    error SendNotSufficientlyFunded(); // This delivery request was not sufficiently funded, and must request redelivery
    error UnexpectedRelayer(); // Specified relayer is not the relayer delivering the message
    error InsufficientRelayerFunds(); // The relayer didn't pass sufficient funds (msg.value does not cover the necessary budget fees)
    error AlreadyDelivered(); // The message was already delivered.
    error TargetChainIsNotThisChain(uint16 targetChainId);

    function send(Send memory request, uint32 nonce, IRelayProvider provider)
        public
        payable
        returns (uint64 sequence)
    {
        Send[] memory requests = new Send[](1);
        requests[0] = request;
        MultichainSend memory container = MultichainSend({relayProviderAddress: address(provider), requests: requests});
        return multichainSend(container, nonce);
    }

    function forward(Send memory request, uint32 nonce, IRelayProvider provider) public payable {
        Send[] memory requests = new Send[](1);
        requests[0] = request;
        MultichainSend memory container = MultichainSend({relayProviderAddress: address(provider), requests: requests});
        return multichainForward(container, request.targetChain, nonce);
    }

    function resend(ResendByTx memory request, uint32 nonce, IRelayProvider provider)
        public
        payable
        returns (uint64 sequence)
    {
        (uint256 requestFee, uint256 maximumRefund, uint256 receiverValueTarget, bool isSufficient, uint8 reason) =
        verifyFunding(
            VerifyFundingCalculation({
                provider: provider,
                sourceChain: chainId(),
                targetChain: request.targetChain,
                maxTransactionFeeSource: request.newMaxTransactionFee,
                receiverValueSource: request.newReceiverValue,
                isDelivery: false
            })
        );
        if (!isSufficient) {
            if (reason == 26) {
                revert MaxTransactionFeeNotEnough();
            } else {
                revert FundsTooMuch();
            }
        }
        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();
        uint256 totalFee = requestFee + wormholeMessageFee;

        //Make sure the msg.value covers the budget they specified
        if (msg.value < totalFee) {
            revert MsgValueTooLow();
        }

        sequence = emitRedelivery(
            request,
            nonce,
            provider.getConsistencyLevel(),
            receiverValueTarget,
            maximumRefund,
            provider,
            wormhole,
            wormholeMessageFee
        );

        //Send the delivery fees to the specified address of the provider.
        pay(provider.getRewardAddress(), msg.value - wormholeMessageFee);
    }

    function emitRedelivery(
        ResendByTx memory request,
        uint32 nonce,
        uint8 consistencyLevel,
        uint256 receiverValueTarget,
        uint256 maximumRefund,
        IRelayProvider provider,
        IWormhole wormhole,
        uint256 wormholeMessageFee
    ) internal returns (uint64 sequence) {
        bytes memory instruction = convertToEncodedRedeliveryByTxHashInstruction(
            request,
            receiverValueTarget,
            maximumRefund,
            calculateTargetGasRedeliveryAmount(request.targetChain, request.newMaxTransactionFee, provider),
            provider
        );

        sequence = wormhole.publishMessage{value: wormholeMessageFee}(nonce, instruction, consistencyLevel);
    }

    /**
     * TODO: Correct this comment
     * @dev `multisend` generates a VAA with DeliveryInstructions to be delivered to the specified target
     * contract based on user parameters.
     * it parses the RelayParameters to determine the target chain ID
     * it estimates the cost of relaying the batch
     * it confirms that the user has passed enough value to pay the relayer
     * it checks that the passed nonce is not zero (VAAs with a nonce of zero will not be batched)
     * it generates a VAA with the encoded DeliveryInstructions
     */
    function multichainSend(MultichainSend memory deliveryRequests, uint32 nonce)
        public
        payable
        returns (uint64 sequence)
    {
        (uint256 totalCost, bool isSufficient, uint8 cause) = sufficientFundsHelper(deliveryRequests, msg.value);
        if (!isSufficient) {
            if (cause == 26) {
                revert MaxTransactionFeeNotEnough();
            } else if (cause == 25) {
                revert MsgValueTooLow();
            } else {
                revert FundsTooMuch();
            }
        }
        if (nonce == 0) {
            revert NonceIsZero();
        }

        // encode the DeliveryInstructions
        bytes memory container = convertToEncodedDeliveryInstructions(deliveryRequests, true);

        // emit delivery message
        IWormhole wormhole = wormhole();
        IRelayProvider provider = IRelayProvider(deliveryRequests.relayProviderAddress);
        uint256 wormholeMessageFee = wormhole.messageFee();

        sequence = wormhole.publishMessage{value: wormholeMessageFee}(nonce, container, provider.getConsistencyLevel());

        //pay fee to provider
        pay(provider.getRewardAddress(), totalCost - wormholeMessageFee);
    }

    /**
     * TODO correct this comment
     * @dev `forward` queues up a 'send' which will be executed after the present delivery is complete
     * & uses the gas refund to cover the costs.
     * contract based on user parameters.
     * it parses the RelayParameters to determine the target chain ID
     * it estimates the cost of relaying the batch
     * it confirms that the user has passed enough value to pay the relayer
     * it checks that the passed nonce is not zero (VAAs with a nonce of zero will not be batched)
     * it generates a VAA with the encoded DeliveryInstructions
     */
    function multichainForward(MultichainSend memory deliveryRequests, uint16 rolloverChain, uint32 nonce)
        public
        payable
    {
        // Can only forward while a delivery is in process.
        if (!isContractLocked()) {
            revert NoDeliveryInProcess();
        }
        if (getForwardingRequest().isValid) {
            revert CantRequestMultipleForwards();
        }

        //We want to catch malformed requests in this function, and only underfunded requests when emitting.
        verifyForwardingRequest(deliveryRequests, rolloverChain, nonce);

        bytes memory encodedMultichainSend = encodeMultichainSend(deliveryRequests);
        setForwardingRequest(
            ForwardingRequest({
                deliveryRequestsContainer: encodedMultichainSend,
                rolloverChain: rolloverChain,
                nonce: nonce,
                msgValue: msg.value,
                sender: msg.sender,
                isValid: true
            })
        );
    }

    function emitForward(uint256 refundAmount, ForwardingRequest memory forwardingRequest)
        internal
        returns (uint64, bool)
    {
        MultichainSend memory container = decodeMultichainSend(forwardingRequest.deliveryRequestsContainer);

        //Add any additional funds which were passed in to the refund amount
        refundAmount = refundAmount + forwardingRequest.msgValue;

        //make sure the refund amount covers the native gas amounts
        (uint256 totalMinimumFees, bool funded,) = sufficientFundsHelper(container, refundAmount);

        //REVISE consider deducting the cost of this process from the refund amount?

        if (funded) {
            //find the delivery instruction for the rollover chain
            uint16 rolloverInstructionIndex = findDeliveryIndex(container, forwardingRequest.rolloverChain);

            //calc how much budget is used by chains other than the rollover chain
            uint256 rolloverChainCostEstimate = container.requests[rolloverInstructionIndex].maxTransactionFee
                + container.requests[rolloverInstructionIndex].receiverValue;
            //uint256 nonrolloverBudget = totalMinimumFees - rolloverChainCostEstimate; //stack too deep
            uint256 rolloverBudget = refundAmount - (totalMinimumFees - rolloverChainCostEstimate)
                - container.requests[rolloverInstructionIndex].receiverValue;

            //overwrite the gas budget on the rollover chain to the remaining budget amount
            container.requests[rolloverInstructionIndex].maxTransactionFee = rolloverBudget;
        }

        //emit forwarding instruction
        bytes memory reencoded = convertToEncodedDeliveryInstructions(container, funded);
        IRelayProvider provider = IRelayProvider(container.relayProviderAddress);
        IWormhole wormhole = wormhole();
        uint64 sequence = wormhole.publishMessage{value: wormhole.messageFee()}(
            forwardingRequest.nonce, reencoded, provider.getConsistencyLevel()
        );

        // if funded, pay out reward to provider. Otherwise, the delivery code will handle sending a refund.
        if (funded) {
            pay(provider.getRewardAddress(), refundAmount);
        }

        //clear forwarding request from cache
        clearForwardingRequest();

        return (sequence, funded);
    }

    function verifyForwardingRequest(MultichainSend memory container, uint16 rolloverChain, uint32 nonce)
        internal
        view
    {
        if (nonce == 0) {
            revert NonceIsZero();
        }

        if (msg.sender != lockedTargetAddress()) {
            revert ForwardRequestFromWrongAddress();
        }

        bool foundRolloverChain = false;
        IRelayProvider selectedProvider = IRelayProvider(container.relayProviderAddress);

        for (uint16 i = 0; i < container.requests.length; i++) {
            // TODO: Optimization opportunity here by reducing multiple calls to only one with all requested addresses.
            if (selectedProvider.getDeliveryAddress(container.requests[i].targetChain) == 0) {
                revert RelayProviderDoesNotSupportTargetChain();
            }
            if (container.requests[i].targetChain == rolloverChain) {
                foundRolloverChain = true;
            }
        }

        if (!foundRolloverChain) {
            revert RolloverChainNotIncluded();
        }
    }

    function findDeliveryIndex(MultichainSend memory container, uint16 chainId)
        internal
        pure
        returns (uint16 deliveryRequestIndex)
    {
        for (uint16 i = 0; i < container.requests.length; i++) {
            if (container.requests[i].targetChain == chainId) {
                deliveryRequestIndex = i;
                return deliveryRequestIndex;
            }
        }

        revert ChainNotFoundInSends(chainId);
    }

    /*
    By the time this function completes, we must be certain that the specified funds are sufficient to cover
    delivery for each one of the deliveryRequests with at least 1 gas on the target chains.
    */
    function sufficientFundsHelper(MultichainSend memory deliveryRequests, uint256 funds)
        internal
        view
        returns (uint256 totalFees, bool isSufficient, uint8 reason)
    {
        totalFees = wormhole().messageFee();
        IRelayProvider provider = IRelayProvider(deliveryRequests.relayProviderAddress);

        for (uint256 i = 0; i < deliveryRequests.requests.length; i++) {
            Send memory request = deliveryRequests.requests[i];

            (uint256 requestFee, uint256 maximumRefund, uint256 receiverValueTarget, bool isSufficient, uint8 reason) =
            verifyFunding(
                VerifyFundingCalculation({
                    provider: provider,
                    sourceChain: chainId(),
                    targetChain: request.targetChain,
                    maxTransactionFeeSource: request.maxTransactionFee,
                    receiverValueSource: request.receiverValue,
                    isDelivery: true
                })
            );

            if (!isSufficient) {
                return (0, false, reason);
            }

            totalFees = totalFees + requestFee;
            if (funds < totalFees) {
                return (0, false, 25); //"Insufficient funds were provided to cover the delivery fees.");
            }
        }

        return (totalFees, true, 0);
    }

    struct VerifyFundingCalculation {
        IRelayProvider provider;
        uint16 sourceChain;
        uint16 targetChain;
        uint256 maxTransactionFeeSource;
        uint256 receiverValueSource;
        bool isDelivery;
    }

    function verifyFunding(VerifyFundingCalculation memory args)
        internal
        view
        returns (
            uint256 requestFee,
            uint256 maximumRefund,
            uint256 receiverValueTarget,
            bool isSufficient,
            uint8 reason
        )
    {
        requestFee = args.maxTransactionFeeSource + args.receiverValueSource;
        receiverValueTarget = convertApplicationBudgetAmount(args.receiverValueSource, args.targetChain, args.provider);
        uint256 overheadFeeSource = args.isDelivery
            ? args.provider.quoteDeliveryOverhead(args.targetChain)
            : args.provider.quoteRedeliveryOverhead(args.targetChain);
        uint256 overheadBudgetTarget =
            assetConversionHelper(args.sourceChain, overheadFeeSource, args.targetChain, 1, 1, true, args.provider);
        maximumRefund = args.isDelivery
            ? calculateTargetDeliveryMaximumRefund(args.targetChain, args.maxTransactionFeeSource, args.provider)
            : calculateTargetRedeliveryMaximumRefund(args.targetChain, args.maxTransactionFeeSource, args.provider);

        //Make sure the maxTransactionFee covers the minimum delivery cost to the targetChain
        if (args.maxTransactionFeeSource < overheadFeeSource) {
            isSufficient = false;
            reason = 26; //Insufficient msg.value to cover minimum delivery costs.";
        }
        //Make sure the budget does not exceed the maximum for the provider on that chain; //This added value is totalBudgetTarget
        else if (
            args.provider.quoteMaximumBudget(args.targetChain)
                < (maximumRefund + overheadBudgetTarget + receiverValueTarget)
        ) {
            isSufficient = false;
            reason = 27; //"Specified budget exceeds the maximum allowed by the provider";
        } else {
            isSufficient = true;
            reason = 0;
        }
    }

    function _executeDelivery(
        IWormhole wormhole,
        DeliveryInstruction memory internalInstruction,
        bytes[] memory encodedVMs,
        bytes32 deliveryVaaHash,
        address payable relayerRefund,
        uint16 sourceChain,
        uint64 sourceSequence
    ) internal {
        //REVISE Decide whether we want to remove the DeliveryInstructionContainer from encodedVMs.

        // lock the contract to prevent reentrancy
        if (isContractLocked()) {
            revert ReentrantCall();
        }
        setContractLock(true);
        setLockedTargetAddress(fromWormholeFormat(internalInstruction.targetAddress));
        // store gas budget pre target invocation to calculate unused gas budget
        uint256 preGas = gasleft();

        // call the receiveWormholeMessages endpoint on the target contract
        (bool success,) = fromWormholeFormat(internalInstruction.targetAddress).call{
            gas: internalInstruction.executionParameters.gasLimit,
            value: internalInstruction.receiverValueTarget
        }(abi.encodeCall(IWormholeReceiver.receiveWormholeMessages, (encodedVMs, new bytes[](0))));

        uint256 postGas = gasleft();
        // There's no easy way to measure the exact cost of the CALL instruction.
        // This is due to the fact that the compiler probably emits DUPN or MSTORE instructions
        // to setup the arguments for the call just after our measurement.
        // This means the refund could be off by a few units of gas.
        // Thus, we ensure the overhead doesn't cause an overflow in our refund formula here.
        uint256 gasUsed = (preGas - postGas) > internalInstruction.executionParameters.gasLimit
            ? internalInstruction.executionParameters.gasLimit
            : (preGas - postGas);

        // refund unused gas budget
        uint256 weiToRefund = internalInstruction.receiverValueTarget;
        if (success) {
            weiToRefund = (internalInstruction.executionParameters.gasLimit - gasUsed)
                * internalInstruction.maximumRefundTarget / internalInstruction.executionParameters.gasLimit;
        }

        // unlock the contract
        setContractLock(false);

        //REVISE decide if we want to always emit a VAA, or only emit a msg when forwarding
        // // emit delivery status message
        // DeliveryStatus memory status = DeliveryStatus({
        //     payloadID: 2,
        //     batchHash: internalParams.batchVM.hash,
        //     emitterAddress: internalParams.deliveryId.emitterAddress,
        //     sequence: internalParams.deliveryId.sequence,
        //     deliveryCount: uint16(stackTooDeep.attemptedDeliveryCount + 1),
        //     deliverySuccess: success
        // });
        // // set the nonce to zero so a batch VAA is not created
        // sequence =
        //     wormhole.publishMessage{value: wormhole.messageFee()}(0, encodeDeliveryStatus(status), consistencyLevel());
        ForwardingRequest memory forwardingRequest = getForwardingRequest();
        if (forwardingRequest.isValid) {
            (, success) = emitForward(weiToRefund, forwardingRequest);
            if (success) {
                emit Delivery({
                    recipientContract: fromWormholeFormat(internalInstruction.targetAddress),
                    sourceChain: sourceChain,
                    sequence: sourceSequence,
                    deliveryVaaHash: deliveryVaaHash,
                    status: DeliveryStatus.FORWARD_REQUEST_SUCCESS
                });
            } else {
                bool sent = pay(payable(fromWormholeFormat(internalInstruction.refundAddress)), weiToRefund);
                if (!sent) {
                    // if refunding fails, pay out full refund to relayer
                    weiToRefund = 0;
                }
                emit Delivery({
                    recipientContract: fromWormholeFormat(internalInstruction.targetAddress),
                    sourceChain: sourceChain,
                    sequence: sourceSequence,
                    deliveryVaaHash: deliveryVaaHash,
                    status: DeliveryStatus.FORWARD_REQUEST_FAILURE
                });
            }
        } else {
            bool sent = pay(payable(fromWormholeFormat(internalInstruction.refundAddress)), weiToRefund);
            if (!sent) {
                // if refunding fails, pay out full refund to relayer
                weiToRefund = 0;
            }

            if (success) {
                emit Delivery({
                    recipientContract: fromWormholeFormat(internalInstruction.targetAddress),
                    sourceChain: sourceChain,
                    sequence: sourceSequence,
                    deliveryVaaHash: deliveryVaaHash,
                    status: DeliveryStatus.SUCCESS
                });
            } else {
                emit Delivery({
                    recipientContract: fromWormholeFormat(internalInstruction.targetAddress),
                    sourceChain: sourceChain,
                    sequence: sourceSequence,
                    deliveryVaaHash: deliveryVaaHash,
                    status: DeliveryStatus.RECEIVER_FAILURE
                });
            }
        }

        uint256 receiverValuePaid = (success ? internalInstruction.receiverValueTarget : 0);
        uint256 wormholeFeePaid = forwardingRequest.isValid ? wormhole.messageFee() : 0;
        uint256 relayerRefundAmount = msg.value - weiToRefund - receiverValuePaid - wormholeFeePaid;
        // refund the rest to relayer
        pay(relayerRefund, relayerRefundAmount);
    }

    //REVISE, consider implementing this system into the RelayProvider.
    // function requestRewardPayout(uint16 rewardChain, bytes32 receiver, uint32 nonce)
    //     public
    //     payable
    //     returns (uint64 sequence)
    // {
    //     uint256 amount = relayerRewards(msg.sender, rewardChain);

    //     require(amount > 0, "no current accrued rewards");

    //     resetRelayerRewards(msg.sender, rewardChain);

    //     sequence = wormhole().publishMessage{value: msg.value}(
    //         nonce,
    //         encodeRewardPayout(
    //             RewardPayout({
    //                 payloadID: 100,
    //                 fromChain: chainId(),
    //                 chain: rewardChain,
    //                 amount: amount,
    //                 receiver: receiver
    //             })
    //         ),
    //         20 //REVISE encode finality
    //     );
    // }

    // function collectRewards(bytes memory encodedVm) public {
    //     (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

    //     require(valid, reason);
    //     require(verifyRelayerVM(vm), "invalid emitter");

    //     RewardPayout memory payout = parseRewardPayout(vm.payload);

    //     require(payout.chain == chainId());

    //     payable(address(uint160(uint256(payout.receiver)))).transfer(payout.amount);
    // }

    function verifyRelayerVM(IWormhole.VM memory vm) internal view returns (bool) {
        return registeredCoreRelayerContract(vm.emitterChainId) == vm.emitterAddress;
    }

    function getDefaultRelayProvider() public view returns (IRelayProvider) {
        return defaultRelayProvider();
    }

    function redeliverSingle(TargetRedeliveryByTxHashParamsSingle memory targetParams) public payable {
        //cache wormhole
        IWormhole wormhole = wormhole();

        //validate the redelivery VM
        (IWormhole.VM memory redeliveryVM, bool valid, string memory reason) =
            wormhole.parseAndVerifyVM(targetParams.redeliveryVM);
        if (!valid) {
            revert InvalidRedeliveryVM(reason);
        }
        if (!verifyRelayerVM(redeliveryVM)) {
            // Redelivery VM has an invalid emitter
            revert InvalidEmitterInRedeliveryVM();
        }

        RedeliveryByTxHashInstruction memory redeliveryInstruction =
            decodeRedeliveryByTxHashInstruction(redeliveryVM.payload);

        //validate the original delivery VM
        IWormhole.VM memory originalDeliveryVM;
        (originalDeliveryVM, valid, reason) =
            wormhole.parseAndVerifyVM(targetParams.sourceEncodedVMs[redeliveryInstruction.deliveryIndex]);
        if (!valid) {
            revert InvalidVaa(redeliveryInstruction.deliveryIndex);
        }
        if (!verifyRelayerVM(originalDeliveryVM)) {
            // Original Delivery VM has a invalid emitter
            revert InvalidEmitterInOriginalDeliveryVM(redeliveryInstruction.deliveryIndex);
        }

        DeliveryInstruction memory instruction;
        (instruction, valid) = validateRedeliverySingle(
            redeliveryInstruction,
            decodeDeliveryInstructionsContainer(originalDeliveryVM.payload).instructions[redeliveryInstruction
                .multisendIndex]
        );

        if (!valid) {
            emit Delivery({
                recipientContract: fromWormholeFormat(instruction.targetAddress),
                sourceChain: redeliveryVM.emitterChainId,
                sequence: redeliveryVM.sequence,
                deliveryVaaHash: redeliveryVM.hash,
                status: DeliveryStatus.INVALID_REDELIVERY
            });
            pay(targetParams.relayerRefundAddress, msg.value);
            return;
        }

        _executeDelivery(
            wormhole,
            instruction,
            targetParams.sourceEncodedVMs,
            originalDeliveryVM.hash,
            targetParams.relayerRefundAddress,
            originalDeliveryVM.emitterChainId,
            originalDeliveryVM.sequence
        );
    }

    function validateRedeliverySingle(
        RedeliveryByTxHashInstruction memory redeliveryInstruction,
        DeliveryInstruction memory originalInstruction
    ) internal view returns (DeliveryInstruction memory deliveryInstruction, bool isValid) {
        // All the same checks as delivery single, with a couple additional

        // The same relay provider must be specified when doing a single VAA redeliver.
        address providerAddress = fromWormholeFormat(redeliveryInstruction.executionParameters.providerDeliveryAddress);
        if (providerAddress != fromWormholeFormat(originalInstruction.executionParameters.providerDeliveryAddress)) {
            revert MismatchingRelayProvidersInRedelivery();
        }

        // relayer must have covered the necessary funds
        if (
            msg.value
                < redeliveryInstruction.newMaximumRefundTarget + redeliveryInstruction.newReceiverValueTarget
                    + wormhole().messageFee()
        ) {
            revert InsufficientRelayerFunds();
        }

        uint16 whChainId = chainId();
        // msg.sender must be the provider
        // "Relay provider differed from the specified address");
        isValid = msg.sender == providerAddress
        // redelivery must target this chain
        // "Redelivery request does not target this chain.");
        && whChainId == redeliveryInstruction.targetChain
        // original delivery must target this chain
        // "Original delivery request did not target this chain.");
        && whChainId == originalInstruction.targetChain
        // gasLimit & receiverValue must be at least as large as the initial delivery
        // "New receiver value is smaller than the original"
        && originalInstruction.receiverValueTarget <= redeliveryInstruction.newReceiverValueTarget
        // "New gasLimit is smaller than the original"
        && originalInstruction.executionParameters.gasLimit <= redeliveryInstruction.executionParameters.gasLimit;

        // Overwrite compute budget and application budget on the original request and proceed.
        deliveryInstruction = originalInstruction;
        deliveryInstruction.maximumRefundTarget = redeliveryInstruction.newMaximumRefundTarget;
        deliveryInstruction.receiverValueTarget = redeliveryInstruction.newReceiverValueTarget;
        deliveryInstruction.executionParameters = redeliveryInstruction.executionParameters;
    }

    function deliverSingle(TargetDeliveryParametersSingle memory targetParams) public payable {
        // cache wormhole instance
        IWormhole wormhole = wormhole();

        // validate the deliveryIndex
        (IWormhole.VM memory deliveryVM, bool valid, string memory reason) =
            wormhole.parseAndVerifyVM(targetParams.encodedVMs[targetParams.deliveryIndex]);
        if (!valid) {
            revert InvalidVaa(targetParams.deliveryIndex);
        }
        if (!verifyRelayerVM(deliveryVM)) {
            revert InvalidEmitter();
        }

        DeliveryInstructionsContainer memory container = decodeDeliveryInstructionsContainer(deliveryVM.payload);
        //ensure this is a funded delivery, not a failed forward.
        if (!container.sufficientlyFunded) {
            revert SendNotSufficientlyFunded();
        }

        // parse the deliveryVM payload into the DeliveryInstructions struct
        DeliveryInstruction memory deliveryInstruction = container.instructions[targetParams.multisendIndex];

        //make sure the specified relayer is the relayer delivering this message
        if (fromWormholeFormat(deliveryInstruction.executionParameters.providerDeliveryAddress) != msg.sender) {
            revert UnexpectedRelayer();
        }

        //make sure relayer passed in sufficient funds
        if (
            msg.value
                < deliveryInstruction.maximumRefundTarget + deliveryInstruction.receiverValueTarget + wormhole.messageFee()
        ) {
            revert InsufficientRelayerFunds();
        }

        //make sure this delivery is intended for this chain
        if (chainId() != deliveryInstruction.targetChain) {
            revert TargetChainIsNotThisChain(deliveryInstruction.targetChain);
        }

        _executeDelivery(
            wormhole,
            deliveryInstruction,
            targetParams.encodedVMs,
            deliveryVM.hash,
            targetParams.relayerRefundAddress,
            deliveryVM.emitterChainId,
            deliveryVM.sequence
        );
    }

    function toWormholeFormat(address addr) public pure returns (bytes32 whFormat) {
        return bytes32(uint256(uint160(addr)));
    }

    function fromWormholeFormat(bytes32 whFormatAddress) public pure returns (address addr) {
        return address(uint160(uint256(whFormatAddress)));
    }

    function getDefaultRelayParams() public pure returns (bytes memory relayParams) {
        return new bytes(0);
    }

    function makeRelayerParams(IRelayProvider provider) public pure returns (bytes memory relayerParams) {
        //current version is just 1,
        relayerParams = abi.encode(1, toWormholeFormat(address(provider)));
    }

    function getDeliveryInstructionsContainer(bytes memory encoded)
        public
        view
        returns (DeliveryInstructionsContainer memory container)
    {
        container = decodeDeliveryInstructionsContainer(encoded);
    }

    function getRedeliveryByTxHashInstruction(bytes memory encoded)
        public
        view
        returns (RedeliveryByTxHashInstruction memory instruction)
    {
        instruction = decodeRedeliveryByTxHashInstruction(encoded);
    }

    /**
     * Given a targetChain, maxTransactionFee, and a relay provider, this function calculates what the gas limit of the delivery transaction
     * should be.
     */
    function calculateTargetGasDeliveryAmount(uint16 targetChain, uint256 maxTransactionFee, IRelayProvider provider)
        internal
        view
        returns (uint32 gasAmount)
    {
        gasAmount = calculateTargetGasDeliveryAmountHelper(
            targetChain, maxTransactionFee, provider.quoteDeliveryOverhead(targetChain), provider
        );
    }

    function calculateTargetDeliveryMaximumRefund(
        uint16 targetChain,
        uint256 maxTransactionFee,
        IRelayProvider provider
    ) internal view returns (uint256 maximumRefund) {
        maximumRefund = calculateTargetDeliveryMaximumRefundHelper(
            targetChain, maxTransactionFee, provider.quoteDeliveryOverhead(targetChain), provider
        );
    }

    /**
     * Given a targetChain, maxTransactionFee, and a relay provider, this function calculates what the gas limit of the redelivery transaction
     * should be.
     */
    function calculateTargetGasRedeliveryAmount(uint16 targetChain, uint256 maxTransactionFee, IRelayProvider provider)
        internal
        view
        returns (uint32 gasAmount)
    {
        gasAmount = calculateTargetGasDeliveryAmountHelper(
            targetChain, maxTransactionFee, provider.quoteRedeliveryOverhead(targetChain), provider
        );
    }

    function calculateTargetRedeliveryMaximumRefund(
        uint16 targetChain,
        uint256 maxTransactionFee,
        IRelayProvider provider
    ) internal view returns (uint256 maximumRefund) {
        maximumRefund = calculateTargetDeliveryMaximumRefundHelper(
            targetChain, maxTransactionFee, provider.quoteRedeliveryOverhead(targetChain), provider
        );
    }

    function calculateTargetGasDeliveryAmountHelper(
        uint16 targetChain,
        uint256 maxTransactionFee,
        uint256 deliveryOverhead,
        IRelayProvider provider
    ) internal view returns (uint32 gasAmount) {
        if (maxTransactionFee <= deliveryOverhead) {
            gasAmount = 0;
        } else {
            uint256 gas = (maxTransactionFee - deliveryOverhead) / provider.quoteGasPrice(targetChain);
            if (gas > type(uint32).max) {
                gasAmount = type(uint32).max;
            } else {
                gasAmount = uint32(gas);
            }
        }
    }

    function calculateTargetDeliveryMaximumRefundHelper(
        uint16 targetChain,
        uint256 maxTransactionFee,
        uint256 deliveryOverhead,
        IRelayProvider provider
    ) internal view returns (uint256 maximumRefund) {
        if (maxTransactionFee >= deliveryOverhead) {
            uint256 remainder = maxTransactionFee - deliveryOverhead;
            maximumRefund = assetConversionHelper(chainId(), remainder, targetChain, 1, 1, false, provider);
        } else {
            maximumRefund = 0;
        }
    }

    function quoteGas(uint16 targetChain, uint32 gasLimit, IRelayProvider provider)
        public
        view
        returns (uint256 deliveryQuote)
    {
        deliveryQuote = provider.quoteDeliveryOverhead(targetChain) + (gasLimit * provider.quoteGasPrice(targetChain));
    }

    function quoteGasResend(uint16 targetChain, uint32 gasLimit, IRelayProvider provider)
        public
        view
        returns (uint256 redeliveryQuote)
    {
        redeliveryQuote =
            provider.quoteRedeliveryOverhead(targetChain) + (gasLimit * provider.quoteGasPrice(targetChain));
    }

    function assetConversionHelper(
        uint16 sourceChain,
        uint256 sourceAmount,
        uint16 targetChain,
        uint256 multiplier,
        uint256 multiplierDenominator,
        bool roundUp,
        IRelayProvider provider
    ) internal view returns (uint256 targetAmount) {
        uint256 srcNativeCurrencyPrice = provider.quoteAssetPrice(sourceChain);
        if (srcNativeCurrencyPrice == 0) {
            revert RelayProviderDoesNotSupportTargetChain();
        }

        uint256 dstNativeCurrencyPrice = provider.quoteAssetPrice(targetChain);
        if (dstNativeCurrencyPrice == 0) {
            revert RelayProviderDoesNotSupportTargetChain();
        }
        uint256 numerator = sourceAmount * srcNativeCurrencyPrice * multiplier;
        uint256 denominator = dstNativeCurrencyPrice * multiplierDenominator;
        if (roundUp) {
            targetAmount = (numerator + denominator - 1) / denominator;
        } else {
            targetAmount = numerator / denominator;
        }
    }

    //If the integrator pays at least nativeQuote, they should receive at least targetAmount as their application budget
    function quoteReceiverValue(uint16 targetChain, uint256 targetAmount, IRelayProvider provider)
        public
        view
        returns (uint256 nativeQuote)
    {
        (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChain);
        nativeQuote = assetConversionHelper(
            targetChain, targetAmount, chainId(), uint256(0) + denominator + buffer, denominator, true, provider
        );
    }

    //This should invert quoteApplicationBudgetAmount, I.E when a user pays the sourceAmount, they receive at least the value of targetAmount they requested from
    //quoteReceiverValue.
    function convertApplicationBudgetAmount(uint256 sourceAmount, uint16 targetChain, IRelayProvider provider)
        internal
        view
        returns (uint256 targetAmount)
    {
        (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChain);

        targetAmount = assetConversionHelper(
            chainId(), sourceAmount, targetChain, denominator, uint256(0) + denominator + buffer, false, provider
        );
    }

    function convertToEncodedRedeliveryByTxHashInstruction(
        ResendByTx memory request,
        uint256 receiverValueTarget,
        uint256 maximumRefund,
        uint32 gasLimit,
        IRelayProvider provider
    ) internal view returns (bytes memory encoded) {
        encoded = abi.encodePacked(
            uint8(2), //version payload number
            uint16(request.sourceChain),
            bytes32(request.sourceTxHash),
            uint32(request.sourceNonce),
            uint16(request.targetChain),
            uint8(request.deliveryIndex),
            uint8(request.multisendIndex),
            maximumRefund,
            receiverValueTarget,
            uint8(1), //version for ExecutionParameters
            gasLimit,
            provider.getDeliveryAddress(request.targetChain)
        );
    }

    function convertToEncodedDeliveryInstructions(MultichainSend memory container, bool isFunded)
        internal
        view
        returns (bytes memory encoded)
    {
        encoded = abi.encodePacked(
            uint8(1), //version payload number
            uint8(isFunded ? 1 : 0), // sufficiently funded
            uint8(container.requests.length) //number of requests in the array
        );

        // TODO: this probably results in a quadratic algorithm. Further optimization can be done here.
        // Append all the messages to the array.
        for (uint256 i = 0; i < container.requests.length; i++) {
            encoded = appendDeliveryInstruction(
                encoded, container.requests[i], IRelayProvider(container.relayProviderAddress)
            );
        }
    }

    function appendDeliveryInstruction(bytes memory encoded, Send memory request, IRelayProvider provider)
        internal
        view
        returns (bytes memory newEncoded)
    {
        newEncoded = abi.encodePacked(
            encoded,
            request.targetChain,
            request.targetAddress,
            request.refundAddress,
            calculateTargetDeliveryMaximumRefund(request.targetChain, request.maxTransactionFee, provider),
            convertApplicationBudgetAmount(request.receiverValue, request.targetChain, provider),
            uint8(1), //version for ExecutionParameters
            calculateTargetGasDeliveryAmount(request.targetChain, request.maxTransactionFee, provider),
            provider.getDeliveryAddress(request.targetChain)
        );
    }

    function pay(address payable receiver, uint256 amount) internal returns (bool success) {
        if (amount > 0) {
            (success,) = receiver.call{value: amount}("");
        } else {
            success = true;
        }
    }
}
