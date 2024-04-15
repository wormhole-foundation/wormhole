

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
import {NFTBridgeShutdown} from "../contracts/nft/NFTBridgeShutdown.sol";
import "forge-std/Script.sol";

contract DeployNFTBridgeShutdown is Script {
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
        NFTBridgeShutdown shutdown = new NFTBridgeShutdown();

        return address(shutdown);
    }
}