// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/bridge/Bridge.sol";
import "forge-std/Test.sol";

// @dev ensure some internal methods are public for testing
contract ExportedBridge is Bridge {
    function _truncateAddressPub(bytes32 b) public pure returns (address) {
        return super._truncateAddress(b);
    }

    function setChainIdPub(uint16 chainId) public {
        return super.setChainId(chainId);
    }

    function setEvmChainIdPub(uint256 evmChainId) public {
        return super.setEvmChainId(evmChainId);
    }
}

contract TestBridge is Test {
    ExportedBridge bridge;

    function setUp() public {
        bridge = new ExportedBridge();
    }

    function testTruncate(bytes32 b) public {
        bool invalidAddress = bytes12(b) != 0;
        if (invalidAddress) {
            vm.expectRevert( "invalid EVM address");
        }
        bytes32 converted = bytes32(uint256(uint160(bytes20(bridge._truncateAddressPub(b)))));

        if (!invalidAddress) {
            require(converted == b, "truncate does not roundrip");
        }
    }

    function testEvmChainId() public {
        vm.chainId(1);
        bridge.setChainIdPub(1);
        bridge.setEvmChainIdPub(1);
        assertEq(bridge.chainId(), 1);
        assertEq(bridge.evmChainId(), 1);

        // fork occurs, block.chainid changes
        vm.chainId(10001);

        bridge.setEvmChainIdPub(10001);
        assertEq(bridge.chainId(), 1);
        assertEq(bridge.evmChainId(), 10001);

        // evmChainId must equal block.chainid
        vm.expectRevert("invalid evmChainId");
        bridge.setEvmChainIdPub(1337);

    }
}
