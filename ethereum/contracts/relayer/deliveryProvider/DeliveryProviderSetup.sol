// contracts/Setup.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "./DeliveryProviderGovernance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

contract DeliveryProviderSetup is DeliveryProviderSetters, ERC1967Upgrade {
    error ImplementationAddressIsZero();
    error FailedToInitializeImplementation(string reason);

    function setup(address implementation, uint16 chainId) public {
        // sanity check initial values
        if (implementation == address(0)) {
            revert ImplementationAddressIsZero();
        }

        setOwner(_msgSender());

        setChainId(chainId);

        _upgradeTo(implementation);

        // call initialize function of the new implementation
        (bool success, bytes memory reason) =
            implementation.delegatecall(abi.encodeWithSignature("initialize()"));
        if (!success) {
            revert FailedToInitializeImplementation(string(reason));
        }
    }
}
