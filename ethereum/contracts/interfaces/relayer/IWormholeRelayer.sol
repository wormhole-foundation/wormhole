// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

//TODO AMO: Decide on use regarding '' vs `` quotation of variables/function names in comments
//            and then make it uniform.

//TODO AMO: 132 because presumably 4 (function selector) + 4*32 (4 32-byte words)
uint256 constant RETURNDATA_TRUNCATION_THRESHOLD = 132;

error InvalidMsgValue(uint256 msgValue, uint256 totalFee);
//Specifically, (msg.value) + (any leftover funds if this is a forward) is less than
//  (maxTransactionFee + receiverValue), summed over all of your requests if this is a
//  multichainSend/multichainForward
error InsufficientMaxTransactionFee(); 
//(maxTransactionFee, converted to target chain currency) + (receiverValue, converted to target
//  chain currency) is greater than what your chosen relay provider allows
error ExceedsMaximumBudget(
  uint256 requested,
  uint256 maximum,
  address relayProvider,
  uint16 chainId
);

error RelayProviderQuotedBogusAssetPrice(address relayProvider, uint16 chainId, uint256 price);
error RelayProviderDoesNotSupportTargetChain(address relayer, uint16 chainId);

//Forwards can only be requested within execution of 'receiveWormholeMessages', or when a delivery
//  is in progress
error NoDeliveryInProgress(); 
// A delivery cannot occur during another delivery
error ReentrantDelivery(address msgSender, address lockedBy);
//A forward was requested from an address that is not the 'targetAddress' of the original delivery.
error ForwardRequestFromWrongAddress(address msgSender, address deliveryTarget);

error InvalidPayloadId(uint8 parsed, uint8 expected);
error InvalidPayloadLength(uint256 received, uint256 expected);
error InvalidVaaKeyType(uint8 parsed);

error InvalidDeliveryVaa(string reason);
// The delivery VAA (signed wormhole message with delivery instructions) was not emitted by the registered CoreRelayer contract
error InvalidEmitter(bytes32 emitter, bytes32 registered, uint16 chainId);
error VaaKeysLengthDoesNotMatchVaasLength(uint256 keys, uint256 vaas); // The VAA array has a different length than the original array of VaaKey descriptions from the source chain
error VaaKeysDoNotMatchVaas(uint8 index); // The VAA at index 'index' does not match the 'index'-th description given on the source chain in the 'messages' field
error RequesterNotCoreRelayer();

error SendNotSufficientlyFunded(); // The container of delivery instructions (for which this current delivery was in) was not fully funded on the source chain
error TargetChainIsNotThisChain(uint16 targetChainId); // The specified target chain is not the current chain
error ForwardNotSufficientlyFunded(uint256 amountOfFunds, uint256 amountOfFundsNeeded); // Should never happen as this should have already been checked for
error InvalidOverrideGasLimit(); // Invalid gas limit override was passed in to the delivery
error InvalidOverrideReceiverValue(); // Invalid receiver value override was passed in to the delivery
error InvalidOverrideMaximumRefund(); // Invalid maximum refund override was passed in to the delivery

error InsufficientRelayerFunds(uint256 msgValue, uint256 minimum); // The relay provider didn't pass in sufficient funds (msg.value does not cover the necessary budget fees)

//duplicated from utils.sol:
error NotAnEvmAddress(bytes32);
error OutOfBounds(uint256 offset, uint256 length);

enum VaaKeyType {
  EMITTER_SEQUENCE,
  VAAHASH
}

/**
 * @notice This 'VaaKey' struct identifies a wormhole message from the current transaction
 *
 * @custom:member infoType - determines which of the remaining fields are actually specified
 * @custom:member chainId - only specified if infoType == VaaKeyType.EMITTER_SEQUENCE
 * @custom:member emitterAddress - only specified if infoType = VaaKeyType.EMITTER_SEQUENCE
 * @custom:member sequence - only specified if infoType = VaaKeyType.EMITTER_SEQUENCE
 * @custom:member vaaHash - only specified if infoType = VaaKeyType.VAAHASH
 */
struct VaaKey {
  VaaKeyType infoType;
  uint16 chainId;
  bytes32 emitterAddress;
  uint64 sequence;
  bytes32 vaaHash;
}

