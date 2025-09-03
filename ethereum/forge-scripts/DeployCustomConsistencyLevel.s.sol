// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import {
    CustomConsistencyLevel,
    customConsistencyLevelVersion
} from "../contracts/custom_consistency_level/CustomConsistencyLevel.sol";
import "forge-std/Script.sol";

// DeployCustomConsistencyLevel is a forge script to deploy the CustomConsistencyLevel contract. Use ./sh/deployCustomConsistencyLevel.sh to invoke this.
// e.g. anvil
// EVM_CHAIN_ID=31337 MNEMONIC=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 ./sh/deployCustomConsistencyLevel.sh
// e.g. anvil --fork-url https://ethereum-rpc.publicnode.com
// EVM_CHAIN_ID=1 MNEMONIC=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 ./sh/deployCustomConsistencyLevel.sh
contract DeployCustomConsistencyLevel is Script {
    function test() public {} // Exclude this from coverage report.

    function dryRun() public {
        _deploy();
    }

    function run() public returns (address deployedAddress) {
        vm.startBroadcast();
        (deployedAddress) = _deploy();
        vm.stopBroadcast();
    }

    function _deploy() internal returns (address deployedAddress) {
        bytes32 salt = keccak256(abi.encodePacked(customConsistencyLevelVersion));
        CustomConsistencyLevel customConsistencyLevel = new CustomConsistencyLevel{salt: salt}();

        return (address(customConsistencyLevel));
    }
}
