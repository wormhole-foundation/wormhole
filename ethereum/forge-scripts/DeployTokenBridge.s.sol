// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {BridgeImplementation} from "../contracts/bridge/BridgeImplementation.sol";
import {BridgeSetup} from "../contracts/bridge/BridgeSetup.sol";
import {TokenImplementation} from "../contracts/bridge/token/TokenImplementation.sol";
import {TokenBridge} from "../contracts/bridge/TokenBridge.sol";
import "forge-std/Script.sol";

contract DeployTokenBridge is Script {
    TokenImplementation tokenImpl;
    BridgeSetup bridgeSetup;
    BridgeImplementation bridgeImpl;

    function dryRun(
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        address weth,
        uint8 finality,
        uint256 evmChainId,
        address wormhole
    ) public {
        _deploy(
            chainId,
            governanceChainId,
            governanceContract,
            weth,
            finality,
            evmChainId,
            wormhole
        );
    }

    function run(
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        address weth,
        uint8 finality,
        uint256 evmChainId,
        address wormhole
    )
        public
        returns (
            address deployedAddress,
            address tokenImplementationAddress,
            address bridgeSetupAddress,
            address bridgeImplementationAddress
        )
    {
        vm.startBroadcast();
        (
            deployedAddress,
            tokenImplementationAddress,
            bridgeSetupAddress,
            bridgeImplementationAddress
        ) = _deploy(
            chainId,
            governanceChainId,
            governanceContract,
            weth,
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
        address weth,
        uint8 finality,
        uint256 evmChainId,
        address wormhole
    )
        internal
        returns (
            address deployedAddress,
            address tokenImplementationAddress,
            address bridgeSetupAddress,
            address bridgeImplementationAddress
        )
    {
        tokenImpl = new TokenImplementation();
        bridgeSetup = new BridgeSetup();
        bridgeImpl = new BridgeImplementation();

        TokenBridge tokenBridge = new TokenBridge(
            address(bridgeSetup),
            abi.encodeWithSignature(
                "setup(address,uint16,address,uint16,bytes32,address,address,uint8,uint256)",
                address(bridgeImpl),
                chainId,
                wormhole,
                governanceChainId,
                governanceContract,
                address(tokenImpl),
                weth,
                finality,
                evmChainId
            )
        );

        return (
            address(tokenBridge),
            address(tokenImpl),
            address(bridgeSetup),
            address(bridgeImpl)
        );
    }
}
