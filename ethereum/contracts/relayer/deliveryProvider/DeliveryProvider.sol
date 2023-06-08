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

    function quoteEvmDeliveryPrice(
        uint16 targetChain,
        Gas gasLimit,
        TargetNative receiverValue
    )
        public
        view
        returns (LocalNative nativePriceQuote, GasPrice targetChainRefundPerUnitGasUnused)
    {
        targetChainRefundPerUnitGasUnused = gasPrice(targetChain);
        Wei costOfProvidingFullGasLimit = gasLimit.toWei(targetChainRefundPerUnitGasUnused);
        Wei transactionFee =
            quoteDeliveryOverhead(targetChain) + gasLimit.toWei(quoteGasPrice(targetChain));
        Wei receiverValueCost = quoteAssetCost(targetChain, receiverValue);
        nativePriceQuote = (
            transactionFee.max(costOfProvidingFullGasLimit) + receiverValueCost
        ).asLocalNative();
        require(
            receiverValue.asNative() + costOfProvidingFullGasLimit <= maximumBudget(targetChain),
            "Exceeds maximum budget"
        );
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
            (buffer),
            (uint32(buffer) + bufferDenominator),
            false
        )
            // round down
            .asTargetNative();
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
    function quoteDeliveryOverhead(uint16 targetChain) public view returns (Wei nativePriceQuote) {
        Gas overhead = deliverGasOverhead(targetChain);
        Wei targetFees = overhead.toWei(gasPrice(targetChain));
        Wei result = assetConversion(targetChain, targetFees, chainId());
        require(result.unwrap() <= type(uint128).max, "Overflow");
        return result;
    }

    //Returns the price of purchasing 1 unit of gas on the target chain, denominated in this chain's wei.
    function quoteGasPrice(uint16 targetChain) public view returns (GasPrice) {
        Wei gasPriceInSourceChainCurrency =
            assetConversion(targetChain, gasPrice(targetChain).priceAsWei(), chainId());
        require(gasPriceInSourceChainCurrency.unwrap() <= type(uint88).max, "Overflow");
        return GasPrice.wrap(uint88(gasPriceInSourceChainCurrency.unwrap()));
    }

    // relevant for chains that have dynamic execution pricing (e.g. Ethereum)
    function assetConversion(
        uint16 sourceChain,
        Wei sourceAmount,
        uint16 targetChain
    ) internal view returns (Wei targetAmount) {
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
    ) internal view returns (Wei currentChainAmount) {
        (uint16 buffer, uint16 bufferDenominator) = assetConversionBuffer(targetChain);
        return targetChainAmount.asNative().convertAsset(
            nativeCurrencyPrice(chainId()),
            nativeCurrencyPrice(targetChain),
            (uint32(buffer) + bufferDenominator),
            (buffer),
            // round up
            true
        );
    }
}
