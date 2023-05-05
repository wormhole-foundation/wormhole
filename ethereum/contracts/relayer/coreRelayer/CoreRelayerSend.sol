// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/relayer/IWormholeRelayer.sol";
import "./Utils.sol";
import "./CoreRelayerMessages.sol";
import "./CoreRelayerSetters.sol";
import "../../interfaces/relayer/IWormholeRelayerInternalStructs.sol";

abstract contract CoreRelayerSend is CoreRelayerMessages, CoreRelayerSetters {
    /**
     *  @notice the 'send' function emits a wormhole message (VAA) that instructs the default wormhole relay provider to
     *  call the 'IWormholeReceiver.receiveWormholeMessage' method of the contract on chain 'sendParams.targetChain' and address 'sendParams.targetAddress'
     *
     *  @param sendParams The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, relayProviderAddress, vaaKeys, consistencyLevel, payload, and relayParameters
     *
     *  This function must be called with a payment of exactly sendParams.maxTransactionFee + sendParams.receiverValue + one wormhole message fee.
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for your specified relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described.
     */
    function send(IWormholeRelayer.Send memory sendParams)
        public
        payable
        returns (uint64 sequence)
    {
        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();
        uint256 totalFee =
            sendParams.maxTransactionFee + sendParams.receiverValue + wormholeMessageFee;

        if (totalFee > msg.value) {
            revert IWormholeRelayer.MsgValueTooLow();
        } else if (msg.value > totalFee) {
            revert IWormholeRelayer.MsgValueTooHigh();
        }

        IRelayProvider relayProvider = IRelayProvider(sendParams.relayProviderAddress);

        if (!relayProvider.isChainSupported(sendParams.targetChain)) {
            revert IWormholeRelayer.RelayProviderDoesNotSupportTargetChain();
        }

        // Calculate how much gas the relay provider can pay for on 'sendParams.targetChain' using 'sendParams.newTransactionFee',
        // and calculate how much value the relay provider will pass into 'sendParams.targetAddress'
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction =
            convertSendToDeliveryInstruction(sendParams);

        // Check that the total amount of value the relay provider needs to use for this send is <= the relayProvider's maximum budget for 'targetChain'
        // and check that the calculated gas is greater than 0
        checkInstruction(instruction, relayProvider);

        // Publish a wormhole message instructing the relay provider
        // to relay this request to the specified chains
        sequence = wormhole.publishMessage{value: wormholeMessageFee}(
            0, encodeDeliveryInstruction(instruction), sendParams.consistencyLevel
        );

        emit Send(sequence, sendParams.maxTransactionFee, sendParams.receiverValue);

        // Pay the relay provider
        Utils.pay(relayProvider.getRewardAddress(), totalFee - wormholeMessageFee);
    }

    /**
     * @notice This 'forward' function can only be called in a IWormholeReceiver within the 'receiveWormholeMessages' function
     * It's purpose is to use any leftover fee from the 'maxTransactionFee' of the current delivery to fund another delivery
     *
     * @dev Specifically, suppose an integrator requested a Send (with parameters oldTargetChain, oldTargetAddress, etc)
     * and sets quoteGas(oldTargetChain, gasLimit, oldRelayProvider) as 'maxTransactionFee' in a Send,
     * but during the delivery on oldTargetChain, the call to oldTargetAddress's receiveWormholeMessages endpoint uses only x units of gas (where x < gasLimit).
     *
     * @dev Normally, (gasLimit - x)/gasLimit * oldMaxTransactionFee, converted to target chain currency, would be refunded to 'oldRefundAddress'.
     * However, if during execution of receiveWormholeMessage the integrator made a call to forward.
     * We instead would use [(gasLimit - x)/gasLimit * oldMaxTransactionFee, converted to target chain currency] + (any additional funds passed into forward)
     * to fund a new delivery (of wormhole messages emitted during execution of oldTargetAddress's receiveWormholeMessages) that is requested in the call to 'forward'.
     *
     * @param sendParams The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters.
     * See struct documentation
     *
     * This function must be called with a payment of exactly sendParams.maxTransactionFee + sendParams.receiverValue + one wormhole message fee OR there must be enough
     * left over gas from the currently in-progress delivery to cover.
     */
    function forward(IWormholeRelayer.Send memory sendParams) public payable {
        if (!isContractLocked()) {
            revert IWormholeRelayer.NoDeliveryInProgress();
        }
        if (msg.sender != lockedTargetAddress()) {
            revert IWormholeRelayer.ForwardRequestFromWrongAddress();
        }

        uint256 wormholeMessageFee = wormhole().messageFee();
        uint256 totalFee =
            sendParams.maxTransactionFee + sendParams.receiverValue + wormholeMessageFee;

        IRelayProvider relayProvider = IRelayProvider(sendParams.relayProviderAddress);

        if (!relayProvider.isChainSupported(sendParams.targetChain)) {
            revert IWormholeRelayer.RelayProviderDoesNotSupportTargetChain();
        }

        checkInstruction(convertSendToDeliveryInstruction(sendParams), relayProvider);

        // Save information about the forward in state, so it can be processed after the execution of 'receiveWormholeMessages',
        // because we will then know how much of the 'maxTransactionFee' of the current delivery is still available for use in this forward
        appendForwardInstruction(
            IWormholeRelayerInternalStructs.ForwardInstruction({
                encodedSend: encodeSend(sendParams),
                msgValue: msg.value,
                totalFee: totalFee
            })
        );
    }

    /**
     * @notice This 'resend' function allows a caller to request an additional delivery of a specified `send` VAA, with an updated provider, maxTransactionFee, and receiveValue.
     * This function is intended to help integrators more eaily resolve ReceiverFailure cases, or other scenarios where an delivery was not able to be correctly performed.
     *
     * No checks about the original delivery VAA are performed prior to the emission of the redelivery instruction. Therefore, caller should be careful not to request
     * redeliveries in the following cases, as they will result in an undeliverable, invalid redelivery instruction that the provider will not be able to perform:
     *
     * - If the specified VaaKey does not correspond to a valid delivery VAA.
     * - If the targetChain does not equal the targetChain of the original delivery.
     * - If the gasLimit calculated from 'newMaxTransactionFee' is less than the original delivery's gas limit.
     * - If the receiverValueTarget (amount of receiver value to pass into the target contract) calculated from newReceiverValue is lower than the original delivery's receiverValueTarget.
     * - If the new calculated maximumRefundTarget (maximum possible refund amount) calculated from 'newMaxTransactionFee' is lower than the original delivery's maximumRefundTarget.
     *
     * Similar to send, you must call this function with msg.value = nexMaxTransactionFee + newReceiverValue + wormhole.messageFee() in order to pay for the delivery.
     *
     *  @param key a VAA Key corresponding to the delivery which should be performed again. This must correspond to a valid delivery instruction VAA.
     *  @param newMaxTransactionFee - the maxTransactionFee (in this chain's wei) that should be used on the redelivery. Must correspond to a gas amount equal to or greater than the original delivery,
     *  as well as a maximum transaction fee refund equal to or greater than the original delivery.
     *  @param newReceiverValue - the receiveValue (in this chain's wei) that should be used on the redelivery. Must result in receiverValue on the target chain which is equal to or greater that the original delivery.
     *  @param targetChain - the chain which the original delivery targetted.
     *  @param relayProviderAddress - the address of the relayProvider (on this chain) which should be used for this redelivery.
     */
    function resend(
        IWormholeRelayer.VaaKey memory key,
        uint256 newMaxTransactionFee,
        uint256 newReceiverValue,
        uint16 targetChain,
        address relayProviderAddress
    ) external payable returns (uint64 sequence) {
        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();
        IRelayProvider relayProvider = IRelayProvider(relayProviderAddress);

        uint256 totalFee = newMaxTransactionFee + newReceiverValue + wormholeMessageFee;
        if (msg.value < totalFee) {
            revert IWormholeRelayer.MsgValueTooLow();
        } else if (msg.value > totalFee) {
            revert IWormholeRelayer.MsgValueTooHigh();
        }

        if (!relayProvider.isChainSupported(targetChain)) {
            revert IWormholeRelayer.RelayProviderDoesNotSupportTargetChain();
        }

        IWormholeRelayerInternalStructs.RedeliveryInstruction memory instruction =
        IWormholeRelayerInternalStructs.RedeliveryInstruction({
            key: key,
            newMaximumRefundTarget: calculateTargetDeliveryMaximumRefund(
                targetChain, newMaxTransactionFee, relayProvider
                ),
            newReceiverValueTarget: convertReceiverValueAmountToTarget(
                newReceiverValue, targetChain, relayProvider
                ),
            sourceRelayProvider: toWormholeFormat(relayProviderAddress),
            targetChain: targetChain,
            executionParameters: IWormholeRelayerInternalStructs.ExecutionParameters({
                version: 1,
                gasLimit: calculateTargetGasDeliveryAmount(targetChain, newMaxTransactionFee, relayProvider)
            })
        });

        if (instruction.executionParameters.gasLimit == 0) {
            revert IWormholeRelayer.MaxTransactionFeeNotEnough();
        }

        if (
            instruction.newMaximumRefundTarget + instruction.newReceiverValueTarget
                > relayProvider.quoteMaximumBudget(targetChain)
        ) {
            revert IWormholeRelayer.MsgValueMoreThanMaximum();
        }

        sequence = wormhole.publishMessage{value: wormholeMessageFee}(
            0,
            encodeRedeliveryInstruction(instruction),
            200 //emit immediately
        );

        emit Send(sequence, newMaxTransactionFee, newReceiverValue);

        Utils.pay(relayProvider.getRewardAddress(), totalFee - wormholeMessageFee);
    }

    /**
     * @notice quoteGas returns how much maxTransactionFee (denominated in current (source) chain currency) must be in order to fund a call to
     * receiveWormholeMessages on a contract on chain 'targetChain' that uses 'gasLimit' units of gas
     *
     * @dev Specifically, for a Send 'request',
     * if 'request.targetAddress''s receiveWormholeMessage function uses 'gasLimit' units of gas,
     * then we must have request.maxTransactionFee >= quoteGas(request.targetChain, gasLimit, relayProvider)
     *
     * @param targetChain the target chain that you wish to use gas on
     * @param gasLimit the amount of gas you wish to use
     * @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *
     * @return maxTransactionFee The 'maxTransactionFee' you pass into your request (to relay messages to 'targetChain' and use 'gasLimit' units of gas) must be at least this amount
     */
    function quoteGas(
        uint16 targetChain,
        uint32 gasLimit,
        address relayProvider
    ) public view returns (uint256 maxTransactionFee) {
        IRelayProvider provider = IRelayProvider(relayProvider);

        // maxTransactionFee is a linear function of the amount of gas desired
        maxTransactionFee = provider.quoteDeliveryOverhead(targetChain)
            + (gasLimit * provider.quoteGasPrice(targetChain));
    }

    /**
     * @notice quoteReceiverValue returns how much receiverValue (denominated in current (source) chain currency) must be
     * in order for the relay provider to pass in 'targetAmount' as msg value when calling receiveWormholeMessages.
     *
     * @dev Specifically, for a send 'request',
     * In order for 'request.targetAddress''s receiveWormholeMessage function to be called with 'targetAmount' of value,
     * then we must have request.receiverValue >= quoteReceiverValue(request.targetChain, targetAmount, relayProvider)
     *
     * @param targetChain the target chain that you wish to receive value on
     * @param targetAmount the amount of value you wish to be passed into receiveWormholeMessages
     * @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *
     * @return receiverValue The 'receiverValue' you pass into your send request (to relay messages to 'targetChain' with 'targetAmount' of value) must be at least this amount
     */
    function quoteReceiverValue(
        uint16 targetChain,
        uint256 targetAmount,
        address relayProvider
    ) public view returns (uint256 receiverValue) {
        IRelayProvider provider = IRelayProvider(relayProvider);

        // Converts 'targetAmount' from target chain currency to source chain currency (using relayProvider's prices)
        // and applies a multiplier of '1 + (buffer / denominator)'
        (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChain);
        receiverValue = assetConversionHelper(
            targetChain,
            targetAmount,
            chainId(),
            uint256(denominator) + buffer,
            denominator,
            true,
            provider
        );
    }

    /**
     * @notice Returns the address of the current default relay provider
     * @return relayProvider The address of (the default relay provider)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     */
    function getDefaultRelayProvider() public view returns (address relayProvider) {
        relayProvider = defaultRelayProvider();
    }

    /**
     * @notice Returns default relay parameters
     * @return relayParams default relay parameters
     */
    function getDefaultRelayParams() public pure returns (bytes memory relayParams) {
        return new bytes(0);
    }

    /**
     * @notice returns the address of the contract which delivers messages on this chain.
     * I.E this is the address which will call receiveWormholeMessages.
     */
    function getDeliveryAddress() external view returns (address deliveryAddress) {
        return getWormholeRelayerCallerAddress();
    }
}
