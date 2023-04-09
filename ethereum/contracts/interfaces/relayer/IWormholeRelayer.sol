// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface IWormholeRelayer {
    /**
     * @title IWormholeRelayer
     * @notice Users may use this interface to have wormhole messages (VAAs) in their transaction
     * relayed to destination contract(s) of their choice
     */

    /**
     * @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
     * call the receiveWormholeMessage(bytes[] memory signedVaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being one wormhole message (VAA) from the current transaction that matches the wormholeMessageEmitterAddress and wormholeMessageSequenceNumber provided (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param refundChain The chain where the refund should be sent. If the refundchain is not the targetchain, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
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
        bytes32 refundAddress,
        uint16 refundChain,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        bytes memory payload
    ) external payable returns (uint64 sequence);

    /**
     * @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
     * call the receiveWormholeMessage(bytes[] memory signedVaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being one wormhole message (VAA) from the current transaction that matches the wormholeMessageEmitterAddress and wormholeMessageSequenceNumber provided (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param refundChain The chain where the refund should be sent. If the refundchain is not the targetchain, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param payload an arbitrary payload which will be sent to the receiver contract.
     *  @param wormholeMessageEmitterAddress emitter address identifying the wormhole message
     *  @param wormholeMessageSequenceNumber sequence number identifying the wormhole message
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
        uint16 refundChain,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        bytes memory payload,
        address wormholeMessageEmitterAddress,
        uint64 wormholeMessageSequenceNumber
    ) external payable returns (uint64 sequence);

    /**
     * @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
     * call the receiveWormholeMessage(bytes[] memory signedVaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'messageInfos' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param refundChain The chain where the refund should be sent. If the refundchain is not the targetchain, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param payload an arbitrary payload which will be sent to the receiver contract.
     *  @param messageInfos Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messageInfos', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
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
        uint16 refundChain,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        bytes memory payload,
        MessageInfo[] memory messageInfos
    ) external payable returns (uint64 sequence);

    enum MessageInfoType {
        EMITTER_SEQUENCE,
        VAAHASH
    }

    /**
     * @notice This 'MessageInfo' struct identifies a wormhole message from the current transaction
     *
     * @custom:member infoType if infoType = MessageInfoType.EMITTER_SEQUENCE, then the wormhole message identified by this struct must have emitterAddress 'emitterAddress' and sequence number 'sequence'
     * else if infoType = MessageInfoType.VAAHASH, then the wormhole message identified by this struct must have VAA hash 'vaaHash'
     * @custom:member emitterAddress the emitterAddress specified (if infoType = MessageInfoType.EMITTER_SEQUENCE); else this field is ignored
     * @custom:member sequence the sequence specified specified (if infoType = MessageInfoType.EMITTER_SEQUENCE); else this field is ignored
     * @custom:member vaaHash the hash specified (if infoType = MessageInfoType.VAAHASH); else this field is ignored
     */
    struct MessageInfo {
        MessageInfoType infoType;
        bytes32 emitterAddress;
        uint64 sequence;
        bytes32 vaaHash;
    }

    /**
     * @notice This 'Send' struct represents a request to relay to a contract at address 'targetAddress' on chain 'targetChain'
     *
     *  @custom:member targetChain The chain that the encoded+signed Wormhole messages (VAAs) are delivered to, in Wormhole Chain ID format
     *  @custom:member targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @custom:member refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @custom:member maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  @custom:member receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  @custom:member an optional payload which will be delivered to the receiving contract.
     *  @custom:member relayParameters This should be 'getDefaultRelayParameters()'
     */
    struct Send {
        uint16 targetChain;
        bytes32 targetAddress;
        bytes32 refundAddress;
        uint16 refundChain;
        uint256 maxTransactionFee;
        uint256 receiverValue;
        bytes payload;
        bytes relayParameters;
    }

    /**
     * @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
     * call the receiveWormholeMessage(bytes[] memory signedVaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'messageInfos' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *
     *  @param request The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters
     *  @param messageInfos Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messageInfos', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
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
    function send(Send memory request, MessageInfo[] memory messageInfos, address relayProvider)
        external
        payable
        returns (uint64 sequence);

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
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'messageInfos' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param refundChain The chain where the refund should be sent. If the refundchain is not the targetchain, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param messageInfos Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messageInfos', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
     *
     *  This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
     */
    function forward(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes32 refundAddress,
        uint16 refundChain,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        bytes memory payload,
        MessageInfo[] memory messageInfos
    ) external payable;

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
     * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'messageInfos' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
     * and with the second argument empty
     *
     *  @param request The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters
     *  (specifically, the send info that will be used to deliver all of the wormhole messages emitted during the execution of oldTargetAddress's receiveWormholeMessages)
     *  This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
     *  @param messageInfos Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messageInfos', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
     *  @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *  If request.maxTransactionFee >= quoteGas(request.targetChain, gasLimit, relayProvider),
     *  then as long as 'request.targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  If request.receiverValue >= quoteReceiverValue(request.targetChain, targetAmount, relayProvider), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  To use the default relay provider, set this field to be getDefaultRelayProvider()
     *
     *  This function must be called with a payment of at least request.maxTransactionFee + request.receiverValue + one wormhole message fee.
     */
    function forward(Send memory request, MessageInfo[] memory messageInfos, address relayProvider) external payable;

    /**
     * @notice This 'MultichainSend' struct represents a collection of send requests 'requests' and a specified relay provider 'relayProviderAddress'
     *  This struct is used to request sending the same array of transactions to many different destination contracts (on many different chains),
     *  With each request in the 'requests' array denoting parameters for one specific destination
     *
     *  @custom:member relayProviderAddress The address of the relay provider's contract on this source chain. This relay provider will perform delivery of all of these Send requests.
     *  Use getDefaultRelayProvider() here to use the default relay provider.
     *  @custom:member messages Array of MessageInfo structs identifying each message to be relayed. Each MessageInfo struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this messages array.
     *  Specifically, the 'signedVaas' array will have the same length as 'messageInfos', and additionally for each 0 <= i < messages.length, signedVaas[i] will match the description in messages[i]
     *  @custom:member requests The array of send requests, each specifying a targetAddress on a targetChain, along with information about the desired maxTransactionFee, receiverValue, refundAddress, and relayParameters
     */
    struct MultichainSend {
        address relayProviderAddress;
        IWormholeRelayer.MessageInfo[] messageInfos;
        Send[] requests;
    }

    /**
     * @notice The multichainSend function delivers the messages in the current transaction specified by the 'messageInfos' array,
     * with each destination specified in a Send struct, describing the desired targetAddress, targetChain, maxTransactionFee, receiverValue, refundAddress, and relayParameters
     *
     * @param sendContainer The MultichainSend struct, containing the array of Send requests, the message array specifying the messages to be relayed, as well as the desired relayProviderAddress
     *
     *  This function must be called with a payment of at least (one wormhole message fee) + Sum_(i=0 -> sendContainer.requests.length - 1) [sendContainer.requests[i].maxTransactionFee + sendContainer.requests[i].receiverValue].
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for the default wormhole relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described
     */
    function multichainSend(MultichainSend memory sendContainer) external payable returns (uint64 sequence);

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
    function multichainForward(MultichainSend memory sendContainer) external payable;

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
        external
        pure
        returns (uint256 maxTransactionFee);

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
        external
        pure
        returns (uint256 receiverValue);

    /**
     * @notice Helper function that converts an EVM address to wormhole format
     * @param addr (EVM 20-byte address)
     * @return whFormat (32-byte address in Wormhole format)
     */
    function toWormholeFormat(address addr) external pure returns (bytes32 whFormat);

    /**
     * @notice Helper function that converts an Wormhole format (32-byte) address to the EVM 'address' 20-byte format
     * @param whFormatAddress (32-byte address in Wormhole format)
     * @return addr (EVM 20-byte address)
     */
    function fromWormholeFormat(bytes32 whFormatAddress) external pure returns (address addr);

    /**
     * @notice Returns the address of the current default relay provider
     * @return relayProvider The address of (the default relay provider)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     */
    function getDefaultRelayProvider() external view returns (address relayProvider);

    /**
     * @notice Returns default relay parameters
     * @return relayParams default relay parameters
     */
    function getDefaultRelayParams() external pure returns (bytes memory relayParams);

    error FundsTooMuch(uint8 multisendIndex); // (maxTransactionFee, converted to target chain currency) + (receiverValue, converted to target chain currency) is greater than what your chosen relay provider allows
    error MaxTransactionFeeNotEnough(uint8 multisendIndex); // maxTransactionFee is less than the minimum needed by your chosen relay provider
    error MsgValueTooLow(); // msg.value is too low
    // Specifically, (msg.value) + (any leftover funds if this is a forward) is less than (maxTransactionFee + receiverValue), summed over all of your requests if this is a multichainSend/multichainForward
    error NoDeliveryInProgress(); // Forwards can only be requested within execution of 'receiveWormholeMessages', or when a delivery is in progress
    error MultipleForwardsRequested(); // Only one forward can be requested in a transaction
    error ForwardRequestFromWrongAddress(); // A forward was requested from an address that is not the 'targetAddress' of the original delivery
    error RelayProviderDoesNotSupportTargetChain(); // Your relay provider does not support the target chain you specified
    error MultichainSendEmpty(); // Your delivery request container has size 0
}
