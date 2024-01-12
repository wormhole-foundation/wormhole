// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

import "forge-std/Test.sol";

import "../../contracts/native_token_transfer/EndpointManager.sol";
import "../../contracts/Wormhole.sol";

// @dev A non-abstract EndpointManager contract
contract EndpointManagerContract is EndpointManager {
    constructor(
        address _token,
        bool _isLockingMode,
        uint16 _chainId,
        uint256 _evmChainId
    ) EndpointManager(_token, _isLockingMode, _chainId, _evmChainId) {}
}

contract TestEndpointManager is Test {
    Wormhole wormhole;
    EndpointManager endpointManager;

    function test_countSetBits() public {
        assertEq(endpointManager.countSetBits(5), 2);
        assertEq(endpointManager.countSetBits(0), 0);
        assertEq(endpointManager.countSetBits(15), 4);
        assertEq(endpointManager.countSetBits(16), 1);
        assertEq(endpointManager.countSetBits(65535), 16);
    }

    function setUp() public {
        endpointManager = new EndpointManagerContract(
            address(0),
            false,
            0,
            0
        );
        // deploy sample token contract
        // deploy wormhole contract
        // wormhole = deployWormholeForTest();
        // deploy endpoint contracts
        // instantiate endpoint manager contract
        // endpointManager = new EndpointManagerContract();
    }
}
