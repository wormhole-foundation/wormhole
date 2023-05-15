// contracts/Getters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/IWormhole.sol";
import "../../interfaces/relayer/TypedUnits.sol";

import "./RelayProviderState.sol";

contract RelayProviderGetters is RelayProviderState {
    function owner() public view returns (address) {
        return _state.owner;
    }

    function pendingOwner() public view returns (address) {
        return _state.pendingOwner;
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

    function gasPrice(uint16 targetChainId) public view returns (GasPrice) {
        return _state.data[targetChainId].gasPrice;
    }

    function nativeCurrencyPrice(uint16 targetChainId) public view returns (WeiPrice) {
        return _state.data[targetChainId].nativeCurrencyPrice;
    }

    function deliverGasOverhead(uint16 targetChainId) public view returns (Gas) {
        return _state.deliverGasOverhead[targetChainId];
    }

    function maximumBudget(uint16 targetChainId) public view returns (Wei) {
        return _state.maximumBudget[targetChainId];
    }

    function targetChainAddress(uint16 targetChainId) public view returns (bytes32) {
        return _state.targetChainAddresses[targetChainId];
    }

    function rewardAddress() public view returns (address payable) {
        return _state.rewardAddress;
    }

    function assetConversionBuffer(uint16 targetChainId)
        public
        view
        returns (uint16 tolerance, uint16 toleranceDenominator)
    {
        RelayProviderStorage.AssetConversion storage assetConversion =
            _state.assetConversion[targetChainId];
        return (assetConversion.buffer, assetConversion.denominator);
    }
}
