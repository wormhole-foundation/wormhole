// contracts/Relayer.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "../libraries/external/BytesLib.sol";

import "./RelayProviderGetters.sol";
import "./RelayProviderSetters.sol";
import "./RelayProviderStructs.sol";

abstract contract RelayProviderGovernance is RelayProviderGetters, RelayProviderSetters, ERC1967Upgrade {
    error ChainIdIsZero();
    error GasPriceIsZero();
    error NativeCurrencyPriceIsZero();
    error FailedToInitializeImplementation(string reason);
    error WrongChainId();
    error AddressIsZero();
    error CallerMustBePendingOwner();
    error CallerMustBeOwner();

    event ContractUpgraded(address indexed oldContract, address indexed newContract);
    event ChainSupportUpdated(uint16 targetChainId, bool isSupported);
    event OwnershipTransfered(address indexed oldOwner, address indexed newOwner);
    event RewardAddressUpdated(address indexed newAddress);
    event TargetChainAddressUpdated(bytes32 indexed newAddress, uint16 indexed targetChain);
    event DeliverGasOverheadUpdated(uint32 indexed oldGasOverhead, uint32 indexed newGasOverhead);
    event CoreRelayerUpdated(address coreRelayer);
    event AssetConversionBufferUpdated(uint16 targetChain, uint16 buffer, uint16 bufferDenominator);

    function updateCoreRelayer(address payable newAddress) public onlyOwner {
        setCoreRelayer(newAddress);
        emit CoreRelayerUpdated(newAddress);
    }

    function updateSupportedChain(uint16 targetChainId, bool isSupported) public onlyOwner {
        setChainSupported(targetChainId, isSupported);
        emit ChainSupportUpdated(targetChainId, isSupported);
    }

    function updateRewardAddress(address payable newAddress) public onlyOwner {
        setRewardAddress(newAddress);
        emit RewardAddressUpdated(newAddress);
    }

    function updateTargetChainAddress(bytes32 newAddress, uint16 targetChain) public onlyOwner {
        setTargetChainAddress(newAddress, targetChain);
        emit TargetChainAddressUpdated(newAddress, targetChain);
    }

    function updateDeliverGasOverhead(uint16 chainId, uint32 newGasOverhead) public onlyOwner {
        uint32 currentGasOverhead = deliverGasOverhead(chainId);
        setDeliverGasOverhead(chainId, newGasOverhead);
        emit DeliverGasOverheadUpdated(currentGasOverhead, newGasOverhead);
    }

    function updatePrice(uint16 updateChainId, uint128 updateGasPrice, uint128 updateNativeCurrencyPrice)
        public
        onlyOwner
    {
        if (updateChainId == 0) {
            revert ChainIdIsZero();
        }
        if (updateGasPrice == 0) {
            revert GasPriceIsZero();
        }
        if (updateNativeCurrencyPrice == 0) {
            revert NativeCurrencyPriceIsZero();
        }

        setPriceInfo(updateChainId, updateGasPrice, updateNativeCurrencyPrice);
    }

    function updatePrices(RelayProviderStructs.UpdatePrice[] memory updates) public onlyOwner {
        uint256 pricesLen = updates.length;
        for (uint256 i = 0; i < pricesLen;) {
            updatePrice(updates[i].chainId, updates[i].gasPrice, updates[i].nativeCurrencyPrice);
            unchecked {
                i += 1;
            }
        }
    }

    function updateMaximumBudget(uint16 targetChainId, uint256 maximumTotalBudget) public onlyOwner {
        setMaximumBudget(targetChainId, maximumTotalBudget);
    }

    function updateAssetConversionBuffer(uint16 targetChain, uint16 buffer, uint16 bufferDenominator)
        public
        onlyOwner
    {
        setAssetConversionBuffer(targetChain, buffer, bufferDenominator);
        emit AssetConversionBufferUpdated(targetChain, buffer, bufferDenominator);
    }

    /// @dev upgrade serves to upgrade contract implementations
    function upgrade(uint16 relayProviderChainId, address newImplementation) public onlyOwner {
        if (relayProviderChainId != chainId()) {
            revert WrongChainId();
        }

        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // call initialize function of the new implementation
        (bool success, bytes memory reason) = newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        if (!success) {
            revert FailedToInitializeImplementation(string(reason));
        }

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    /**
     * @dev submitOwnershipTransferRequest serves to begin the ownership transfer process of the contracts
     * - it saves an address for the new owner in the pending state
     */
    function submitOwnershipTransferRequest(uint16 thisRelayerChainId, address newOwner) public onlyOwner {
        if (thisRelayerChainId != chainId()) {
            revert WrongChainId();
        }
        if (newOwner == address(0)) {
            revert AddressIsZero();
        }

        setPendingOwner(newOwner);
    }

    /**
     * @dev confirmOwnershipTransferRequest serves to finalize an ownership transfer
     * - it checks that the caller is the pendingOwner to validate the wallet address
     * - it updates the owner state variable with the pendingOwner state variable
     */
    function confirmOwnershipTransferRequest() public {
        // cache the new owner address
        address newOwner = pendingOwner();

        if (msg.sender != newOwner) {
            revert CallerMustBePendingOwner();
        }

        // cache currentOwner for Event
        address currentOwner = owner();

        // update the owner in the contract state and reset the pending owner
        setOwner(newOwner);
        setPendingOwner(address(0));

        emit OwnershipTransfered(currentOwner, newOwner);
    }

    modifier onlyOwner() {
        if (owner() != _msgSender()) {
            revert CallerMustBeOwner();
        }
        _;
    }
}
