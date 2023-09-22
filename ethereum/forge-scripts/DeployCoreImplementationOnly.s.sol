

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
import {Implementation} from "../contracts/Implementation.sol";
import "forge-std/Script.sol";

contract DeployCoreImplementationOnly is Script {
    // DryRun - Deploy the system
    // dry run: forge script DeployCoreImplementationOnly --sig "dryRun()" --rpc-url $RPC
    function dryRun() public {
        _deploy();
    }
    // Deploy the system
    // deploy:  forge script DeployCoreImplementationOnly --sig "run()" --rpc-url $RPC --etherscan-api-key $ETHERSCAN_API_KEY --private-key $RAW_PRIVATE_KEY --broadcast --verify
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