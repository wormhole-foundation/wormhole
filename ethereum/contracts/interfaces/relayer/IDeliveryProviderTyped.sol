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
