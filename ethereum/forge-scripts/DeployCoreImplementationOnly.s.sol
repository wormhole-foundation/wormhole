

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {Implementation} from "../contracts/Implementation.sol";
import "forge-std/Script.sol";

contract DeployCoreImplementationOnly is Script {
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
        Implementation impl = new Implementation();

        return address(impl);

        // TODO: initialize?
    }
}