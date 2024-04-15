

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {BridgeImplementation} from "../contracts/bridge/BridgeImplementation.sol";
import "forge-std/Script.sol";

contract DeployTokenBridgeImplementationOnly is Script {
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
        BridgeImplementation impl = new BridgeImplementation();

        return address(impl);
    }
}