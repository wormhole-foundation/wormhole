// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./TypedUnits.sol";

interface IDeliveryProvider {
    function quoteDeliveryPrice(
        uint16 targetChain,
        TargetNative receiverValue,
        bytes memory encodedExecutionParams
    ) external view returns (LocalNative nativePriceQuote, bytes memory encodedExecutionInfo);

    function quoteAssetConversion(
        uint16 targetChain,
        LocalNative currentChainAmount
    ) external view returns (TargetNative targetChainAmount);

    /**
     * @notice This function should return a payable address on this (source) chain where all awards
     *     should be sent for the relay provider.
     *
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
     * @notice This function determines whether a relay provider supports the provided keyType.
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
