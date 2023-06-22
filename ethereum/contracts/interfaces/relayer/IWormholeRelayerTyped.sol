// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "./TypedUnits.sol";

/**
 * @title WormholeRelayer
 * @author 
 * @notice This project allows developers to build cross-chain applications powered by Wormhole without needing to 
 * write and run their own relaying infrastructure
 * 
 * We implement the IWormholeRelayer interface that allows users to request a delivery provider to relay a payload (and/or additional VAAs) 
 * to a chain and address of their choice.
 */

/**
 * @notice VaaKey identifies a wormhole message
 *
 * @custom:member chainId Wormhole chain ID of the chain where this VAA was emitted from
 * @custom:member emitterAddress Address of the emitter of the VAA, in Wormhole bytes32 format
 * @custom:member sequence Sequence number of the VAA
 */
struct VaaKey {
    uint16 chainId;
    bytes32 emitterAddress;
    uint64 sequence;
}

interface IWormholeRelayerBase {
    event SendEvent(
        uint64 indexed sequence, LocalNative deliveryQuote, LocalNative paymentForExtraReceiverValue
    );

    function getRegisteredWormholeRelayerContract(uint16 chainId) external view returns (bytes32);
}

/**
 * @title IWormholeRelayerSend
 * @notice The interface to request deliveries
 */
interface IWormholeRelayerSend is IWormholeRelayerBase {

