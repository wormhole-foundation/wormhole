// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "./RelayProviderGovernance.sol";
import "./RelayProviderStructs.sol";
import "../../interfaces/relayer/IRelayProvider.sol";
import "../../interfaces/relayer/TypedUnits.sol";
import "../../libraries/relayer/ExecutionParameters.sol";

contract RelayProvider is RelayProviderGovernance, IRelayProvider {
    using WeiLib for Wei;
    using GasLib for Gas;
    using GasPriceLib for GasPrice;
    using WeiPriceLib for WeiPrice;

    error CallerNotApproved(address msgSender);

    function quoteEvmDeliveryPrice(
        uint16 targetChainId,
        Gas gasLimit,
        Wei receiverValue
    )
        public
        view
        returns (Wei nativePriceQuote, GasPrice targetChainRefundPerUnitGasUnused)
    {
        targetChainRefundPerUnitGasUnused = gasPrice(targetChainId);
        Wei costOfProvidingFullGasLimit = gasLimit.toWei(targetChainRefundPerUnitGasUnused);
        Wei transactionFee =
            quoteDeliveryOverhead(targetChainId) + gasLimit.toWei(quoteGasPrice(targetChainId));
        Wei receiverValueCost = quoteAssetCost(targetChainId, receiverValue);
        nativePriceQuote =
            transactionFee.max(costOfProvidingFullGasLimit) + receiverValueCost;
        require(
            receiverValue + costOfProvidingFullGasLimit <= maximumBudget(targetChainId),
            "Exceeds maximum budget"
        );
        //require(nativePriceQuote.unwrap() <= type(uint128).max, "Overflow");
    }

    function quoteDeliveryPrice(
        uint16 targetChainId,
        Wei receiverValue,
        bytes memory encodedExecutionParams
    ) external view returns (Wei nativePriceQuote, bytes memory encodedExecutionInfo) {
        ExecutionParamsVersion version = decodeExecutionParamsVersion(encodedExecutionParams);
        if (version == ExecutionParamsVersion.EVM_V1) {
            EvmExecutionParamsV1 memory parsed = decodeEvmExecutionParamsV1(encodedExecutionParams);
            GasPrice targetChainRefundPerUnitGasUnused;
            (nativePriceQuote, targetChainRefundPerUnitGasUnused) = quoteEvmDeliveryPrice(targetChainId, parsed.gasLimit, receiverValue);
            return (
                nativePriceQuote,
                encodeEvmExecutionInfoV1(EvmExecutionInfoV1(parsed.gasLimit, targetChainRefundPerUnitGasUnused))
            );
        } else {
            revert UnsupportedExecutionParamsVersion(uint8(version));
        }
    }

    function quoteAssetConversion(
        uint16 targetChainId,
        Wei currentChainAmount
    ) public view returns (Wei targetChainAmount) {
        return quoteAssetConversion(chainId(), targetChainId, currentChainAmount);
    }

    function quoteAssetConversion(
        uint16 sourceChainId,
        uint16 targetChainId,
        Wei sourceChainAmount
    ) internal view returns (Wei targetChainAmount) {
        (uint16 buffer, uint16 bufferDenominator) = assetConversionBuffer(targetChainId);
        return sourceChainAmount.convertAsset(
            nativeCurrencyPrice(sourceChainId),
            nativeCurrencyPrice(targetChainId),
            (buffer),
            (uint32(buffer) + bufferDenominator),
            // round down
            false
        );
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

    /**
     *
     * HELPER METHODS
     *
     */

    //Returns the delivery overhead fee required to deliver a message to the target chain, denominated in this chain's wei.
    function quoteDeliveryOverhead(uint16 targetChainId)
        public
        view
        returns (Wei nativePriceQuote)
    {
        Gas overhead = deliverGasOverhead(targetChainId);
        Wei targetFees = overhead.toWei(gasPrice(targetChainId));
        Wei result = assetConversion(targetChainId, targetFees, chainId());
        require(result.unwrap() <= type(uint128).max, "Overflow");
        return result;
    }

    //Returns the price of purchasing 1 unit of gas on the target chain, denominated in this chain's wei.
    function quoteGasPrice(uint16 targetChainId) public view returns (GasPrice) {
        Wei gasPriceInSourceChainCurrency =
            assetConversion(targetChainId, gasPrice(targetChainId).priceAsWei(), chainId());
        require(gasPriceInSourceChainCurrency.unwrap() <= type(uint88).max, "Overflow");
        return GasPrice.wrap(uint88(gasPriceInSourceChainCurrency.unwrap()));
    }

    // relevant for chains that have dynamic execution pricing (e.g. Ethereum)
    function assetConversion(
        uint16 sourceChainId,
        Wei sourceAmount,
        uint16 targetChainId
    ) internal view returns (Wei targetAmount) {
        return sourceAmount.convertAsset(
            nativeCurrencyPrice(sourceChainId),
            nativeCurrencyPrice(targetChainId),
            1,
            1,
            // round up
            true
        );
    }

    function quoteAssetCost(
        uint16 targetChainId,
        Wei targetChainAmount
    ) internal view returns (Wei currentChainAmount) {
        (uint16 buffer, uint16 bufferDenominator) = assetConversionBuffer(targetChainId);
        return targetChainAmount.convertAsset(
            nativeCurrencyPrice(chainId()),
            nativeCurrencyPrice(targetChainId),
            (uint32(buffer) + bufferDenominator),
            (buffer),
            // round up
            true
        );
    }
}
