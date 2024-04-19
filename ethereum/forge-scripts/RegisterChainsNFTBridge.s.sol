

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {INFTBridge} from "../contracts/nft/interfaces/INFTBridge.sol";
import "forge-std/Script.sol";

contract RegisterChainsNFTBridge is Script {
    // DryRun - Register chains
    // dry run: forge script RegisterChainsNFTBridge --sig "dryRun(address,bytes[])" --rpc-url $RPC
    function dryRun(address nftBridge, bytes[] memory registrationVaas) public {
        _registerChains(nftBridge, registrationVaas);
    }
    // Register chains
    // forge script RegisterChainsNFTBridge --sig "run(address,bytes[])" --rpc-url $RPC --private-key $RAW_PRIVATE_KEY --broadcast
    function run(address nftBridge, bytes[] memory registrationVaas) public {
        vm.startBroadcast();
        _registerChains(nftBridge, registrationVaas);
        vm.stopBroadcast();
    }

    function _registerChains(address nftBridge, bytes[] memory registrationVaas) internal {
        INFTBridge nftBridgeContract = INFTBridge(nftBridge);
        uint256 len = registrationVaas.length;
        for(uint256 i=0; i<len; i++) {
            nftBridgeContract.registerChain(registrationVaas[i]);
        }
    }
}