struct ExecutionParameters {
  uint32 gasLimit;
}

struct DeliveryInstruction {
  uint16 targetChainId;
  bytes32 targetAddress;
  uint16 refundChainId;
  bytes32 refundAddress;
  uint256 maximumRefundTarget;
  uint256 receiverValueTarget;
  bytes32 sourceRelayProvider;
  bytes32 targetRelayProvider;
  bytes32 senderAddress;
  VaaKey[] vaaKeys;
  uint8 consistencyLevel;
  ExecutionParameters executionParameters;
  bytes payload;
}

struct RedeliveryInstruction {
  VaaKey key;
  uint256 newMaximumRefundTarget;
  uint256 newReceiverValueTarget;
  bytes32 sourceRelayProvider;
  uint16 targetChainId;
  ExecutionParameters executionParameters;
}

/**
 * @notice DeliveryOverride is a struct which can alter several aspects of a delivery.
 *
 * @custom:member gaslimit override, must be greater than the gasLimit specified in the delivery instruction.
 * @custom:member maximumRefund override, must be greater than or equal to the maximumRefund specified in the delivery instruction.
 * @custom:member receiverValue override, must be greater than or equal to the receiverValue specified in the delivery instruction.
 * @custom:member the hash of the redelivery which is being performed, or 0 if none.
 */
struct DeliveryOverride {
  uint32 gasLimit;
  uint256 maximumRefund;
  uint256 receiverValue;
  bytes32 redeliveryHash;
}

/**
 * @notice This 'Send' struct represents a request to relay to a contract at address 'targetAddress' on chain 'targetChainId'
 *
 * @custom:member targetChainId The chain that the encoded+signed Wormhole messages (VAAs) are delivered to, in Wormhole Chain ID format
 * @custom:member targetAddress The address (in Wormhole 32-byte format) on chain 'targetChainId' of the contract to which the vaas are delivered.
 * This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData)' endpoint
 * @custom:member refundAddress The address (in Wormhole 32-byte format) on chain 'targetChainId' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
 * @custom:member maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
 * If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
 * Any unused value out of this fee will be refunded to 'refundAddress'
 * @custom:member receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
 * @custom:member relayProviderAddress The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
 * If request.maxTransactionFee >= quoteGas(request.targetChainId, gasLimit, relayProvider),
 * then as long as 'request.targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
 * If request.receiverValue >= quoteReceiverValue(request.targetChainId, targetAmount, relayProvider), then at least 'targetAmount' of targetChainId currency will be passed into the 'receiveWormholeFunction' as value.
 * To use the default relay provider, set this field to be getDefaultRelayProvider()
 * @custom:member vaaKeys Array of VaaKey structs identifying each message to be relayed. Each VaaKey struct specifies a wormhole message, either by the VAA hash, or by the (chain id, emitter address, sequence number) triple.
 * The relay provider will set the second parameter in the call to receiveWormholeMessages to be an array of signed VAAs specified by this vaaKeys array.
 * Specifically, the 'signedVaas' array will have the same length as 'vaaKeys', and additionally for each 0 <= i < vaaKeys.length, signedVaas[i] will match the description in vaaKeys[i]
 * @custom:member consistencyLevel The level of finality to reach before emitting the Wormhole VAA corresponding to this 'send' request. See https://book.wormhole.com/wormhole/3_coreLayerContracts.html#consistency-levels
 * @custom:member payload an optional payload which will be delivered to the receiving contract.
 * @custom:member relayParameters This should be 'getDefaultRelayParameters()'
 */
  //TODO AMO: Why does this struct exist as an externally visible struct?
  //TODO AMO: Reconsider order of parameters to be consistent with function parameter
  //            (either change struct or functions)
  struct Send {
    uint16 targetChainId;
    bytes32 targetAddress;
    uint16 refundChainId;
    bytes32 refundAddress;
    uint256 maxTransactionFee;
    uint256 receiverValue;
    address relayProviderAddress;
    VaaKey[] vaaKeys;
    uint8 consistencyLevel;
    bytes payload;
    bytes relayParameters;
  }

interface IWormholeRelayerBase {
  event SendEvent(uint64 indexed sequence, uint256 maxTxFee, uint256 receiverValue);

