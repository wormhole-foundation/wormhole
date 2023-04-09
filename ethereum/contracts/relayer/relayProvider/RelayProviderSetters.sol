// contracts/Setters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/utils/Context.sol";

import "./RelayProviderState.sol";

contract RelayProviderSetters is Context, RelayProviderState {
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

    function setDeliverGasOverhead(uint16 chainId, uint32 deliverGasOverhead) internal {
        _state.deliverGasOverhead[chainId] = deliverGasOverhead;
    }

    function setRewardAddress(address payable rewardAddress) internal {
        _state.rewardAddress = rewardAddress;
    }

    function setTargetChainAddress(bytes32 newAddress, uint16 targetChain) internal {
        _state.targetChainAddresses[targetChain] = newAddress;
    }

    function setMaximumBudget(uint16 targetChainId, uint256 amount) internal {
        _state.maximumBudget[targetChainId] = amount;
    }

    function setPriceInfo(uint16 updateChainId, uint128 updateGasPrice, uint128 updateNativeCurrencyPrice) internal {
        _state.data[updateChainId].gasPrice = updateGasPrice;
        _state.data[updateChainId].nativeCurrencyPrice = updateNativeCurrencyPrice;
    }

    function setAssetConversionBuffer(uint16 targetChain, uint16 tolerance, uint16 toleranceDenominator) internal {
        RelayProviderStorage.AssetConversion storage assetConversion = _state.assetConversion[targetChain];
        assetConversion.buffer = tolerance;
        assetConversion.denominator = toleranceDenominator;
    }
}
