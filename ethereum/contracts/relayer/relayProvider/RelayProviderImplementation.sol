// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "./RelayProvider.sol";

contract RelayProviderImplementation is RelayProvider {
    error ImplementationAlreadyInitialized();

    function initialize() public virtual initializer {
        // this function needs to be exposed for an upgrade to pass
    }

    modifier initializer() {
        address impl = ERC1967Upgrade._getImplementation();

        if (isInitialized(impl)) {
            revert ImplementationAlreadyInitialized();
        }

        setInitialized(impl);

        _;
    }
}
