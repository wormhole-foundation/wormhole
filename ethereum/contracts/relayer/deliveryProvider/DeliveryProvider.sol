// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "./DeliveryProviderGovernance.sol";
import "./DeliveryProviderStructs.sol";
import "../../interfaces/relayer/IDeliveryProviderTyped.sol";
import "../../interfaces/relayer/TypedUnits.sol";
import "../../libraries/relayer/ExecutionParameters.sol";
import {IWormhole} from "../../interfaces/IWormhole.sol";

contract DeliveryProvider is DeliveryProviderGovernance, IDeliveryProvider {
    using WeiLib for Wei;
    using GasLib for Gas;
    using GasPriceLib for GasPrice;
    using WeiPriceLib for WeiPrice;
    using TargetNativeLib for TargetNative;
    using LocalNativeLib for LocalNative;

    error CallerNotApproved(address msgSender);
    error PriceIsZero(uint16 chain);
    error Overflow(uint256 value, uint256 max);
    error MaxRefundGreaterThanGasLimitCost(uint256 maxRefund, uint256 gasLimitCost);
    error MaxRefundGreaterThanGasLimitCostOnSourceChain(uint256 maxRefund, uint256 gasLimitCost);
    error ExceedsMaximumBudget(uint16 targetChain, uint256 exceedingValue, uint256 maximumBudget);

    function quoteEvmDeliveryPrice(
        uint16 targetChain,
        Gas gasLimit,
        TargetNative receiverValue
    )
        public
        view
        returns (LocalNative nativePriceQuote, GasPrice targetChainRefundPerUnitGasUnused)
    {
        (uint16 buffer, uint16 denominator) = assetConversionBuffer(targetChain);
        targetChainRefundPerUnitGasUnused = GasPrice.wrap(gasPrice(targetChain).unwrap() * (denominator) / (uint256(denominator) + buffer));
        
        TargetNative maxRefund = gasLimit.toWei(targetChainRefundPerUnitGasUnused).asTargetNative();
        TargetNative gasLimitCost = gasLimit.toWei(gasPrice(targetChain)).asTargetNative();
        LocalNative maxRefundInSourceCurrency = quoteAssetCost(targetChain, maxRefund);
        LocalNative gasLimitCostInSourceCurrency = quoteGasCost(targetChain, gasLimit);
        LocalNative receiverValueCost = quoteAssetCost(targetChain, receiverValue);
        nativePriceQuote = quoteDeliveryOverhead(targetChain) + gasLimitCostInSourceCurrency + receiverValueCost;
        if(maxRefundInSourceCurrency.unwrap() > gasLimitCostInSourceCurrency.unwrap()) {
            revert MaxRefundGreaterThanGasLimitCostOnSourceChain(maxRefundInSourceCurrency.unwrap(), gasLimitCostInSourceCurrency.unwrap());
        } 
        if(maxRefund.unwrap() > gasLimitCost.unwrap()) {
            revert MaxRefundGreaterThanGasLimitCost(maxRefund.unwrap(), gasLimitCost.unwrap());
        }
        if(receiverValue.asNative() + gasLimitCost.asNative() > maximumBudget(targetChain).asNative()) {
            revert ExceedsMaximumBudget(targetChain, receiverValue.unwrap() + gasLimitCost.unwrap(), maximumBudget(targetChain).unwrap());
        }
    }

    function quoteDeliveryPrice(
        uint16 targetChain,
        TargetNative receiverValue,
        bytes memory encodedExecutionParams
    ) external view returns (LocalNative nativePriceQuote, bytes memory encodedExecutionInfo) {
        ExecutionParamsVersion version = decodeExecutionParamsVersion(encodedExecutionParams);
        if (version == ExecutionParamsVersion.EVM_V1) {
            EvmExecutionParamsV1 memory parsed = decodeEvmExecutionParamsV1(encodedExecutionParams);
            GasPrice targetChainRefundPerUnitGasUnused;
            (nativePriceQuote, targetChainRefundPerUnitGasUnused) =
                quoteEvmDeliveryPrice(targetChain, parsed.gasLimit, receiverValue);
            return (
                nativePriceQuote,
                encodeEvmExecutionInfoV1(
                    EvmExecutionInfoV1(parsed.gasLimit, targetChainRefundPerUnitGasUnused)
                    )
            );
        } else {
            revert UnsupportedExecutionParamsVersion(uint8(version));
        }
    }

    function quoteAssetConversion(
        uint16 targetChain,
        LocalNative currentChainAmount
    ) public view returns (TargetNative targetChainAmount) {
        return quoteAssetConversion(chainId(), targetChain, currentChainAmount);
    }

    function quoteAssetConversion(
        uint16 sourceChain,
        uint16 targetChain,
        LocalNative sourceChainAmount
    ) internal view returns (TargetNative targetChainAmount) {
        (uint16 buffer, uint16 bufferDenominator) = assetConversionBuffer(targetChain);
        return sourceChainAmount.asNative().convertAsset(
            nativeCurrencyPrice(sourceChain),
            nativeCurrencyPrice(targetChain),
            (bufferDenominator),
            (uint32(buffer) + bufferDenominator),
            false  // round down
        ).asTargetNative();
    }

    //Returns the address on this chain that rewards should be sent to
    function getRewardAddress() public view override returns (address payable) {
        return rewardAddress();
    }

    function isChainSupported(uint16 targetChain) public view override returns (bool supported) {
        return _state.supportedChains[targetChain];
    }

    function getTargetChainAddress(uint16 targetChain)
        public
        view
        override
        returns (bytes32 deliveryProviderAddress)
    {
        return targetChainAddress(targetChain);
    }

    /**
     *
     * HELPER METHODS
     *
     */

    //Returns the delivery overhead fee required to deliver a message to the target chain, denominated in this chain's wei.
    function quoteDeliveryOverhead(uint16 targetChain) public view returns (LocalNative nativePriceQuote) {
        nativePriceQuote = quoteGasCost(targetChain, deliverGasOverhead(targetChain));
        if(nativePriceQuote.unwrap() > type(uint128).max) {
            revert Overflow(nativePriceQuote.unwrap(), type(uint128).max);
        }
    }

    //Returns the price of purchasing gasAmount units of gas on the target chain, denominated in this chain's wei.
    function quoteGasCost(uint16 targetChain, Gas gasAmount) public view returns (LocalNative totalCost) {
        Wei gasCostInSourceChainCurrency =
            assetConversion(targetChain, gasAmount.toWei(gasPrice(targetChain)), chainId());
        totalCost = LocalNative.wrap(gasCostInSourceChainCurrency.unwrap());
    }

    function quoteGasPrice(uint16 targetChain) public view returns (GasPrice price) {
        price = GasPrice.wrap(quoteGasCost(targetChain, Gas.wrap(1)).unwrap());
        if(price.unwrap() > type(uint88).max) {
            revert Overflow(price.unwrap(), type(uint88).max);
        }
    }

    // relevant for chains that have dynamic execution pricing (e.g. Ethereum)
    function assetConversion(
        uint16 sourceChain,
        Wei sourceAmount,
        uint16 targetChain
    ) internal view returns (Wei targetAmount) {
        if(nativeCurrencyPrice(sourceChain).unwrap() == 0) {
            revert PriceIsZero(sourceChain);
        } 
        if(nativeCurrencyPrice(targetChain).unwrap() == 0) {
            revert PriceIsZero(targetChain);
        }
        return sourceAmount.convertAsset(
            nativeCurrencyPrice(sourceChain),
            nativeCurrencyPrice(targetChain),
            1,
            1,
            // round up
            true
        );
    }

    function quoteAssetCost(
        uint16 targetChain,
        TargetNative targetChainAmount
    ) internal view returns (LocalNative currentChainAmount) {
        (uint16 buffer, uint16 bufferDenominator) = assetConversionBuffer(targetChain);
        if(nativeCurrencyPrice(chainId()).unwrap() == 0) {
            revert PriceIsZero(chainId());
        } 
        if(nativeCurrencyPrice(targetChain).unwrap() == 0) {
            revert PriceIsZero(targetChain);
        }
        return targetChainAmount.asNative().convertAsset(
            nativeCurrencyPrice(targetChain),
            nativeCurrencyPrice(chainId()),
            (uint32(buffer) + bufferDenominator),
            (bufferDenominator),
            // round up
            true
        ).asLocalNative();
    }
}
