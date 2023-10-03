// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
import {NFTBridgeImplementation} from "../contracts/nft/NFTBridgeImplementation.sol";
import {NFTBridgeSetup} from "../contracts/nft/NFTBridgeSetup.sol";
import {NFTImplementation} from "../contracts/nft/token/NFTImplementation.sol";
import {NFTBridgeEntrypoint} from "../contracts/nft/NFTBridgeEntrypoint.sol";
import "forge-std/Script.sol";

contract DeployNFTBridge is Script {
    // DryRun - Deploy the system
    // dry run: forge script ./forge-scripts/DeployNFTBridge.s.sol:DeployNFTBridge --sig "dryRun()" --rpc-url $RPC
    function dryRun() public {
        _deploy();
    }

    // Deploy the system
    // deploy:  forge script ./forge-scripts/DeployNFTBridge.s.sol:DeployNFTBridge --sig "run()" --rpc-url $RPC --etherscan-api-key $ETHERSCAN_API_KEY --private-key $RAW_PRIVATE_KEY --broadcast --verify
    function run()
        public
        returns (
            address deployedAddress,
            address nftImplementationAddress,
            address setupAddress,
            address implementationAddress
        )
    {
        vm.startBroadcast();
        (
            deployedAddress,
            nftImplementationAddress,
            setupAddress,
            implementationAddress
        ) = _deploy();
        vm.stopBroadcast();
    }

    function _deploy()
        internal
        returns (
            address deployedAddress,
            address nftImplementationAddress,
            address setupAddress,
            address implementationAddress
        )
    {
        NFTImplementation nftImpl = new NFTImplementation();
        NFTBridgeSetup nftBridgeSetup = new NFTBridgeSetup();
        NFTBridgeImplementation nftBridgeImpl = new NFTBridgeImplementation();

        uint16 chainId = uint16(vm.envUint("BRIDGE_INIT_CHAIN_ID"));
        uint16 governanceChainId = uint16(
            vm.envUint("BRIDGE_INIT_GOV_CHAIN_ID")
        );
        bytes32 governanceContract = bytes32(
            vm.envBytes32("BRIDGE_INIT_GOV_CONTRACT")
        );
        uint8 finality = uint8(vm.envUint("BRIDGE_INIT_FINALITY"));
        uint256 evmChainId = vm.envUint("INIT_EVM_CHAIN_ID");

        address wormhole = vm.envAddress("WORMHOLE_ADDRESS");

        bytes memory setupAbi = abi.encodeCall(
            NFTBridgeSetup.setup,
            (
                address(nftBridgeImpl),
                chainId,
                wormhole,
                governanceChainId,
                governanceContract,
                address(nftImpl),
                finality,
                evmChainId
            )
        );

        NFTBridgeEntrypoint nftBridge = new NFTBridgeEntrypoint(
            address(nftBridgeSetup),
            setupAbi
        );

        return (
            address(nftBridge),
            address(nftImpl),
            address(nftBridgeSetup),
            address(nftBridgeImpl)
        );
    }
}