  function getRegisteredCoreRelayerContract(uint16 chainId) external view returns (bytes32);
}

interface IWormholeRelayerSend is IWormholeRelayerBase {
  /**
   * @title IWormholeRelayer
   * @notice Users may use this interface to have wormhole messages (VAAs) in their transaction
   *   relayed to destination contract(s) of their choice.
   */

  /**
   * @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole
   * relay provider to call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[]
   * memory signedVaas) endpoint of the contract on chain 'targetChainId' and address
   //TODO AMO: another duplication of DeliveryData documentation
   *
   * 'targetAddress' with the first argument being a DeliveryData struct with fields:
   *  - sourceAddress:
   *      Address (in wormhole 32-byte format) that called 'send' on the source chain.
   *  - sourceChainId:
   *      The wormhole chainID of the source chain (this current chain).
   *  - maximumRefund:
   *      The maximum transaction fee refund that can possibly be awarded (to refundAddress) at
   *        the end of this delivery, assuming no gas is used by receiveWormholeMessages.
   *      This is calculated by subtracting the relayer's base fee for the targetChainId from
   *        'maxTransactionFee' and then converting to target chain currency.
   *  - deliveryHash:
   *      The VAA hash of the deliveryVAA. If you do not want to potentially process this
   *        delivery multiple times, you should store this hash in state for replay protection.
   *  - payload:
   *      The arbitrary payload (bytes).
   * and with the second argument (signedVaas) empty.
   *
   *
   * @param targetChainId The chain that the vaas are delivered to, in Wormhole Chain ID format
   * @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChainId' of the
   *     contract to which the vaas are delivered.
   *   This contract must implement the IWormholeReceiver interface, which simply requires a
   *     'receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas)'
   *     endpoint.
   * @param refundChainId The chain where the refund should be sent. If the refundChainId is not the
   *     targetChainId, a new empty delivery will be initiated in order to perform the refund,
   *     which is subject to the provider's rates on the target chain.
   * @param refundAddress The address (in Wormhole 32-byte format) on chain 'refundChainId' to
   *     which any leftover funds (that weren't used for target chain gas or passed into
   *     targetAddress as value) should be sent.
   * @param maxTransactionFee The maximum amount (denominated in source chain (this chain)
   *     currency) that you wish to spend on funding gas for the target chain.
   *   If more gas is needed on the target chain than is paid for, there will be a Receiver
   *     Failure and the call to targetAddress will revert.
   *   Any unused value out of this fee will be refunded to refundAddress'.
   *   If maxTransactionFee >= quoteGas(targetChainId, gasLimit, getDefaultRelayProvider()), then
   *     as long as targetAddress's receiveWormholeMessage function uses at most 'gasLimit'
   *     units of gas (and doesn't revert), the delivery will succeed.
   * @param receiverValue The amount (denominated in source chain currency) that will be converted
   *     to target chain currency and passed into the receiveWormholeMessage endpoint as value.
   *   If receiverValue >= quoteReceiverValue(targetChainId, targetAmount,
   *     getDefaultRelayProvider()), then at least 'targetAmount' of targetChainId currency will be
   *     passed into the 'receiveWormholeFunction' as value.
   * @param payload An arbitrary payload which will be sent to the receiver contract.
   *
   * This function must be called with a payment of exactly:
   *   maxTransactionFee + receiverValue + one wormhole message fee
   *
   * @return sequence The sequence number for the emitted wormhole message, which contains
   *     encoded delivery instructions meant for the default wormhole relay provider.
   *   The relay provider will listen for these messages, and then execute the delivery as
   *     described.
   */
  function send(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload
  ) external payable returns (uint64 sequence);

