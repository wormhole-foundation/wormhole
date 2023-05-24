// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "./TypedUnits.sol";

/**
 * @notice VaaKey identifies a wormhole message
 *
 * @custom:member chainId - only specified if `infoType == VaaKeyType.EMITTER_SEQUENCE`
 * @custom:member emitterAddress - only specified if `infoType = VaaKeyType.EMITTER_SEQUENCE`
 * @custom:member sequence - only specified if `infoType = VaaKeyType.EMITTER_SEQUENCE`
 */
struct VaaKey {
  uint16 chainId;
  bytes32 emitterAddress;
  uint64 sequence;
}

interface IWormholeRelayerBase {

  event SendEvent(uint64 indexed sequence, Wei deliveryQuote, Wei paymentForExtraReceiverValue);

  function getRegisteredCoreRelayerContract(uint16 chainId) external view returns (bytes32);
}

/**
  * IWormholeRelayer
  * @notice Users may use this interface to have wormhole messages (VAAs) in their transaction
  *   relayed to destination contract(s) of their choice.
  */
interface IWormholeRelayerSend is IWormholeRelayerBase {

  function sendToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Gas gasLimit
  ) external payable returns (uint64 sequence);

  /**
    * TODO
    */
  function sendToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Gas gasLimit,
    uint16 refundChainId,
    address refundAddress
  ) external payable returns (uint64 sequence);

   /**
   * @notice This `send` function emits a wormhole message (VAA) that alerts the default wormhole
   *     relay provider to call the `receiveWormholeMessage(DeliveryData memory deliveryData,
   *     bytes[] memory signedVaas)` endpoint of the contract on chain `targetChainId` and address
   *     `targetAddress` with the first argument being a `DeliveryData` struct and with the second
   *     argument (`signedVaas`) empty.
   *   This endpoint can be found in IWormholeReceiver.sol
   *
   *
   * @param targetChainId The chain that the vaas are delivered to, in Wormhole Chain ID format
   * @param targetAddress The address (in Wormhole 32-byte format) on chain `targetChainId` of the
   *     contract to which the vaas are delivered.
   *   This contract must implement the IWormholeReceiver interface, which simply requires a
   *     `receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas)`
   *     endpoint.
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
  function sendToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Wei paymentForExtraReceiverValue,
    Gas gasLimit,
    uint16 refundChainId,
    address refundAddress,
    address relayProviderAddress,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) external payable returns (uint64 sequence);

  function send(
    uint16 targetChainId,
    bytes32 targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Wei paymentForExtraReceiverValue,
    bytes memory encodedExecutionParameters,
    uint16 refundChainId,
    bytes32 refundAddress,
    address relayProviderAddress,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) external payable returns (uint64 sequence);


  /**
   * @notice This `forward` function can only be called in a IWormholeReceiver within the `receiveWormholeMessages` function
   * It's purpose is to use any leftover fee from the `maxTransactionFee` of the current delivery to fund another delivery
   *
   * Specifically, suppose an integrator requested a Send (with parameters oldTargetChain, oldTargetAddress, etc)
   * and sets quoteGas(oldTargetChain, gasLimit, oldRelayProvider) as `maxTransactionFee` in a Send,
   * but during the delivery on oldTargetChain, the call to oldTargetAddress's receiveWormholeMessages endpoint uses only x units of gas (where x < gasLimit).
   *
   * Normally, (gasLimit - x)/gasLimit * oldMaxTransactionFee, converted to target chain currency, would be refunded to `oldRefundAddress`.
   * However, if during execution of receiveWormholeMessage the integrator made a call to forward,
   *
   * We instead would use [(gasLimit - x)/gasLimit * oldMaxTransactionFee, converted to target chain currency] + (any additional funds passed into forward)
   * to fund a new delivery (of wormhole messages emitted during execution of oldTargetAddress's receiveWormholeMessages) that is requested in the call to `forward`.
   *
   * Specifically, this `forward` function is only callable within a delivery (during receiveWormholeMessages) and indicates the in-progress delivery to use any leftover funds from the current delivery to fund a new delivery
   * or equivalently, indicates the in-progress delivery to call the receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas) endpoint of the contract on chain `targetChainId` and address `targetAddress`
   * with the first argument being wormhole messages (VAAs) from the current transaction that match the descriptions in the `vaaKeys` array (which have additionally been encoded and signed by the Guardian set to form `signed VAAs`),
   * and with the second argument empty
   *
   * @param targetChainId The chain that the vaas are delivered to, in Wormhole Chain ID format
   * @param targetAddress The address (in Wormhole 32-byte format) on chain `targetChainId` of the contract to which the vaas are delivered.
   * This contract must implement the IWormholeReceiver interface, which simply requires a `receiveWormholeMessage(DeliveryData memory deliveryData, bytes[] memory signedVaas)` endpoint
   */
  function forwardToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Gas gasLimit,
    uint16 refundChainId,
    address refundAddress
  ) external payable;

  function forwardToEvm(
    uint16 targetChainId,
    address targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Wei paymentForExtraReceiverValue,
    Gas gasLimit,
    uint16 refundChainId,
    address refundAddress,
    address relayProviderAddress,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) external payable;

  function forward(
    uint16 targetChainId,
    bytes32 targetAddress,
    bytes memory payload,
    Wei receiverValue,
    Wei paymentForExtraReceiverValue,
    bytes memory encodedExecutionParameters,
    uint16 refundChainId,
    bytes32 refundAddress,
    address relayProviderAddress,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel
  ) external payable;


  /**
   * @notice This `resend` function allows a caller to request an additional delivery of a specified
   *   `send` VAA, with an updated provider, maxTransactionFee, and receiveValue.
   * This function is intended to help integrators more eaily resolve ReceiverFailure cases, or
   *   other scenarios where an delivery was not able to be correctly performed.
   *
   * No checks about the original delivery VAA are performed prior to the emission of the redelivery
   *   instruction. Therefore, caller should be careful not to request redeliveries in the following
   *   cases, as they will result in an undeliverable, invalid redelivery instruction that the
   *   provider will not be able to perform:
   *
   * - If the specified VaaKey does not correspond to a valid delivery VAA.
   * - If the targetChainId does not equal the targetChainId of the original delivery.
   * - If the gasLimit calculated from `newMaxTransactionFee` is less than the original delivery's
   *     gas limit.
   * - If the receiverValueTarget (amount of receiver value to pass into the target contract)
   *     calculated from newReceiverValue is lower than the original delivery's receiverValueTarget.
   * - If the new calculated maximumRefundTarget (maximum possible refund amount) calculated from
   *     `newMaxTransactionFee` is lower than the original delivery's maximumRefundTarget.
   *
   * Similar to send, you must call this function with `msg.value = nexMaxTransactionFee +
   *  newReceiverValue + wormhole.messageFee()` in order to pay for the delivery.
   *
   * @param deliveryVaaKey a VAA Key corresponding to the delivery which should be performed again. This must
   *     correspond to a valid delivery instruction VAA.
   * @param targetChainId - the chain which the original delivery targetted.
   * @param newRelayProviderAddress - the address of the relayProvider (on this chain) which should be
   *     used for this redelivery.
   */
  function resendToEvm(
    VaaKey memory deliveryVaaKey,
    uint16 targetChainId,
    Wei newReceiverValue,
    Gas newGasLimit,
    address newRelayProviderAddress
  ) external payable returns (uint64 sequence);

  function resend(
    VaaKey memory deliveryVaaKey,
    uint16 targetChainId,
    Wei newReceiverValue,
    bytes memory newEncodedExecutionParameters,
    address newRelayProviderAddress
  ) external payable returns (uint64 sequence);


  function quoteEVMDeliveryPrice(uint16 targetChainId, uint128 receiverValue, uint32 gasLimit) external view returns (uint256 nativePriceQuote, uint256 targetChainRefundPerGasUnused);

  function quoteEVMDeliveryPrice(uint16 targetChainId, uint128 receiverValue, uint32 gasLimit, address relayProviderAddress) external view returns (uint256 nativePriceQuote, uint256 targetChainRefundPerGasUnused);

  function quoteDeliveryPrice(uint16 targetChainId, uint128 receiverValue, bytes memory encodedExecutionParameters, address relayProviderAddress) external view returns (uint256 nativePriceQuote, bytes memory encodedExecutionInfo);

  function quoteAssetConversion(
    uint16 targetChainId,
    uint128 currentChainAmount,
    address relayProviderAddress
  ) external view returns (uint256 targetChainAmount);

  /**
   * @notice Returns the address of the current default relay provider
   * @return relayProvider The address of (the default relay provider)'s contract on this source
   *   chain. This must be a contract that implements IRelayProvider.
   */
  function getDefaultRelayProvider() external view returns (address relayProvider);
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
    CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED,
    CROSS_CHAIN_REFUND_FAIL_NOT_ENOUGH
  }

  /**
   * @custom:member recipientContract - The target contract address
   * @custom:member sourceChainId - The chain which this delivery was requested from (in wormhole
   *     ChainID format)
   * @custom:member sequence - The wormhole sequence number of the delivery VAA on the source chain
   *     corresponding to this delivery request
   * @custom:member deliveryVaaHash - The hash of the delivery VAA corresponding to this delivery
   *     request
   * @custom:member gasUsed - The amount of gas that was used to call your target contract (and, if
   *     there was a forward, to ensure that there were enough funds to complete the forward)
   * @custom:member status:
   *   - RECEIVER_FAILURE, if the target contract reverts
   *   - SUCCESS, if the target contract doesn't revert and no forwards were requested
   *   - FORWARD_REQUEST_FAILURE, if the target contract doesn't revert, forwards were requested,
   *       but provided/leftover funds were not sufficient to cover them all
   *   - FORWARD_REQUEST_SUCCESS, if the target contract doesn't revert and all forwards are covered
   * @custom:member additionalStatusInfo:
   *   - If status is SUCCESS or FORWARD_REQUEST_SUCCESS, then this is empty.
   *   - If status is RECEIVER_FAILURE, this is `RETURNDATA_TRUNCATION_THRESHOLD` bytes of the
   *       return data (i.e. potentially truncated revert reason information).
   *   - If status is FORWARD_REQUEST_FAILURE, this is also the return data, which is specifically
   *       an error ForwardNotSufficientlyFunded(uint256 amountOfFunds, uint256 amountOfFundsNeeded)
   * @custom:member refundStatus - Result of the refund. REFUND_SUCCESS or REFUND_FAIL are for
   *     refunds where targetChainId=refundChainId; the others are for targetChainId!=refundChainId,
   *     where a cross chain refund is necessary
   * @custom:member overridesInfo:
   *   - If not an override: empty bytes array
   *   - Otherwise: An encoded `DeliveryOverride`
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
   * @notice The relay provider calls `deliver` to relay messages as described by one delivery instruction
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
   * @param encodedVMs - An array of signed wormhole messages (all from the same source chain
   *     transaction)
   * @param encodedDeliveryVAA - Signed wormhole message from the source chain's CoreRelayer
   *     contract with payload being the encoded delivery instruction container
   * @param relayerRefundAddress - The address to which any refunds to the relay provider
   *     should be sent
   * @param deliveryOverrides - Optional overrides field which must be either an empty bytes array or
   *     it must be an encoded DeliveryOverride struct
   */
  function deliver(
    bytes[] memory encodedVMs,
    bytes memory encodedDeliveryVAA,
    address payable relayerRefundAddress,
    bytes memory deliveryOverrides
  ) external payable;
}

