// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface IRelayProvider {
    function quoteDeliveryPrice(
        uint16 targetChainId,
        uint256 receiverValue,
        bytes memory encodedExecutionParams
    ) external view returns (uint256 nativePriceQuote, bytes memory encodedExecutionInfo);

    function quoteAssetConversion(
        uint16 targetChainId,
        uint256 currentChainAmount
    ) external view returns (uint256 targetChainAmount);

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
     * @param targetChainId - The chain which is being delivered to.
     */
    function isChainSupported(uint16 targetChainId) external view returns (bool supported);

    /**
     * @notice If a RelayProvider supports a given chain, this function should provide the contract
     *      address (in wormhole format) of the relay provider on that chain.
     *
     * @param targetChainId - The chain which is being delivered to.
     */
    function getTargetChainAddress(uint16 targetChainId)
        external
        view
        returns (bytes32 relayProviderAddress);
}
