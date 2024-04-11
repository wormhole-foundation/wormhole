

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {ITokenBridge} from "../contracts/bridge/interfaces/ITokenBridge.sol";
import "forge-std/Script.sol";

contract RegisterChainsTokenBridge is Script {
    // DryRun - Register chains
    // dry run: forge script RegisterChainsTokenBridge --sig "dryRun(address,bytes[])" --rpc-url $RPC
    function dryRun(address tokenBridge, bytes[] memory registrationVaas) public {
        _registerChains(tokenBridge, registrationVaas);
    }
    // Register chains
    // forge script RegisterChainsTokenBridge --sig "run(address,bytes[])" --rpc-url $RPC --private-key $RAW_PRIVATE_KEY --broadcast
    function run(address tokenBridge, bytes[] memory registrationVaas) public {
        vm.startBroadcast();
        _registerChains(tokenBridge, registrationVaas);
        vm.stopBroadcast();
    }

    function _registerChains(address tokenBridge, bytes[] memory registrationVaas) internal {
        ITokenBridge tokenBridgeContract = ITokenBridge(tokenBridge);
        uint256 len = registrationVaas.length;
        for(uint256 i=0; i<len; i++) {
            tokenBridgeContract.registerChain(registrationVaas[i]);
        }
    }
}