// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "../interfaces/IWormhole.sol";

contract MockBatchMessageIntegration {
    address wormholeAddress;
    bytes[] payloads;

    function parseAndVerifyVM2(bytes memory encodedVm) public {
        // parse and verify a batch VAA
        (IWormhole.VM2 memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM2(encodedVm);
        require(valid, reason);
        require(vm.header.version == 2, "wrong version type");
    }

    function consumeBatchVAA(bytes memory encodedVm2) public {
        // parse and verify a batch VAA
        (IWormhole.VM2 memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM2(encodedVm2);
        require(valid, reason);
        require(vm.header.version == 2, "wrong version type");

        uint256 observationsLen = vm.observations.length;
        for (uint256 i = 0; i < observationsLen; i++) {
            consumeVM3(vm.observations[i]);
        }

        // clear the batch cache
        wormhole().clearBatchCache(vm.header);
    }

    function consumeVM3(bytes memory encodedVm3) internal {
        (IWormhole.Observation memory observation, bool valid, string memory reason) = wormhole().parseAndVerifyVAA(encodedVm3);
        require(valid, reason);
        payloads.push(observation.payload);
    }

    function getPayloads() public view returns (bytes[] memory) {
        return payloads;
    }

    function wormhole() internal view returns (IWormhole) {
        return IWormhole(wormholeAddress);
    }

    function setup(address _wormhole) public {
        wormholeAddress = _wormhole;
    }
}