  /**
   *  @notice This 'send' function emits a wormhole message (VAA) that alerts the default wormhole relay provider to
   *  call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas) endpoint of the contract on chain 'targetChainId' and address 'targetAddress'
   *  with the first argument being a DeliveryData struct with fields:
   *    - sourceAddress: address (in wormhole 32-byte format) that called 'send' on the source chain
   *    - sourceChainId: The wormhole chainID of the source chain (this current chain)
   *    - maximumRefund: The maximum transaction fee refund that can possibly be awarded (to refundAddress) at the end of this delivery, assuming no gas is used by receiveWormholeMessages
   *             This is calculated by subtracting the relayer's base fee for the targetChainId from 'maxTransactionFee' and then converting to target chain currency
   *    - deliveryHash: The VAA hash of the deliveryVAA. If you do not want to potentially process this delivery multiple times, you should store this hash in state for replay protection
   *    - payload: the arbitrary payload (bytes).
   *  and with the second argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'vaaKeys' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs')
   *
   *
   *  @param targetChainId The chain that the vaas are delivered to, in Wormhole Chain ID format
   *  @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChainId' of the contract to which the vaas are delivered.
   *  This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas)' endpoint
   *  @param refundChainId The chain where the refund should be sent. If the refundChainId is not the targetChainId, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
   *  @param refundAddress The address (in Wormhole 32-byte format) on chain 'refundChainId' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
   *  @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
   *  If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
   *  Any unused value out of this fee will be refunded to 'refundAddress'
   *  If maxTransactionFee >= quoteGas(targetChainId, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
   *  @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
   *  If receiverValue >= quoteReceiverValue(targetChainId, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChainId currency will be passed into the 'receiveWormholeFunction' as value.
   *  @param payload an arbitrary payload which will be sent to the receiver contract.
   *  @param vaaKeys Array of VaaKey structs identifying each message to be relayed. Each VaaKey struct specifies a wormhole message, either by the VAA hash, or by the (chain id, emitter address, sequence number) triple.
   *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this vaaKeys array.
   *  Specifically, the 'signedVaas' array will have the same length as 'vaaKeys', and additionally for each 0 <= i < vaaKeys.length, signedVaas[i] will match the description in vaaKeys[i]
   *  @param consistencyLevel  The level of finality to reach before emitting the Wormhole VAA corresponding to this 'send' request. See https://book.wormhole.com/wormhole/3_coreLayerContracts.html#consistency-levels
   *
   * This function must be called with a payment of exactly:
   *   maxTransactionFee + receiverValue + one wormhole message fee
   *
   *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for the default wormhole relay provider.
   *  The relay provider will listen for these messages, and then execute the delivery as described.
   */
  function send(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) external payable returns (uint64 sequence);

