// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../Implementation.sol";

contract MockImplementation is Implementation {
    function testNewImplementationActive() external pure returns (bool) {
        return true;
    }
}
