// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import {
    DelegatedManagerSet,
    DELEGATED_MANAGER_SET_VERSION
} from "../contracts/delegated_manager_set/DelegatedManagerSet.sol";
import "forge-std/Script.sol";

// DeployDelegatedManagerSet is a forge script to deploy the DelegatedManagerSet contract.
// Use ./sh/deployDelegatedManagerSet.sh to invoke this.
//
// Required environment variables:
//   WORMHOLE_ADDRESS - The address of the Wormhole core contract
//
// Example usage:
//   tilt: WORMHOLE_ADDRESS=0xC89Ce4735882C9F0f0FE26686c53074E09B0D550 ./sh/deployDelegatedManagerSet.sh
//   anvil: EVM_CHAIN_ID=31337 MNEMONIC=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 WORMHOLE_ADDRESS=0x... ./sh/deployDelegatedManagerSet.sh
contract DeployDelegatedManagerSet is Script {
    function test() public {} // Exclude this from coverage report.

    function dryRun(address wormhole) public {
        _deploy(wormhole);
    }

    function run(address wormhole) public returns (address deployedAddress) {
        vm.startBroadcast();
        deployedAddress = _deploy(wormhole);
        vm.stopBroadcast();
    }

    function _deploy(address wormhole) internal returns (address deployedAddress) {
        require(wormhole != address(0), "Wormhole address cannot be zero");

        bytes32 salt = keccak256(abi.encodePacked(DELEGATED_MANAGER_SET_VERSION));
        DelegatedManagerSet delegatedManagerSet = new DelegatedManagerSet{salt: salt}(wormhole);

        return address(delegatedManagerSet);
    }
}
