

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {NFTBridgeImplementation} from "../contracts/nft/NFTBridgeImplementation.sol";
import "forge-std/Script.sol";

contract DeployNFTBridgeImplementationOnly is Script {
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
        NFTBridgeImplementation impl = new NFTBridgeImplementation();

        return address(impl);
    }
}
