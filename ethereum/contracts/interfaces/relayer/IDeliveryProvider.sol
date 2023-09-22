// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface IDeliveryProvider {
    
    /**
     * @notice This function returns 
     *
     * 1) nativePriceQuote: the price of a delivery (by this delivery provider) to chain
     * 'targetChain', giving the user's contract 'receiverValue' target chain wei and performing the 
     * relay with the execution parameters (e.g. the gas limit) specified in 'encodedExecutionParameters'
     * 
     * 2) encodedExecutionInfo: information relating to how this delivery provider
     * will perform such a delivery (e.g. the gas limit, and the amount it will refund per gas unused)
     *
     * encodedExecutionParameters and encodedExecutionInfo both are encodings of versioned structs - 
     * version EVM_V1 of ExecutionParameters specifies the gas limit,
     * and version EVM_V1 of ExecutionInfo specifies the gas limit and the amount that this delivery provider 
     * will refund per unit of gas unused
     */
    function quoteDeliveryPrice(
        uint16 targetChain,
        uint256 receiverValue,
        bytes memory encodedExecutionParams
    ) external view returns (uint256 nativePriceQuote, bytes memory encodedExecutionInfo);

    /**
     * @notice This function returns the amount of extra 'receiverValue' (msg.value on the target chain) 
     * that will be sent to your contract, if you specify 'currentChainAmount' in the 
     * 'paymentForExtraReceiverValue' field on 'send'
     */
    function quoteAssetConversion(
        uint16 targetChain,
        uint256 currentChainAmount
    ) external view returns (uint256 targetChainAmount);

    /**
     * @notice This function should return a payable address on this (source) chain where all awards
     *     should be sent for the relay provider.
     */
    function getRewardAddress() external view returns (address payable rewardAddress);

    /**
     * @notice This function determines whether a relay provider supports deliveries to a given chain
     *     or not.
     *
     * @param targetChain - The chain which is being delivered to.
     */
    function isChainSupported(uint16 targetChain) external view returns (bool supported);

    /**
     * @notice This function determines whether a relay provider supports the given keyType.
     *      
     * Note: 0-127 are reserved for standardized keyTypes and 128-255 are allowed to be custom per DeliveryProvider
     *       Practically this means that 0-127 must mean the same thing for all DeliveryProviders,
     *       while x within 128-255 may have different meanings between DeliveryProviders 
     *       (e.g. 130 for provider A means pyth price quotes while 130 for provider B means tweets, 
     *       but 8 must mean the same for both)
     *
     * @param keyType - The keyType within MessageKey that specifies what the encodedKey within a MessageKey means
     */
    function isMessageKeyTypeSupported(uint8 keyType) external view returns (bool supported);

    /**
     * @notice This function returns a bitmap encoding all the keyTypes this provider supports
     *      
     * Note: 0-127 are reserved for standardized keyTypes and 128-255 are allowed to be custom per DeliveryProvider
     *       Practically this means that 0-127 must mean the same thing for all DeliveryProviders,
     *       while x within 128-255 may have different meanings between DeliveryProviders 
     *       (e.g. 130 for provider A means pyth price quotes while 130 for provider B means tweets, 
     *       but 8 must mean the same for both)
     */
    function getSupportedKeys() external view returns (uint256 bitmap);

    /**
     * @notice If a DeliveryProvider supports a given chain, this function should provide the contract
     *      address (in wormhole format) of the relay provider on that chain.
     *
     * @param targetChain - The chain which is being delivered to.
     */
    function getTargetChainAddress(uint16 targetChain)
        external
        view
        returns (bytes32 deliveryProviderAddress);
}
