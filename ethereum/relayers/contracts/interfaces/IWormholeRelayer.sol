// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface IWormholeRelayer {
    /**
     * @title IWormholeRelayer
     * @notice Users may use this interface to have wormhole messages in their transaction
     * relayed to destination contract(s) of their choice
     */

    /**
     * @notice This 'send' function emits a wormhole message that alerts the default wormhole relay provider to
     * call the receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being all of the wormhole message in the current transaction that have nonce 'nonce' (which have additionally been encoded and signed by the Guardian set to form 'VAAs'),
     * (including the one emitted from this function, which can be ignored)
     * (these messages will be ordered in the 'vaas' array in the order they were emitted in the source transaction)
     * and with the second argument empty
     *
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a DeliveryFailure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param nonce The messages to be relayed are all of the emitted wormhole messages in the current transaction that have nonce 'nonce'.
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
        uint32 nonce
    ) external payable returns (uint64 sequence);

    /**
     * @notice This 'Send' struct represents a request to relay to a contract at address 'targetAddress' on chain 'targetChain'
     *
     *  @custom:member targetChain The chain that the encoded+signed Wormhole messages (VAAs) are delivered to, in Wormhole Chain ID format
     *  @custom:member targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @custom:member refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @custom:member maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a DeliveryFailure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  @custom:member receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  @custom:member relayParameters This should be 'getDefaultRelayParameters()'
     */
    struct Send {
        uint16 targetChain;
        bytes32 targetAddress;
        bytes32 refundAddress;
        uint256 maxTransactionFee;
        uint256 receiverValue;
        bytes relayParameters;
    }

    /**
     * @notice This 'send' function emits a wormhole message that alerts a relay provider to
     * call the receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChain' and address 'targetAddress'
     * with the first argument being all of the wormhole message in the current transaction that have nonce 'nonce' (which have additionally been encoded and signed by the Guardian set to form 'VAAs'),
     * (including the one emitted from this function, which can be ignored)
     * (these messages will be ordered in the 'vaas' array in the order they were emitted in the source transaction)
     * and with the second argument empty
     *
     *
     *  @param request The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters
     *  @param nonce The messages to be relayed are all of the emitted wormhole messages in the current transaction that have nonce 'nonce'.
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
    function send(Send memory request, uint32 nonce, address relayProvider)
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
     * with the first argument being all of the wormhole message in the current transaction that have nonce 'nonce' (which have additionally been encoded and signed by the Guardian set to form 'VAAs'),
     * (which will be all of the wormhole messages emitted during the execution of oldTargetAddress's receiveWormholeMessages in the order that they were emitted, as well as one wormhole message *that is always at the end* that can be ignored)
     * and with the second argument empty
     *
     *  @param targetChain The chain that the vaas are delivered to, in Wormhole Chain ID format
     *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChain' of the contract to which the vaas are delivered.
     *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
     *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'targetChain' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
     *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a DeliveryFailure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  If maxTransactionFee >= quoteGas(targetChain, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  If receiverValue >= quoteReceiverValue(targetChain, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  @param nonce The messages to be relayed are all of the emitted wormhole messages in the current transaction that have nonce 'nonce'.
     *
     *  This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
     */
    function forward(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes32 refundAddress,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        uint32 nonce
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
     * with the first argument being all of the wormhole message in the current transaction that have nonce 'nonce' (which have additionally been encoded and signed by the Guardian set to form 'VAAs'),
     * (which will be all of the wormhole messages emitted during the execution of oldTargetAddress's receiveWormholeMessages in the order that they were emitted, as well as one wormhole message *that is always at the end* that can be ignored)
     * and with the second argument empty
     *
     *  @param request The Send request containing info about the targetChain, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters
     *  (specifically, the send info that will be used to deliver all of the wormhole messages emitted during the execution of oldTargetAddress's receiveWormholeMessages)
     *  This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
     *  @param nonce The messages to be relayed are all of the emitted wormhole messages in the current transaction (during execution of oldTargetAddress's receiveWormholeMessages) that have nonce 'nonce'.
     *  @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *  If request.maxTransactionFee >= quoteGas(request.targetChain, gasLimit, relayProvider),
     *  then as long as 'request.targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
     *  If request.receiverValue >= quoteReceiverValue(request.targetChain, targetAmount, relayProvider), then at least 'targetAmount' of targetChain currency will be passed into the 'receiveWormholeFunction' as value.
     *  To use the default relay provider, set this field to be getDefaultRelayProvider()
     *
     *  This function must be called with a payment of at least request.maxTransactionFee + request.receiverValue + one wormhole message fee.
     */
    function forward(Send memory request, uint32 nonce, address relayProvider) external payable;

    /**
     * @notice This 'ResendByTx' struct represents a request to resend an array of messages that have been previously requested to be sent
     *  Specifically, if a user in transaction 'txHash' on chain 'sourceChain' emits many wormhole messages of nonce 'sourceNonce' and then
     *  makes a call to 'send' requesting these messages to be sent to 'targetAddress' on 'targetChain',
     *  then the user can request a redelivery of these wormhole messages any time in the future through a call to 'resend' using this struct
     *
     *  @custom:member sourceChain The chain (that the original Send was initiated from (or equivalent, the chain that the original wormhole messages were emitted from).
     *  Important note: This does not need to be the current chain. A resend can be requested from any chain.
     *  @custom:member sourceTxHash The transaction hash of the original source chain transaction that contained the original wormhole messages and the original 'Send' request
     *  @custom:member sourceNonce The nonce of the original wormhole messages and original 'Send' request
     *  @custom:member targetChain The chain that the encoded+signed Wormhole messages (VAAs) were originally delivered to (and will be redelivered to), in Wormhole Chain ID format
     *  @custom:member deliveryIndex If all the originally emitted wormhole messages are ordered, *including* the wormhole message emitted from the original Send request,
     *  this is the (0-indexed) index of the wormhole message emitted from the original Send request. So, if originally the 'send' request was made after the publishing of x wormhole messages,
     *  deliveryIndex here would be 'x'.
     *  @custom:member multisendIndex If the 'send' (or forward) function was used in the original transaction, this should be 0. Otherwise if the multichainSend (or multichainForward) function was used,
     *  then this should be the index of the specific Send request in the requests array that you wish to be redelivered
     *  @custom:member newMaxTransactionFee The new maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
     *  If more gas is needed on the target chain than is paid for, there will be a DeliveryFailure.
     *  Any unused value out of this fee will be refunded to 'refundAddress'
     *  This must be greater than or equal to the original maxTransactionFee paid in the original request
     *  @custom:member receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
     *  This must be greater than or equal to the original receiverValue paid in the original request
     *  @custom:member newRelayParameters This should be 'getDefaultRelayParameters()'
     */
    struct ResendByTx {
        uint16 sourceChain;
        bytes32 sourceTxHash;
        uint32 sourceNonce;
        uint16 targetChain;
        uint8 deliveryIndex;
        uint8 multisendIndex;
        uint256 newMaxTransactionFee;
        uint256 newReceiverValue;
        bytes newRelayParameters;
    }

    /**
     * @notice This 'ResendByTx' struct represents a request to resend an array of messages that have been previously requested to be sent
     *  Specifically, if a user in transaction 'txHash' on chain 'sourceChain' emits many wormhole messages of nonce 'sourceNonce' and then
     *  makes a call to 'send' requesting these messages to be sent to 'targetAddress' on 'targetChain',
     *  then the user can request a redelivery of these wormhole messages any time in the future through a call to 'resend' using this struct
     *
     *  @param request Information about the resend request, including the source chain and source transaction hash,
     *  @param nonce This should be 0
     *  @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
     *  If the targetAddress's receiveWormholeMessage function uses 'gasLimit' units of gas, then we must have newMaxTransactionFee >= quoteGasResend(targetChain, gasLimit, relayProvider)
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for your specified relay provider.
     *  The relay provider will listen for these messages, and then execute the redelivery as described
     */
    function resend(ResendByTx memory request, uint32 nonce, address relayProvider)
        external
        payable
        returns (uint64 sequence);

    /**
     * @notice This 'MultichainSend' struct represents a collection of send requests 'requests' and a specified relay provider 'relayProviderAddress'
     *  This struct is used to request sending the same array of transactions to many different destination contracts (on many different chains),
     *  With each request in the 'requests' array denoting parameters for one specific destination
     *
     *  @custom:member relayProviderAddress The address of the relay provider's contract on this source chain. This relay provider will perform delivery of all of these Send requests.
     *  Use getDefaultRelayProvider() here to use the default relay provider.
     *  @custom:member requests The array of send requests, each specifying a targetAddress on a targetChain, along with information about the desired maxTransactionFee, receiverValue, refundAddress, and relayParameters
     */
    struct MultichainSend {
        address relayProviderAddress;
        Send[] requests;
    }

    /**
     * @notice The multichainSend function delivers all wormhole messages in the current transaction of nonce 'nonce' to many destinations,
     * with each destination specified in a Send struct, describing the desired targetAddress, targetChain, maxTransactionFee, receiverValue, refundAddress, and relayParameters
     *
     * @param requests The MultichainSend struct, containing the array of Send requests, as well as the desired relayProviderAddress
     * @param nonce The messages to be relayed are all of the emitted wormhole messages in the current transaction that have nonce 'nonce'
     *
     *  This function must be called with a payment of at least (one wormhole message fee) + Sum_(i=0 -> requests.requests.length - 1) [requests.requests[i].maxTransactionFee + requests.requests[i].receiverValue].
     *
     *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for the default wormhole relay provider.
     *  The relay provider will listen for these messages, and then execute the delivery as described
     */
    function multichainSend(MultichainSend memory requests, uint32 nonce) external payable returns (uint64 sequence);

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
     *  @param requests The MultichainSend struct, containing the array of Send requests, as well as the desired relayProviderAddress
     *  @param rolloverChain If LEFTOVER_VALUE > NEEDED_VALUE, then the maxTransactionFee of one of the requests in the array of sends will be incremented by 'LEFTOVER_VALUE - NEEDED_VALUE'
     *  Specifically, the 'send' that will have it's maxTransactionFee incremented is the first send in the 'requests.requests' array that has targetChain equal to 'rolloverChain'
     *  @param nonce The messages to be relayed are all of the emitted wormhole messages in the current transaction that have nonce 'nonce'
     *
     */
    function multichainForward(MultichainSend memory requests, uint16 rolloverChain, uint32 nonce) external payable;

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
        external
        pure
        returns (uint256 maxTransactionFee);

    /**
     * @notice quoteReceiverValue tells you how much receiverValue (denominated in current (source) chain currency) must be
     * in order for the relayer to pass in 'targetAmount' as msg value when calling receiveWormholeMessages.
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

    error FundsTooMuch(); // (maxTransactionFee, converted to target chain currency) + (receiverValue, converted to target chain currency) is greater than what your chosen relay provider allows
    error MaxTransactionFeeNotEnough(); // maxTransactionFee is less than the minimum needed by your chosen relay provider
    error MsgValueTooLow(); // msg.value is too low
    // Specifically, (msg.value) + (any leftover funds if this is a forward) is less than (maxTransactionFee + receiverValue), summed over all of your requests if this is a multichainSend/multichainForward
    error NonceIsZero(); // Nonce cannot be 0
    error NoDeliveryInProcess(); // Forwards can only be requested within execution of 'receiveWormholeMessages', or when a delivery is in progress
    error MultipleForwardsRequested(); // Only one forward can be requested in a transaction
    error RelayProviderDoesNotSupportTargetChain(); // Your relay provider does not support the target chain you specified
    error RolloverChainNotIncluded(); // None of the Send structs in your multiForward are for the target chain 'rolloverChain'
    error ChainNotFoundInSends(uint16 chainId); // This should never happen. Post a Github Issue if this occurs
    error ReentrantCall(); // A delivery cannot occur during another delivery
}
