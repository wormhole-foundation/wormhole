// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import {ITestCustomConsistencyLevel} from
    "../contracts/custom_consistency_level/interfaces/ITestCustomConsistencyLevel.sol";
import "forge-std/Script.sol";

contract PostTestCustomConsistencyLevelMsg is Script {
    function test() public {} // Exclude this from coverage report.

    function dryRun(address _tccl, string calldata _payload) public {
        _publishMessage(_tccl, _payload);
    }

    function run(address _tccl, string calldata _payload) public {
        vm.startBroadcast();
        _publishMessage(_tccl, _payload);
        vm.stopBroadcast();
    }

    function _publishMessage(address _tccl, string calldata _payload) internal {
        ITestCustomConsistencyLevel tccl = ITestCustomConsistencyLevel(_tccl);
        tccl.publishMessage(_payload);
    }
}
