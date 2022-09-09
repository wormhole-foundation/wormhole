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

    function testEvmChainId() public {
        vm.chainId(1);
        setChainId(1);
        setEvmChainId(1);
        assertEq(chainId(), 1);
        assertEq(evmChainId(), 1);

        // fork occurs, block.chainid changes
        vm.chainId(10001);

        setEvmChainId(10001);
        assertEq(chainId(), 1);
        assertEq(evmChainId(), 10001);

        // evmChainId must equal block.chainid
        vm.expectRevert("invalid evmChainId");
        setEvmChainId(1337);

    }
}
