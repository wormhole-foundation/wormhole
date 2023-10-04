// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
import {NFTBridgeImplementation} from "../contracts/nft/NFTBridgeImplementation.sol";
import {NFTBridgeSetup} from "../contracts/nft/NFTBridgeSetup.sol";
import {NFTImplementation} from "../contracts/nft/token/NFTImplementation.sol";
import {NFTBridgeEntrypoint} from "../contracts/nft/NFTBridgeEntrypoint.sol";
import "forge-std/Script.sol";

contract DeployNFTBridge is Script {
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
        NFTImplementation nftImpl = new NFTImplementation();
        NFTBridgeSetup nftBridgeSetup = new NFTBridgeSetup();
        NFTBridgeImplementation nftBridgeImpl = new NFTBridgeImplementation();

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
