// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../token/TokenImplementation.sol";

contract MockTokenImplementation is TokenImplementation {
    function testNewImplementationActive() external pure returns (bool) {
        return true;
    }
}
