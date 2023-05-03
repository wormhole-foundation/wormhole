// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./CoreRelayerSend.sol";

abstract contract CoreRelayerSendOverloads is CoreRelayerSend {
    /**
     *  @notice the 'send' function emits a wormhole message (VAA) that instructs the default wormhole relay provider to
     *  call the 'IWormholeReceiver.receiveWormholeMessage' method of the contract at targetAddress on targetChain.
     *  No additional signed vaas will be passed to `receiveWormholeMessage` when using this overload.
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
     *  @notice the 'send' function emits a wormhole message (VAA) that instructs the default wormhole relay provider to
     *  call the 'IWormholeReceiver.receiveWormholeMessage' method of the contract at targetAddress on targetChain.
     *  This method accepts a list of additional wormhole message keys (vaaKeys param) that should be submitted along with the payload
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
     *  The relay provider will call receiveWormholeMessages with an array of signed VAAs specified by this vaaKeys array with order preserved.
     *  @param consistencyLevel  The level of finality to reach before emitting the Wormhole VAA corresponding to this 'send' request. See https://book.wormhole.com/wormhole/3_coreLayerContracts.html#consistency-levels
     *
     *  This function must be called with a payment of at least maxTransactionFee + receiverValue + one wormhole message fee.
     *
     *  @return sequence The sequence number for the emitted wormhole message
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
     * @notice This 'forward' function can only be called from within a delivery, i.e. inside something that implements `IWormholeReceiver.receiveWormholeMessages'
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
}
