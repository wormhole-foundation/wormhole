

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {Shutdown} from "../contracts/Shutdown.sol";
import "forge-std/Script.sol";

contract DeployCoreShutdown is Script {
    // DryRun - Deploy the system
    function dryRun() public {
        _deploy();
    }
    // Deploy the system
    function run() public returns (address deployedAddress) {
        vm.startBroadcast();
        deployedAddress = _deploy();
        vm.stopBroadcast();
    }
    function _deploy() internal returns (address deployedAddress) {
        Shutdown shutdown = new Shutdown();

        return address(shutdown);
    }
}