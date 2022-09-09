// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../NFTBridgeImplementation.sol";

contract MockNFTBridgeImplementation is NFTBridgeImplementation {
    function initialize() initializer public override {
        // this function needs to be exposed for an upgrade to pass
    }

    function testNewImplementationActive() external pure returns (bool) {
        return true;
    }

    function testOverwriteEVMChainId(uint16 fakeChainId, uint256 fakeEvmChainId) external {
        _state.provider.chainId = fakeChainId;
        _state.evmChainId = fakeEvmChainId;
    }
}
