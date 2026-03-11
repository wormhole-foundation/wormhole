// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import "forge-std/Script.sol";
import {WormholeDelegatedGuardians} from "../contracts/delegated_guardians/WormholeDelegatedGuardians.sol";

contract DeployWormholeDelegatedGuardians is Script {
  bytes32 internal constant DEPLOY_SALT = keccak256(abi.encodePacked("WormholeDelegatedGuardians"));

  function dryRun(address wormholeCore) public {
    _deploy(wormholeCore);
  }

  function run(address wormholeCore) public returns (address deployedDelegatedGuardians) {
    vm.startBroadcast();
    deployedDelegatedGuardians = _deploy(wormholeCore);
    vm.stopBroadcast();
  }

  function _deploy(address wormholeCore) internal returns (address deployedDelegatedGuardians) {
    WormholeDelegatedGuardians delegated = new WormholeDelegatedGuardians{salt: DEPLOY_SALT}(wormholeCore);
    return address(delegated);
  }
}
