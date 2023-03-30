// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/relayer/IWormholeRelayer.sol";
import "./CoreRelayerDelivery.sol";
import "./CoreRelayerStructs.sol";

contract CoreRelayer is CoreRelayerDelivery {
    /**
     * @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
     * call the receiveWormholeMessage(bytes[] memory signedVaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'messages' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param messages Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messages', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
     *
     *  This function must be called with a payment of at least maxTransactionFee + receiverValue + one wormhole message fee.
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for the default wormhole relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described.
     */
    function send(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes32 refundAddress,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        IWormholeRelayer.MessageInfo[] memory messages
    ) external payable returns (uint64 sequence) {
        sequence = send(
            IWormholeRelayer.Send(
                targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, getDefaultRelayParams()
            ),
            messages,
            getDefaultRelayProvider()
        );
    }

    /**
     * @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
     * call the receiveWormholeMessage(bytes[] memory signedVaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'messages' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *
     *  @param request The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters
     *  @param messages Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messages', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
     *  @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *  If request.maxTransactionFee >= quoteGas(request.targetChain, gasLimit, relayProvider),
     *  then as long as 'request.targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  If request.receiverValue >= quoteReceiverValue(request.targetChain, targetAmount, relayProvider), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  To use the default relay provider, set this field to be getDefaultRelayProvider()
     *
     *  This function must be called with a payment of at least request.maxTransactionFee + request.receiverValue + one wormhole message fee.
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for your specified relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described.
     */
    function send(
        IWormholeRelayer.Send memory request,
        IWormholeRelayer.MessageInfo[] memory messages,
        address relayProvider
    ) public payable returns (uint64 sequence) {
        // call multichainSend with one 'Send' in the requests array
        sequence = multichainSend(multichainSendContainer(request, relayProvider, messages));
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
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'messages' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param messages Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messages', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
     *
     *  This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
     */
    function forward(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes32 refundAddress,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        IWormholeRelayer.MessageInfo[] memory messages
    ) external payable {
        forward(
            IWormholeRelayer.Send(
                targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, getDefaultRelayParams()
            ),
            messages,
            getDefaultRelayProvider()
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
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'messages' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *  @param request The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters
     *  (specifically, the send info that will be used to deliver all of the wormhole messages emitted during the execution of oldTargetAddress's receiveWormholeMessages)
     *  This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
     *  @param messages Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messages', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
     *  @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *  If request.maxTransactionFee >= quoteGas(request.targetChain, gasLimit, relayProvider),
     *  then as long as 'request.targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  If request.receiverValue >= quoteReceiverValue(request.targetChain, targetAmount, relayProvider), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  To use the default relay provider, set this field to be getDefaultRelayProvider()
     *
     *  This function must be called with a payment of at least request.maxTransactionFee + request.receiverValue + one wormhole message fee.
     */
    function forward(
        IWormholeRelayer.Send memory request,
        IWormholeRelayer.MessageInfo[] memory messages,
        address relayProvider
    ) public payable {
        // call multichainForward with one 'Send' in the requests array
        multichainForward(multichainSendContainer(request, relayProvider, messages));
    }

    /**
     * @notice The multichainSend function delivers the messages in the current transaction specified by the 'messages' array,
     * with each destination specified in a Send struct, describing the desired targetAddress, targetChain, maxTransactionFee, receiverValue, refundAddress, and relayParameters
     *
     * @param sendContainer The MultichainSend struct, containing the array of Send requests, the message array specifying the messages to be relayed, as well as the desired relayProviderAddress
     *
     *  This function must be called with a payment of at least (one wormhole message fee) + Sum_(i=0 -> sendContainer.requests.length - 1) [sendContainer.requests[i].maxTransactionFee + sendContainer.requests[i].receiverValue].
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for the default wormhole relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described
     */
    function multichainSend(IWormholeRelayer.MultichainSend memory sendContainer)
        public
        payable
        returns (uint64 sequence)
    {
        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();
        uint256 totalFee = getTotalFeeMultichainSend(sendContainer, wormholeMessageFee);
        if (totalFee > msg.value) {
            revert IWormholeRelayer.MsgValueTooLow();
        }
        if (sendContainer.requests.length == 0) {
            revert IWormholeRelayer.MultichainSendEmpty();
        }

        IRelayProvider relayProvider = IRelayProvider(sendContainer.relayProviderAddress);

        // For each 'Send' request,
        // calculate how much gas the relay provider can pay for on 'request.targetChain' using 'request.newTransactionFee',
        // and calculate how much value the relay provider will pass into 'request.targetAddress'
        DeliveryInstructionsContainer memory instructionsContainer =
            convertMultichainSendToDeliveryInstructionsContainer(sendContainer);

        // For each 'Send' request,
        // Check that the total amount of value the relay provider needs to use for this send is <= the relayProvider's maximum budget for 'targetChain'
        // and check that the calculated gas is greater than 0
        checkInstructions(instructionsContainer, IRelayProvider(sendContainer.relayProviderAddress));

        // Mark the container as 'sufficientlyFunded', since the user passed in msg.value to fund all of the requests
        instructionsContainer.sufficientlyFunded = true;

        // Publish a wormhole message indicating to the relay provider (who is watching wormhole messages from this contract)
        // to relay the messages from this transaction (of nonce 'nonce') to the specified chains, each with the calculated amount of gas and receiverValue
        sequence = wormhole.publishMessage{value: wormholeMessageFee}(
            0, encodeDeliveryInstructionsContainer(instructionsContainer), relayProvider.getConsistencyLevel()
        );

        // Pay the relay provider
        pay(relayProvider.getRewardAddress(), totalFee - wormholeMessageFee);
    }

    /**
     * @notice The multichainForward function can only be called in a IWormholeReceiver within the 'receiveWormholeMessages' function
     * It's purpose is to use any leftover fee from the 'maxTransactionFee' of the current delivery to fund another delivery, specifically a multichain delivery to many destinations
     * See the description of 'forward' for further explanation of what a forward is.
     * multichainForward provides the same functionality of forward, while additionally allowing the same array of wormhole messages to be sent to many destinations
     *
     * Let LEFTOVER_VALUE = (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward)
     * and let NEEDED_VALUE = (one wormhole message fee) + Sum_(i=0 -> requests.requests.length - 1) [requests.requests[i].maxTransactionFee + requests.requests[i].receiverValue].
     * The multichainForward will succeed if LEFTOVER_VALUE >= NEEDED_VALUE
     *
     * note: If LEFTOVER_VALUE > NEEDED_VALUE, then the maxTransactionFee of the first request in the array of sends will be incremented by 'LEFTOVER_VALUE - NEEDED_VALUE'
     *
     *  @param sendContainer The MultichainSend struct, containing the array of Send requests, the message array specifying the messages to be relayed, as well as the desired relayProviderAddress
     *
     */
    function multichainForward(IWormholeRelayer.MultichainSend memory sendContainer) public payable {
        if (!isContractLocked()) {
            revert IWormholeRelayer.NoDeliveryInProgress();
        }
        if (getForwardInstruction().isValid) {
            revert IWormholeRelayer.MultipleForwardsRequested();
        }
        if (msg.sender != lockedTargetAddress()) {
            revert IWormholeRelayer.ForwardRequestFromWrongAddress();
        }
        if (sendContainer.requests.length == 0) {
            revert IWormholeRelayer.MultichainSendEmpty();
        }

        uint256 wormholeMessageFee = wormhole().messageFee();
        uint256 totalFee = getTotalFeeMultichainSend(sendContainer, wormholeMessageFee);

        // For each 'Send' request,
        // calculate how much gas the relay provider can pay for on 'request.targetChain' using 'request.newTransactionFee',
        // and calculate how much value the relay provider will pass into 'request.targetAddress'
        DeliveryInstructionsContainer memory instructionsContainer =
            convertMultichainSendToDeliveryInstructionsContainer(sendContainer);

        // For each 'Send' request,
        // Check that the total amount of value the relay provider needs to use for this send is <= the relayProvider's maximum budget for 'targetChain'
        // and check that the calculated gas is greater than 0
        checkInstructions(instructionsContainer, IRelayProvider(sendContainer.relayProviderAddress));

        // Save information about the forward in state, so it can be processed after the execution of 'receiveWormholeMessages',
        // because we will then know how much of the 'maxTransactionFee' of the current delivery is still available for use in this forward
        setForwardInstruction(
            ForwardInstruction({
                container: encodeDeliveryInstructionsContainer(instructionsContainer),
                msgValue: msg.value,
                totalFee: totalFee,
                sender: msg.sender,
                relayProvider: sendContainer.relayProviderAddress,
                isValid: true
            })
        );
    }

    /**
     * @notice This 'resend' function emits a wormhole message requesting to resend an array of messages that have been previously requested to be sent
     *  Specifically, if a user in transaction 'txHash' on chain 'sourceChain' emits many wormhole messages and then
     *  makes a call to 'send' requesting these messages to be sent to 'targetAddress' on 'targetChain' (which returns a sequence number of 'deliveryVAASequence'),
     *  then the user can request a redelivery of these wormhole messages any time in the future through a call to 'resend' using this function
     *
     *  @param request Information about the resend request, including the source chain and source transaction hash,
     *  @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *  If the targetAddress's receiveWormholeMessage function uses 'gasLimit' units of gas, then we must have newMaxTransactionFee >= quoteGasResend(targetChain, gasLimit, relayProvider)
     *  This must be the same relayProvider that was designated to relay these messages previously
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for your specified relay provider.
     *  The relay provider will listen for these messages, and then execute the redelivery as described
     */
    function resend(IWormholeRelayer.ResendByTx memory request, address relayProvider)
        public
        payable
        returns (uint64 sequence)
    {
        IWormhole wormhole = wormhole();
        uint256 wormholeMessageFee = wormhole.messageFee();
        if (request.newMaxTransactionFee + request.newReceiverValue + wormholeMessageFee > msg.value) {
            revert IWormholeRelayer.MsgValueTooLow();
        }

        IRelayProvider provider = IRelayProvider(relayProvider);

        // Calculate how much gas the relay provider can pay for on 'request.targetChain' using 'request.newTransactionFee',
        // and calculate how much value the relay provider will pass into 'request.targetAddress'
        RedeliveryByTxHashInstruction memory instruction = convertResendToRedeliveryInstruction(request, provider);

        // Check that the total amount of value the relay provider needs to use for this redelivery is <= the relayProvider's maximum budget for 'targetChain'
        // and check that the calculated gas is greater than 0
        checkRedeliveryInstruction(instruction, provider);

        // Publish a wormhole message indicating to the relay provider (who is watching wormhole messages from this contract)
        // to re-relay the messages from transaction 'request.txHash' with the calculated amount of gas and receiverValue
        sequence = wormhole.publishMessage{value: wormholeMessageFee}(
            0, encodeRedeliveryInstruction(instruction), provider.getConsistencyLevel()
        );

        // Pay the relay provider
        pay(provider.getRewardAddress(), msg.value - wormholeMessageFee);
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
     * @notice quoteGasResend tells you how much maxTransactionFee (denominated in current (source) chain currency) must be in order to fund a *resend* call to
     * receiveWormholeMessages on a contract on chain 'targetChain' that uses 'gasLimit' units of gas
     *
     * Specifically, for a ResendByTx 'request',
     * If 'request.targetAddress''s receiveWormholeMessage function uses 'gasLimit' units of gas,
     * then we must have request.maxTransactionFee >= quoteGasResend(request.targetChain, gasLimit, relayProvider)
     *
     * @param targetChain the target chain that you wish to use gas on
     * @param gasLimit the amount of gas you wish to use
     * @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *
     * @return maxTransactionFee The 'maxTransactionFee' you pass into your resend request (to relay messages to 'targetChain' and use 'gasLimit' units of gas) must be at least this amount
     */
    function quoteGasResend(uint16 targetChain, uint32 gasLimit, address relayProvider)
        public
        view
        returns (uint256 maxTransactionFee)
    {
        IRelayProvider provider = IRelayProvider(relayProvider);

        // maxTransactionFee is a linear function of the amount of gas desired
        maxTransactionFee =
            provider.quoteRedeliveryOverhead(targetChain) + (gasLimit * provider.quoteGasPrice(targetChain));
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

    // Helper to put one Send struct into a MultichainSend struct
    function multichainSendContainer(
        IWormholeRelayer.Send memory request,
        address relayProvider,
        IWormholeRelayer.MessageInfo[] memory messages
    ) internal pure returns (IWormholeRelayer.MultichainSend memory container) {
        IWormholeRelayer.Send[] memory requests = new IWormholeRelayer.Send[](1);
        requests[0] = request;
        container = IWormholeRelayer.MultichainSend({
            relayProviderAddress: relayProvider,
            requests: requests,
            messages: messages
        });
    }
}
