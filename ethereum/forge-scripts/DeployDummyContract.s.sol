

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {Implementation} from "../contracts/Implementation.sol";
import {Setup} from "../contracts/Setup.sol";
import "forge-std/Script.sol";

contract DeployDummyContract is Script {
    function run(uint256 num) public {
        vm.startBroadcast();
        for(uint256 i=0; i<num; i++) {
            deploy();
        }
        vm.stopBroadcast();
    }
    function deploy() internal {
        Setup setup = new Setup();
    }
}