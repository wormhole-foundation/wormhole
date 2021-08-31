// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../token/NFTImplementation.sol";

contract MockNFTImplementation is NFTImplementation {
    function testNewImplementationActive() external pure returns (bool) {
        return true;
    }
}
