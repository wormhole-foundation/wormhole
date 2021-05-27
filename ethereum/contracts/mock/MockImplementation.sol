// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../Implementation.sol";

contract MockImplementation is Implementation {
    function initialize() initializer public {
        // this function needs to be exposed for an upgrade to pass
    }

    function testNewImplementationActive() external pure returns (bool) {
        return true;
    }
}
