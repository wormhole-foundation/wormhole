// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";
import "./ForwardWrapper.sol";

import "./CoreRelayer.sol";

contract CoreRelayerImplementation is CoreRelayer {
    error ImplementationAlreadyInitialized();

    function initialize() public virtual initializer {
        setForwardWrapper(address(new ForwardWrapper(address(this), address(wormhole()))));
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
