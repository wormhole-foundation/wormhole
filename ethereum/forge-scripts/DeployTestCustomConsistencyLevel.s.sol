// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import {
    TestCustomConsistencyLevel,
    testCustomConsistencyLevelVersion
} from "../contracts/custom_consistency_level/TestCustomConsistencyLevel.sol";
import "forge-std/Script.sol";

// DeployTestCustomConsistencyLevel is a forge script to deploy the TestCustomConsistencyLevel contract. Use ./sh/deployTestCustomConsistencyLevel.sh to invoke this.
// e.g. anvil
// EVM_CHAIN_ID=31337 MNEMONIC=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 WORMHOLE_ADDRESS= CUSTOM_CONSISTENCY_LEVEL= ./sh/deployTestCustomConsistencyLevel.sh
// e.g. anvil --fork-url https://ethereum-rpc.publicnode.com
// EVM_CHAIN_ID=1 MNEMONIC=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 WORMHOLE_ADDRESS= CUSTOM_CONSISTENCY_LEVEL= ./sh/deployTestCustomConsistencyLevel.sh
contract DeployTestCustomConsistencyLevel is Script {
    function test() public {} // Exclude this from coverage report.

    function dryRun(address _wormhole, address _customConsistencyLevel) public {
        _deploy(_wormhole, _customConsistencyLevel);
    }

    function run(
        address _wormhole,
        address _customConsistencyLevel
    ) public returns (address deployedAddress) {
        vm.startBroadcast();
        (deployedAddress) = _deploy(_wormhole, _customConsistencyLevel);
        vm.stopBroadcast();
    }

    function _deploy(
        address _wormhole,
        address _customConsistencyLevel
    ) internal returns (address deployedAddress) {
        bytes32 salt = keccak256(abi.encodePacked(testCustomConsistencyLevelVersion));
        TestCustomConsistencyLevel customConsistencyLevel =
            new TestCustomConsistencyLevel{salt: salt}(_wormhole, _customConsistencyLevel);

        return (address(customConsistencyLevel));
    }
}