    /**
     * @notice Publishes an instruction for the default delivery provider
     * to relay a payload to the address `targetAddress` on chain `targetChain` 
     * with gas limit `gasLimit` and `msg.value` equal to `receiverValue`
     * 
     * `targetAddress` must implement the IWormholeReceiver interface
     * 
     * This function must be called with `msg.value` equal to `quoteEVMDeliveryPrice(targetChain, receiverValue, gasLimit)`
     * 
     * Any refunds (from leftover gas) will be paid to the delivery provider. In order to receive the refunds, use the `sendPayloadToEvm` function 
     * with `refundChain` and `refundAddress` as parameters
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver) 
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param gasLimit gas limit with which to call `targetAddress`.
     * @return sequence sequence number of published VAA containing delivery instructions
     */
    function sendPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit
    ) external payable returns (uint64 sequence);

    /**
     * @notice Publishes an instruction for the default delivery provider
     * to relay a payload to the address `targetAddress` on chain `targetChain` 
     * with gas limit `gasLimit` and `msg.value` equal to `receiverValue`
     * 
     * Any refunds (from leftover gas) will be sent to `refundAddress` on chain `refundChain`
     * `targetAddress` must implement the IWormholeReceiver interface
     * 
     * This function must be called with `msg.value` equal to `quoteEVMDeliveryPrice(targetChain, receiverValue, gasLimit)`
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver) 
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param gasLimit gas limit with which to call `targetAddress`. Any units of gas unused will be refunded according to the
     *        `targetChainRefundPerGasUnused` rate quoted by the delivery provider
     * @param refundChain The chain to deliver any refund to, in Wormhole Chain ID format
     * @param refundAddress The address on `refundChain` to deliver any refund to
     * @return sequence sequence number of published VAA containing delivery instructions
     */
    function sendPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit,
        uint16 refundChain,
        address refundAddress
    ) external payable returns (uint64 sequence);

    /**
     * @notice Publishes an instruction for the default delivery provider
     * to relay a payload and VAAs specified by `vaaKeys` to the address `targetAddress` on chain `targetChain` 
     * with gas limit `gasLimit` and `msg.value` equal to `receiverValue`
     * 
     * `targetAddress` must implement the IWormholeReceiver interface
     * 
     * This function must be called with `msg.value` equal to `quoteEVMDeliveryPrice(targetChain, receiverValue, gasLimit)`
     * 
     * Any refunds (from leftover gas) will be paid to the delivery provider. In order to receive the refunds, use the `sendVaasToEvm` function 
     * with `refundChain` and `refundAddress` as parameters
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver) 
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param gasLimit gas limit with which to call `targetAddress`. 
     * @param vaaKeys Additional VAAs to pass in as parameter in call to `targetAddress`
     * @return sequence sequence number of published VAA containing delivery instructions
     */
    function sendVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit,
        VaaKey[] memory vaaKeys
    ) external payable returns (uint64 sequence);

    /**
     * @notice Publishes an instruction for the default delivery provider
     * to relay a payload and VAAs specified by `vaaKeys` to the address `targetAddress` on chain `targetChain` 
     * with gas limit `gasLimit` and `msg.value` equal to `receiverValue`
     * 
     * Any refunds (from leftover gas) will be sent to `refundAddress` on chain `refundChain`
     * `targetAddress` must implement the IWormholeReceiver interface
     * 
     * This function must be called with `msg.value` equal to `quoteEVMDeliveryPrice(targetChain, receiverValue, gasLimit)`
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver) 
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param gasLimit gas limit with which to call `targetAddress`. Any units of gas unused will be refunded according to the 
     *        `targetChainRefundPerGasUnused` rate quoted by the delivery provider
     * @param vaaKeys Additional VAAs to pass in as parameter in call to `targetAddress`
     * @param refundChain The chain to deliver any refund to, in Wormhole Chain ID format
     * @param refundAddress The address on `refundChain` to deliver any refund to
     * @return sequence sequence number of published VAA containing delivery instructions
     */
    function sendVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit,
        VaaKey[] memory vaaKeys,
        uint16 refundChain,
        address refundAddress
    ) external payable returns (uint64 sequence);

    /**
     * @notice Publishes an instruction for the delivery provider at `deliveryProviderAddress` 
     * to relay a payload and VAAs specified by `vaaKeys` to the address `targetAddress` on chain `targetChain` 
     * with gas limit `gasLimit` and `msg.value` equal to 
     * receiverValue + (arbitrary amount that is paid for by paymentForExtraReceiverValue of this chain's wei) in targetChain wei.
     * 
     * Any refunds (from leftover gas) will be sent to `refundAddress` on chain `refundChain`
     * `targetAddress` must implement the IWormholeReceiver interface
     * 
     * This function must be called with `msg.value` equal to 
     * quoteEVMDeliveryPrice(targetChain, receiverValue, gasLimit, deliveryProviderAddress) + paymentForExtraReceiverValue
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver) 
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param paymentForExtraReceiverValue amount (in current chain currency units) to spend on extra receiverValue 
     *        (in addition to the `receiverValue` specified)
     * @param gasLimit gas limit with which to call `targetAddress`. Any units of gas unused will be refunded according to the  
     *        `targetChainRefundPerGasUnused` rate quoted by the delivery provider
     * @param refundChain The chain to deliver any refund to, in Wormhole Chain ID format
     * @param refundAddress The address on `refundChain` to deliver any refund to
     * @param deliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @param vaaKeys Additional VAAs to pass in as parameter in call to `targetAddress`
     * @param consistencyLevel Consistency level with which to publish the delivery instructions - see 
     *        https://book.wormhole.com/wormhole/3_coreLayerContracts.html?highlight=consistency#consistency-levels
     * @return sequence sequence number of published VAA containing delivery instructions
     */
    function sendToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        Gas gasLimit,
        uint16 refundChain,
        address refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence);
    
    /**
     * @notice Publishes an instruction for the delivery provider at `deliveryProviderAddress` 
     * to relay a payload and VAAs specified by `vaaKeys` to the address `targetAddress` on chain `targetChain` 
     * with `msg.value` equal to 
     * receiverValue + (arbitrary amount that is paid for by paymentForExtraReceiverValue of this chain's wei) in targetChain wei.
     * 
     * Any refunds (from leftover gas) will be sent to `refundAddress` on chain `refundChain`
     * `targetAddress` must implement the IWormholeReceiver interface
     * 
     * This function must be called with `msg.value` equal to 
     * quoteDeliveryPrice(targetChain, receiverValue, encodedExecutionParameters, deliveryProviderAddress) + paymentForExtraReceiverValue  
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver), in Wormhole bytes32 format
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param paymentForExtraReceiverValue amount (in current chain currency units) to spend on extra receiverValue 
     *        (in addition to the `receiverValue` specified)
     * @param encodedExecutionParameters encoded information on how to execute delivery that may impact pricing
     *        e.g. for version EVM_V1, this is a struct that encodes the `gasLimit` with which to call `targetAddress`
     * @param refundChain The chain to deliver any refund to, in Wormhole Chain ID format
     * @param refundAddress The address on `refundChain` to deliver any refund to, in Wormhole bytes32 format
     * @param deliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @param vaaKeys Additional VAAs to pass in as parameter in call to `targetAddress`
     * @param consistencyLevel Consistency level with which to publish the delivery instructions - see 
     *        https://book.wormhole.com/wormhole/3_coreLayerContracts.html?highlight=consistency#consistency-levels
     * @return sequence sequence number of published VAA containing delivery instructions
     */
    function send(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        bytes memory encodedExecutionParameters,
        uint16 refundChain,
        bytes32 refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence);

    /**
     * @notice Performs the same function as a `send`, except:
     * 1)  Can only be used during a delivery (i.e. in execution of `receiveWormholeMessages`)
     * 2)  Is paid for (along with any other calls to forward) by (any msg.value passed in) + (refund leftover from current delivery)
     * 3)  Only executes after `receiveWormholeMessages` is completed (and thus does not return a sequence number)
     * 
     * The refund from the delivery currently in progress will not be sent to the user; it will instead
     * be paid to the delivery provider to perform the instruction specified here
     * 
     * Publishes an instruction for the same delivery provider (or default, if the same one doesn't support the new target chain)
     * to relay a payload to the address `targetAddress` on chain `targetChain` 
     * with gas limit `gasLimit` and with `msg.value` equal to `receiverValue`
     * 
     * The following equation must be satisfied (sum_f indicates summing over all forwards requested in `receiveWormholeMessages`):
     * (refund amount from current execution of receiveWormholeMessages) + sum_f [msg.value_f]
     * >= sum_f [quoteEVMDeliveryPrice(targetChain_f, receiverValue_f, gasLimit_f)]
     * 
     * The difference between the two sides of the above inequality will be added to `paymentForExtraReceiverValue` of the first forward requested
     * 
     * Any refunds (from leftover gas) from this forward will be paid to the same refundChain and refundAddress specified for the current delivery.
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver), in Wormhole bytes32 format
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param gasLimit gas limit with which to call `targetAddress`.
     */
    function forwardPayloadToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit
    ) external payable;

    /**
     * @notice Performs the same function as a `send`, except:
     * 1)  Can only be used during a delivery (i.e. in execution of `receiveWormholeMessages`)
     * 2)  Is paid for (along with any other calls to forward) by (any msg.value passed in) + (refund leftover from current delivery)
     * 3)  Only executes after `receiveWormholeMessages` is completed (and thus does not return a sequence number)
     * 
     * The refund from the delivery currently in progress will not be sent to the user; it will instead
     * be paid to the delivery provider to perform the instruction specified here
     * 
     * Publishes an instruction for the same delivery provider (or default, if the same one doesn't support the new target chain)
     * to relay a payload and VAAs specified by `vaaKeys` to the address `targetAddress` on chain `targetChain` 
     * with gas limit `gasLimit` and with `msg.value` equal to `receiverValue`
     * 
     * The following equation must be satisfied (sum_f indicates summing over all forwards requested in `receiveWormholeMessages`):
     * (refund amount from current execution of receiveWormholeMessages) + sum_f [msg.value_f]
     * >= sum_f [quoteEVMDeliveryPrice(targetChain_f, receiverValue_f, gasLimit_f)]
     * 
     * The difference between the two sides of the above inequality will be added to `paymentForExtraReceiverValue` of the first forward requested
     * 
     * Any refunds (from leftover gas) from this forward will be paid to the same refundChain and refundAddress specified for the current delivery.
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver), in Wormhole bytes32 format
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param gasLimit gas limit with which to call `targetAddress`. 
     * @param vaaKeys Additional VAAs to pass in as parameter in call to `targetAddress`
     */
    function forwardVaasToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        Gas gasLimit,
        VaaKey[] memory vaaKeys
    ) external payable;

    /**
     * @notice Performs the same function as a `send`, except:
     * 1)  Can only be used during a delivery (i.e. in execution of `receiveWormholeMessages`)
     * 2)  Is paid for (along with any other calls to forward) by (any msg.value passed in) + (refund leftover from current delivery)
     * 3)  Only executes after `receiveWormholeMessages` is completed (and thus does not return a sequence number)
     * 
     * The refund from the delivery currently in progress will not be sent to the user; it will instead
     * be paid to the delivery provider to perform the instruction specified here
     * 
     * Publishes an instruction for the delivery provider at `deliveryProviderAddress` 
     * to relay a payload and VAAs specified by `vaaKeys` to the address `targetAddress` on chain `targetChain` 
     * with gas limit `gasLimit` and with `msg.value` equal to 
     * receiverValue + (arbitrary amount that is paid for by paymentForExtraReceiverValue of this chain's wei) in targetChain wei.
     * 
     * Any refunds (from leftover gas) will be sent to `refundAddress` on chain `refundChain`
     * `targetAddress` must implement the IWormholeReceiver interface
     * 
     * The following equation must be satisfied (sum_f indicates summing over all forwards requested in `receiveWormholeMessages`):
     * (refund amount from current execution of receiveWormholeMessages) + sum_f [msg.value_f]
     * >= sum_f [quoteEVMDeliveryPrice(targetChain_f, receiverValue_f, gasLimit_f, deliveryProviderAddress_f) + paymentForExtraReceiverValue_f]
     * 
     * The difference between the two sides of the above inequality will be added to `paymentForExtraReceiverValue` of the first forward requested
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver), in Wormhole bytes32 format
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param paymentForExtraReceiverValue amount (in current chain currency units) to spend on extra receiverValue 
     *        (in addition to the `receiverValue` specified)
     * @param gasLimit gas limit with which to call `targetAddress`. Any units of gas unused will be refunded according to the  
     *        `targetChainRefundPerGasUnused` rate quoted by the delivery provider
     * @param refundChain The chain to deliver any refund to, in Wormhole Chain ID format
     * @param refundAddress The address on `refundChain` to deliver any refund to, in Wormhole bytes32 format
     * @param deliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @param vaaKeys Additional VAAs to pass in as parameter in call to `targetAddress`
     * @param consistencyLevel Consistency level with which to publish the delivery instructions - see 
     *        https://book.wormhole.com/wormhole/3_coreLayerContracts.html?highlight=consistency#consistency-levels
     */
    function forwardToEvm(
        uint16 targetChain,
        address targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        Gas gasLimit,
        uint16 refundChain,
        address refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) external payable;

    /**
     * @notice Performs the same function as a `send`, except:
     * 1)  Can only be used during a delivery (i.e. in execution of `receiveWormholeMessages`)
     * 2)  Is paid for (along with any other calls to forward) by (any msg.value passed in) + (refund leftover from current delivery)
     * 3)  Only executes after `receiveWormholeMessages` is completed (and thus does not return a sequence number)
     * 
     * The refund from the delivery currently in progress will not be sent to the user; it will instead
     * be paid to the delivery provider to perform the instruction specified here
     * 
     * Publishes an instruction for the delivery provider at `deliveryProviderAddress` 
     * to relay a payload and VAAs specified by `vaaKeys` to the address `targetAddress` on chain `targetChain` 
     * with `msg.value` equal to 
     * receiverValue + (arbitrary amount that is paid for by paymentForExtraReceiverValue of this chain's wei) in targetChain wei.
     * 
     * Any refunds (from leftover gas) will be sent to `refundAddress` on chain `refundChain`
     * `targetAddress` must implement the IWormholeReceiver interface
     * 
     * The following equation must be satisfied (sum_f indicates summing over all forwards requested in `receiveWormholeMessages`):
     * (refund amount from current execution of receiveWormholeMessages) + sum_f [msg.value_f]
     * >= sum_f [quoteDeliveryPrice(targetChain_f, receiverValue_f, encodedExecutionParameters_f, deliveryProviderAddress_f) + paymentForExtraReceiverValue_f]
     * 
     * The difference between the two sides of the above inequality will be added to `paymentForExtraReceiverValue` of the first forward requested
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param targetAddress address to call on targetChain (that implements IWormholeReceiver), in Wormhole bytes32 format
     * @param payload arbitrary bytes to pass in as parameter in call to `targetAddress`
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param paymentForExtraReceiverValue amount (in current chain currency units) to spend on extra receiverValue 
     *        (in addition to the `receiverValue` specified)
     * @param encodedExecutionParameters encoded information on how to execute delivery that may impact pricing
     *        e.g. for version EVM_V1, this is a struct that encodes the `gasLimit` with which to call `targetAddress`
     * @param refundChain The chain to deliver any refund to, in Wormhole Chain ID format
     * @param refundAddress The address on `refundChain` to deliver any refund to, in Wormhole bytes32 format
     * @param deliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @param vaaKeys Additional VAAs to pass in as parameter in call to `targetAddress`
     * @param consistencyLevel Consistency level with which to publish the delivery instructions - see 
     *        https://book.wormhole.com/wormhole/3_coreLayerContracts.html?highlight=consistency#consistency-levels
     */
    function forward(
        uint16 targetChain,
        bytes32 targetAddress,
        bytes memory payload,
        TargetNative receiverValue,
        LocalNative paymentForExtraReceiverValue,
        bytes memory encodedExecutionParameters,
        uint16 refundChain,
        bytes32 refundAddress,
        address deliveryProviderAddress,
        VaaKey[] memory vaaKeys,
        uint8 consistencyLevel
    ) external payable;

    /**
     * @notice Requests a previously published delivery instruction to be redelivered 
     * (e.g. with a different delivery provider)
     *
     * This function must be called with `msg.value` equal to 
     * quoteEVMDeliveryPrice(targetChain, newReceiverValue, newGasLimit, newDeliveryProviderAddress)
     * 
     *  @notice *** This will only be able to succeed if the following is true **
     *         - newGasLimit >= gas limit of the old instruction
     *         - newReceiverValue >= receiver value of the old instruction
     *         - newDeliveryProvider's `targetChainRefundPerGasUnused` >= old relay provider's `targetChainRefundPerGasUnused`
     * 
     * @param deliveryVaaKey VaaKey identifying the wormhole message containing the 
     *        previously published delivery instructions
     * @param targetChain The target chain that the original delivery targeted. Must match targetChain from original delivery instructions
     * @param newReceiverValue new msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param newGasLimit gas limit with which to call `targetAddress`. Any units of gas unused will be refunded according to the  
     *        `targetChainRefundPerGasUnused` rate quoted by the delivery provider, to the refund chain and address specified in the original request
     * @param newDeliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @return sequence sequence number of published VAA containing redelivery instructions
     *
     * @notice *** This will only be able to succeed if the following is true **
     *         - newGasLimit >= gas limit of the old instruction
     *         - newReceiverValue >= receiver value of the old instruction
     *         - newDeliveryProvider's `targetChainRefundPerGasUnused` >= old relay provider's `targetChainRefundPerGasUnused`
     */
    function resendToEvm(
        VaaKey memory deliveryVaaKey,
        uint16 targetChain,
        TargetNative newReceiverValue,
        Gas newGasLimit,
        address newDeliveryProviderAddress
    ) external payable returns (uint64 sequence);

    /**
     * @notice Requests a previously published delivery instruction to be redelivered 
     * 
     *
     * This function must be called with `msg.value` equal to 
     * quoteDeliveryPrice(targetChain, newReceiverValue, newEncodedExecutionParameters, newDeliveryProviderAddress)
     * 
     * @param deliveryVaaKey VaaKey identifying the wormhole message containing the 
     *        previously published delivery instructions
     * @param targetChain The target chain that the original delivery targeted. Must match targetChain from original delivery instructions
     * @param newReceiverValue new msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param newEncodedExecutionParameters new encoded information on how to execute delivery that may impact pricing
     *        e.g. for version EVM_V1, this is a struct that encodes the `gasLimit` with which to call `targetAddress`
     * @param newDeliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @return sequence sequence number of published VAA containing redelivery instructions
     * 
     *  @notice *** This will only be able to succeed if the following is true **
     *         - (For EVM_V1) newGasLimit >= gas limit of the old instruction
     *         - newReceiverValue >= receiver value of the old instruction
     *         - (For EVM_V1) newDeliveryProvider's `targetChainRefundPerGasUnused` >= old relay provider's `targetChainRefundPerGasUnused`
     */
    function resend(
        VaaKey memory deliveryVaaKey,
        uint16 targetChain,
        TargetNative newReceiverValue,
        bytes memory newEncodedExecutionParameters,
        address newDeliveryProviderAddress
    ) external payable returns (uint64 sequence);

    /**
     * @notice Returns the price to request a relay to chain `targetChain`, using the default delivery provider
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param gasLimit gas limit with which to call `targetAddress`. 
     * @return nativePriceQuote Price, in units of current chain currency, that the delivery provider charges to perform the relay
     * @return targetChainRefundPerGasUnused amount of target chain currency that will be refunded per unit of gas unused, 
     *         if a refundAddress is specified
     */
    function quoteEVMDeliveryPrice(
        uint16 targetChain,
        TargetNative receiverValue,
        Gas gasLimit
    ) external view returns (LocalNative nativePriceQuote, GasPrice targetChainRefundPerGasUnused);

    /**
     * @notice Returns the price to request a relay to chain `targetChain`, using delivery provider `deliveryProviderAddress`
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param gasLimit gas limit with which to call `targetAddress`. 
     * @param deliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @return nativePriceQuote Price, in units of current chain currency, that the delivery provider charges to perform the relay
     * @return targetChainRefundPerGasUnused amount of target chain currency that will be refunded per unit of gas unused, 
     *         if a refundAddress is specified
     */
    function quoteEVMDeliveryPrice(
        uint16 targetChain,
        TargetNative receiverValue,
        Gas gasLimit,
        address deliveryProviderAddress
    ) external view returns (LocalNative nativePriceQuote, GasPrice targetChainRefundPerGasUnused);

    /**
     * @notice Returns the price to request a relay to chain `targetChain`, using delivery provider `deliveryProviderAddress`
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param receiverValue msg.value that delivery provider should pass in for call to `targetAddress` (in targetChain currency units)
     * @param encodedExecutionParameters encoded information on how to execute delivery that may impact pricing
     *        e.g. for version EVM_V1, this is a struct that encodes the `gasLimit` with which to call `targetAddress`
     * @param deliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @return nativePriceQuote Price, in units of current chain currency, that the delivery provider charges to perform the relay
     * @return encodedExecutionInfo encoded information on how the delivery will be executed
     *        e.g. for version EVM_V1, this is a struct that encodes the `gasLimit` and `targetChainRefundPerGasUnused`
     *             (which is the amount of target chain currency that will be refunded per unit of gas unused, 
     *              if a refundAddress is specified)
     */
    function quoteDeliveryPrice(
        uint16 targetChain,
        TargetNative receiverValue,
        bytes memory encodedExecutionParameters,
        address deliveryProviderAddress
    ) external view returns (LocalNative nativePriceQuote, bytes memory encodedExecutionInfo);

    /**
     * @notice Returns the (extra) amount of target chain currency that `targetAddress`
     * will be called with, if the `paymentForExtraReceiverValue` field is set to `currentChainAmount`
     * 
     * @param targetChain in Wormhole Chain ID format
     * @param currentChainAmount The value that `paymentForExtraReceiverValue` will be set to
     * @param deliveryProviderAddress The address of the desired delivery provider's implementation of IDeliveryProvider
     * @return targetChainAmount The amount such that if `targetAddress` will be called with `msg.value` equal to
     *         receiverValue + targetChainAmount
     */
    function quoteNativeForChain(
        uint16 targetChain,
        LocalNative currentChainAmount,
        address deliveryProviderAddress
    ) external view returns (TargetNative targetChainAmount);

    /**
     * @notice Returns the address of the current default delivery provider
     * @return deliveryProvider The address of (the default delivery provider)'s contract on this source
     *   chain. This must be a contract that implements IDeliveryProvider.
     */
    function getDefaultDeliveryProvider() external view returns (address deliveryProvider);
}

