// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "contracts/Implementation.sol";

contract MyImplementation is Implementation {
    constructor(uint256 evmChain, uint16 chain) {
        setEvmChainId(evmChain);
        setChainId(chain);
    }

    function getImplementation() public view returns (address impl) {
        impl = _getImplementation();
        return impl;
    }

    function upgradeImpl(address newImplementation) public {
        upgradeImplementation(newImplementation);
    }
}
