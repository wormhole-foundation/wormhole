// contracts/Setters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/utils/Context.sol";

import "./DeliveryProviderState.sol";
import "../../interfaces/relayer/IDeliveryProviderTyped.sol";

contract DeliveryProviderSetters is Context, DeliveryProviderState {
    using GasPriceLib for GasPrice;
    using WeiLib for Wei;

    function setOwner(address owner_) internal {
        _state.owner = owner_;
    }

    function setPendingOwner(address newOwner) internal {
        _state.pendingOwner = newOwner;
    }

    function setInitialized(address implementation) internal {
        _state.initializedImplementations[implementation] = true;
    }

    function setChainId(uint16 thisChain) internal {
        _state.chainId = thisChain;
    }

    function setPricingWallet(address newPricingWallet) internal {
        _state.pricingWallet = newPricingWallet;
    }

    function setWormholeRelayer(address payable coreRelayer) internal {
        _state.coreRelayer = coreRelayer;
    }

    function setChainSupported(uint16 targetChain, bool isSupported) internal {
        _state.supportedChains[targetChain] = isSupported;
    }

    function setDeliverGasOverhead(uint16 chainId, Gas deliverGasOverhead) internal {
        require(Gas.unwrap(deliverGasOverhead) <= type(uint32).max, "deliverGasOverhead too large");
        _state.deliverGasOverhead[chainId] = deliverGasOverhead;
    }

    function setRewardAddress(address payable rewardAddress) internal {
        _state.rewardAddress = rewardAddress;
    }

    function setTargetChainAddress(uint16 targetChain, bytes32 newAddress) internal {
        _state.targetChainAddresses[targetChain] = newAddress;
    }

    function setMaximumBudget(uint16 targetChain, Wei amount) internal {
        require(amount.unwrap() <= type(uint192).max, "amount too large");
        _state.maximumBudget[targetChain] = amount.asTargetNative();
    }

    function setPriceInfo(
        uint16 updateChainId,
        GasPrice updateGasPrice,
        WeiPrice updateNativeCurrencyPrice
    ) internal {
        require(updateGasPrice.unwrap() <= type(uint64).max, "gas price must be < 2^64");
        _state.data[updateChainId].gasPrice = updateGasPrice;
        _state.data[updateChainId].nativeCurrencyPrice = updateNativeCurrencyPrice;
    }

    function setAssetConversionBuffer(
        uint16 targetChain,
        uint16 tolerance,
        uint16 toleranceDenominator
    ) internal {
        DeliveryProviderStorage.AssetConversion storage assetConversion =
            _state.assetConversion[targetChain];
        assetConversion.buffer = tolerance;
        assetConversion.denominator = toleranceDenominator;
    }
}