/**
 * @title IWormholeRelayerDelivery
 * @notice The interface to execute deliveries. Only relevant for Delivery Providers 
 */
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
     * @custom:member sourceChain - The chain which this delivery was requested from (in wormhole
     *     ChainID format)
     * @custom:member sequence - The wormhole sequence number of the delivery VAA on the source chain
     *     corresponding to this delivery request
     * @custom:member deliveryVaaHash - The hash of the delivery VAA corresponding to this delivery
     *     request
     * @custom:member gasUsed - The amount of gas that was used to call your target contract 
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
     *   - If status is FORWARD_REQUEST_FAILURE, this is also the revert data - the reason the forward failed.
     *     This will be either an encoded Cancelled, DeliveryProviderReverted, or DeliveryProviderPaymentFailed error
     * @custom:member refundStatus - Result of the refund. REFUND_SUCCESS or REFUND_FAIL are for
     *     refunds where targetChain=refundChain; the others are for targetChain!=refundChain,
     *     where a cross chain refund is necessary
     * @custom:member overridesInfo:
     *   - If not an override: empty bytes array
     *   - Otherwise: An encoded `DeliveryOverride`
     */
    event Delivery(
        address indexed recipientContract,
        uint16 indexed sourceChain,
        uint64 indexed sequence,
        bytes32 deliveryVaaHash,
        DeliveryStatus status,
        Gas gasUsed,
        RefundStatus refundStatus,
        bytes additionalStatusInfo,
        bytes overridesInfo
    );

    /**
     * @notice The delivery provider calls `deliver` to relay messages as described by one delivery instruction
     * 
     * The delivery provider must pass in the specified (by VaaKeys[]) signed wormhole messages (VAAs) from the source chain
     * as well as the signed wormhole message with the delivery instructions (the delivery VAA)
     *
     * The messages will be relayed to the target address (with the specified gas limit and receiver value) iff the following checks are met:
     * - the delivery VAA has a valid signature
     * - the delivery VAA's emitter is one of these WormholeRelayer contracts
     * - the delivery provider passed in at least enough of this chain's currency as msg.value (enough meaning the maximum possible refund)     
     * - the instruction's target chain is this chain
     * - the relayed signed VAAs match the descriptions in container.messages (the VAA hashes match, or the emitter address, sequence number pair matches, depending on the description given)
     *
     * @param encodedVMs - An array of signed wormhole messages (all from the same source chain
     *     transaction)
     * @param encodedDeliveryVAA - Signed wormhole message from the source chain's WormholeRelayer
     *     contract with payload being the encoded delivery instruction container
     * @param relayerRefundAddress - The address to which any refunds to the delivery provider
     *     should be sent
     * @param deliveryOverrides - Optional overrides field which must be either an empty bytes array or
     *     an encoded DeliveryOverride struct
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

// Bound chosen by the following formula: `memoryWord * 4 + selectorSize`.
// This means that an error identifier plus four fixed size arguments should be available to developers.
// In the case of a `require` revert with error message, this should provide 2 memory word's worth of data.
uint256 constant RETURNDATA_TRUNCATION_THRESHOLD = 132;

//When msg.value was not equal to `delivery provider's quoted delivery price` + `paymentForExtraReceiverValue`
error InvalidMsgValue(LocalNative msgValue, LocalNative totalFee);

error RequestedGasLimitTooLow();

error DeliveryProviderDoesNotSupportTargetChain(address relayer, uint16 chainId);
error DeliveryProviderCannotReceivePayment();

//When calling `forward()` on the WormholeRelayer if no delivery is in progress
error NoDeliveryInProgress();
//When calling `delivery()` a second time even though a delivery is already in progress
error ReentrantDelivery(address msgSender, address lockedBy);
//When any other contract but the delivery target calls `forward()` on the WormholeRelayer while a
//  delivery is in progress
error ForwardRequestFromWrongAddress(address msgSender, address deliveryTarget);

error InvalidPayloadId(uint8 parsed, uint8 expected);
error InvalidPayloadLength(uint256 received, uint256 expected);
error InvalidVaaKeyType(uint8 parsed);

error InvalidDeliveryVaa(string reason);
//When the delivery VAA (signed wormhole message with delivery instructions) was not emitted by the
//  registered WormholeRelayer contract
error InvalidEmitter(bytes32 emitter, bytes32 registered, uint16 chainId);
error VaaKeysLengthDoesNotMatchVaasLength(uint256 keys, uint256 vaas);
error VaaKeysDoNotMatchVaas(uint8 index);
//When someone tries to call an external function of the WormholeRelayer that is only intended to be
//  called by the WormholeRelayer itself (to allow retroactive reverts for atomicity)
error RequesterNotWormholeRelayer();

//When trying to relay a `DeliveryInstruction` to any other chain but the one it was specified for
error TargetChainIsNotThisChain(uint16 targetChain);
error ForwardNotSufficientlyFunded(LocalNative amountOfFunds, LocalNative amountOfFundsNeeded);
//When a `DeliveryOverride` contains a gas limit that's less than the original
error InvalidOverrideGasLimit();
//When a `DeliveryOverride` contains a receiver value that's less than the original
error InvalidOverrideReceiverValue();
//When a `DeliveryOverride` contains a 'refund per unit of gas unused' that's less than the original
error InvalidOverrideRefundPerGasUnused();

//When the delivery provider doesn't pass in sufficient funds (i.e. msg.value does not cover the
// maximum possible refund to the user)
error InsufficientRelayerFunds(LocalNative msgValue, LocalNative minimum);

//When a bytes32 field can't be converted into a 20 byte EVM address, because the 12 padding bytes
//  are non-zero (duplicated from Utils.sol)
error NotAnEvmAddress(bytes32);
