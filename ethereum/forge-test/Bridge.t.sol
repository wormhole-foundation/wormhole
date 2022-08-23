// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/bridge/Bridge.sol";
import "forge-std/Test.sol";

contract TestBridge is Bridge, Test {
    function testTruncate(bytes32 b) public {
        if (bytes12(b) != 0) {
            vm.expectRevert( "invalid EVM address");
        }
        bytes32 converted = bytes32(uint256(uint160(bytes20(_truncateAddress(b)))));
        require(converted == b, "truncate does not roundrip");
    }
}
