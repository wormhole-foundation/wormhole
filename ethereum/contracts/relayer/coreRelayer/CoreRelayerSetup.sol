// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./CoreRelayerGovernance.sol";
import "./ForwardWrapper.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

contract CoreRelayerSetup is CoreRelayerSetters, ERC1967Upgrade {
    error ImplementationAddressIsZero();
    error WormholeAddressIsZero();
    error DefaultRelayProviderAddressIsZero();
    error FailedToInitializeImplementation(string reason);

    function setup(
        address implementation,
        uint16 chainId,
        address wormhole,
        address defaultRelayProvider,
        uint16 governanceChainId,
        bytes32 governanceContract,
        uint256 evmChainId
    ) public {
        // sanity check initial values
        if (implementation == address(0)) {
            revert ImplementationAddressIsZero();
        }
        if (wormhole == address(0)) {
            revert WormholeAddressIsZero();
        }
        if (defaultRelayProvider == address(0)) {
            revert DefaultRelayProviderAddressIsZero();
        }

        setChainId(chainId);

        setWormhole(wormhole);

        setRelayProvider(defaultRelayProvider);

        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);
        setEvmChainId(evmChainId);

        setForwardWrapper(address(new ForwardWrapper(address(this), wormhole)));

        _upgradeTo(implementation);

        // call initialize function of the new implementation
        (bool success, bytes memory reason) = implementation.delegatecall(abi.encodeWithSignature("initialize()"));
        if (!success) {
            revert FailedToInitializeImplementation(string(reason));
        }
    }
}
