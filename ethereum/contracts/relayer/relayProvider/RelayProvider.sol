// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "./RelayProviderGovernance.sol";
import "./RelayProviderStructs.sol";
import "../../interfaces/relayer/IRelayProvider.sol";
import "../../interfaces/relayer/TypedUnits.sol";

contract RelayProvider is RelayProviderGovernance, IRelayProvider {
    using WeiLib for Wei;
    using GasLib for Gas;
    using GasPriceLib for GasPrice;
    using WeiPriceLib for WeiPrice;

    error CallerNotApproved(address msgSender);

    //Returns the delivery overhead fee required to deliver a message to the target chain, denominated in this chain's wei.
    function quoteDeliveryOverhead(uint16 targetChainId)
        public
        view
        override
        returns (Wei nativePriceQuote)
    {
        Gas overhead = deliverGasOverhead(targetChainId);
        Wei targetFees = overhead.toWei(gasPrice(targetChainId));
        Wei result = quoteAssetConversion(targetChainId, targetFees, chainId());
        require(result.unwrap() <= type(uint128).max, "Overflow");
        return result;
    }

    //Returns the price of purchasing 1 unit of gas on the target chain, denominated in this chain's wei.
    function quoteGasPrice(uint16 targetChainId) public view override returns (GasPrice) {
        Wei gasPriceInSourceChainCurrency =
            quoteAssetConversion(targetChainId, gasPrice(targetChainId).toWei(), chainId());
        require(gasPriceInSourceChainCurrency.unwrap() <= type(uint88).max, "Overflow");
        return GasPrice.wrap(uint88(gasPriceInSourceChainCurrency.unwrap()));
    }

    //Returns the price of chainId's native currency in USD 10^-6 units
    function quoteAssetPrice(uint16 chainId) public view override returns (WeiPrice) {
        return nativeCurrencyPrice(chainId);
    }

    //Returns the maximum budget that is allowed for a delivery on target chain, denominated in the target chain's wei.
    function quoteMaximumBudget(uint16 targetChainId)
        public
        view
        override
        returns (Wei maximumTargetBudget)
    {
        return maximumBudget(targetChainId);
    }

    //Returns the address on this chain that rewards should be sent to
    function getRewardAddress() public view override returns (address payable) {
        return rewardAddress();
    }

    function isChainSupported(uint16 targetChainId) public view override returns (bool supported) {
        return _state.supportedChains[targetChainId];
    }

    function getTargetChainAddress(uint16 targetChainId)
        public
        view
        override
        returns (bytes32 relayProviderAddress)
    {
        return targetChainAddress(targetChainId);
    }

    //Returns a buffer amount, and a buffer denominator, whereby the bufferAmount / bufferDenominator will be reduced from
    //receiverValue conversions, giving an overhead to the provider on each conversion
    function getAssetConversionBuffer(uint16 targetChainId)
        public
        view
        override
        returns (uint16 tolerance, uint16 toleranceDenominator)
    {
        return assetConversionBuffer(targetChainId);
    }

    /**
     *
     * HELPER METHODS
     *
     */

    // relevant for chains that have dynamic execution pricing (e.g. Ethereum)
    function quoteAssetConversion(
        uint16 sourceChainId,
        Wei sourceAmount,
        uint16 targetChainId
    ) internal view returns (Wei targetAmount) {
        return sourceAmount.convertAsset(
            quoteAssetPrice(sourceChainId),
            quoteAssetPrice(targetChainId),
            1,
            1,
            // round up
            true
        );
    }
}
