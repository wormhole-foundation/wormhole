// contracts/Setters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/utils/Context.sol";

import "./RelayProviderState.sol";
import "../../interfaces/relayer/IRelayProvider.sol";

contract RelayProviderSetters is Context, RelayProviderState {
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

    function setCoreRelayer(address payable coreRelayer) internal {
        _state.coreRelayer = coreRelayer;
    }

    function setChainSupported(uint16 targetChainId, bool isSupported) internal {
        _state.supportedChains[targetChainId] = isSupported;
    }

    function setDeliverGasOverhead(uint16 chainId, Gas deliverGasOverhead) internal {
        require(Gas.unwrap(deliverGasOverhead) <= type(uint32).max, "deliverGasOverhead too large");
        _state.deliverGasOverhead[chainId] = deliverGasOverhead;
    }

    function setRewardAddress(address payable rewardAddress) internal {
        _state.rewardAddress = rewardAddress;
    }

    function setTargetChainAddress(uint16 targetChainId, bytes32 newAddress) internal {
        _state.targetChainAddresses[targetChainId] = newAddress;
    }

    function setMaximumBudget(uint16 targetChainId, Wei amount) internal {
        require(amount.unwrap() <= type(uint192).max, "amount too large");
        _state.maximumBudget[targetChainId] = amount;
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
        uint16 targetChainId,
        uint16 tolerance,
        uint16 toleranceDenominator
    ) internal {
        RelayProviderStorage.AssetConversion storage assetConversion =
            _state.assetConversion[targetChainId];
        assetConversion.buffer = tolerance;
        assetConversion.denominator = toleranceDenominator;
    }
}
