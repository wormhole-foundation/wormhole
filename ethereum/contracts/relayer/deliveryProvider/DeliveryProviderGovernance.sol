// contracts/Relayer.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "../../libraries/external/BytesLib.sol";
import "../../interfaces/relayer/TypedUnits.sol";

import "./DeliveryProviderGetters.sol";
import "./DeliveryProviderSetters.sol";
import "./DeliveryProviderStructs.sol";

abstract contract DeliveryProviderGovernance is
    DeliveryProviderGetters,
    DeliveryProviderSetters,
    ERC1967Upgrade
{
    using WeiLib for Wei;
    using GasLib for Gas;
    using DollarLib for Dollar;
    using WeiPriceLib for WeiPrice;
    using GasPriceLib for GasPrice;

    error ChainIdIsZero();
    error GasPriceIsZero();
    error NativeCurrencyPriceIsZero();
    error FailedToInitializeImplementation(string reason);
    error WrongChainId();
    error AddressIsZero();
    error CallerMustBePendingOwner();
    error CallerMustBeOwner();
    error CallerMustBeOwnerOrPricingWallet();

    event ContractUpgraded(address indexed oldContract, address indexed newContract);
    event ChainSupportUpdated(uint16 targetChain, bool isSupported);
    event OwnershipTransfered(address indexed oldOwner, address indexed newOwner);
    event RewardAddressUpdated(address indexed newAddress);
    event TargetChainAddressUpdated(uint16 indexed targetChain, bytes32 indexed newAddress);
    event DeliverGasOverheadUpdated(Gas indexed oldGasOverhead, Gas indexed newGasOverhead);
    event WormholeRelayerUpdated(address coreRelayer);
    event AssetConversionBufferUpdated(uint16 targetChain, uint16 buffer, uint16 bufferDenominator);

    function updateWormholeRelayer(address payable newAddress) external onlyOwner {
        updateWormholeRelayerImpl(newAddress);
    }

    function updateWormholeRelayerImpl(address payable newAddress) internal {
        setWormholeRelayer(newAddress);
        emit WormholeRelayerUpdated(newAddress);
    }

    function updateSupportedChain(uint16 targetChain, bool isSupported) public onlyOwner {
        setChainSupported(targetChain, isSupported);
        emit ChainSupportUpdated(targetChain, isSupported);
    }

    function updateSupportedChains(DeliveryProviderStructs.SupportedChainUpdate[] memory updates)
        public
        onlyOwner
    {
        uint256 updatesLength = updates.length;
        for (uint256 i = 0; i < updatesLength;) {
            DeliveryProviderStructs.SupportedChainUpdate memory update = updates[i];
            updateSupportedChainImpl(update.chainId, update.isSupported);
            unchecked {
                i += 1;
            }
        }
    }

    function updatePricingWallet(address newPricingWallet) external onlyOwner {
        setPricingWallet(newPricingWallet);
    }

    function updateSupportedChainImpl(uint16 targetChain, bool isSupported) internal {
        setChainSupported(targetChain, isSupported);
        emit ChainSupportUpdated(targetChain, isSupported);
    }

    function updateRewardAddress(address payable newAddress) external onlyOwner {
        updateRewardAddressImpl(newAddress);
    }

    function updateRewardAddressImpl(address payable newAddress) internal {
        setRewardAddress(newAddress);
        emit RewardAddressUpdated(newAddress);
    }

    function updateTargetChainAddress(uint16 targetChain, bytes32 newAddress) public onlyOwner {
        updateTargetChainAddressImpl(targetChain, newAddress);
    }

    function updateTargetChainAddresses(DeliveryProviderStructs.TargetChainUpdate[] memory updates)
        external
        onlyOwner
    {
        uint256 updatesLength = updates.length;
        for (uint256 i = 0; i < updatesLength;) {
            DeliveryProviderStructs.TargetChainUpdate memory update = updates[i];
            updateTargetChainAddressImpl(update.chainId, update.targetChainAddress);
            unchecked {
                i += 1;
            }
        }
    }

    function updateTargetChainAddressImpl(uint16 targetChain, bytes32 newAddress) internal {
        setTargetChainAddress(targetChain, newAddress);
        emit TargetChainAddressUpdated(targetChain, newAddress);
    }

    function updateDeliverGasOverhead(uint16 chainId, Gas newGasOverhead) external onlyOwnerOrPricingWallet {
        updateDeliverGasOverheadImpl(chainId, newGasOverhead);
    }

    function updateDeliverGasOverheads(
        DeliveryProviderStructs.DeliverGasOverheadUpdate[] memory overheadUpdates
    ) external onlyOwnerOrPricingWallet {
        uint256 updatesLength = overheadUpdates.length;
        for (uint256 i = 0; i < updatesLength;) {
            DeliveryProviderStructs.DeliverGasOverheadUpdate memory update = overheadUpdates[i];
            updateDeliverGasOverheadImpl(update.chainId, update.newGasOverhead);
            unchecked {
                i += 1;
            }
        }
    }

    function updateDeliverGasOverheadImpl(uint16 chainId, Gas newGasOverhead) internal {
        Gas currentGasOverhead = deliverGasOverhead(chainId);
        setDeliverGasOverhead(chainId, newGasOverhead);
        emit DeliverGasOverheadUpdated(currentGasOverhead, newGasOverhead);
    }

    function updatePrice(
        uint16 updateChainId,
        GasPrice updateGasPrice,
        WeiPrice updateNativeCurrencyPrice
    ) external onlyOwnerOrPricingWallet {
        updatePriceImpl(updateChainId, updateGasPrice, updateNativeCurrencyPrice);
    }

    function updatePrices(DeliveryProviderStructs.UpdatePrice[] memory updates)
        external
        onlyOwnerOrPricingWallet
    {
        uint256 pricesLength = updates.length;
        for (uint256 i = 0; i < pricesLength;) {
            DeliveryProviderStructs.UpdatePrice memory update = updates[i];
            updatePriceImpl(update.chainId, update.gasPrice, update.nativeCurrencyPrice);
            unchecked {
                i += 1;
            }
        }
    }

    function updatePriceImpl(
        uint16 updateChainId,
        GasPrice updateGasPrice,
        WeiPrice updateNativeCurrencyPrice
    ) internal {
        if (updateChainId == 0) {
            revert ChainIdIsZero();
        }
        if (updateGasPrice.unwrap() == 0) {
            revert GasPriceIsZero();
        }
        if (updateNativeCurrencyPrice.unwrap() == 0) {
            revert NativeCurrencyPriceIsZero();
        }

        setPriceInfo(updateChainId, updateGasPrice, updateNativeCurrencyPrice);
    }

    function updateMaximumBudget(uint16 targetChain, Wei maximumTotalBudget) external onlyOwnerOrPricingWallet {
        setMaximumBudget(targetChain, maximumTotalBudget);
    }

    function updateMaximumBudgets(DeliveryProviderStructs.MaximumBudgetUpdate[] memory updates)
        external
        onlyOwnerOrPricingWallet
    {
        uint256 updatesLength = updates.length;
        for (uint256 i = 0; i < updatesLength;) {
            DeliveryProviderStructs.MaximumBudgetUpdate memory update = updates[i];
            setMaximumBudget(update.chainId, update.maximumTotalBudget);
            unchecked {
                i += 1;
            }
        }
    }

    function updateAssetConversionBuffer(
        uint16 targetChain,
        uint16 buffer,
        uint16 bufferDenominator
    ) external onlyOwnerOrPricingWallet {
        updateAssetConversionBufferImpl(targetChain, buffer, bufferDenominator);
    }

    function updateAssetConversionBuffers(
        DeliveryProviderStructs.AssetConversionBufferUpdate[] memory updates
    ) external onlyOwnerOrPricingWallet {
        uint256 updatesLength = updates.length;
        for (uint256 i = 0; i < updatesLength;) {
            DeliveryProviderStructs.AssetConversionBufferUpdate memory update = updates[i];
            updateAssetConversionBufferImpl(update.chainId, update.buffer, update.bufferDenominator);
            unchecked {
                i += 1;
            }
        }
    }

    function updateAssetConversionBufferImpl(
        uint16 targetChain,
        uint16 buffer,
        uint16 bufferDenominator
    ) internal {
        setAssetConversionBuffer(targetChain, buffer, bufferDenominator);
        emit AssetConversionBufferUpdated(targetChain, buffer, bufferDenominator);
    }

    function updateConfig(
        DeliveryProviderStructs.Update[] memory updates,
        DeliveryProviderStructs.CoreConfig memory coreConfig
    ) external onlyOwner {
        uint256 updatesLength = updates.length;
        for (uint256 i = 0; i < updatesLength;) {
            DeliveryProviderStructs.Update memory update = updates[i];
            if (update.updatePrice) {
                updatePriceImpl(update.chainId, update.gasPrice, update.nativeCurrencyPrice);
            }
            if (update.updateTargetChainAddress) {
                updateTargetChainAddressImpl(update.chainId, update.targetChainAddress);
            }
            if (update.updateDeliverGasOverhead) {
                updateDeliverGasOverheadImpl(update.chainId, update.newGasOverhead);
            }
            if (update.updateMaximumBudget) {
                setMaximumBudget(update.chainId, update.maximumTotalBudget);
            }
            if (update.updateAssetConversionBuffer) {
                updateAssetConversionBufferImpl(
                    update.chainId, update.buffer, update.bufferDenominator
                );
            }
            if (update.updateSupportedChain) {
                updateSupportedChainImpl(update.chainId, update.isSupported);
            }
            unchecked {
                i += 1;
            }
        }

        if (coreConfig.updateWormholeRelayer) {
            updateWormholeRelayerImpl(coreConfig.coreRelayer);
        }

        if (coreConfig.updateRewardAddress) {
            updateRewardAddressImpl(coreConfig.rewardAddress);
        }
    }

    /// @dev upgrade serves to upgrade contract implementations
    function upgrade(uint16 deliveryProviderChainId, address newImplementation) public onlyOwner {
        if (deliveryProviderChainId != chainId()) {
            revert WrongChainId();
        }

        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // call initialize function of the new implementation
        (bool success, bytes memory reason) =
            newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        if (!success) {
            revert FailedToInitializeImplementation(string(reason));
        }

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    /**
     * @dev submitOwnershipTransferRequest serves to begin the ownership transfer process of the contracts
     * - it saves an address for the new owner in the pending state
     */
    function submitOwnershipTransferRequest(
        uint16 thisRelayerChainId,
        address newOwner
    ) external onlyOwner {
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
    function confirmOwnershipTransferRequest() external {
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

    modifier onlyOwnerOrPricingWallet() {
        if ((pricingWallet() != _msgSender()) && (owner() != _msgSender())) {
            revert CallerMustBeOwnerOrPricingWallet();
        }
        _;
    }
}
