// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/relayer/IWormholeRelayer.sol";
import "./CoreRelayerDelivery.sol";
import "../../interfaces/relayer/IWormholeRelayerInternalStructs.sol";

abstract contract CoreRelayer is CoreRelayerDelivery {
    /**
     * @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
     * call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being a DeliveryData struct with fields:
     *      - sourceAddress: address (in wormhole 32-byte format) that called 'send' on the source chain
     *      - sourceChain: The wormhole chainID of the source chain (this current chain)
     *      - maximumRefund: The maximum transaction fee refund that can possibly be awarded (to refundAddress) at the end of this delivery, assuming no gas is used by receiveWormholeMessages
     *                       This is calculated by subtracting the relayer's base fee for the targetChain from 'maxTransactionFee' and then converting to target chain currency
     *      - deliveryHash: The VAA hash of the deliveryVAA. If you do not want to potentially process this delivery multiple times, you should store this hash in state for replay protection
     *      - payload: the arbitrary payload (bytes).
     * and with the second argument empty
     *
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas)' endpoint
     *  @param refundChain The chain where the refund should be sent. If the refundChain is not the targetChain, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'refundChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure and the call to targetAddress will revert.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param payload an arbitrary payload which will be sent to the receiver contract.
     *
     *  This function must be called with a payment of at least maxTransactionFee + receiverValue + one wormhole message fee.
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for the default wormhole relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described.
     */
    function send(
        uint16 targetChain,
        bytes32 targetAddress,
        uint16 refundChain,
        bytes32 refundAddress,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        bytes memory payload
    ) public payable returns (uint64 sequence) {
        sequence = send(
            IWormholeRelayer.Send(
                targetChain,
                targetAddress,
                refundChain,
                refundAddress,
                maxTransactionFee,
                receiverValue,
                getDefaultRelayProvider(),
                new IWormholeRelayer.VaaKey[](0),
                15, //finality on all EVM chains
                payload,
                getDefaultRelayParams()
            )
        );
    }

    /**
     *  @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
     *  call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     *  with the first argument being a DeliveryData struct with fields:
     *      - sourceAddress: address (in wormhole 32-byte format) that called 'send' on the source chain
     *      - sourceChain: The wormhole chainID of the source chain (this current chain)
     *      - maximumRefund: The maximum transaction fee refund that can possibly be awarded (to refundAddress) at the end of this delivery, assuming no gas is used by receiveWormholeMessages
     *                       This is calculated by subtracting the relayer's base fee for the targetChain from 'maxTransactionFee' and then converting to target chain currency
     *      - deliveryHash: The VAA hash of the deliveryVAA. If you do not want to potentially process this delivery multiple times, you should store this hash in state for replay protection
     *      - payload: the arbitrary payload (bytes).
     *  and with the second argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'vaaKeys' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs')
     *
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas)' endpoint
     *  @param refundChain The chain where the refund should be sent. If the refundChain is not the targetChain, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'refundChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param payload an arbitrary payload which will be sent to the receiver contract.
     *  @param vaaKeys Array of VaaKey structs identifying each message to be relayed. Each VaaKey struct specifies a wormhole message, either by the VAA hash, or by the (chain id, emitter address, sequence number) triple.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this vaaKeys array.
     *  Specifically, the 'signedVaas' array will have the same length as 'vaaKeys', and additionally for each 0 <= i < vaaKeys.length, signedVaas[i] will match the description in vaaKeys[i]
     *  @param consistencyLevel  The level of finality to reach before emitting the Wormhole VAA corresponding to this 'send' request. See https://book.wormhole.com/wormhole/3_coreLayerContracts.html#consistency-levels
     *
     *  This function must be called with a payment of at least maxTransactionFee + receiverValue + one wormhole message fee.
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for the default wormhole relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described.
     */
    function send(
        uint16 targetChain,
        bytes32 targetAddress,
        uint16 refundChain,
        bytes32 refundAddress,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        bytes memory payload,
        IWormholeRelayer.VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence) {
        sequence = send(
            IWormholeRelayer.Send(
                targetChain,
                targetAddress,
                refundChain,
                refundAddress,
                maxTransactionFee,
                receiverValue,
                getDefaultRelayProvider(),
                vaaKeys,
                consistencyLevel,
                payload,
                getDefaultRelayParams()
            )
        );
    }

    /**
     *  @notice This 'send' function emits a wormhole message (VAA) that alerts the relay provider specified by sendParams.relayProviderAddress to
     *  call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas) endpoint of the contract on chain 'sendParams.targetChain' and address 'sendParams.targetAddress'
     *  with the first argument being a DeliveryData struct with fields:
     *      - sourceAddress: address (in wormhole 32-byte format) that called 'send' on the source chain
     *      - sourceChain: The wormhole chainID of the source chain (this current chain)
     *      - maximumRefund: The maximum transaction fee refund that can possibly be awarded (to sendParams.refundAddress) at the end of this delivery, assuming no gas is used by receiveWormholeMessages
     *                       This is calculated by subtracting the relayer's base fee for the targetChain from 'maxTransactionFee' and then converting to target chain currency
     *      - deliveryHash: The VAA hash of the deliveryVAA. If you do not want to potentially process this delivery multiple times, you should store this hash in state for replay protection
     *      - payload: the arbitrary payload (bytes).
     *  and with the second argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'sendParams.vaaKeys' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs')
     *
     *
     *  @param sendParams The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, relayProviderAddress, vaaKeys, consistencyLevel, payload, and relayParameters
     *
     *  This function must be called with a payment of at least sendParams.maxTransactionFee + sendParams.receiverValue + one wormhole message fee.
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for your specified relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described.
     */
    function send(IWormholeRelayer.Send memory sendParams) public payable returns (uint64 sequence) {
        // call multichainSend with one 'Send' in the requests array
        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();
        uint256 totalFee = sendParams.maxTransactionFee + sendParams.receiverValue + wormholeMessageFee;

        if (totalFee > msg.value) {
            revert IWormholeRelayer.MsgValueTooLow();
        }

        IRelayProvider relayProvider = IRelayProvider(sendParams.relayProviderAddress);

        if (!relayProvider.isChainSupported(sendParams.targetChain)) {
            revert IWormholeRelayer.RelayProviderDoesNotSupportTargetChain();
        }

        // For each 'Send' request,
        // calculate how much gas the relay provider can pay for on 'sendParams.targetChain' using 'sendParams.newTransactionFee',
        // and calculate how much value the relay provider will pass into 'sendParams.targetAddress'
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction =
            convertSendToDeliveryInstruction(sendParams);

        // For each 'Send' request,
        // Check that the total amount of value the relay provider needs to use for this send is <= the relayProvider's maximum budget for 'targetChain'
        // and check that the calculated gas is greater than 0
        checkInstruction(instruction, relayProvider);

        // Publish a wormhole message indicating to the relay provider (who is watching wormhole messages from this contract)
        // to relay the messages from this transaction (of nonce 'nonce') to the specified chains, each with the calculated amount of gas and receiverValue
        sequence = wormhole.publishMessage{value: wormholeMessageFee}(
            0, encodeDeliveryInstruction(instruction), sendParams.consistencyLevel
        );

        // Pay the relay provider
        pay(relayProvider.getRewardAddress(), totalFee - wormholeMessageFee);
    }

    /**
     * @notice This 'forward' function can only be called in a IWormholeReceiver within the 'receiveWormholeMessages' function
     * It's purpose is to use any leftover fee from the 'maxTransactionFee' of the current delivery to fund another delivery
     *
     * Specifically, suppose an integrator requested a Send (with parameters oldTargetChain, oldTargetAddress, etc)
     * and sets quoteGas(oldTargetChain, gasLimit, oldRelayProvider) as 'maxTransactionFee' in a Send,
     * but during the delivery on oldTargetChain, the call to oldTargetAddress's receiveWormholeMessages endpoint uses only x units of gas (where x < gasLimit).
     *
     * Normally, (gasLimit - x)/gasLimit * oldMaxTransactionFee, converted to target chain currency, would be refunded to 'oldRefundAddress'.
     * However, if during execution of receiveWormholeMessage the integrator made a call to forward,
     *
     * We instead would use [(gasLimit - x)/gasLimit * oldMaxTransactionFee, converted to target chain currency] + (any additional funds passed into forward)
     * to fund a new delivery (of wormhole messages emitted during execution of oldTargetAddress's receiveWormholeMessages) that is requested in the call to 'forward'.
     *
     * Specifically, this 'forward' function is only callable within a delivery (during receiveWormholeMessages) and indicates the in-progress delivery to use any leftover funds from the current delivery to fund a new delivery
     * or equivalently, indicates the in-progress delivery to call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'vaaKeys' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'refundChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param refundChain The chain where the refund should be sent. If the refundChain is not the targetChain, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param vaaKeys Array of VaaKey structs identifying each message to be relayed. Each VaaKey struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this vaaKeys array.
     *  Specifically, the 'signedVaas' array will have the same length as 'vaaKeys', and additionally for each 0 <= i < vaaKeys.length, signedVaas[i] will match the description in vaaKeys[i]
     *  @param consistencyLevel The level of finality to reach before emitting the Wormhole VAA corresponding to this 'forward' request. See https://book.wormhole.com/wormhole/3_coreLayerContracts.html#consistency-levels
     *
     *  This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
     */
    function forward(
        uint16 targetChain,
        bytes32 targetAddress,
        uint16 refundChain,
        bytes32 refundAddress,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        bytes memory payload,
        IWormholeRelayer.VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) external payable {
        forward(
            IWormholeRelayer.Send(
                targetChain,
                targetAddress,
                refundChain,
                refundAddress,
                maxTransactionFee,
                receiverValue,
                getDefaultRelayProvider(),
                vaaKeys,
                consistencyLevel,
                payload,
                getDefaultRelayParams()
            )
        );
    }

    /**
     * @notice This 'forward' function can only be called in a IWormholeReceiver within the 'receiveWormholeMessages' function
     * It's purpose is to use any leftover fee from the 'maxTransactionFee' of the current delivery to fund another delivery
     *
     * Specifically, suppose an integrator requested a Send (with parameters oldTargetChain, oldTargetAddress, etc)
     * and sets quoteGas(oldTargetChain, gasLimit, oldRelayProvider) as 'maxTransactionFee' in a Send,
     * but during the delivery on oldTargetChain, the call to oldTargetAddress's receiveWormholeMessages endpoint uses only x units of gas (where x < gasLimit).
     *
     * Normally, (gasLimit - x)/gasLimit * oldMaxTransactionFee, converted to target chain currency, would be refunded to 'oldRefundAddress'.
     * However, if during execution of receiveWormholeMessage the integrator made a call to forward,
     *
     * We instead would use [(gasLimit - x)/gasLimit * oldMaxTransactionFee, converted to target chain currency] + (any additional funds passed into forward)
     * to fund a new delivery (of wormhole messages emitted during execution of oldTargetAddress's receiveWormholeMessages) that is requested in the call to 'forward'.
     *
     * Specifically, this 'forward' function is only callable within a delivery (during receiveWormholeMessages) and indicates the in-progress delivery to use any leftover funds from the current delivery to fund a new delivery
     * or equivalently, indicates the in-progress delivery to call the receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'vaaKeys' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *  @param sendParams The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters
     *  (specifically, the send info that will be used to deliver all of the wormhole messages emitted during the execution of oldTargetAddress's receiveWormholeMessages)
     *  This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
     *  notparam vaaKeys Array of VaaKey structs identifying each message to be relayed. Each VaaKey struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this vaaKeys array.
     *  Specifically, the 'signedVaas' array will have the same length as 'vaaKeys', and additionally for each 0 <= i < vaaKeys.length, signedVaas[i] will match the description in vaaKeys[i]
     *  notparam relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *  If sendParams.maxTransactionFee >= quoteGas(sendParams.targetChain, gasLimit, relayProvider),
     *  then as long as 'sendParams.targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  If sendParams.receiverValue >= quoteReceiverValue(sendParams.targetChain, targetAmount, relayProvider), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  To use the default relay provider, set this field to be getDefaultRelayProvider()
     *
     *  This function must be called with a payment of at least sendParams.maxTransactionFee + sendParams.receiverValue + one wormhole message fee.
     */
    function forward(IWormholeRelayer.Send memory sendParams) public payable {
        if (!isContractLocked()) {
            revert IWormholeRelayer.NoDeliveryInProgress();
        }
        if (msg.sender != lockedTargetAddress()) {
            revert IWormholeRelayer.ForwardRequestFromWrongAddress();
        }

        uint256 wormholeMessageFee = wormhole().messageFee();
        uint256 totalFee = sendParams.maxTransactionFee + sendParams.receiverValue + wormholeMessageFee;

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
     * @notice This 'resend' function allows a caller to request a second delivery of a specified VAA, with an updated provider, maxTransactionFee, and receiveValue.
     * This function is intended to help integrators more eaily resolve Receiver Failure cases, or other scenarios where an delivery was not able to be correctly performed.
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
        }

        if (!relayProvider.isChainSupported(targetChain)) {
            revert IWormholeRelayer.RelayProviderDoesNotSupportTargetChain();
        }

        IWormholeRelayerInternalStructs.RedeliveryInstruction memory instruction = IWormholeRelayerInternalStructs
            .RedeliveryInstruction({
            key: key,
            newMaximumRefundTarget: calculateTargetDeliveryMaximumRefund(targetChain, newMaxTransactionFee, relayProvider),
            newReceiverValueTarget: convertReceiverValueAmount(newReceiverValue, targetChain, relayProvider),
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
            revert IWormholeRelayer.FundsTooMuch();
        }

        sequence = wormhole.publishMessage{value: wormholeMessageFee}(
            0,
            encodeRedeliveryInstruction(instruction),
            200 //emit immediately
        );

        // Pay the relay provider
        pay(relayProvider.getRewardAddress(), totalFee - wormholeMessageFee);
    }

    /**
     * @notice quoteGas tells you how much maxTransactionFee (denominated in current (source) chain currency) must be in order to fund a call to
     * receiveWormholeMessages on a contract on chain 'targetChain' that uses 'gasLimit' units of gas
     *
     * Specifically, for a Send 'request',
     * If 'request.targetAddress''s receiveWormholeMessage function uses 'gasLimit' units of gas,
     * then we must have request.maxTransactionFee >= quoteGas(request.targetChain, gasLimit, relayProvider)
     *
     * @param targetChain the target chain that you wish to use gas on
     * @param gasLimit the amount of gas you wish to use
     * @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *
     * @return maxTransactionFee The 'maxTransactionFee' you pass into your request (to relay messages to 'targetChain' and use 'gasLimit' units of gas) must be at least this amount
     */
    function quoteGas(uint16 targetChain, uint32 gasLimit, address relayProvider)
        public
        view
        returns (uint256 maxTransactionFee)
    {
        IRelayProvider provider = IRelayProvider(relayProvider);

        // maxTransactionFee is a linear function of the amount of gas desired
        maxTransactionFee =
            provider.quoteDeliveryOverhead(targetChain) + (gasLimit * provider.quoteGasPrice(targetChain));
    }

    /**
     * @notice quoteReceiverValue tells you how much receiverValue (denominated in current (source) chain currency) must be
     * in order for the relay provider to pass in 'targetAmount' as msg value when calling receiveWormholeMessages.
     *
     * Specifically, for a send 'request',
     * In order for 'request.targetAddress''s receiveWormholeMessage function to be called with 'targetAmount' of value,
     * then we must have request.receiverValue >= quoteReceiverValue(request.targetChain, targetAmount, relayProvider)
     *
     * @param targetChain the target chain that you wish to receive value on
     * @param targetAmount the amount of value you wish to be passed into receiveWormholeMessages
     * @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *
     * @return receiverValue The 'receiverValue' you pass into your send request (to relay messages to 'targetChain' with 'targetAmount' of value) must be at least this amount
     */
    function quoteReceiverValue(uint16 targetChain, uint256 targetAmount, address relayProvider)
        public
        view
        returns (uint256 receiverValue)
    {
        IRelayProvider provider = IRelayProvider(relayProvider);

        // Converts 'targetAmount' from target chain currency to source chain currency (using relayProvider's prices)
        // and applies a multiplier of '1 + (buffer / denominator)'
        (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChain);
        receiverValue = assetConversionHelper(
            targetChain, targetAmount, chainId(), uint256(0) + denominator + buffer, denominator, true, provider
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