interface IWormholeRelayer is IWormholeRelayerDelivery, IWormholeRelayerSend {}

/*
 *  Errors thrown by IWormholeRelayer contract
 */

//132 is chosen because 132 = 4 (function selector) + 4*32 (4 32-byte words)
uint256 constant RETURNDATA_TRUNCATION_THRESHOLD = 132;

//When msg.value was not equal to (one wormhole message fee) + `maxTransactionFee` + `receiverValue`
error InvalidMsgValue(Wei msgValue, Wei totalFee);

error RequestedGasLimitTooLow(); 

error RelayProviderDoesNotSupportTargetChain(address relayer, uint16 chainId);

//When calling `forward()` on the CoreRelayer if no delivery is in progress
error NoDeliveryInProgress(); 
//When calling `delivery()` a second time even though a delivery is already in progress
error ReentrantDelivery(address msgSender, address lockedBy);
//When any other contract but the delivery target calls `forward()` on the CoreRelayer while a
//  delivery is in progress
error ForwardRequestFromWrongAddress(address msgSender, address deliveryTarget);

error InvalidPayloadId(uint8 parsed, uint8 expected);
error InvalidPayloadLength(uint256 received, uint256 expected);
error InvalidVaaKeyType(uint8 parsed);

error InvalidDeliveryVaa(string reason);
//When the delivery VAA (signed wormhole message with delivery instructions) was not emitted by the
//  registered CoreRelayer contract
error InvalidEmitter(bytes32 emitter, bytes32 registered, uint16 chainId);
error VaaKeysLengthDoesNotMatchVaasLength(uint256 keys, uint256 vaas);
error VaaKeysDoNotMatchVaas(uint8 index);
//When someone tries to call an external function of the CoreRelayer that is only intended to be
//  called by the CoreRelayer itself (to allow retroactive reverts for atomicity)
error RequesterNotCoreRelayer();

//When trying to relay a `DeliveryInstruction` to any other chain but the one it was specified for
error TargetChainIsNotThisChain(uint16 targetChainId);
error ForwardNotSufficientlyFunded(Wei amountOfFunds, Wei amountOfFundsNeeded);
//When a `DeliveryOverride` contains a gas limit that's less than the original
error InvalidOverrideGasLimit();
//When a `DeliveryOverride` contains a receiver value that's less than the original
error InvalidOverrideReceiverValue();
//When a `DeliveryOverride` contains a maximum refund that's less than the original
error InvalidOverrideRefundPerGasUnused();

//When the relay provider doesn't pass in sufficient funds (i.e. msg.value does not cover the
//  necessary budget fees)
error InsufficientRelayerFunds(Wei msgValue, Wei minimum);

//When a bytes32 field can't be converted into a 20 byte EVM address, because the 12 padding bytes
//  are non-zero (duplicated from Utils.sol)
error NotAnEvmAddress(bytes32);