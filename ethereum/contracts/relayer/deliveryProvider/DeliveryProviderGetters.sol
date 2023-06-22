// contracts/Getters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/IWormhole.sol";
import "../../interfaces/relayer/TypedUnits.sol";

import "./DeliveryProviderState.sol";

contract DeliveryProviderGetters is DeliveryProviderState {
    function owner() public view returns (address) {
        return _state.owner;
    }

    function pendingOwner() public view returns (address) {
        return _state.pendingOwner;
    }

    function pricingWallet() public view returns (address) {
        return _state.pricingWallet;
    }

    function isInitialized(address impl) public view returns (bool) {
        return _state.initializedImplementations[impl];
    }

    function chainId() public view returns (uint16) {
        return _state.chainId;
    }

    function coreRelayer() public view returns (address) {
        return _state.coreRelayer;
    }

    function gasPrice(uint16 targetChain) public view returns (GasPrice) {
        return _state.data[targetChain].gasPrice;
    }

    function nativeCurrencyPrice(uint16 targetChain) public view returns (WeiPrice) {
        return _state.data[targetChain].nativeCurrencyPrice;
    }

    function deliverGasOverhead(uint16 targetChain) public view returns (Gas) {
        return _state.deliverGasOverhead[targetChain];
    }

    function maximumBudget(uint16 targetChain) public view returns (TargetNative) {
        return _state.maximumBudget[targetChain];
    }

    function targetChainAddress(uint16 targetChain) public view returns (bytes32) {
        return _state.targetChainAddresses[targetChain];
    }

    function rewardAddress() public view returns (address payable) {
        return _state.rewardAddress;
    }

    function assetConversionBuffer(uint16 targetChain)
        public
        view
        returns (uint16 tolerance, uint16 toleranceDenominator)
    {
        DeliveryProviderStorage.AssetConversion storage assetConversion =
            _state.assetConversion[targetChain];
        return (assetConversion.buffer, assetConversion.denominator);
    }
}