  /**
   *  @notice This 'send' function emits a wormhole message (VAA) that alerts the relay provider specified by sendParams.relayProviderAddress to
   *  call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas) endpoint of the contract on chain 'sendParams.targetChainId' and address 'sendParams.targetAddress'
   //TODO AMO: Jesus H Christ, how often is this duplicated?!?
   *  with the first argument being a DeliveryData struct with fields:
   *    - sourceAddress: address (in wormhole 32-byte format) that called 'send' on the source chain
   *    - sourceChainId: The wormhole chainID of the source chain (this current chain)
   *    - maximumRefund: The maximum transaction fee refund that can possibly be awarded (to sendParams.refundAddress) at the end of this delivery, assuming no gas is used by receiveWormholeMessages
   *             This is calculated by subtracting the relayer's base fee for the targetChainId from 'maxTransactionFee' and then converting to target chain currency
   *    - deliveryHash: The VAA hash of the deliveryVAA. If you do not want to potentially process this delivery multiple times, you should store this hash in state for replay protection
   *    - payload: the arbitrary payload (bytes).
   *  and with the second argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'sendParams.vaaKeys' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs')
   *
   *
   *  @param sendParams The Send request containing info about the targetChainId, targetAddress, refundAddress, maxTransactionFee, receiverValue, relayProviderAddress, vaaKeys, consistencyLevel, payload, and relayParameters
   *
   * This function must be called with a payment of exactly:
   *   maxTransactionFee + receiverValue + one wormhole message fee
   *
   *  @return sequence The sequence number for the emitted wormhole message, which contains encoded delivery instructions meant for your specified relay provider.
   *  The relay provider will listen for these messages, and then execute the delivery as described.
   */
  function send(Send memory sendParams) external payable returns (uint64 sequence);

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
   * or equivalently, indicates the in-progress delivery to call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas) endpoint of the contract on chain 'targetChainId' and address 'targetAddress'
   * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'vaaKeys' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
   * and with the second argument empty
   *
   * @param targetChainId The chain that the vaas are delivered to, in Wormhole Chain ID format
   * @param targetAddress The address (in Wormhole 32-byte format) on chain 'targetChainId' of the contract to which the vaas are delivered.
   * This contract must implement the IWormholeReceiver interface, which simply requires a 'receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas)' endpoint
   * @param refundAddress The address (in Wormhole 32-byte format) on chain 'refundChainId' to which any leftover funds (that weren't used for target chain gas or passed into targetAddress as value) should be sent
   * @param refundChainId The chain where the refund should be sent. If the refundChainId is not the targetChainId, a new empty delivery will be initiated in order to perform the refund, which is subject to the provider's rates on the target chain.
   * @param maxTransactionFee The maximum amount (denominated in source chain (this chain) currency) that you wish to spend on funding gas for the target chain.
   * If more gas is needed on the target chain than is paid for, there will be a Receiver Failure.
   * Any unused value out of this fee will be refunded to 'refundAddress'
   * If maxTransactionFee >= quoteGas(targetChainId, gasLimit, getDefaultRelayProvider()), then as long as 'targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
   * @param receiverValue The amount (denominated in source chain currency) that will be converted to target chain currency and passed into the receiveWormholeMessage endpoint as value.
   * If receiverValue >= quoteReceiverValue(targetChainId, targetAmount, getDefaultRelayProvider()), then at least 'targetAmount' of targetChainId currency will be passed into the 'receiveWormholeFunction' as value.
   * @param vaaKeys Array of VaaKey structs identifying each message to be relayed. Each VaaKey struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
   * The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this vaaKeys array.
   * Specifically, the 'signedVaas' array will have the same length as 'vaaKeys', and additionally for each 0 <= i < vaaKeys.length, signedVaas[i] will match the description in vaaKeys[i]
   * @param consistencyLevel The level of finality to reach before emitting the Wormhole VAA corresponding to this 'forward' request. See https://book.wormhole.com/wormhole/3_coreLayerContracts.html#consistency-levels
   *
   * This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
   */
  function forward(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
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
   * or equivalently, indicates the in-progress delivery to call the receiveWormholeMessage(bytes[] memory vaas, bytes[] memory additionalData) endpoint of the contract on chain 'targetChainId' and address 'targetAddress'
   * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the 'vaaKeys' array (which have additionally been encoded and signed by the Guardian set to form 'signed VAAs'),
   * and with the second argument empty
   *
   * @param sendParams The Send request containing info about the targetChainId, targetAddress, refundAddress, maxTransactionFee, receiverValue, and relayParameters
   * (specifically, the send info that will be used to deliver all of the wormhole messages emitted during the execution of oldTargetAddress's receiveWormholeMessages)
   * This forward will succeed if (leftover funds from the current delivery that would have been refunded) + (any extra msg.value passed into forward) is at least maxTransactionFee + receiverValue + one wormhole message fee.
   * notparam vaaKeys Array of VaaKey structs identifying each message to be relayed. Each VaaKey struct specifies a wormhole message in the current transaction, either by the VAA hash, or by the (emitter address, sequence number) pair.
   * The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this vaaKeys array.
   * Specifically, the 'signedVaas' array will have the same length as 'vaaKeys', and additionally for each 0 <= i < vaaKeys.length, signedVaas[i] will match the description in vaaKeys[i]
   * notparam relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
   * If sendParams.maxTransactionFee >= quoteGas(sendParams.targetChainId, gasLimit, relayProvider),
   * then as long as 'sendParams.targetAddress''s receiveWormholeMessage function uses at most 'gasLimit' units of gas (and doesn't revert), the delivery will succeed
   * If sendParams.receiverValue >= quoteReceiverValue(sendParams.targetChainId, targetAmount, relayProvider), then at least 'targetAmount' of targetChainId currency will be passed into the 'receiveWormholeFunction' as value.
   * To use the default relay provider, set this field to be getDefaultRelayProvider()
   *
   * This function must be called with a payment of exactly sendParams.maxTransactionFee + sendParams.receiverValue + one wormhole message fee.
   */
  function forward(Send memory sendParams) external payable;

  /**
   * @notice This 'resend' function allows a caller to request an additional delivery of a specified `send` VAA, with an updated provider, maxTransactionFee, and receiveValue.
   * This function is intended to help integrators more eaily resolve ReceiverFailure cases, or other scenarios where an delivery was not able to be correctly performed.
   *
   * No checks about the original delivery VAA are performed prior to the emission of the redelivery instruction. Therefore, caller should be careful not to request
   * redeliveries in the following cases, as they will result in an undeliverable, invalid redelivery instruction that the provider will not be able to perform:
   *
   * - If the specified VaaKey does not correspond to a valid delivery VAA.
   * - If the targetChainId does not equal the targetChainId of the original delivery.
   * - If the gasLimit calculated from 'newMaxTransactionFee' is less than the original delivery's gas limit.
   * - If the receiverValueTarget (amount of receiver value to pass into the target contract) calculated from newReceiverValue is lower than the original delivery's receiverValueTarget.
   * - If the new calculated maximumRefundTarget (maximum possible refund amount) calculated from 'newMaxTransactionFee' is lower than the original delivery's maximumRefundTarget.
   *
   * Similar to send, you must call this function with msg.value = nexMaxTransactionFee + newReceiverValue + wormhole.messageFee() in order to pay for the delivery.
   *
   * @param key a VAA Key corresponding to the delivery which should be performed again. This must correspond to a valid delivery instruction VAA.
   * @param newMaxTransactionFee - the maxTransactionFee (in this chain's wei) that should be used for the redelivery. Must be greater than or equal to the new relayProvider's quoted price for the original gas amount (i.e. must not result in a lower gas limit)
   * AND must result in a maximum transaction fee refund equal to or greater than the original delivery
   * @param newReceiverValue - the receiverValue (in this chain's wei) that should be used for the redelivery. Must result in receiverValue on the target chain which is equal to or greater than the original delivery.
   * @param targetChainId - the chain which the original delivery targetted.
   * @param relayProviderAddress - the address of the relayProvider (on this chain) which should be used for this redelivery.
   */
  function resend(
    VaaKey memory key,
    uint256 newMaxTransactionFee,
    uint256 newReceiverValue,
    uint16 targetChainId,
    address relayProviderAddress
  ) external payable returns (uint64 sequence);

  /**
   * @notice quoteGas tells you how much maxTransactionFee (denominated in current (source) chain currency) must be in order to fund a call to
   * receiveWormholeMessages on a contract on chain 'targetChainId' that uses 'gasLimit' units of gas
   *
   * Specifically, for a Send 'request',
   * If 'request.targetAddress''s receiveWormholeMessage function uses 'gasLimit' units of gas,
   * then we must have request.maxTransactionFee >= quoteGas(request.targetChainId, gasLimit, relayProvider)
   *
   * @param targetChainId the target chain that you wish to use gas on
   * @param gasLimit the amount of gas you wish to use
   * @param relayProvider The address of (the relay provider you wish to deliver the messages)'s contract on this source chain. This must be a contract that implements IRelayProvider.
   *
   * @return maxTransactionFee The 'maxTransactionFee' you pass into your request (to relay messages to 'targetChainId' and use 'gasLimit' units of gas) must be at least this amount
   */
  function quoteGas(
    uint16 targetChainId,
    uint32 gasLimit,
    address relayProvider
  ) external view returns (uint256 maxTransactionFee);

  /**
   * @notice quoteReceiverValue tells you how much receiverValue (denominated in current (source)
   *   chain currency) must be in order for the relay provider to pass in 'targetAmount' as msg
   *   value when calling receiveWormholeMessages.
   *
   * Specifically, for a send 'request',
   * In order for 'request.targetAddress''s receiveWormholeMessage function to be called with
   *   'targetAmount' of value, then we must have request.receiverValue >=
   *   quoteReceiverValue(request.targetChainId, targetAmount, relayProvider)
   *
   * @param targetChainId the target chain that you wish to receive value on
   * @param targetAmount the amount of value you wish to be passed into receiveWormholeMessages
   * @param relayProvider The address of (the relay provider you wish to deliver the messages)'s
   *   contract on this source chain. This must be a contract that implements IRelayProvider.
   *
   * @return receiverValue The 'receiverValue' you pass into your send request (to relay messages
   *   to 'targetChainId' with 'targetAmount' of value) must be at least this amount
   */
  function quoteReceiverValue(
    uint16 targetChainId,
    uint256 targetAmount,
    address relayProvider
  ) external view returns (uint256 receiverValue);

  /**
   * @notice Returns the address of the current default relay provider
   * @return relayProvider The address of (the default relay provider)'s contract on this source
   *   chain. This must be a contract that implements IRelayProvider.
   */
  function getDefaultRelayProvider() external view returns (address relayProvider);

  /**
   * @notice Returns default relay parameters
   * @return relayParams default relay parameters
   */
  function getDefaultRelayParams() external view returns (bytes memory relayParams);
}

/**
 * @notice TargetDeliveryParameters is the struct that the relay provider passes into 'deliver'
 * containing an array of the signed wormhole messages that are to be relayed
 *
 * @custom:member encodedVMs An array of signed wormhole messages (all from the same source chain transaction)
 * @custom:member encodedDeliveryVAA signed wormhole message from the source chain's CoreRelayer contract with payload being the encoded delivery instruction container
 * @custom:member relayerRefundAddress The address to which any refunds to the relay provider should be sent
 * @custom:member overrides. Optional overrides field which must parse to executionParameters. //TODO AMO: this seems wrong
 */
//TODO AMO: Why does this struct exist in the first place?
struct TargetDeliveryParameters {
  bytes[] encodedVMs;
  bytes encodedDeliveryVAA;
  address payable relayerRefundAddress;
  bytes overrides; //optional, encoded DeliveryOverride struct
}

interface IWormholeRelayerDelivery is IWormholeRelayerBase {
  enum DeliveryStatus {
    SUCCESS,
    RECEIVER_FAILURE,
    FORWARD_REQUEST_FAILURE,
    FORWARD_REQUEST_SUCCESS
  }

  enum RefundStatus {
    REFUND_SENT,
    REFUND_FAIL,
    CROSS_CHAIN_REFUND_SENT,
    CROSS_CHAIN_REFUND_SENT_MAXIMUM_BUDGET,
    CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED,
    CROSS_CHAIN_REFUND_FAIL_NOT_ENOUGH
  }

  /**
   * @custom:member recipientContract The target contract address
   * @custom:member sourceChainId The chain which this delivery was requested from (in wormhole ChainID format)
   * @custom:member sequence The wormhole sequence number of the delivery VAA on the source chain corresponding to this delivery request
   * @custom:member deliveryVaaHash The hash of the delivery VAA corresponding to this delivery request
   * @custom:member gasUsed The amount of gas that was used to call your target contract (and, if there was a forward, to ensure that there were enough funds to complete the forward)
   * @custom:member status either RECEIVER_FAILURE (if target contract reverts), SUCCESS (target contract doesn't revert, and no forwards requested),
   * FORWARD_REQUEST_FAILURE (target contract doesn't revert, at least one forward requested, not enough funds to cover all forwards), or FORWARD_REQUEST_SUCCESS (target contract doesn't revert, enough funds for all forwards).
   * @custom:member additionalStatusInfo empty if status is SUCCESS.
   * If status is FORWARD_REQUEST_SUCCESS, this is the amount of the leftover transaction fee used to fund the request(s) (not including any additional msg.value sent in the call(s) to forward).
   * If status is RECEIVER_FAILURE, this is 132 bytes of the return data (with revert reason information).
   * If status is FORWARD_REQUEST_FAILURE, this is also the return data,
   * which is specifically an error ForwardNotSufficientlyFunded(uint256 amountOfFunds, uint256 amountOfFundsNeeded)
   * @custom:member refundStatus Result of the refund. REFUND_SUCCESS or REFUND_FAIL are for refunds where targetChainId=refundChainId; the others are for targetChainId!=refundChainId,
   * where a cross chain refund is necessary
   * @custom:member overridesInfo // empty if not a override, else is the encoded DeliveryOverride struct
   */
  event Delivery(
    address indexed recipientContract,
    uint16 indexed sourceChainId,
    uint64 indexed sequence,
    bytes32 deliveryVaaHash,
    DeliveryStatus status,
    uint32 gasUsed,
    RefundStatus refundStatus,
    bytes additionalStatusInfo,
    bytes overridesInfo
  );

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
  function deliver(TargetDeliveryParameters memory targetParams) external payable;
}

interface IWormholeRelayer is IWormholeRelayerSend, IWormholeRelayerDelivery {}
