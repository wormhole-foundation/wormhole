// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {NFTBridgeImplementation} from "../contracts/nft/NFTBridgeImplementation.sol";
import {NFTBridgeSetup} from "../contracts/nft/NFTBridgeSetup.sol";
import {NFTImplementation} from "../contracts/nft/token/NFTImplementation.sol";
import {NFTBridgeEntrypoint} from "../contracts/nft/NFTBridgeEntrypoint.sol";
import "forge-std/Script.sol";

contract DeployNFTBridge is Script {
    NFTImplementation nftImpl;
    NFTBridgeSetup nftBridgeSetup;
    NFTBridgeImplementation nftBridgeImpl;

    function dryRun(
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        uint8 finality,
        uint256 evmChainId,
        address wormhole
    ) public {
        _deploy(
            chainId,
            governanceChainId,
            governanceContract,
            finality,
            evmChainId,
            wormhole
        );
    }

    function run(
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        uint8 finality,
        uint256 evmChainId,
        address wormhole
    )
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
        ) = _deploy(
            chainId,
            governanceChainId,
            governanceContract,
            finality,
            evmChainId,
            wormhole
        );
        vm.stopBroadcast();
    }

    function _deploy(
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        uint8 finality,
        uint256 evmChainId,
        address wormhole
    )
        internal
        returns (
            address deployedAddress,
            address nftImplementationAddress,
            address setupAddress,
            address implementationAddress
        )
    {
        nftImpl = new NFTImplementation();
        nftBridgeSetup = new NFTBridgeSetup();
        nftBridgeImpl = new NFTBridgeImplementation();

        NFTBridgeEntrypoint nftBridge = new NFTBridgeEntrypoint(
            address(nftBridgeSetup),
            abi.encodeWithSignature(
                "setup(address,uint16,address,uint16,bytes32,address,uint8,uint256)",
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

        return (
            address(nftBridge),
            address(nftImpl),
            address(nftBridgeSetup),
            address(nftBridgeImpl)
        );
    }
}
