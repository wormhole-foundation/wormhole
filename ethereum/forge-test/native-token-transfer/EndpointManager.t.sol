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

    function setUp() public {
        // deploy sample token contract
        // deploy wormhole contract
        // wormhole = deployWormholeForTest();
        // deploy endpoint contracts
        // instantiate endpoint manager contract
        // endpointManager = new EndpointManagerContract();
    }
}
