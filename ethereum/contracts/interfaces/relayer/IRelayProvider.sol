// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface IRelayProvider {

    /**
     * @notice This function should provide a fixed overhead fee that will be applied to any delivery on targetChain.
     * NOTE: The fee should be quoted in the native wei of this chain.
     *
     * @param targetChain - the chain that should be quoted for.
     */
    function quoteDeliveryOverhead(uint16 targetChain) external view returns (uint256 deliveryOverhead);

    /**
     * @notice This function should provide a fixed fee for 1 unit of gas on targetChain.
     * 
     * NOTE: The fee should be quoted in the native wei of this chain.
     *
     * @param targetChain - the chain that should be quoted for.
     */
    function quoteGasPrice(uint16 targetChain) external view returns (uint256 gasPriceSource);

    /**
     * @notice This function should provide a quote in USD of the native asset price for all supported chains.
     * 
     * NOTE: The fee should be quoted in 10^-6 dollars. I.E, a quote of 1 dollar should be quoted as 1000000.
     *
     * @param chainId - the chain that should be quoted for.
     */
    function quoteAssetPrice(uint16 chainId) external view returns (uint256 usdPrice);

    /**
     * @notice When calculating the receiverValue of a delivery or performing a refund, a portion of the value is 
     * awarded to the RelayProvider. This function defines the portion, which can differ based on the target chain.
     * 
     * toleranceDenominator denotes how many 'parts' the fee should be broken into, whereas tolerance defines how many
     * parts should be awarded to the relayer.
     * I.E, if toleranceDenominator is 100 and tolerance is 2, 2% of the value will be awarded to the relayer.
     *
     * @param targetChain - The chain which is being delivered to.
     */
    function getAssetConversionBuffer(uint16 targetChain)
        external
        view
        returns (uint16 tolerance, uint16 toleranceDenominator);

    /**
     * @notice This function should return the maximumBudget (receiverValue + maxTransactionFee) that the relay provider is
     * willing to support in a single delivery. 
     *
     * Note: Unlike the other quote functions, this function should return a quote in the wei (or other base currency) of the
     * targetChain, not sourceChain.
     *
     * @param targetChain - The chain which is being delivered to.
     */
    function quoteMaximumBudget(uint16 targetChain) external view returns (uint256 maximumTargetBudget);

    /**
     * @notice This function should return a payable address on this (source) chain where all awards should be sent
     * for the relay provider.
     *
     */
    function getRewardAddress() external view returns (address payable rewardAddress);

    /**
     * @notice This function specifies the consistencyLevel that will be put on all delivery VAAs for this provider.
     * This function should return 200 if the RelayProvider supports delivery of instant VAAs.
     *
     */
    function getConsistencyLevel() external view returns (uint8 consistencyLevel);

    /**
     * @notice This function determines whether a relay provider supports deliveries to a given chain or not.
     *
     * @param targetChain - The chain which is being delivered to.
     */
    function isChainSupported(uint16 targetChain) external view returns (bool supported);

    /**
     * @notice If a RelayProvider supports a given chain, this function should provide the contract address (in wormhole format)
     * of the relay provider on that chain.
     *
     * @param targetChain - The chain which is being delivered to.
     */
    function getTargetChainAddress(uint16 targetChain) external view returns (bytes32 relayProviderAddress);
}
