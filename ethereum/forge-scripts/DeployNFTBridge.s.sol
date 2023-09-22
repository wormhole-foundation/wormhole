

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
import {NFTBridgeImplementation} from "../contracts/nft/NFTBridgeImplementation.sol";
import {NFTBridgeSetup} from "../contracts/nft/NFTBridgeSetup.sol";
import {NFTImplementation} from "../contracts/nft/token/NFTImplementation.sol";
import "forge-std/Script.sol";

contract DeployNFTBridge is Script {
    // DryRun - Deploy the system
    // dry run: forge script DeployNFTBridge --sig "dryRun()" --rpc-url $RPC
    function dryRun() public {
        _deploy();
    }
    // Deploy the system
    // deploy:  forge script DeployNFTBridge --sig "run()" --rpc-url $RPC --etherscan-api-key $ETHERSCAN_API_KEY --private-key $RAW_PRIVATE_KEY --broadcast --verify
    function run() public returns (address deployedAddress) {
        vm.startBroadcast();
        deployedAddress = _deploy();
        vm.stopBroadcast();
    }

    function _deploy() internal returns (address deployedAddress) {
        NFTBridgeImplementation nftBridgeImpl = new NFTBridgeImplementation();
        NFTImplementation nftImpl = new NFTImplementation();
        NFTBridgeSetup nftBridgeSetup = new NFTBridgeSetup();

        uint16 chainId = uint16(vm.envUint("BRIDGE_INIT_CHAIN_ID"));
        uint16 governanceChainId = uint16(vm.envUint("BRIDGE_INIT_GOV_CHAIN_ID"));
        bytes32 governanceContract = bytes32(vm.envBytes32("BRIDGE_INIT_GOV_CONTRACT"));
        uint8 finality = uint8(vm.envUint("BRIDGE_INIT_FINALITY"));
        uint256 evmChainId = vm.envUint("INIT_EVM_CHAIN_ID");

        address wormhole = vm.envAddress("WORMHOLE_ADDRESS");

        nftBridgeSetup.setup(
            address(nftBridgeImpl),
            chainId,
            wormhole,
            governanceChainId,
            governanceContract,
            address(nftImpl),
            finality,
            evmChainId
        );

        return address(nftBridgeSetup);

        // TODO: initialize?
    }
}