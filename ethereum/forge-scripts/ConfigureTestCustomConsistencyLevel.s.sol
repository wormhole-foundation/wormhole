// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import {ITestCustomConsistencyLevel} from
    "../contracts/custom_consistency_level/interfaces/ITestCustomConsistencyLevel.sol";
import "forge-std/Script.sol";

contract ConfigureTestCustomConsistencyLevel is Script {
    function test() public {} // Exclude this from coverage report.

    function dryRun(
        address _tccl
    ) public {
        _configure(_tccl);
    }

    function run(
        address _tccl
    ) public {
        vm.startBroadcast();
        _configure(_tccl);
        vm.stopBroadcast();
    }

    function _configure(
        address _tccl
    ) internal {
        ITestCustomConsistencyLevel tccl = ITestCustomConsistencyLevel(_tccl);
        tccl.configure();
    }
}
