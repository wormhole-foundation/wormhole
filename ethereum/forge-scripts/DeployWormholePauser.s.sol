// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Script.sol";
import {WormholePauser} from "../contracts/wormhole_pauser/WormholePauser.sol";

contract DeployWormholePauser is Script {
    bytes32 internal constant DEPLOY_SALT = keccak256(abi.encodePacked("WormholePauser"));

    function dryRun(address wormholeCore) public {
        _deploy(wormholeCore);
    }

    function run(address wormholeCore) public returns (address deployedWormholePauser) {
        vm.startBroadcast();
        deployedWormholePauser = _deploy(wormholeCore);
        vm.stopBroadcast();
    }

    function _deploy(address wormholeCore) internal returns (address) {
        WormholePauser pauser = new WormholePauser{salt: DEPLOY_SALT}(wormholeCore);
        return address(pauser);
    }
}
