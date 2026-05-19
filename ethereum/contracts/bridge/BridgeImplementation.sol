// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "./Bridge.sol";


contract BridgeImplementation is Bridge {
    error AlreadyInitialized();

    // Beacon getter for the token contracts
    function implementation() public view returns (address) {
        return tokenImplementation();
    }

    function initialize() initializer public virtual {
        // this function needs to be exposed for an upgrade to pass
        if (evmChainId() == 10) { // optimism
            address weth = 0x4200000000000000000000000000000000000006;
            setWETH(weth);
        }
    }

    modifier initializer() {
        address impl = ERC1967Upgrade._getImplementation();

        if (isInitialized(impl)) revert AlreadyInitialized();

        setInitialized(impl);

        _;
    }
}
